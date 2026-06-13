# arche

Template for Go microservices. Uses [chi](https://github.com/go-chi/chi),
[zap](https://github.com/uber-go/zap), [pgx](https://github.com/jackc/pgx),
and [go-redis](https://github.com/redis/go-redis), wired together behind a
config-driven `internal/app`.

## Layout

```
.
├── cmd/app/            # main.go — load config, wire signal context, call app.Run
├── internal/
│   ├── app/            # composition root: build deps, start HTTP server
│   ├── config/         # aggregated Config (HTTP + pkg configs)
│   ├── domain/         # entities and domain errors
│   ├── repository/     # postgres-backed repositories
│   ├── service/        # business logic, uses repo + cache
│   └── http/           # router, handlers, middleware
├── pkg/                # reusable, independent libraries — plain Config structs, no env tags
│   ├── logger/         #   zap wrapper
│   ├── postgres/       #   pgx pool
│   └── redis/          #   go-redis client
├── migrations/         # golang-migrate SQL files
├── .env.example        # single source of runtime config
├── docker-compose.yml  # app + postgres + redis + migrate
├── Dockerfile          # multi-stage build, Go 1.25
└── Makefile
```

## Quickstart

```sh
cp .env.example .env
make up           # build & start app + postgres + redis + run migrations
curl localhost:8080/healthz
```

Local dev without containers:

```sh
make tidy
docker compose up -d postgres redis
make migrate-up
make run          # loads .env from the project root
```

## Config

All runtime config comes from environment variables. `internal/config` reads
`.env` (if present) plus the process environment into a `Config` struct
whose sub-structs own the env tags. The `/pkg/*` libraries are independent
and use plain `Config` structs with no tags; `internal/app/app.go` maps
between them with explicit struct conversion:

```go
log, _  := logger.New(logger.Config(cfg.Logger))
pg,  _  := postgres.New(ctx, postgres.Config(cfg.Postgres))
rdb, _  := redis.New(ctx, redis.Config(cfg.Redis))
```

If a field is added on one side but not the other, this stops compiling.

## API

| Method | Path                | Description     |
| ------ | ------------------- | --------------- |
| GET    | `/healthz`          | liveness probe  |
| POST   | `/api/v1/users`     | create a user   |
| GET    | `/api/v1/users/{id}`| fetch by id     |

## Cloning for a new service

1. `git clone` and `rm -rf .git && git init`
2. Open the project in GoLand, open `go.mod`, right-click the module path on
   the `module` line → **Refactor → Rename** → enter the new path. GoLand
   rewrites every import across the repo in one pass.
3. Rename containers and `APP_NAME` in `docker-compose.yml` / `.env`
4. Replace the `user` domain/service/repository/handler with yours
5. `make tidy && make up`
