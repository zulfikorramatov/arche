package redis

import (
	"context"
	"fmt"
	"time"

	redisprefix "github.com/githubzhaoqian/go-redis-prefix/v9"
	"github.com/redis/go-redis/v9"
)

type Client = redis.Client

var Nil = redis.Nil

type Config struct {
	Host      string
	Port      int
	Username  string
	Password  string
	DB        int
	KeyPrefix string

	SentinelEnabled    bool
	SentinelAddrs      []string
	SentinelMasterName string
	SentinelPassword   string
}

func New(ctx context.Context, cfg Config, opts ...Option) (*Client, error) {
	o := defaultOptions()
	for _, opt := range opts {
		opt(&o)
	}
	if o.maxAttempts < 1 {
		o.maxAttempts = 1
	}

	var client *redis.Client
	if cfg.SentinelEnabled {
		client = redis.NewFailoverClient(&redis.FailoverOptions{
			MasterName:    cfg.SentinelMasterName,
			SentinelAddrs: cfg.SentinelAddrs,
			Username:      cfg.Username,
			Password:      cfg.SentinelPassword,
			DB:            cfg.DB,
			PoolSize:      o.poolSize,
			DialTimeout:   o.dialTimeout,
			ReadTimeout:   o.readTimeout,
			WriteTimeout:  o.writeTimeout,
		})
	} else {
		client = redis.NewClient(&redis.Options{
			Addr:         fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
			Username:     cfg.Username,
			Password:     cfg.Password,
			DB:           cfg.DB,
			PoolSize:     o.poolSize,
			DialTimeout:  o.dialTimeout,
			ReadTimeout:  o.readTimeout,
			WriteTimeout: o.writeTimeout,
		})
	}

	if cfg.KeyPrefix != "" {
		client.AddHook(redisprefix.NewKeyPrefixHook(cfg.KeyPrefix))
	}

	var err error
	for attempt := 1; attempt <= o.maxAttempts; attempt++ {
		err = ping(ctx, client, o.dialTimeout)
		if err == nil {
			return client, nil
		}

		if attempt == o.maxAttempts {
			break
		}

		select {
		case <-ctx.Done():
			_ = client.Close()
			return nil, fmt.Errorf("connect redis: %w", ctx.Err())
		case <-time.After(o.retryDelay):
		}
	}

	_ = client.Close()
	return nil, fmt.Errorf("connect redis after %d attempts: %w", o.maxAttempts, err)
}

func ping(ctx context.Context, client *redis.Client, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("ping: %w", err)
	}

	return nil
}
