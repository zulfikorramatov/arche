# arche

Шаблон для Go-микросервисов. Использует [chi](https://github.com/go-chi/chi),
[zap](https://github.com/uber-go/zap), [pgx](https://github.com/jackc/pgx)
и [go-redis](https://github.com/redis/go-redis), связанные через
конфиг-ориентированный `internal/app`.

## Структура проекта

```
.
├── cmd/app/            # main.go — загрузка конфига, signal context, вызов app.Run
├── internal/
│   ├── app/            # composition root: сборка зависимостей, запуск HTTP-сервера
│   ├── config/         # агрегированный Config (HTTP + pkg-конфиги)
│   ├── domain/         # сущности и доменные ошибки
│   ├── repository/     # postgres-репозитории
│   ├── service/        # бизнес-логика, использует repo + cache
│   └── http/           # роутер, хэндлеры, middleware
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
curl localhost:8080/healthz
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

## Клонирование для нового сервиса

1. `git clone` → `rm -rf .git && git init`
2. Переименовать модуль — заменить `github.com/zulfikorramatov/arche` на новый путь:
   ```sh
   grep -rl "github.com/zulfikorramatov/arche" . --include="*.go" --include="go.mod" | xargs sed -i '' 's|github.com/zulfikorramatov/arche|github.com/your-org/your-service|g'
   ```
3. Переименовать контейнеры и `APP_NAME` в `docker-compose.yml` / `.env`
4. Заменить домен `user` (service/repository/handler) на свой
5. `make tidy && make up`

## Стиль кода

Конвенции описаны в [STYLEGUIDE.md](STYLEGUIDE.md).
