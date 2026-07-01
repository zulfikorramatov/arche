# Style Guide

This document describes the coding conventions used throughout the project.
When in doubt, follow the existing code — consistency trumps personal preference.

## Naming

### Packages

- Single lowercase word: `domain`, `service`, `handler`, `postgres`.
- Never plural (`utils`, `helpers`, `models`) — find a specific name.

### Constructors and types

- Constructors: `NewXxx` (e.g. `NewUserService`, `NewRouter`).
- Exported types: PascalCase. Unexported: camelCase.
- Avoid stuttering: `service.UserService` is fine, `service.ServiceUser` is not.

### Tags

| Context | Convention | Example |
|---------|-----------|---------|
| JSON | `snake_case` | `json:"created_at"` |
| Env | `UPPER_SNAKE_CASE` | `env:"POSTGRES_HOST"` |
| DB column | `snake_case` | positional `$1` in queries |

### Imports

Group in three blocks separated by blank lines:

```go
import (
    "context"        // 1. stdlib
    "fmt"

    "github.com/go-chi/chi/v5"  // 2. external
    "github.com/google/uuid"

    "github.com/zulfikorramatov/arche/internal/domain"  // 3. internal
)
```

`goimports` with `local-prefixes: github.com/zulfikorramatov/arche` handles this
automatically (configured in `.golangci.yml`).

## Interfaces

**Consumer-defined.** The package that _uses_ a dependency declares its own
interface describing what it needs — not what the implementation offers.

```go
// internal/http/handler/server.go
type userService interface {
    List(ctx context.Context) ([]domain.User, error)
}
```

Rules:
- **Unexported** — only visible within the declaring package.
- **Named by role** — `userService`, `userRepository`, not `UserServiceInterface`.
- **Declared in the file that uses them** — not in a shared `interfaces.go`.
- **Concrete types wired in `app.go`** — manual DI, no framework.

This enables testing each layer with a minimal fake.

## Errors

### Wrapping

Always lowercase, describing the action that failed:

```go
return fmt.Errorf("insert user: %w", err)
```

Never start with uppercase. Never include the function name if it's obvious
from context.

### Domain sentinels

Expected (handleable) errors live in `internal/domain`:

```go
var ErrUserNotFound = errors.New("user not found")
var ErrUserExists   = errors.New("user already exists")
```

### Boundary mapping

- **Repository → domain:** map infrastructure errors to domain sentinels.
  Example: PostgreSQL unique violation `23505` → `domain.ErrUserExists`.
- **Handler → HTTP:** map domain sentinels to response types.
  Example: `domain.ErrUserNotFound` → `api.DeleteUser404JSONResponse`.

### Comparison

Always `errors.Is` or `errors.As` — never `==`.

### Discardable errors

When an error is intentionally ignored (cache writes, log sync), assign to `_`
explicitly:

```go
_ = s.cache.Del(ctx, key).Err()
```

## SQL

- **Raw pgx** — no ORM, no query builder.
- Queries are declared as `const q` inside the method:

```go
func (r *UserRepository) List(ctx context.Context) ([]domain.User, error) {
    const q = `SELECT id, username, password, created_at FROM users`
    rows, err := r.pool.Query(ctx, q)
    // ...
}
```

- **Positional placeholders**: `$1`, `$2`, etc.
- **Uppercase SQL keywords**: `SELECT`, `INSERT INTO`, `WHERE`, `DELETE FROM`.
- Keep queries single-purpose; combine at the service layer if needed.

## Testing

- **Table-driven** with `t.Run` subtests.
- Naming: `TestType_Method_scenario` (e.g. `TestUserService_Create_duplicate`).
- Race detector always on: `go test -race -count=1 ./...`
- Fakes over mocking frameworks — implement the consumer interface by hand.
- No test helpers in `pkg/` — keep tests self-contained.

```go
func TestUserService_List_empty(t *testing.T) {
    repo := &fakeUserRepo{users: nil}
    svc := service.NewUserService(repo)

    got, err := svc.List(context.Background())

    require.NoError(t, err)
    assert.Empty(t, got)
}
```

## Comments

**Default: none.** Names should carry meaning.

Exceptions:
- Package-level doc comments on `pkg/*` packages (they're libraries).
- Notes on genuinely non-obvious decisions or tradeoffs.
- `// TODO:` for known technical debt (rare).

Never:
- Per-function doc comments on self-evident code.
- Ticket/PR references in code (use git history).
- Commented-out code (delete it; git remembers).

## `pkg/*` vs `internal/*`

| | `pkg/*` | `internal/*` |
|---|---------|-------------|
| Knowledge of app | Zero | Full |
| Config | Plain struct, no tags | Env-tagged struct |
| Dependencies | Only stdlib + its own deps | Anything |
| Reusability | Copy-pasteable to another project | Project-specific |

The bridge is `internal/app/app.go` which does explicit struct conversion:

```go
pg, err := postgres.New(ctx, postgres.Config(cfg.Postgres))
```

If fields drift between the two sides, the build breaks — this is intentional.

## HTTP layer conventions

- **Spec-first**: the OpenAPI spec is the source of truth for endpoints.
- **Never register routes manually** in `router.go` — they come from generated code.
- Handlers implement `api.StrictServerInterface` — they receive typed request
  objects and return typed response objects. No `http.ResponseWriter`, no
  `json.Decode`.
- One `Server` struct in `handler/` implements all operations. Different domains
  get separate files with methods on `*Server`.
- Map domain errors → response types; let unknown errors bubble up (strict
  adapter returns 500).

## Middleware

Middleware lives in `internal/http/middleware/`. Order in the router matters:

1. `RequestID` — assigns a unique ID to each request.
2. `Logger` — structured request logging (method, path, status, duration).
3. `ErrorHandler` — normalizes non-JSON error responses to `{"error": "..."}`.
4. `BasicAuth` — authenticates via `userAuthenticator` interface, puts `AuthUser` in context.

## Configuration

- All runtime config from environment variables (cleanenv + godotenv).
- `.env.example` is the single source of truth for variable names.
- Defaults live in `env-default` tags in `internal/config/config.go`.
- In containers `.env` is absent — runtime supplies vars directly.
- Never read `os.Getenv` directly outside `internal/config`.
