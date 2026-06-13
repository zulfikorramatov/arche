package config

import (
	"fmt"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/joho/godotenv"
)

// Config aggregates all configuration for the service. The sub-structs here
// own the env tags and are the ONLY place env names live. /pkg structs are
// plain Go types — see app.go for the env→pkg mapping.
//
// PostgresConfig and RedisConfig mirror their /pkg counterparts exactly so
// app.go can use explicit struct conversion (e.g. `postgres.Config(cfg.Postgres)`);
// drift is caught at build time. LoggerConfig does not include AppName —
// that field is sourced from AppConfig.Name when constructing logger.Config
// in app.go.
type Config struct {
	App      AppConfig
	HTTP     HTTPConfig
	Logger   LoggerConfig
	Postgres PostgresConfig
	Redis    RedisConfig
}

type AppConfig struct {
	Name string `env:"APP_NAME" env-default:"arche"`
	Env  string `env:"APP_ENV"  env-default:"dev"`
}

type HTTPConfig struct {
	Addr            string        `env:"HTTP_ADDR" env-default:":8080"`
	ReadTimeout     time.Duration `env:"HTTP_READ_TIMEOUT" env-default:"10s"`
	WriteTimeout    time.Duration `env:"HTTP_WRITE_TIMEOUT" env-default:"10s"`
	ShutdownTimeout time.Duration `env:"HTTP_SHUTDOWN_TIMEOUT" env-default:"15s"`
}

type LoggerConfig struct {
	Level  string `env:"LOG_LEVEL"  env-default:"info"`
	Format string `env:"LOG_FORMAT" env-default:"json"`
}

type PostgresConfig struct {
	Host        string        `env:"POSTGRES_HOST"         env-default:"localhost"`
	Port        int           `env:"POSTGRES_PORT"         env-default:"5432"`
	User        string        `env:"POSTGRES_USER"         env-default:"postgres"`
	Password    string        `env:"POSTGRES_PASSWORD"`
	Database    string        `env:"POSTGRES_DB"           env-default:"app"`
	SSLMode     string        `env:"POSTGRES_SSL_MODE"     env-default:"disable"`
	MaxConns    int32         `env:"POSTGRES_MAX_CONNS"    env-default:"10"`
	MinConns    int32         `env:"POSTGRES_MIN_CONNS"    env-default:"1"`
	ConnTimeout time.Duration `env:"POSTGRES_CONN_TIMEOUT" env-default:"5s"`
}

type RedisConfig struct {
	Host        string        `env:"REDIS_HOST"         env-default:"localhost"`
	Port        int           `env:"REDIS_PORT"         env-default:"6379"`
	Password    string        `env:"REDIS_PASSWORD"`
	DB          int           `env:"REDIS_DB"           env-default:"0"`
	PoolSize    int           `env:"REDIS_POOL_SIZE"    env-default:"10"`
	DialTimeout time.Duration `env:"REDIS_DIAL_TIMEOUT" env-default:"5s"`
}

// Load reads variables from .env (if present in the working directory) and
// then from the process environment. In containers .env is usually absent
// and the runtime supplies the variables directly — both paths work.
func Load() (*Config, error) {
	_ = godotenv.Load() // optional; ignore "no such file"

	var cfg Config
	if err := cleanenv.ReadEnv(&cfg); err != nil {
		return nil, fmt.Errorf("read env: %w", err)
	}
	return &cfg, nil
}
