package postgres

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.elastic.co/apm/module/apmpgxv5/v2"
)

type Pool = pgxpool.Pool

type Config struct {
	Host        string
	Port        int
	User        string
	Password    string
	Database    string
	SSLMode     string
	MaxConns    int32
	MinConns    int32
	ConnTimeout time.Duration
	RetryDelay  time.Duration
	MaxAttempts int
}

func (c Config) DSN() string {
	u := url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(c.User, c.Password),
		Host:   fmt.Sprintf("%s:%d", c.Host, c.Port),
		Path:   c.Database,
	}
	q := u.Query()
	q.Set("sslmode", c.SSLMode)
	u.RawQuery = q.Encode()
	return u.String()
}

func New(ctx context.Context, cfg Config) (*Pool, error) {
	if cfg.MaxAttempts < 1 {
		cfg.MaxAttempts = 1
	}

	poolCfg, err := pgxpool.ParseConfig(cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("parse pool config: %w", err)
	}
	if cfg.MaxConns > 0 {
		poolCfg.MaxConns = cfg.MaxConns
	}
	if cfg.MinConns > 0 {
		poolCfg.MinConns = cfg.MinConns
	}
	apmpgxv5.Instrument(poolCfg.ConnConfig)

	var pool *Pool
	for attempt := 1; attempt <= cfg.MaxAttempts; attempt++ {
		pool, err = connect(ctx, poolCfg, cfg.ConnTimeout)
		if err == nil {
			return pool, nil
		}

		if attempt == cfg.MaxAttempts {
			break
		}

		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("connect postgres: %w", ctx.Err())
		case <-time.After(cfg.RetryDelay):
		}
	}

	return nil, fmt.Errorf("connect postgres after %d attempts: %w", cfg.MaxAttempts, err)
}

func connect(ctx context.Context, poolCfg *pgxpool.Config, timeout time.Duration) (*Pool, error) {
	pingCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(pingCtx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("new pool: %w", err)
	}
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping: %w", err)
	}

	return pool, nil
}
