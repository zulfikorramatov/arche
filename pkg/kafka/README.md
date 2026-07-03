# pkg/kafka

Обёртка над [franz-go](https://github.com/twmb/franz-go) с двумя гарантиями:

- **Producer — гарантированная доставка.** `acks=all` + идемпотентный producer + синхронная отправка.
  Вызов `Produce` возвращается только после подтверждения брокером; если подтверждения нет — вернётся
  ошибка. Дубли при ретраях брокер отбрасывает сам (идемпотентность включена по умолчанию).
- **Consumer — без потерь (at-least-once).** Авто-коммит **выключен**. Offset коммитится **только
  после** успешной обработки сообщения. Если сервис упал между обработкой и коммитом — сообщение
  будет прочитано заново.

> ⚠️ **Обработчик обязан быть идемпотентным.** At-least-once допускает повторную доставку (дубли),
> поэтому обработка одного и того же сообщения дважды не должна ломать данные (например
> `INSERT ... ON CONFLICT DO NOTHING` или проверка «уже обработано»).

Пакет самодостаточен: он не знает про `internal/*`, env-переменные и `.env` — вызывающий код
заполняет `Config` и передаёт его внутрь (как `pkg/postgres` и `pkg/redis`).

## Config

```go
type Config struct {
    Brokers      []string      // адреса брокеров, напр. []string{"localhost:19092"}
    GroupID      string        // consumer group id
    Topics       []string      // топики для consumer
    DialTimeout  time.Duration // таймаут подключения и ping
    Username     string        // опционально: SASL/SCRAM-SHA-512
    Password     string
    MaxRetries   int           // consumer: ретраев на сообщение до фатальной ошибки
    RetryBackoff time.Duration // consumer: задержка между ретраями
}
```

`Config` зеркалится один-в-один структурой `config.KafkaConfig` в `internal/config`, поэтому в
`app.go` используется прямая конвертация `kafka.Config(cfg.Kafka)` — если поля разойдутся, проект
**не соберётся** (это и есть защита от рассинхрона).

## Producer

```go
producer, err := kafka.NewProducer(ctx, kafka.Config(cfg.Kafka))
if err != nil {
    return fmt.Errorf("new kafka producer: %w", err)
}
defer producer.Close()

// Синхронная отправка: ошибка возвращается, если брокер не подтвердил запись.
if err := producer.Produce(ctx, "arche.example", []byte("user-42"), payload); err != nil {
    return fmt.Errorf("publish event: %w", err)
}
```

В реальном сервисе продюсер обычно дёргается из бизнес-слоя — например, опубликовать `user.created`
после создания пользователя. Producer потокобезопасен, его можно держать одним инстансом на сервис.

## Consumer

```go
consumer, err := kafka.NewConsumer(ctx, kafka.Config(cfg.Kafka))
if err != nil {
    return fmt.Errorf("new kafka consumer: %w", err)
}
defer consumer.Close()

// Run блокируется до отмены ctx. Запускать в отдельной горутине.
err = consumer.Run(ctx, func(ctx context.Context, msg kafka.Message) error {
    // обработка; верните nil — offset закоммитится,
    // верните ошибку — сообщение НЕ коммитится и будет доставлено снова.
    return process(ctx, msg)
})
```

Логику коммита делает сам `Run`: после успешной обработки сообщение коммитится. Если обработчик
вернул ошибку, сообщение повторяется до `MaxRetries` раз с задержкой `RetryBackoff`.

### Обработка ошибок и Kubernetes (fail-fast)

Поведение рассчитано на несколько подов в одной consumer group:

- **Транзиентная ошибка** обработчика → `Run` ретраит сообщение `MaxRetries` раз с backoff. Чаще
  всего этого достаточно (БД моргнула, downstream поднялся).
- **Ретраи исчерпаны** или **неустранимая ошибка клиента/poll** → `Run` возвращает **фатальную
  ошибку**. Вызывающий код (`app.go`) отдаёт её в `errCh`, сервис гасится gracefully и выходит с
  ненулевым кодом. Kubernetes перезапускает под, а его партиции на время рестарта покрывают
  остальные поды группы через **rebalance**.
- **Отмена `ctx`** (обычный graceful shutdown) → `Run` возвращает `nil`, без ошибки.

> ⚠️ Это **fail-fast**, а не dead-letter queue. «Ядовитое» сообщение (которое не обработается
> никогда) будет валить под в цикле (CrashLoopBackOff), пока не починят обработчик или не уберут
> сообщение. Если такой риск критичен — следующий шаг — DLQ-топик: после исчерпания ретраев слать
> сообщение в `<topic>.dlq` и коммитить, вместо падения.

### Как отключить consumer

Consumer стартует в `app.go` только если задан хотя бы один топик. Оставьте `KAFKA_TOPICS` пустым —
consumer не запустится (producer при этом доступен).

## Локальный запуск

В `docker-compose.yml` поднят брокер [Redpanda](https://redpanda.com) (Kafka-совместимый, без
ZooKeeper). Слушатели:

- внутри docker-сети — `redpanda:9092`;
- с хоста — `localhost:19092` (используется при `make run`, см. `.env.example`).

```sh
docker compose up -d redpanda   # поднять брокер
make run                        # запустить сервис на хосте
```

При старте сервис в качестве демо публикует одно сообщение `service.started` в первый топик, и
consumer его логирует — так сразу видно, что цепочка producer → consumer работает. Прочитать топик
вручную:

```sh
docker exec arche-redpanda rpk topic consume arche.example --num 1
```

## Чего здесь нет (и почему)

Реализован **at-least-once**, без транзакций и **exactly-once (EOS)**. EOS в Kafka покрывает только
цепочку **Kafka → Kafka** (атомарно «запись в топик + коммит offset» через транзакции). Как только
результат уходит во внешнюю систему (Postgres, Redis, отправка письма), транзакции брокера не
помогают — «ровно один раз» там достигается идемпотентностью приёмника или паттерном
transactional outbox. Если конкретный сценарий потребует EOS, его можно добавить поверх этой базы.
