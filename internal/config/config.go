package config

import (
	"fmt"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/joho/godotenv"
)

type Config struct {
	App      AppConfig
	HTTP     HTTPConfig
	Logger   LoggerConfig
	Postgres PostgresConfig
	Redis    RedisConfig
	Kafka    KafkaConfig
}

type AppConfig struct {
	Name string `env:"APP_NAME"`
	Env  string `env:"APP_ENV"  env-default:"local"`
}

type HTTPConfig struct {
	Addr            string        `env:"HTTP_ADDR"             env-default:":8080"`
	ReadTimeout     time.Duration `env:"HTTP_READ_TIMEOUT"     env-default:"10s"`
	WriteTimeout    time.Duration `env:"HTTP_WRITE_TIMEOUT"    env-default:"10s"`
	ShutdownTimeout time.Duration `env:"HTTP_SHUTDOWN_TIMEOUT" env-default:"15s"`
}

type LoggerConfig struct {
	Level string `env:"LOG_LEVEL" env-default:"info"`
}

type PostgresConfig struct {
	Host        string        `env:"POSTGRES_HOST"          env-default:"localhost"`
	Port        int           `env:"POSTGRES_PORT"          env-default:"5432"`
	User        string        `env:"POSTGRES_USER"          env-default:"postgres"`
	Password    string        `env:"POSTGRES_PASSWORD"`
	Database    string        `env:"POSTGRES_DB"            env-default:"app"`
	SSLMode     string        `env:"POSTGRES_SSL_MODE"      env-default:"disable"`
	MaxConns    int32         `env:"POSTGRES_MAX_CONNS"     env-default:"10"`
	MinConns    int32         `env:"POSTGRES_MIN_CONNS"     env-default:"1"`
	ConnTimeout time.Duration `env:"POSTGRES_CONN_TIMEOUT"  env-default:"5s"`
	RetryDelay  time.Duration `env:"POSTGRES_RETRY_DELAY"   env-default:"5s"`
	MaxAttempts int           `env:"POSTGRES_MAX_ATTEMPTS"  env-default:"5"`
}

type RedisConfig struct {
	Host         string        `env:"REDIS_HOST"          env-default:"localhost"`
	Port         int           `env:"REDIS_PORT"          env-default:"6379"`
	Username     string        `env:"REDIS_USERNAME"`
	Password     string        `env:"REDIS_PASSWORD"`
	DB           int           `env:"REDIS_DB"            env-default:"0"`
	PoolSize     int           `env:"REDIS_POOL_SIZE"     env-default:"10"`
	DialTimeout  time.Duration `env:"REDIS_DIAL_TIMEOUT"  env-default:"1s"`
	ReadTimeout  time.Duration `env:"REDIS_READ_TIMEOUT"  env-default:"3s"`
	WriteTimeout time.Duration `env:"REDIS_WRITE_TIMEOUT" env-default:"3s"`
	KeyPrefix    string        `env:"REDIS_CACHE_PREFIX"`

	SentinelEnabled    bool   `env:"REDIS_SENTINEL_ENABLED"  env-default:"false"`
	SentinelHost1      string `env:"REDIS_SENTINEL_HOST_1"`
	SentinelHost2      string `env:"REDIS_SENTINEL_HOST_2"`
	SentinelHost3      string `env:"REDIS_SENTINEL_HOST_3"`
	SentinelPort       int    `env:"REDIS_SENTINEL_PORT"     env-default:"26379"`
	SentinelMasterName string `env:"REDIS_SENTINEL_SERVICE"  env-default:"mymaster"`
	SentinelPassword   string `env:"REDIS_SENTINEL_PASSWORD"`
}

type KafkaConfig struct {
	Brokers      []string      `env:"KAFKA_BROKERS"       env-separator:"," env-default:"localhost:19092"`
	GroupID      string        `env:"KAFKA_GROUP_ID"      env-default:"arche"`
	Topics       []string      `env:"KAFKA_TOPICS"        env-separator:"," env-default:"arche.example"`
	DialTimeout  time.Duration `env:"KAFKA_DIAL_TIMEOUT"  env-default:"10s"`
	Username     string        `env:"KAFKA_USERNAME"`
	Password     string        `env:"KAFKA_PASSWORD"`
	MaxRetries   int           `env:"KAFKA_MAX_RETRIES"   env-default:"3"`
	RetryBackoff time.Duration `env:"KAFKA_RETRY_BACKOFF" env-default:"2s"`
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	var cfg Config
	if err := cleanenv.ReadEnv(&cfg); err != nil {
		return nil, fmt.Errorf("read env: %w", err)
	}

	return &cfg, nil
}
