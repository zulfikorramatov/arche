# arche

Шаблон для Go-микросервисов. Использует [chi](https://github.com/go-chi/chi),
[zap](https://github.com/uber-go/zap), [pgx](https://github.com/jackc/pgx)
и [go-redis](https://github.com/redis/go-redis), связанные через
конфиг-ориентированный `internal/app`.

## Структура проекта

```
.
├── cmd/app/            # main.go — загрузка конфига, signal context, вызов app.Run
├── generated/api/      # spec-first HTTP контракт: OpenAPI spec + сгенерированный код
│   ├── definitions.yaml/.go  # общие schemas/parameters (x-go-type маппинг)
│   ├── api.yaml              # paths/operations
│   └── *-cfg.yaml, gen.go    # конфиги генерации + сгенерированный strict-server (НЕ РЕДАКТИРОВАТЬ)
├── internal/
│   ├── app/            # composition root: сборка зависимостей, запуск HTTP-сервера
│   ├── config/         # агрегированный Config (HTTP + pkg-конфиги)
│   ├── domain/         # сущности и доменные ошибки
│   ├── repository/     # postgres-репозитории
│   ├── service/        # бизнес-логика, использует repo + cache
│   └── http/           # роутер (strict adapter + OpenAPI validator), реализация StrictServerInterface
├── pkg/                # переиспользуемые библиотеки — plain Config, без env-тегов
│   ├── logger/         #   обёртка над zap
│   ├── postgres/       #   pgx pool
│   └── redis/          #   go-redis клиент
├── migrations/         # SQL-миграции (golang-migrate)
├── .env.example        # единый источник runtime-конфигурации
├── docker-compose.yml  # app + postgres + redis + migrate
├── Dockerfile          # multi-stage сборка, Go 1.25
└── Makefile
```

## Требования

| Зависимость | Версия | Назначение |
|-------------|--------|------------|
| [Go](https://go.dev/dl/) | 1.25+ | компиляция и запуск |
| [Docker](https://docs.docker.com/get-docker/) + Compose | — | контейнеры для postgres, redis, app |
| [golangci-lint](https://golangci-lint.run/welcome/install/) | 2.x | линтинг (`make lint`) |
| [golang-migrate](https://github.com/golang-migrate/migrate) | 4.x | миграции БД (`make migrate-*`) |

## Быстрый старт

```sh
cp .env.example .env
make up           # собрать и запустить app + postgres + redis + миграции
curl localhost:8080/ping        # -> pong
```

Локальная разработка без контейнеров для приложения:

```sh
make tidy
docker compose up -d postgres redis
make migrate-up
make run          # загружает .env из корня проекта
```

## Make-команды

| Команда | Описание |
|---------|----------|
| `make help` | Показать список всех целей |
| `make generate` | Сгенерировать код сервера из OpenAPI спеки |
| `make tidy` | `go mod tidy` |
| `make build` | Собрать бинарник в `bin/arche` |
| `make run` | Запустить локально (читает `.env`) |
| `make test` | Тесты с race detector |
| `make lint` | Запуск golangci-lint |
| `make fmt` | Форматирование + `go vet` |
| `make up` | Запуск docker-compose стека |
| `make down` | Остановка docker-compose стека |
| `make logs` | Логи приложения в реальном времени |
| `make migrate-up` | Применить все миграции |
| `make migrate-down` | Откатить одну миграцию |
| `make migrate-new name=add_xxx` | Создать новую миграцию |
| `make clean` | Удалить `bin/` |

## Архитектура

Запрос проходит через слои сверху вниз; зависимости направлены только внутрь
(внешние слои знают о внутренних, не наоборот):

```
HTTP-запрос
   │
   ▼
generated/api/gen.go            сгенерированный роутер + OpenAPI-валидатор
   │                            (отклоняет некорректный запрос ДО хендлера)
   ▼
internal/http/router.go         chi: middleware (RequestID, RealIP, Logger,
   │                            Recoverer) + strict adapter + validator
   ▼
internal/http/handler           реализация api.StrictServerInterface:
   │                            получает типизированный *RequestObject,
   │                            возвращает типизированный *ResponseObject,
   │                            мапит доменные ошибки в коды ответов
   ▼
internal/service                бизнес-логика; использует repository + cache
   │
   ▼
internal/repository             raw SQL через pgx; мапит ошибки БД в доменные
   │
   ▼
internal/domain                 сущности (User) + sentinel-ошибки (ErrUserNotFound)
```

| Слой | Пакет | Ответственность | Знает о |
|------|-------|-----------------|---------|
| Точка входа | `cmd/app` | загрузка конфига, signal context, вызов `app.Run` | `app`, `config` |
| Composition root | `internal/app` | создаёт все зависимости, запускает/останавливает HTTP-сервер | всё |
| Контракт | `generated/api` | сгенерированные типы, роутер, strict-интерфейс, embedded-спека | — (не редактируется) |
| Транспорт | `internal/http` | роутер, middleware, реализация `StrictServerInterface` | `generated/api`, `domain`, `service` (через интерфейс) |
| Бизнес-логика | `internal/service` | оркестрация, кэширование, доменные правила | `domain`, `repository` (через интерфейс), `pkg/redis` |
| Данные | `internal/repository` | SQL-запросы, маппинг ошибок БД | `domain`, `pkg/postgres` |
| Домен | `internal/domain` | сущности и sentinel-ошибки | — |

**Ключевые принципы:**

- **Интерфейсы объявляет потребитель.** `internal/http/handler` объявляет
  `userService`, `internal/service` объявляет `userRepository` — каждый в своём
  файле, неэкспортируемые, по роли зависимости. Конкретные типы (`*service.UserService`,
  `*repository.UserRepository`) подставляются в `app.go`. Это позволяет
  тестировать каждый слой с фейком (см. `internal/http/router_test.go`).
- **Доменные ошибки — граница между слоями.** Repository превращает ошибки
  pgx/PostgreSQL в `domain.ErrXxx` (например, нарушение unique-constraint
  `23505` → `domain.ErrUserExists`); handler превращает `domain.ErrXxx` в
  HTTP-коды. Сравнение только через `errors.Is`.
- **Wiring — вручную, без DI-фреймворка.** Вся сборка зависимостей видна в
  одном месте — `internal/app/app.go`.

## Конфигурация

Вся runtime-конфигурация приходит из переменных окружения. `internal/config`
читает `.env` (если есть) и окружение процесса в структуру `Config`,
чьи подструктуры владеют env-тегами. Библиотеки `/pkg/*` независимы и
используют plain `Config`-структуры без тегов; `internal/app/app.go`
связывает их через явное преобразование структур:

```go
log, err := logger.New(logger.Config(cfg.Logger))
pg,  err := postgres.New(ctx, postgres.Config(cfg.Postgres))
rdb, err := redis.New(ctx, redis.Config(cfg.Redis))
```

Если поле добавлено на одной стороне, но не на другой — код перестаёт компилироваться.

## Spec-first API

HTTP-контракт — **единственный источник истины**. Сервер описывается в OpenAPI-спеке,
а код роутера, типов запросов/ответов и интерфейса хендлеров генерируется из неё
через [oapi-codegen](https://github.com/oapi-codegen/oapi-codegen) в strict-server
режиме. Бизнес-логика пишется вручную; «сантехника» (парсинг JSON, чтение
параметров, сериализация, регистрация роутов, валидация) — генерируется.

### Из чего состоит `generated/api/`

| Файл | Редактируется | Назначение |
|------|:---:|------------|
| `definitions.yaml` | ✍️ вручную | Переиспользуемые `schemas` и `parameters`. `x-go-type` маппит OpenAPI-типы на готовые Go-типы (например `UUID` → `uuid.UUID`). |
| `api.yaml` | ✍️ вручную | `paths` и `operations`; ссылается на `definitions.yaml` через `$ref`. Здесь же базовый путь (`servers: /api/v1`). |
| `definitions-cfg.yaml`, `cfg.yaml` | ✍️ вручную | Конфиги генерации (типы и сервер соответственно). |
| `definitions.go` | 🤖 генерируется | Go-типы из `definitions.yaml`. |
| `gen.go` | 🤖 генерируется | `StrictServerInterface`, `*RequestObject`/`*ResponseObject`, роутер `HandlerWithOptions`, embedded-спека `GetSwagger()`. |

`cfg.yaml` содержит `import-mapping: { ./definitions.yaml: "-" }` — это говорит
генератору переиспользовать типы из уже сгенерированного `definitions.go`, а не
дублировать их в `gen.go`. Поэтому при генерации **порядок важен**: сначала
`definitions`, потом `api` (этот порядок уже зашит в `make generate`).

### Что даёт strict-server режим

- Хендлер **не знает про HTTP**: нет `http.ResponseWriter`, нет `json.Decode` —
  на вход приходит готовый типизированный `*RequestObject`, на выход
  возвращается типизированный `*ResponseObject`.
- **Compile-time контроль контракта**: добавили операцию в спеку → проект не
  собирается, пока не реализован соответствующий метод `StrictServerInterface`.
- **Нельзя вернуть код ответа, которого нет в спеке** — для каждого описанного
  кода генерируется отдельный тип ответа (`CreateUser201JSONResponse`,
  `CreateUser409JSONResponse` и т.д.).
- **Runtime-валидация**: middleware `OapiRequestValidator` сверяет входящий
  запрос с embedded-спекой (обязательные параметры, схема тела, `Content-Type`)
  и отклоняет некорректный запрос с `400` **до** попадания в хендлер.

### Как добавить новый endpoint (пошагово)

Пример: добавим `DELETE /api/v1/users/{id}`.

**Шаг 1 — описать операцию в `api.yaml`.** Добавьте операцию к существующему
path `/users/{id}` (параметр `id` уже описан в `definitions.yaml`):

```yaml
  /users/{id}:
    # ... get: ...
    delete:
      operationId: deleteUser          # → метод DeleteUser в Go (PascalCase)
      summary: Delete a user by ID
      parameters:
        - $ref: './definitions.yaml#/components/parameters/UserID'
      responses:
        '204':
          description: user deleted
        '404':
          description: user not found
          content:
            application/json:
              schema:
                $ref: './definitions.yaml#/components/schemas/Error'
```

Если нужны новые типы (тело запроса, новая сущность) — добавьте их в
`definitions.yaml` в `components/schemas` и сошлитесь через `$ref`.

**Шаг 2 — сгенерировать код:**

```sh
make generate
```

Появятся `DeleteUserRequestObject`, `DeleteUser204Response`,
`DeleteUser404JSONResponse` и новый метод в `StrictServerInterface`.

**Шаг 3 — убедиться, что компилятор требует реализацию:**

```sh
go build ./...
# *handler.Server does not implement api.StrictServerInterface
#   (missing method DeleteUser)
```

**Шаг 4 — добавить бизнес-логику вниз по слоям** (если её ещё нет). В
`internal/domain` — при необходимости sentinel-ошибка; в
`internal/repository/user.go` — SQL; в `internal/service/user.go` — метод и
расширение интерфейса `userRepository`:

```go
// internal/repository/user.go
func (r *UserRepository) Delete(ctx context.Context, id uuid.UUID) error {
    const q = `DELETE FROM users WHERE id = $1`
    tag, err := r.pool.Exec(ctx, q, id)
    if err != nil {
        return fmt.Errorf("delete user: %w", err)
    }
    if tag.RowsAffected() == 0 {
        return domain.ErrUserNotFound
    }
    return nil
}

// internal/service/user.go
func (s *UserService) Delete(ctx context.Context, id uuid.UUID) error {
    if err := s.repo.Delete(ctx, id); err != nil {
        return err
    }
    _ = s.cache.Del(ctx, userCacheKey(id)).Err()
    return nil
}
```

**Шаг 5 — реализовать метод хендлера** в `internal/http/handler/user.go`.
Добавьте `Delete` в потребительский интерфейс `userService` (в
`internal/http/handler/server.go`), затем сам метод:

```go
func (s *Server) DeleteUser(ctx context.Context, req api.DeleteUserRequestObject) (api.DeleteUserResponseObject, error) {
    if err := s.users.Delete(ctx, req.Id); err != nil {
        if errors.Is(err, domain.ErrUserNotFound) {
            return api.DeleteUser404JSONResponse{Error: "user not found"}, nil
        }
        s.log.Error("delete user", zap.Error(err))
        return nil, err     // → strict adapter вернёт 500
    }
    return api.DeleteUser204Response{}, nil
}
```

**Шаг 6 — проверить:**

```sh
go build ./...      # компилируется → контракт реализован полностью
make test           # тесты с race detector
```

Регистрировать роут вручную в `router.go` **не нужно** — он попадает в роутер
автоматически при следующей генерации.

### Как добавить новый домен

Те же шаги плюс новый вертикальный срез по образцу `user`:
`internal/domain/<name>.go` → `internal/repository/<name>.go` →
`internal/service/<name>.go` → методы хендлера в `internal/http/handler/<name>.go`.
Затем создать `*Service`/`*Repository` в `internal/app/app.go` и передать
зависимости в `handler.NewServer`. Один `Server` реализует весь
`StrictServerInterface`, поэтому хендлеры разных доменов — это просто отдельные
файлы с методами на `*Server`.

### Частые ошибки

- **Забыли `make generate`** после правки спеки — старый `gen.go`, новых типов нет.
- **Ручная правка `gen.go`/`definitions.go`** — затрётся при следующей генерации
  (файлы помечены `DO NOT EDIT`).
- **Неуникальный `operationId`** — имена сгенерированных методов конфликтуют.
- **Код ответа не описан в спеке** — соответствующего `*ResponseObject` нет,
  вернуть его не получится; сначала добавьте код в `responses` и перегенерируйте.

## Клонирование для нового сервиса

1. `git clone` → `rm -rf .git && git init`
2. Переименовать модуль — заменить `github.com/zulfikorramatov/arche` на новый путь:
   ```sh
   grep -rl "github.com/zulfikorramatov/arche" . --include="*.go" --include="go.mod" | xargs sed -i '' 's|github.com/zulfikorramatov/arche|github.com/your-org/your-service|g'
   ```
3. Переименовать контейнеры и `APP_NAME` в `docker-compose.yml` / `.env`
4. Заменить домен `user` на свой: спека (`api.yaml`, `definitions.yaml`) →
   `make generate` → `domain`/`repository`/`service`/`handler` → миграции
5. `make tidy && make up`

## Стиль кода

Конвенции описаны в [STYLEGUIDE.md](STYLEGUIDE.md).
