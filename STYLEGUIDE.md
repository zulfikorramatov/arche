# Go Style Guide

This document codifies the coding conventions used in this repository.
It serves as a reference for contributors and AI-assisted development tools.

## Packages

- **Names**: single lowercase word — `handler`, `service`, `repository`, `domain`.
  Never `userHandler` or `user_service`.
- **`pkg/*`** packages are self-contained libraries with zero knowledge of
  `internal/*`, env vars, or the application config.
  They must be copy-pasteable to another project without modification.
- **`internal/*`** packages hold application-specific logic.

## Naming

| Kind              | Convention          | Example                          |
| ----------------- | ------------------- | -------------------------------- |
| Constructors      | `NewXxx`            | `NewUserHandler`, `New`          |
| Interfaces        | unexported, by role | `userService`, `userRepository`  |
| Sentinel errors   | `ErrXxxYyy`         | `ErrUserNotFound`, `ErrUserExists` |
| Constants         | CamelCase           | `FormatJSON`, `userCacheTTL`     |
| Struct fields      | PascalCase          | `CreatedAt`, `SSLMode`           |
| JSON tags         | snake_case          | `json:"created_at"`             |
| Env tags          | UPPER_SNAKE_CASE    | `env:"POSTGRES_HOST"`           |

## Interfaces

Interfaces are declared by the **consumer**, not the provider:

```go
// in handler/user.go — handler declares what it needs from the service layer
type userService interface {
    Create(ctx context.Context, email, name string) (*domain.User, error)
    GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
}
```

- Always unexported.
- Named after the dependency role, not the concrete type.
- Defined in the file that uses them, not in a separate `interfaces.go`.

## Imports

Three groups separated by blank lines:

```go
import (
    "context"
    "fmt"

    "github.com/go-chi/chi/v5"
    "go.uber.org/zap"

    "github.com/zulfikorramatov/arche/internal/domain"
)
```

1. Standard library
2. Third-party
3. Project-internal

**Aliases** only when there is a name conflict:

```go
httpserver    "github.com/zulfikorramatov/arche/internal/http"
chimiddleware "github.com/go-chi/chi/v5/middleware"
appmiddleware "github.com/zulfikorramatov/arche/internal/http/middleware"
goredis       "github.com/redis/go-redis/v9"
```

## Error handling

- Wrap with context: `fmt.Errorf("insert user: %w", err)` — lowercase, no
  trailing period.
- Use domain sentinel errors for expected cases:
  ```go
  var ErrUserNotFound = errors.New("user not found")
  ```
- Check with `errors.Is()` / `errors.As()`, never compare strings.
- Map infrastructure errors to domain errors at the boundary:
  ```go
  var pgErr *pgconn.PgError
  if errors.As(err, &pgErr) && pgErr.Code == "23505" {
      return domain.ErrUserExists
  }
  ```
- Non-critical errors (cache writes, logger sync) may be silently discarded:
  `_ = rdb.Set(...)`.

## SQL

- Queries as `const` inside the method that uses them:
  ```go
  func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
      const q = `
          SELECT id, email, name, created_at, updated_at
          FROM users
          WHERE id = $1
      `
      // ...
  }
  ```
- Positional placeholders: `$1`, `$2`, ... (pgx style).
- No ORM — raw SQL with `pgx`.

## HTTP handlers

- One struct per domain: `UserHandler`, `OrderHandler`, etc.
- Private helpers `writeJSON` / `writeError` for consistent responses.
- Validation inline; map domain errors to HTTP status codes in the handler:
  ```go
  if errors.Is(err, domain.ErrUserNotFound) {
      writeError(w, http.StatusNotFound, "user not found")
      return
  }
  ```
- Early return — happy path is not nested.

## Config

- Env tags live **only** in `internal/config/config.go`.
- `pkg/*` Config structs use plain fields — no tags.
- Field layout between `config.XxxConfig` and `pkg/xxx.Config` must match
  exactly (names, types, order) so `pkg.Config(cfg.Xxx)` compiles.
  Drift is caught at build time.

## Comments

- Default: **no comments**. Well-named identifiers are self-documenting.
- Exceptions: package-level doc comments on `pkg/*`, and comments explaining
  non-obvious design decisions or constraints.
- Never write per-function doc comments on self-evident code.
- Never reference tickets, PRs, or task descriptions in code comments.

## Resource lifecycle

- Create resources in `app.Run`, clean up with `defer`:
  ```go
  pg, err := postgres.New(ctx, postgres.Config(cfg.Postgres))
  if err != nil { return fmt.Errorf("new postgres: %w", err) }
  defer pg.Close()
  ```
- Graceful HTTP shutdown via `signal.NotifyContext` + `srv.Shutdown`.

## Tests

- Table-driven tests with `t.Run` subtests.
- Test naming: `TestTypeName_Method_scenario` — e.g.,
  `TestUserService_Create_duplicateEmail`.
- Race detector always on: `go test -race`.
- Mock dependencies via consumer-defined interfaces, not framework mocks.
