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
	Host     string
	Port     int
	User     string
	Password string
	Database string
	SSLMode  string
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

func New(ctx context.Context, cfg Config, opts ...Option) (*Pool, error) {
	o := defaultOptions()
	for _, opt := range opts {
		opt(&o)
	}
	if o.maxAttempts < 1 {
		o.maxAttempts = 1
	}

	poolCfg, err := pgxpool.ParseConfig(cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("parse pool config: %w", err)
	}
	if o.maxConns > 0 {
		poolCfg.MaxConns = o.maxConns
	}
	if o.minConns > 0 {
		poolCfg.MinConns = o.minConns
	}
	apmpgxv5.Instrument(poolCfg.ConnConfig)

	var pool *Pool
	for attempt := 1; attempt <= o.maxAttempts; attempt++ {
		pool, err = connect(ctx, poolCfg, o.connTimeout)
		if err == nil {
			return pool, nil
		}

		if attempt == o.maxAttempts {
			break
		}

		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("connect postgres: %w", ctx.Err())
		case <-time.After(o.retryDelay):
		}
	}

	return nil, fmt.Errorf("connect postgres after %d attempts: %w", o.maxAttempts, err)
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
