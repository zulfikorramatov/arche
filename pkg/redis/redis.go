// Package redis wraps go-redis with a Config-driven constructor. The package
// is independent of the rest of the project: it has no knowledge of env vars
// or .env files — callers populate Config and pass it in. Client and Nil are
// re-exported so callers can keep importing only this package.
package redis

import (
	"context"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

type Client = goredis.Client

// Nil mirrors goredis.Nil so callers can compare errors without importing
// the upstream package.
var Nil = goredis.Nil

type Config struct {
	Host        string
	Port        int
	Password    string
	DB          int
	PoolSize    int
	DialTimeout time.Duration
}

func New(ctx context.Context, cfg Config) (*Client, error) {
	client := goredis.NewClient(&goredis.Options{
		Addr:        fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password:    cfg.Password,
		DB:          cfg.DB,
		PoolSize:    cfg.PoolSize,
		DialTimeout: cfg.DialTimeout,
	})

	pingCtx, cancel := context.WithTimeout(ctx, cfg.DialTimeout)
	defer cancel()
	if err := client.Ping(pingCtx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("ping: %w", err)
	}
	return client, nil
}
