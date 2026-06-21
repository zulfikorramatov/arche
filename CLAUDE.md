# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

`arche` is a **template** for Go microservices (chi + zap + pgx + go-redis). It ships a single example domain (`user`) wired end to end so it can be cloned and reshaped per service. README.md has the clone-and-rename checklist. When adding a feature, mirror the `user` vertical slice rather than inventing new layering.

## Commands

```sh
make generate                  # regenerate generated/api/*.go from the OpenAPI spec
make run                       # run locally, loads .env from project root
make test                      # go test -race -count=1 ./...
make lint                      # golangci-lint run ./...
make fmt                       # gofmt -s -w . && go vet ./...
make build                     # -> bin/arche
make up / make down / make logs   # docker-compose stack (app + postgres + redis + migrate)
make migrate-up / migrate-down
make migrate-new name=add_xxx  # scaffold a new SQL migration pair

go test -race -run TestUserService_Create ./internal/service   # single test
```

Local dev without containerizing the app: `docker compose up -d postgres redis` → `make migrate-up` → `make run`. Requires Go 1.25+, golangci-lint 2.x, golang-migrate 4.x.

## Architecture

Request flow and dependency direction:

```
generated/api/*.yaml    OpenAPI spec — the single source of truth for the HTTP contract
  → generated/api/*.go  oapi-codegen output (strict server, models, embedded spec) — DO NOT EDIT
cmd/app/main.go         config.Load + signal.NotifyContext, calls app.Run
  → internal/app        composition root: builds every dependency, owns lifecycle
      → internal/http   chi router: strict adapter + OpenAPI validator middleware
          → handler     implements api.StrictServerInterface; maps domain <-> generated types
              → service business logic, repo + redis cache
                  → repository  raw pgx SQL, maps pg errors to domain errors
                      → domain  entities + sentinel errors (ErrUserNotFound, ...)
```

Wiring is **manual** in `internal/app/app.go` (no DI framework) — construct repo → service → handler `Server` → router, then start `http.Server` with graceful shutdown driven by the signal context.

### Spec-first HTTP layer (the important convention)

The HTTP contract is **generated, not hand-written**. `generated/api/` holds two specs and two generator configs:
- `definitions.yaml` — reusable schemas/parameters; `x-go-type` maps OpenAPI types onto existing Go types (e.g. `UUID` → `uuid.UUID`). Generated into `definitions.go`.
- `api.yaml` — paths/operations; `$ref`s into `definitions.yaml`. `cfg.yaml` uses `import-mapping: { ./definitions.yaml: "-" }` so it reuses the `definitions.go` types instead of regenerating them. Generated into `gen.go`.
- `gen.go` / `definitions.go` are **regenerated, never edited** (run `make generate`).

`internal/http/handler` implements the generated `api.StrictServerInterface` (the `var _ api.StrictServerInterface = (*Server)(nil)` check enforces this at compile time). Strict handlers receive a typed `*RequestObject` (body already decoded, params already parsed) and return a typed `*ResponseObject` — they never touch `http.ResponseWriter` or `json`. They only translate between generated types and the service layer and map domain sentinel errors to the response variant for that status (e.g. `ErrUserExists` → `CreateUser409JSONResponse`).

`internal/http/router.go` wires it: chi router → `NewStrictHandlerWithOptions` (with request/response error funcs that emit JSON 400/500) → `HandlerWithOptions` with `BaseURL: "/api/v1"` and the `OapiRequestValidator` middleware. The validator uses the embedded spec to reject malformed requests (missing params, bad body, wrong content-type) **before** they reach a handler. Spec `Servers` stays set to `/api/v1` so the validator's router matches the prefixed paths; `SilenceServersWarning: true` quiets the (irrelevant, relative-URL) host-check warning. `/ping` is registered directly on the base router, outside the validator.

**To add or change an endpoint:** edit `api.yaml` (and `definitions.yaml` for new types) → `make generate` → the build breaks until you implement the new `StrictServerInterface` method in `internal/http/handler` → routes are registered automatically. Do not register routes by hand in `router.go`.

### `pkg/*` vs `internal/*` (the central convention)

- `pkg/{logger,postgres,redis}` are **self-contained libraries**: zero knowledge of `internal/*`, env vars, or app config. They expose a plain `Config` struct with **no tags** and must be copy-pasteable to another project unchanged.
- `internal/config/config.go` is the **only** place env tags live. Its sub-structs (`PostgresConfig`, `RedisConfig`, `LoggerConfig`) mirror the matching `pkg/*.Config` field-for-field (names, types, order).
- `app.go` bridges them with explicit struct conversion: `postgres.New(ctx, postgres.Config(cfg.Postgres))`. If a field drifts between the two sides, **the build breaks** — that is the intended safety net, so keep the structs in lockstep.

### Config

All runtime config comes from env (cleanenv + optional `.env` via godotenv). `.env.example` is the single source of truth for variable names; defaults live in `env-default` tags. In containers `.env` is absent and the runtime supplies vars directly — both paths work.

## Conventions (see STYLEGUIDE.md for the full list)

- **Interfaces are consumer-defined**: declared unexported in the file that uses them, named by role (`userService`, `userRepository`), never in a shared `interfaces.go`. Mock via these, not a framework.
- **Errors**: wrap with lowercase context (`fmt.Errorf("insert user: %w", err)`); domain sentinels (`ErrXxx`) for expected cases; compare with `errors.Is/As`; map infra errors → domain errors at the repository boundary (e.g. pg `23505` → `domain.ErrUserExists`). Discardable errors (cache writes, `log.Sync`) use `_ =`.
- **SQL**: raw pgx, no ORM. Queries are `const q` inside the method, positional `$1` placeholders.
- **Comments**: default to none — names carry meaning. Only package docs on `pkg/*` and notes on genuinely non-obvious decisions. No per-function doc comments on self-evident code; no ticket/PR references.
- **Naming**: package names are single lowercase words; constructors `NewXxx`; JSON tags snake_case; env tags UPPER_SNAKE_CASE.
- **Tests**: table-driven with `t.Run` subtests, named `TestType_Method_scenario`, race detector always on.
