package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"go.uber.org/zap"

	"github.com/zulfikorramatov/arche/internal/config"
	httpserver "github.com/zulfikorramatov/arche/internal/http"
	"github.com/zulfikorramatov/arche/internal/http/handler"
	"github.com/zulfikorramatov/arche/internal/repository"
	"github.com/zulfikorramatov/arche/internal/service"
	"github.com/zulfikorramatov/arche/pkg/logger"
	"github.com/zulfikorramatov/arche/pkg/postgres"
	"github.com/zulfikorramatov/arche/pkg/redis"
)

func Run(ctx context.Context, cfg *config.Config) error {
	log, err := logger.New(logger.Config(cfg.Logger))
	if err != nil {
		return fmt.Errorf("new logger: %w", err)
	}
	defer func() { _ = log.Sync() }()

	pg, err := postgres.New(ctx, postgres.Config(cfg.Postgres))
	if err != nil {
		return fmt.Errorf("new postgres: %w", err)
	}
	defer pg.Close()

	rdb, err := redis.New(ctx, redis.Config(cfg.Redis))
	if err != nil {
		return fmt.Errorf("new redis: %w", err)
	}
	defer func() { _ = rdb.Close() }()

	userRepo := repository.NewUserRepository(pg)
	userSvc := service.NewUserService(userRepo, rdb)
	userHandler := handler.NewUserHandler(userSvc, log)

	router := httpserver.NewRouter(log, userHandler)

	srv := &http.Server{
		Addr:         cfg.HTTP.Addr,
		Handler:      router,
		ReadTimeout:  cfg.HTTP.ReadTimeout,
		WriteTimeout: cfg.HTTP.WriteTimeout,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Info("http server starting", zap.String("addr", cfg.HTTP.Addr))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		log.Info("shutdown signal received")
	case err := <-errCh:
		log.Error("http server error", zap.Error(err))
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.HTTP.ShutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}
	log.Info("http server stopped")
	return nil
}
