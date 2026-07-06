# arche

`arche` — шаблон HTTP-сервиса на Go (1.25) с чистой слоёной архитектурой,
design-first HTTP-слоем на OpenAPI и набором готовой инфраструктуры
(Postgres, Redis, Kafka, логирование, Elastic APM).

Собирается в два бинарника:

- **HTTP-сервер** — `cmd/app` → `bin/app`
- **CLI (Cobra)** — `cmd/cli` → `bin/cli`

## Требования

- **Go 1.25+**
- **Docker** + Docker Compose — для локального стека (Postgres, Redis, миграции)
- [`golang-migrate`](https://github.com/golang-migrate/migrate) CLI — для ручных миграций (`make migrate-*`)
- [`golangci-lint`](https://golangci-lint.run) — для `make lint`

`oapi-codegen` подключён как Go tool dependency и отдельной установки не требует.

## Быстрый старт

```bash
# 1. Конфиг
cp .env.example .env

# 2. Поднять инфраструктуру (Postgres + Redis + one-shot миграции)
make up

# 3. Запустить сервер (читает .env из корня)
make run
```

Сервис слушает `HTTP_ADDR` (по умолчанию `:8080`). Все эндпоинты защищены Basic-Auth,
поэтому сначала создайте пользователя через CLI:

```bash
make build-cli
./bin/cli user create --username alice --password secret

# Проверка
curl -u alice:secret http://localhost:8080/users
```

## Команды

Всё идёт через `Makefile` (`make help` покажет полный список):

| Команда | Описание |
| --- | --- |
| `make run` | Запустить HTTP-сервер локально |
| `make build` / `make build-cli` | Собрать сервер / CLI в `bin/` |
| `make test` | `go test -race -count=1 ./...` |
| `make lint` | `golangci-lint run` (конфиг `.golangci.yml`) |
| `make fmt` | `gofmt -s -w .` + `go vet ./...` |
| `make generate` | Регенерировать HTTP-код из OpenAPI-спеки (после любой правки `.yaml`) |
| `make up` / `make down` | Поднять / остановить docker-compose стек |
| `make migrate-up` / `make migrate-down` | Накатить / откатить миграции |
| `make migrate-new name=add_xxx` | Создать новую миграцию |

## Структура проекта

```
cmd/
  app/            HTTP-сервер (entrypoint)
  cli/            Cobra CLI; поддомены в cli/<domain>/, общие зависимости в cli/deps/
internal/
  app/            Композиционный корень (app.Run): config → infra → repo → service → handler → server
  config/         Загрузка конфига (cleanenv + godotenv), зеркалит Config-структуры из pkg/
  entity/         Доменные типы
  repository/     SQL поверх pgx, возвращает entity-типы
  service/        Бизнес-логика (auth и т.п.); зависит от интерфейса репозитория
  http/
    handler/      Реализация сгенерированного api.StrictServerInterface, маппинг entity ↔ api
    middleware/   Валидатор запросов из встроенной OpenAPI-спеки, Basic-Auth
    response/      Хелперы ответов
    router.go     chi-роутер, middleware, обёртка Elastic APM
generated/api/    Сгенерированный код + OpenAPI-спеки (руками НЕ править .go)
migrations/       SQL-миграции (golang-migrate)
pkg/              Самодостаточная инфраструктура (postgres, redis, kafka, logger, money)
```

## API

HTTP-контракт — источник истины. Код в `generated/api/*.go` **регенерируется**
и правится руками только через спеку:

- `generated/api/definitions.yaml` — схемы данных (`User`, `Error`, тела запросов)
- `generated/api/api.yaml` — операции (пути, параметры, тела, ответы, security)

После правки спеки — `make generate`. Валидация (типы, `required`, форматы, Basic-Auth)
навешивается автоматически из встроенной спеки в
[middleware/validator.go](internal/http/middleware/validator.go) до вызова хендлеров —
дублировать её в коде не нужно.

Полный разбор «как добавить эндпоинт» — в [OAPI_GUIDE.md](OAPI_GUIDE.md).

## Конфигурация

Все переменные окружения — в [.env.example](.env.example). Загружаются один раз
в entrypoint через `cleanenv` + `godotenv` и передаются вниз явно.

| Группа | Переменные | Назначение |
| --- | --- | --- |
| App | `APP_NAME`, `APP_ENV` | Имя и окружение сервиса |
| HTTP | `HTTP_ADDR`, `HTTP_READ_TIMEOUT`, `HTTP_WRITE_TIMEOUT`, `HTTP_SHUTDOWN_TIMEOUT` | Адрес и таймауты сервера |
| Log | `LOG_LEVEL` | Уровень логирования (slog) |
| Postgres | `POSTGRES_HOST`, `POSTGRES_PORT`, `POSTGRES_USER`, `POSTGRES_PASSWORD`, `POSTGRES_DB`, `POSTGRES_SSL_MODE` | Подключение к БД |
| Redis | `REDIS_HOST`, `REDIS_PORT`, `REDIS_USERNAME`, `REDIS_PASSWORD`, `REDIS_DB`, `REDIS_CACHE_PREFIX` | Подключение к Redis |
| Redis Sentinel | `REDIS_SENTINEL_ENABLED`, `REDIS_SENTINEL_HOST_*`, `REDIS_SENTINEL_PORT`, `REDIS_SENTINEL_SERVICE`, `REDIS_SENTINEL_PASSWORD` | Опциональный режим Sentinel |
| Kafka | `KAFKA_BROKERS`, `KAFKA_GROUP_ID`, `KAFKA_TOPICS`, `KAFKA_USERNAME`, `KAFKA_PASSWORD` | Брокеры и группа консьюмера |
| APM | `ELASTIC_APM_SERVER_URL`, `ELASTIC_APM_SERVICE_NAME`, `ELASTIC_APM_ENVIRONMENT`, `ELASTIC_APM_SECRET_TOKEN` | Elastic APM (можно оставить пустым локально) |

## Миграции

Миграции лежат в `migrations/` в формате golang-migrate (`NNNNNN_name.up.sql` / `.down.sql`).

- `make up` при старте стека прогоняет их one-shot контейнером `migrate`.
- Вручную: `make migrate-up` / `make migrate-down` (требуют запущенный Postgres;
  DSN переопределяется через `POSTGRES_DSN`).
- Новая: `make migrate-new name=add_orders`.

## Архитектура

Слоёная архитектура с направлением зависимостей строго внутрь (через интерфейсы,
объявленные на стороне потребителя). Подробнее и с диаграммой — в
[docs/architecture.md](docs/architecture.md).

## Использование как шаблона

`arche` — стартовый шаблон. Для нового сервиса:

1. Смените module path в `go.mod` (`github.com/zulfikorramatov/arche`) и поправьте импорты
   (`go mod edit -module <new>` + замена по коду).
2. Переименуйте `APP_NAME` / `ELASTIC_APM_SERVICE_NAME` в `.env.example` и `Makefile`.
3. Замените доменный пример `user` (entity/repository/service/handler/CLI) на свои сущности.
4. Опишите свой API в `generated/api/*.yaml` и выполните `make generate`.
5. Замените начальную миграцию `000001_init` под свою схему.

## Разработка

Перед PR прогоняйте:

```bash
make generate   # если менялась OpenAPI-спека
make fmt
make lint
make test
```
