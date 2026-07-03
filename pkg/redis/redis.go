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
	Host         string
	Port         int
	Username     string
	Password     string
	DB           int
	KeyPrefix    string
	PoolSize     int
	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	RetryDelay   time.Duration
	MaxAttempts  int

	SentinelEnabled    bool
	SentinelAddrs      []string
	SentinelMasterName string
	SentinelPassword   string
}

func New(ctx context.Context, cfg Config) (*Client, error) {
	if cfg.MaxAttempts < 1 {
		cfg.MaxAttempts = 1
	}

	var client *redis.Client
	if cfg.SentinelEnabled {
		client = redis.NewFailoverClient(&redis.FailoverOptions{
			MasterName:    cfg.SentinelMasterName,
			SentinelAddrs: cfg.SentinelAddrs,
			Username:      cfg.Username,
			Password:      cfg.SentinelPassword,
			DB:            cfg.DB,
			PoolSize:      cfg.PoolSize,
			DialTimeout:   cfg.DialTimeout,
			ReadTimeout:   cfg.ReadTimeout,
			WriteTimeout:  cfg.WriteTimeout,
		})
	} else {
		client = redis.NewClient(&redis.Options{
			Addr:         fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
			Username:     cfg.Username,
			Password:     cfg.Password,
			DB:           cfg.DB,
			PoolSize:     cfg.PoolSize,
			DialTimeout:  cfg.DialTimeout,
			ReadTimeout:  cfg.ReadTimeout,
			WriteTimeout: cfg.WriteTimeout,
		})
	}

	if cfg.KeyPrefix != "" {
		client.AddHook(redisprefix.NewKeyPrefixHook(cfg.KeyPrefix))
	}

	var err error
	for attempt := 1; attempt <= cfg.MaxAttempts; attempt++ {
		err = ping(ctx, client, cfg.DialTimeout)
		if err == nil {
			return client, nil
		}

		if attempt == cfg.MaxAttempts {
			break
		}

		select {
		case <-ctx.Done():
			_ = client.Close()
			return nil, fmt.Errorf("connect redis: %w", ctx.Err())
		case <-time.After(cfg.RetryDelay):
		}
	}

	_ = client.Close()
	return nil, fmt.Errorf("connect redis after %d attempts: %w", cfg.MaxAttempts, err)
}

func ping(ctx context.Context, client *redis.Client, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("ping: %w", err)
	}

	return nil
}
