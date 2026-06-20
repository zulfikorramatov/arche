# Plan: Spec-First OpenAPI подход — Гайд для Claude Code

## TL;DR

Подробный гайд по реализации spec-first подхода для Go-проектов, основанный на архитектуре ledger-hub. Цель — дать Claude Code (или разработчику) пошаговую инструкцию для создания HTTP API, где контракт (OpenAPI spec) является единственным источником истины, а серверный код генерируется автоматически.

---

## Архитектура (на примере ledger-hub)

### Цепочка: Spec → Generated Code → Handler → Router

```
definitions.yaml (shared types)
       ↓
api.yaml (endpoints + request/response schemas)
       ↓
oapi-codegen (генерация)
       ↓
├── definitions.go    (Go-типы из definitions.yaml)
└── gen.go            (ServerInterface, StrictServerInterface, Router, Request/Response objects, embedded spec)
       ↓
handlers/server.go   (реализация StrictServerInterface)
       ↓
app.go               (wiring: strict adapter → chi router → middleware → HTTP server)
```

---

## Шаги реализации

### Phase 1: Подготовка проекта

**Step 1.** Инициализировать Go-модуль и установить зависимости:
- `github.com/oapi-codegen/oapi-codegen/v2` — кодогенератор (как tool dependency)
- `github.com/oapi-codegen/runtime` — runtime для strict server
- `github.com/getkin/kin-openapi` — парсинг OpenAPI для валидации
- `github.com/go-chi/chi/v5` — HTTP-роутер
- `github.com/oapi-codegen/nethttp-middleware` — middleware валидации запросов по спеке

**Step 2.** Создать структуру директорий:
```
generated/api/
  api.yaml              # Основная спецификация (paths, operations)
  definitions.yaml      # Общие schemas и parameters
  cfg.yaml              # Конфиг генерации сервера
  definitions-cfg.yaml  # Конфиг генерации типов
  docs/                 # Опционально: redocly конфиг
internal/
  http/handlers/        # Реализация хендлеров
  http/middleware/      # Middleware
  app/                  # Wiring (DI, lifecycle)
```

---

### Phase 2: Написание OpenAPI спецификации

**Step 3.** Создать `definitions.yaml` — общие типы и параметры:
- Определить reusable schemas (статусы, типы, кастомные форматы)
- Использовать `x-go-type` и `x-go-type-import` для маппинга на существующие Go-типы
- Определить общие parameters (headers: X-Request-ID, X-Tenant-ID и т.д.)

Пример:
```yaml
components:
  schemas:
    MoneyAmountSums:
      type: string
      format: decimal
      x-go-type: money.Money
      x-go-type-import:
        path: myproject/pkg/money
  parameters:
    XRequestID:
      name: X-Request-ID
      in: header
      required: true
      schema:
        type: string
        minLength: 1
```

**Step 4.** Создать `api.yaml` — основная спецификация:
- Каждый path ссылается на definitions через `$ref: './definitions.yaml#/...'`
- Каждая операция имеет уникальный `operationId` (из него генерируются имена методов интерфейса)
- Описать request body и все возможные response коды со схемами
- Security schemes определяются здесь

Ключевые правила:
- `operationId` → имя метода в Go (PascalCase): `createCustomer` → `CreateCustomer`
- Каждый response code → отдельный тип: `CreateCustomer201JSONResponse`, `CreateCustomer400JSONResponse`
- Parameters (path, header, query) → поля в `*RequestObject`

---

### Phase 3: Конфигурация кодогенератора

**Step 5.** Создать `definitions-cfg.yaml`:
```yaml
package: api
generate:
  models: true
output: generated/api/definitions.go
output-options:
  nullable-type: true
  skip-prune: true
```

**Step 6.** Создать `cfg.yaml`:
```yaml
package: api
generate:
  chi-server: true      # Генерирует HandlerWithOptions для Chi роутера
  models: true          # Генерирует request/response модели
  strict-server: true   # Генерирует StrictServerInterface (типобезопасные хендлеры)
  embedded-spec: true   # Встраивает спеку в бинарник (для runtime валидации)
output: generated/api/gen.go
output-options:
  nullable-type: true
import-mapping:
  ./definitions.yaml: "-"   # Не генерировать типы из definitions.yaml повторно
```

Опция `import-mapping: ./definitions.yaml: "-"` критически важна — она говорит генератору использовать типы из уже сгенерированного `definitions.go` вместо дублирования.

---

### Phase 4: Генерация кода

**Step 7.** Добавить команды генерации в Makefile:
```make
.PHONY: generate
generate:
	go tool oapi-codegen -config generated/api/definitions-cfg.yaml generated/api/definitions.yaml
	go tool oapi-codegen -config generated/api/cfg.yaml generated/api/api.yaml
```

Порядок важен: сначала definitions (типы), потом api (сервер, который ссылается на эти типы).

**Step 8.** Запустить генерацию — получить:
- `definitions.go` — Go-типы, константы, type aliases
- `gen.go` — содержит:
    - `ServerInterface` — low-level интерфейс (принимает http.ResponseWriter)
    - `StrictServerInterface` — high-level интерфейс (принимает typed RequestObject, возвращает typed ResponseObject)
    - `NewStrictHandlerWithOptions()` — адаптер strict → server
    - `HandlerWithOptions()` — регистрация роутов в Chi
    - `GetSwagger()` — загрузка embedded спеки
    - Request/Response типы для каждой операции

---

### Phase 5: Реализация хендлеров

