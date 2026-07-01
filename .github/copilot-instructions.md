# Copilot Instructions

This is a **Go microservice template** (chi + slog + pgx + go-redis + franz-go).
It ships a single example domain (`user`) wired end-to-end. New features should
mirror the `user` vertical slice.

## Architecture (dependency direction: outside → inside)

```
cmd/app/main.go         → internal/app (composition root)
  → internal/http         router + middleware + handler (StrictServerInterface)
    → internal/service    business logic
      → internal/repository  raw pgx SQL
        → internal/domain    entities + sentinel errors
```

Wiring is manual in `internal/app/app.go` — no DI framework.

## Spec-first HTTP workflow

The HTTP contract is **generated, not hand-written**.

1. Edit `generated/api/api.yaml` (paths) and/or `definitions.yaml` (schemas).
2. Run `make generate` — produces `gen.go` + `definitions.go`.
3. The build breaks until you implement the new `StrictServerInterface` method.
4. Implement the handler in `internal/http/handler/`.
5. Routes register automatically — **never add routes to `router.go` manually**.

Key types: `*RequestObject` (input), `*ResponseObject` (output). Handlers never
touch `http.ResponseWriter` or do manual JSON encoding.

## Key conventions

- **Interfaces are consumer-defined**: unexported, named by role (`userService`,
  `userRepository`), declared in the file that uses them.
- **Errors**: wrap with lowercase context (`fmt.Errorf("insert user: %w", err)`);
  domain sentinels for expected cases; compare with `errors.Is`.
- **SQL**: raw pgx, `const q` inside the method, `$1` positional placeholders.
- **No comments on self-evident code**. Only package docs on `pkg/*` and notes on
  non-obvious decisions.
- **`pkg/*`** are self-contained libraries with plain `Config` (no env tags).
  **`internal/config`** owns all env tags. `app.go` bridges via struct conversion.

## Do

- Follow the existing vertical slice pattern (domain → repo → service → handler).
- Use `errors.Is` for sentinel comparison.
- Return typed `*ResponseObject` variants from handlers (e.g. `ListUsers200JSONResponse`).
- Declare SQL as `const q` inside the repository method.
- Use table-driven tests with `t.Run`, named `TestType_Method_scenario`.
- Keep middleware in `internal/http/middleware/`.

## Don't

- Don't edit `generated/api/gen.go` or `definitions.go` — they're regenerated.
- Don't register routes manually in `router.go`.
- Don't put env tags in `pkg/*` config structs.
- Don't use an ORM or query builder — raw pgx only.
- Don't use a mocking framework — write minimal fakes implementing consumer interfaces.
- Don't add per-function doc comments on obvious code.
- Don't use `os.Getenv` outside `internal/config`.

## Commands

```sh
make generate    # regenerate from OpenAPI spec
make run         # run locally (loads .env)
make test        # go test -race -count=1 ./...
make lint        # golangci-lint
make build       # → bin/arche
make build-cli   # → bin/arche-cli
make up          # docker-compose infra (postgres + redis + redpanda + migrate)
```

## Full style reference

See [STYLEGUIDE.md](../STYLEGUIDE.md) for complete coding conventions.
