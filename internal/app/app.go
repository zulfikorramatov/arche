package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/zulfikorramatov/arche/pkg/redis"
	"go.elastic.co/apm/v2"

	"github.com/zulfikorramatov/arche/internal/config"
	httpserver "github.com/zulfikorramatov/arche/internal/http"
	"github.com/zulfikorramatov/arche/internal/http/handler"
	"github.com/zulfikorramatov/arche/internal/repository"
	"github.com/zulfikorramatov/arche/internal/service"
	"github.com/zulfikorramatov/arche/pkg/kafka"
	"github.com/zulfikorramatov/arche/pkg/logger"
	"github.com/zulfikorramatov/arche/pkg/postgres"
)

func Run(ctx context.Context, cfg *config.Config) error {
	log, err := logger.New(logger.Config(cfg.Logger))
	if err != nil {
		return fmt.Errorf("new logger: %w", err)
	}

	pg, err := postgres.New(ctx, postgres.Config(cfg.Postgres))
	if err != nil {
		return fmt.Errorf("new postgres: %w", err)
	}
	defer pg.Close()

	rdb, err := redis.New(ctx, buildRedisConfig(cfg.Redis))
	if err != nil {
		return fmt.Errorf("new redis: %w", err)
	}
	defer func() { _ = rdb.Close() }()

	producer, err := kafka.NewProducer(ctx, kafka.Config(cfg.Kafka))
	if err != nil {
		return fmt.Errorf("new kafka producer: %w", err)
	}
	defer producer.Close()

	// errCh collects fatal errors from the background workers (http server and
	// kafka consumer); either one triggers graceful shutdown and a non-zero exit.
	errCh := make(chan error, 2)

	if err := startKafkaConsumer(ctx, cfg, log, producer, errCh); err != nil {
		return err
	}

	userRepo := repository.NewUserRepository(pg)
	userSvc := service.NewUserService(userRepo)

	srv, err := newHttpServer(cfg, log, userSvc)
	if err != nil {
		return fmt.Errorf("new http server: %w", err)
	}

	go func() {
		log.Info("http server starting", "addr", cfg.HTTP.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("http server: %w", err)
		}
	}()

	var runErr error
	select {
	case <-ctx.Done():
		log.Info("shutdown signal received")
	case err := <-errCh:
		log.Error("fatal worker error, shutting down", "error", err)
		runErr = err
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.HTTP.ShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}

	apm.DefaultTracer().Flush(nil)
	log.Info("http server stopped")

	return runErr
}

// startKafkaConsumer starts the at-least-once consumer in the background when
// at least one topic is configured (empty KAFKA_TOPICS disables it). A fatal
// consumer error (retries exhausted, or an unrecoverable client error) is sent
// to errCh so the whole service shuts down and exits non-zero — Kubernetes then
// restarts the pod and the partitions rebalance across the survivors. A clean
// ctx cancellation does not produce an error.
//
// It also emits one demo message so a local run immediately shows the
// produce→consume flow; replace this with a real producer call from your service.
func startKafkaConsumer(ctx context.Context, cfg *config.Config, log *logger.Logger, producer *kafka.Producer, errCh chan<- error) error {
	if len(cfg.Kafka.Topics) == 0 {
		return nil
	}

	consumer, err := kafka.NewConsumer(ctx, kafka.Config(cfg.Kafka))
	if err != nil {
		return fmt.Errorf("new kafka consumer: %w", err)
	}

	go func() {
		defer consumer.Close()

		handler := func(ctx context.Context, msg kafka.Message) error {
			log.InfoContext(ctx, "kafka message received",
				"topic", msg.Topic,
				"key", string(msg.Key),
				"value", string(msg.Value),
			)
			return nil
		}

		if err := consumer.Run(ctx, handler); err != nil {
			errCh <- fmt.Errorf("kafka consumer: %w", err)
		}
	}()

	if err := producer.Produce(ctx, cfg.Kafka.Topics[0], []byte("service"), []byte("service.started")); err != nil {
		log.Error("kafka demo produce failed", "error", err)
	}

	return nil
}

func buildRedisConfig(cfg config.RedisConfig) redis.Config {
	var sentinelAddrs []string
	if cfg.SentinelEnabled {
		port := fmt.Sprintf("%d", cfg.SentinelPort)
		for _, host := range []string{cfg.SentinelHost1, cfg.SentinelHost2, cfg.SentinelHost3} {
			if host != "" {
				sentinelAddrs = append(sentinelAddrs, host+":"+port)
			}
		}
	}

	return redis.Config{
		Host:               cfg.Host,
		Port:               cfg.Port,
		Username:           cfg.Username,
		Password:           cfg.Password,
		DB:                 cfg.DB,
		PoolSize:           cfg.PoolSize,
		DialTimeout:        cfg.DialTimeout,
		ReadTimeout:        cfg.ReadTimeout,
		WriteTimeout:       cfg.WriteTimeout,
		KeyPrefix:          cfg.KeyPrefix,
		SentinelEnabled:    cfg.SentinelEnabled,
		SentinelAddrs:      sentinelAddrs,
		SentinelMasterName: cfg.SentinelMasterName,
		SentinelPassword:   cfg.SentinelPassword,
	}
}

func newHttpServer(
	cfg *config.Config,
	log *logger.Logger,
	userSvc *service.UserService,
) (*http.Server, error) {
	server := handler.NewServer(log, userSvc)

	router, err := httpserver.NewRouter(log, server, userSvc)
	if err != nil {
		return nil, fmt.Errorf("new router: %w", err)
	}

	return &http.Server{
		Addr:         cfg.HTTP.Addr,
		Handler:      router,
		ReadTimeout:  cfg.HTTP.ReadTimeout,
		WriteTimeout: cfg.HTTP.WriteTimeout,
	}, nil
}
