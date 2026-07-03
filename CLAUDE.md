# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

`arche` is a Go 1.25 service template: a chi-based HTTP API generated from an OpenAPI spec, backed by Postgres, with self-contained `pkg/` infrastructure wrappers (Redis, Kafka/Redpanda, money) and Elastic APM tracing. It ships with two binaries — the HTTP server (`cmd/app`) and an admin CLI (`cmd/cli`).

## Commands

All common tasks go through the Makefile (`make help` lists them):

- `make run` — run the HTTP server locally (loads `.env` via `godotenv`)
- `make build` / `make build-cli` — build `bin/arche` and `bin/arche-cli`
- `make test` — `go test -race -count=1 ./...` (run a single test: `go test -race -run TestName ./internal/...`)
- `make lint` — `golangci-lint run ./...`
- `make fmt` — `gofmt -s -w .` + `go vet ./...`
- `make generate` — regenerate API code from OpenAPI (see below); **run after editing any `.yaml` under `generated/api/`**
- `make up` / `make down` / `make logs` — docker-compose stack (Postgres, Redis, Redpanda, one-shot migrate)
- `make migrate-up` / `make migrate-down` / `make migrate-new name=add_xxx` — golang-migrate against `POSTGRES_DSN`

Local dev flow: `make up` (starts deps + applies migrations) then `make run`. Config comes entirely from env vars (`.env`, see `.env.example`); `config.Load()` reads them via `cleanenv` struct tags in `internal/config/config.go`.

## Architecture

### Layered request flow

Wiring happens in `internal/app/app.go` (`Run`), which constructs dependencies bottom-up and injects them: `postgres.Pool` → `repository.UserRepository` → `service.UserService` → `handler.Server` → router. Layers depend on **narrow interfaces defined at the consumer**, not on concrete types — e.g. `service.userRepository`, `handler.userService`, and `http.userAuthenticator` are each declared in the package that uses them. Add new features by following this chain: entity → repository (raw SQL via pgx) → service (business logic) → handler.

### OpenAPI is the source of truth for the HTTP layer

The HTTP contract lives in `generated/api/*.yaml`, **not** in hand-written Go:

- `generated/api/api.yaml` — paths/operations; `definitions.yaml` — schemas. Editing these requires `make generate`.
- `make generate` runs `oapi-codegen` twice (configs `definitions-cfg.yaml` then `cfg.yaml`), producing `definitions.go` and `gen.go`. **These generated files are committed; never edit them by hand** (golangci-lint also skips `generated/api`).
- Handlers implement `api.StrictServerInterface` (`internal/http/handler/server.go`, guarded by `var _ api.StrictServerInterface = (*Server)(nil)`). Adding an endpoint = add it to the spec, regenerate, then implement the new method.
- `internal/http/router.go` wires the strict handler through chi with middleware order: **request validation → RequestID → logging → error handling**. Requests are validated against the embedded spec (`api.GetSpec()`) by `nethttp-middleware` before reaching handlers.

### Authentication

The API uses HTTP Basic auth declared in the OpenAPI `securitySchemes`. Enforcement is wired into the **OpenAPI validator middleware** (`internal/http/middleware/validator.go`), whose `AuthenticationFunc` calls `UserService.Authenticate` (bcrypt compare). There is no separate auth middleware — auth is a side effect of spec validation.

### `pkg/` is deliberately self-contained infrastructure

`pkg/postgres`, `pkg/redis`, `pkg/kafka`, `pkg/money`, `pkg/logger` know nothing about `internal/*`, env vars, or `.env`. Each exposes a plain `Config` struct that the caller fills. The `internal/config` structs mirror these one-to-one so `app.go` can convert directly, e.g. `postgres.Config(cfg.Postgres)` / `kafka.Config(cfg.Kafka)` — **if the fields drift, the project won't compile**, which is the intended guard against config drift. Preserve this: keep `pkg/*` free of `internal` imports and keep the mirror structs in sync.

- **`pkg/kafka`** (franz-go over Redpanda): producer is synchronous with `acks=all` + idempotency; consumer is manual-commit at-least-once (commit only after successful handling). See `pkg/kafka/README.md` (in Russian) for the delivery-guarantee contract and fail-fast/K8s semantics. Handlers must be idempotent. Note: Kafka config exists but is not yet wired into `app.go`.
- **`pkg/money`** wraps `bojanz/currency`; default currency is `UZS`, with `FromSums`/`FromTiyins` constructors and SQL `driver.Valuer`/JSON marshalling.
- **`pkg/postgres`** aliases `pgxpool.Pool` and instruments queries with Elastic APM (`apmpgxv5`).

### CLI (`cmd/cli`)

Cobra-based admin tool. `PersistentPreRunE` in `cmd/cli/main.go` loads config and opens a Postgres pool into a shared `deps.Deps` struct, closed in `PersistentPostRun`. Commands (e.g. `user create`/`user delete`) run **raw SQL directly against `deps.Pool`** — they do not go through the repository/service layers. Passwords are bcrypt-hashed here.

## Conventions

- Wrap every returned error with context: `fmt.Errorf("doing x: %w", err)`. This is used throughout and expected by the `nilerr`/`errcheck` linters.
- Lint config (`.golangci.yml`) enables `govet` with `enable-all` (only `fieldalignment` disabled), plus `errcheck`, `staticcheck`, `errname`, `nilerr`, `unparam`, etc. Run `make lint` before finishing.
- goimports local-prefix is `github.com/zulfikorramatov/arche` — imports are grouped stdlib / third-party / local.
- Migrations are paired `.up.sql`/`.down.sql` files under `migrations/`, sequentially numbered; create them with `make migrate-new`.
