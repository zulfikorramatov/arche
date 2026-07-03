# OpenAPI Guide

Практическое руководство: как добавлять и менять HTTP-эндпоинты в этом проекте.

## Введение (коротко)

HTTP-слой построен как **design-first**: контракт API описывается в OpenAPI-спеке
(`generated/api/*.yaml`), а Go-код (роутинг, модели, интерфейсы, валидация) **генерируется**
из него командой `make generate`. Спек — единственный источник истины.

Отсюда два железных правила:

1. **Не редактируй `*.go` в `generated/api/` руками.** Они перезатираются генератором.
   Меняешь `.yaml` → гоняешь `make generate`.
2. Валидация входящих запросов (типы, `required`, форматы, auth) происходит **автоматически**
   в middleware ещё до хендлера. Не дублируй её в коде — описывай в спеке.

### Два файла спека — кто за что отвечает

| Файл | Отвечает за | Генерится в |
| --- | --- | --- |
| `api.yaml` | **Операции**: пути, параметры, `requestBody`, `responses`, security | `gen.go` (роутинг + `StrictServerInterface`) |
| `definitions.yaml` | **Схемы данных**: переиспользуемые структуры (`User`, `Error`, ...) | `definitions.go` (чистые модели) |

Правило простое: всё, что описывает **эндпоинт** — в `api.yaml`; всё, что описывает
**форму данных** — в `definitions.yaml`. Операция ссылается на схему через `$ref`.

Аналогия: `definitions.yaml` — это как `struct`-определения в Go, `api.yaml` — как сигнатуры
хендлеров, которые эти структуры используют.

---

## Пошаговый пример: добавляем `POST /users`

Пройдём весь путь: от строчки в YAML до рабочего эндпоинта, который создаёт пользователя.
Эндпоинт принимает `username` + `password` в теле запроса и возвращает созданного `User`.

### Шаг 1. Описать форму данных в `definitions.yaml`

Тело запроса — это новая структура. Добавляем её в `components/schemas`:

```yaml
    CreateUserRequest:
      type: object
      required: [username, password]     # оба поля обязательны — валидатор сам отвергнет запрос без них
      properties:
        username:
          type: string
          minLength: 3
          maxLength: 32
        password:
          type: string
          format: password
          minLength: 8
```

Схему ответа (`User`) заводить не надо — она уже есть в `definitions.yaml`.

> Почему тело запроса — здесь, а не в `api.yaml`? Потому что это **форма данных**. В `api.yaml`
> мы лишь сошлёмся на неё. Так `CreateUserRequest` можно переиспользовать, а генерация моделей
> остаётся отдельной.

### Шаг 2. Описать операцию в `api.yaml`

Добавляем метод `post` к уже существующему пути `/users`:

```yaml
paths:
  /users:
    get:
      operationId: ListUsers
      # ... существующий код без изменений ...
    post:
      operationId: CreateUser          # -> имя Go-метода: CreateUser(...)
      summary: Create a new user
      requestBody:                      # ФАКТ "принимаю тело" — здесь, в операции
        required: true
        content:
          application/json:
            schema:
              $ref: './definitions.yaml#/components/schemas/CreateUserRequest'
      responses:
        '201':
          description: Created
          content:
            application/json:
              schema:
                $ref: './definitions.yaml#/components/schemas/User'
        '400':
          description: Bad Request
          content:
            application/json:
              schema:
                $ref: './definitions.yaml#/components/schemas/Error'
        '500':
          description: Internal Server Error
          content:
            application/json:
              schema:
                $ref: './definitions.yaml#/components/schemas/Error'
```

Что здесь важно:

- `operationId` → имя метода в Go (`CreateUser`). Держи его в PascalCase и уникальным.
- Каждый код ответа (`201`, `400`, `500`) генерит отдельный типизированный ответ
  (`CreateUser201JSONResponse` и т.д.). Описывай все коды, которые реально возвращаешь.
