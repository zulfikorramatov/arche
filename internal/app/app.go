package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"go.elastic.co/apm/v2"

	"github.com/zulfikorramatov/arche/internal/config"
	httpserver "github.com/zulfikorramatov/arche/internal/http"
	"github.com/zulfikorramatov/arche/internal/http/handler"
	"github.com/zulfikorramatov/arche/internal/repository"
	"github.com/zulfikorramatov/arche/internal/service"
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

	userRepo := repository.NewUserRepository(pg)
	userSvc := service.NewUserService(userRepo)

	srv, err := newHttpServer(cfg, log, userSvc)
	if err != nil {
		return fmt.Errorf("new http server: %w", err)
	}

	errCh := make(chan error, 1)
	go func() {
		log.Info("http server starting", "addr", cfg.HTTP.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		log.Info("shutdown signal received")
	case err := <-errCh:
		log.Error("http server error", "error", err)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.HTTP.ShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}

	apm.DefaultTracer().Flush(nil)
	log.Info("http server stopped")

	return nil
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
