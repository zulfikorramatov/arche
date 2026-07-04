package app

import (
	"fmt"

	"github.com/zulfikorramatov/arche/internal/config"
	"github.com/zulfikorramatov/arche/pkg/postgres"
	"github.com/zulfikorramatov/arche/pkg/redis"
)

func buildPostgresConfig(cfg config.PostgresConfig) postgres.Config {
	return postgres.Config{
		Host:     cfg.Host,
		Port:     cfg.Port,
		User:     cfg.User,
		Password: cfg.Password,
		Database: cfg.Database,
		SSLMode:  cfg.SSLMode,
	}
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
		KeyPrefix:          cfg.KeyPrefix,
		SentinelEnabled:    cfg.SentinelEnabled,
		SentinelAddrs:      sentinelAddrs,
		SentinelMasterName: cfg.SentinelMasterName,
		SentinelPassword:   cfg.SentinelPassword,
	}
}