**Step 9.** Создать `internal/http/handlers/server.go`:
```go
package handlers

import "myproject/generated/api"

// Compile-time check: убеждаемся что Server реализует интерфейс
var _ api.StrictServerInterface = (*Server)(nil)

type Server struct {
    // зависимости (сервисы)
}

func NewServer(...) *Server { return &Server{...} }
```

**Step 10.** Реализовать методы интерфейса:
- Каждый метод принимает `context.Context` и типизированный `*RequestObject`
- Возвращает типизированный `ResponseObject` и `error`
- Не нужно парсить JSON, читать headers, писать в ResponseWriter — всё делает strict adapter

Пример:
```go
func (s *Server) CreateCustomer(ctx context.Context, req api.CreateCustomerRequestObject) (api.CreateCustomerResponseObject, error) {
    // req.Body — уже распарсенный JSON
    // req.Params.XTenantID — header
    result, err := s.customerService.Create(ctx, ...)
    if err != nil {
        return api.CreateCustomer400JSONResponse{Message: err.Error()}, nil
    }
    return api.CreateCustomer201JSONResponse{...}, nil
}
```

---

### Phase 6: Wiring (связывание компонентов)

**Step 11.** В app.go собрать всё вместе:

```go
func httpHandler(server *handlers.Server) http.Handler {
    // 1. Strict adapter: оборачивает typed-хендлер в HTTP-хендлер
    handler := api.NewStrictHandlerWithOptions(server, nil, api.StrictHTTPServerOptions{
        RequestErrorHandlerFunc:  errorHandler(400),
        ResponseErrorHandlerFunc: errorHandler(500),
    })

    // 2. Загрузить embedded спеку для runtime валидации
    swagger, _ := api.GetSwagger()
    swagger.Servers = nil // убрать серверы чтобы валидатор не проверял хост

    // 3. Зарегистрировать роуты с middleware
    return api.HandlerWithOptions(handler, api.ChiServerOptions{
        BaseRouter: chi.NewMux(),
        Middlewares: []api.MiddlewareFunc{
            // OpenAPI validator middleware — валидирует request по спеке
            nethttpmiddleware.OapiRequestValidatorWithOptions(swagger, &nethttpmiddleware.Options{...}),
            // Другие middleware...
        },
    })
}
```

---

### Phase 7: Runtime валидация

**Step 12.** Настроить OpenAPI validator middleware:
- Использует embedded спеку для проверки:
    - Required parameters присутствуют
    - Request body соответствует schema
    - Content-Type корректен
    - Security requirements выполнены (через AuthenticationFunc callback)
- Невалидные запросы отклоняются ДО попадания в хендлер

---

### Phase 8: Документация

**Step 13.** (Опционально) Настроить генерацию HTML-документации:
```make
generate-docs:
	docker compose run --rm redoc build-docs api.yaml --output docs/index.html

generate-lint:
	docker compose run --rm redoc lint api.yaml
```

---

## Ключевые паттерны и решения

### Что генерируется автоматически:
- Все request/response Go-типы
- Router registration (path → handler mapping)
- Parameter binding (path, query, header → struct fields)
- JSON serialization/deserialization
- Embedded OpenAPI spec для runtime валидации
- Compile-time type safety через интерфейсы

### Что пишется вручную:
- OpenAPI spec (api.yaml, definitions.yaml)
- Бизнес-логика в хендлерах (реализация StrictServerInterface)
- Middleware
- Wiring (DI, lifecycle)

### Преимущества strict-server режима:
- Хендлер не знает про HTTP (нет ResponseWriter, нет json.Decode)
- Compile-time проверка: если добавить endpoint в спеку, код не скомпилируется пока не реализуешь метод
- Невозможно вернуть response код, не описанный в спеке
- Request body уже распарсен и типизирован

### import-mapping трюк:
Разделение на два файла (definitions.yaml + api.yaml) с `import-mapping: "-"` позволяет:
- Переиспользовать типы между спеками
- Маппить OpenAPI types на существующие Go-типы через `x-go-type`
- Избежать дублирования кода

---

## Файлы для модификации/создания (при новом проекте)

- `generated/api/api.yaml` — OpenAPI спецификация endpoints
- `generated/api/definitions.yaml` — Shared schemas и parameters
- `generated/api/cfg.yaml` — Конфиг генерации сервера
- `generated/api/definitions-cfg.yaml` — Конфиг генерации типов
- `generated/api/gen.go` — **НЕ ТРОГАТЬ**, автогенерируемый
- `generated/api/definitions.go` — **НЕ ТРОГАТЬ**, автогенерируемый
- `internal/http/handlers/server.go` — Реализация StrictServerInterface
- `internal/http/handlers/*.go` — Методы хендлеров по доменам
- `internal/app/app.go` — Wiring (strict adapter → router → middleware)
- `Makefile` — Команды генерации

---

## Workflow для добавления нового endpoint

1. Добавить path + operation в `api.yaml` (с уникальным `operationId`)
2. При необходимости добавить новые schemas в `definitions.yaml`
3. Запустить `make generate`
4. Компилятор покажет ошибку: Server не реализует новый метод
5. Реализовать метод в handlers
6. Готово — роут автоматически зарегистрирован

---

## Verification

1. `make generate` — генерация проходит без ошибок
2. `go build ./...` — проект компилируется (compile-time check интерфейсов)
3. `make generate-lint` — спека проходит линтинг Redocly
4. Запуск сервера → `GET /ping` возвращает 200
5. Отправка невалидного request → middleware возвращает 400 до хендлера
6. Отправка request без auth → middleware возвращает 401