- Auth наследуется из глобального `security: [BasicAuth: []]` наверху `api.yaml`.
  Чтобы сделать эндпоинт публичным, переопредели `security: []` на уровне операции.

### Шаг 3. Сгенерировать код

```bash
make generate
```

Из спека появятся новые типы (имена выводятся из `operationId` и схем):

- `api.CreateUserRequestObject` — обёртка запроса; тело внутри как `Body *api.CreateUserRequest`.
- `api.CreateUser201JSONResponse` — ответ 201 (обёртка над `User`).
- `api.CreateUser400JSONResponse`, `api.CreateUser500JSONResponse` — ответы-ошибки (обёртки над `Error`).
- Метод `CreateUser(...)` добавится в `api.StrictServerInterface`.

С этого момента проект **не скомпилируется**, пока `Server` не реализует новый метод —
это гарантия из `server.go` (`var _ api.StrictServerInterface = (*Server)(nil)`). Так генератор
заставляет тебя дописать хендлер.

### Шаг 4. Реализовать хендлер

Хендлеры живут в `internal/http/handler/server.go` и реализуют
`api.StrictServerInterface`. Сигнатура строго типизированная: тело уже распарсено и
провалидировано, тебе приходит готовая структура.

Сначала расширь узкий интерфейс `userService` (объявлен у потребителя — в `handler`):

```go
type userService interface {
	List(ctx context.Context) ([]entity.User, error)
	Create(ctx context.Context, username, password string) (entity.User, error) // новый
}
```

Затем сам метод:

```go
func (s *Server) CreateUser(ctx context.Context, req api.CreateUserRequestObject) (api.CreateUserResponseObject, error) {
	user, err := s.userSvc.Create(ctx, req.Body.Username, req.Body.Password)
	if err != nil {
		s.log.Error("create user", "error", err)
		return api.CreateUser500JSONResponse{Error: "internal error"}, nil
	}

	return api.CreateUser201JSONResponse{
		Id:        user.ID,
		Username:  user.Username,
		CreatedAt: user.CreatedAt,
	}, nil
}
```

Обрати внимание: бизнес-ошибку возвращаем **как значение ответа** (`...500JSONResponse`), а
`error` из метода — `nil`. `error != nil` резервируется для неожиданных сбоев и уходит в
`ResponseErrorHandlerFunc` (см. `internal/http/router.go`), который отдаёт 500 и логирует.

### Шаг 5. Дописать бизнес-логику по слоям

Хендлер — только верх цепочки `handler → service → repository`. Спускаемся вниз
(подробности слоёв — в `CLAUDE.md`):

**`internal/service/user.go`** — бизнес-логика, здесь хешируем пароль:

```go
func (s *UserService) Create(ctx context.Context, username, password string) (entity.User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return entity.User{}, fmt.Errorf("hash password: %w", err)
	}

	user, err := s.repo.Create(ctx, username, string(hash))
	if err != nil {
		return entity.User{}, fmt.Errorf("create user: %w", err)
	}
	return user, nil
}
```

Добавь `Create` и в узкий интерфейс `userRepository` в этом же файле.

**`internal/repository/user.go`** — сырой SQL через pgx:

```go
func (r *UserRepository) Create(ctx context.Context, username, passwordHash string) (entity.User, error) {
	const q = `INSERT INTO users (username, password)
	           VALUES ($1, $2)
	           RETURNING id, username, password, created_at`

	var u entity.User
	err := r.pool.QueryRow(ctx, q, username, passwordHash).
		Scan(&u.ID, &u.Username, &u.Password, &u.CreatedAt)
	if err != nil {
		return entity.User{}, fmt.Errorf("insert user: %w", err)
	}
	return u, nil
}
```

### Шаг 6. Проверить

```bash
make generate   # ещё раз, если правил спек
make fmt        # gofmt + go vet
make lint       # golangci-lint
make test       # go test -race
make run        # поднять сервер локально (нужен make up для зависимостей)
```

Ручная проверка:

