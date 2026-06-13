# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this repo is

A Go 1.25 microservice **template**, not a finished service. The sample `user`
domain exists to demonstrate the layout — when this template is cloned for a
new service it should be replaced. Cloning steps live in `README.md` under
"Cloning for a new service".

## Common commands

| Task                  | Command                                          |
| --------------------- | ------------------------------------------------ |
| Tidy modules          | `make tidy`                                      |
| Build binary          | `make build`                                     |
| Run locally           | `make run` (reads `.env` via godotenv)           |
| Tests with race       | `make test`                                      |
| Single test           | `go test -race -run TestName ./internal/service` |
| Lint                  | `make lint` (requires `golangci-lint`)           |
| Format + vet          | `make fmt`                                       |
| Start full stack      | `make up`                                        |
| Tail app logs         | `make logs`                                      |
| Apply migrations      | `make migrate-up`                                |
| New migration         | `make migrate-new name=add_xxx`                  |

Local dev without containers: `docker compose up -d postgres redis && make migrate-up && make run`.

## Architecture: composition root + tag-free pkg

The non-obvious shape of this repo is the **strict separation** between
env-aware config and reusable libraries:

- `internal/config/config.go` is the **only** place env-var names exist
  (struct tags `env:"..."`). It loads `.env` (optional) and the process
  environment via `cleanenv` + `godotenv`.
- `pkg/{logger,postgres,redis}` each expose a `Config` struct with **plain Go
  fields and no tags**. They have zero knowledge of `.env`, `cleanenv`, or
  `internal/*`. Each is meant to be copy-pasted into another service unchanged.
- `internal/app/app.go` is the composition root and the **only** place these
  two worlds meet. It maps env-config → pkg-config:

  ```go
  log, _ := logger.New(logger.Config{
      AppName: cfg.App.Name,           // sourced from cfg.App, not cfg.Logger
      Level:   cfg.Logger.Level,
      Format:  cfg.Logger.Format,
  })
  pg,  _ := postgres.New(ctx, postgres.Config(cfg.Postgres)) // struct conversion
  rdb, _ := redis.New(ctx, redis.Config(cfg.Redis))          // struct conversion
  ```

### Invariants that protect this design

- **`pkg/*` must never import from `internal/*`** and must never reference
  env vars, `.env`, or `cleanenv`. If you need a new config knob in a `/pkg`
  library, add a plain field; add the env tag on the matching
  `internal/config.*Config` sub-struct.
- **Field layout of `config.PostgresConfig` and `pkg/postgres.Config` must
  match exactly** (names, types, order). Same for redis. The explicit
  `postgres.Config(cfg.Postgres)` conversion fails to compile on drift —
  this is the intended safety net. Don't refactor it into a reflective copy.
- **Logger is the exception**: `pkg/logger.Config` has an `AppName` field
  that has no counterpart on `config.LoggerConfig`. It's sourced from
  `cfg.App.Name` in `app.go`. That's why logger uses field-by-field
  construction instead of struct conversion.

### Reserved fields (currently unused — don't remove)

- `pkg/logger.Config.AppName` — reserved for future logger wiring; populated
  but not yet consumed by `logger.New`.
- `config.AppConfig.Env` — read from `APP_ENV`, not currently wired anywhere.

### Request-flow layering

`cmd/app/main.go` → `internal/app.Run` → `internal/http.NewRouter` → handler
→ service (interface) → repository. The handler depends on a `userService`
interface defined in the handler package; the service depends on a
`userRepository` interface defined in the service package. The concrete
`UserRepository` lives in `internal/repository/`. This direction (interfaces
declared by the consumer) is intentional — keep it.

### Re-exported types

- `pkg/postgres.Pool` is a type alias for `*pgxpool.Pool`.
- `pkg/redis.Client` is a type alias for `*goredis.Client`; `pkg/redis.Nil`
  mirrors `goredis.Nil`.

This lets `internal/*` consumers depend only on `pkg/*` and not import
pgx/go-redis directly. Preserve these aliases when extending.

## Docker / env

`.env` defaults hostnames to `localhost` so `make run` works on the host.
`docker-compose.yml` overrides `POSTGRES_HOST=postgres` and `REDIS_HOST=redis`
via the app service's `environment:` block (which takes precedence over
`env_file`). Don't move these overrides back into `.env` — that would break
`make run` on the host.

The `migrate` service in compose is a one-shot runner that the `app` service
depends on with `condition: service_completed_successfully`.
