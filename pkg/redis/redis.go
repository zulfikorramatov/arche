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
	PoolSize     int
	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	KeyPrefix    string

	SentinelEnabled    bool
	SentinelAddrs      []string
	SentinelMasterName string
	SentinelPassword   string
}

func New(ctx context.Context, cfg Config) (*Client, error) {
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

	pingCtx, cancel := context.WithTimeout(ctx, cfg.DialTimeout)
	defer cancel()

	if err := client.Ping(pingCtx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("ping: %w", err)
	}

	return client, nil
}