```bash
curl -u admin:secret -X POST http://localhost:8080/users \
  -H 'Content-Type: application/json' \
  -d '{"username":"alice","password":"password123"}'
```

Проверь, что валидатор работает: пошли запрос без `password` или с `username` короче 3 символов —
получишь **400 ещё до хендлера**, из middleware.

---

## Чек-лист «добавить эндпоинт»

1. [ ] Схемы тел запроса/ответа → `definitions.yaml` (если новые).
2. [ ] Операция (`path`, `requestBody`, `responses`, `parameters`) → `api.yaml`.
3. [ ] `make generate`.
4. [ ] Реализовать метод в `internal/http/handler/server.go`.
5. [ ] Расширить интерфейсы + логику: `service` → `repository`.
6. [ ] Миграция БД, если нужна новая таблица/колонка (`make migrate-new name=...`).
7. [ ] `make fmt && make lint && make test`.

---

## Справочник по схемам (частые случаи)

### Path- и query-параметры (НЕ request body)

Параметры пути и строки запроса объявляются в `api.yaml`, в блоке `parameters` операции —
**не** в `definitions.yaml`:

```yaml
  /users/{id}:
    get:
      operationId: GetUser
      parameters:
        - name: id
          in: path
          required: true                # path-параметры всегда required
          schema:
            $ref: './definitions.yaml#/components/schemas/UUID'
        - name: verbose
          in: query
          required: false
          schema:
            type: boolean
```

В хендлере они приходят типизированными полями `RequestObject` (`req.Id`, `req.Params.Verbose`).

### Типы и форматы

```yaml
type: string
type: string
  format: uuid            # -> uuid.UUID (см. схему UUID)
type: string
  format: date-time       # -> time.Time
type: string
  format: email
type: integer
  format: int64
  minimum: 0
type: number
  format: double
type: boolean
type: string
  enum: [active, blocked, pending]     # ограниченный набор значений
```

### Ограничения (валидируются автоматически)

```yaml
minLength / maxLength     # для строк
pattern: '^[a-z0-9_]+$'   # regex для строк
minimum / maximum         # для чисел
minItems / maxItems       # для массивов
```

### Массивы и вложенность

```yaml
    UserList:
      type: array
      items:
        $ref: '#/components/schemas/User'

    UserWithRoles:
      type: object
      required: [user, roles]
      properties:
        user:
          $ref: '#/components/schemas/User'
        roles:
          type: array
          items:
            type: string
```

### Nullable-поля

В проекте включён `nullable-type` (см. `cfg.yaml`), поэтому:

```yaml
    deleted_at:
      type: string
      format: date-time
      nullable: true        # -> *time.Time в Go
```

### Кастомный Go-тип

Как сделано для UUID в `definitions.yaml` — маппинг схемы на существующий Go-тип:

```yaml
    UUID:
      type: string
      format: uuid
      x-go-type: uuid.UUID
      x-go-type-import:
        path: github.com/google/uuid
```

---

## Частые ошибки

- **Забыл `make generate`** — код и спек разошлись, компиляция падает или ведёт себя странно.
  Первым делом после правки `.yaml` — регенерация.
- **Отредактировал `gen.go`/`definitions.go` руками** — изменения затрутся. Правь только `.yaml`.
- **Продублировал валидацию в хендлере** — не нужно; `required`, форматы, длины проверяет
  middleware. В сервисе проверяй только то, что спек выразить не может (например, «такой username
  уже занят»).
- **Не описал код ответа в `responses`** — не будет сгенерирован соответствующий
  `...JSONResponse`, и ты не сможешь его вернуть типобезопасно.
- **Вернул `error` вместо `...JSONResponse` для бизнес-ошибки** — уйдёт в общий 500-обработчик.
  Ожидаемые ошибки возвращай как типизированный ответ с нужным кодом, `error = nil`.
- **`operationId` не уникален или не PascalCase** — сломает или испортит генерацию имён методов.
