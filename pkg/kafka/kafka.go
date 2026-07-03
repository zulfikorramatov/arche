package kafka

import (
	"time"

	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/pkg/sasl/scram"
)

type Config struct {
	Brokers      []string
	GroupID      string
	Topics       []string
	DialTimeout  time.Duration
	Username     string
	Password     string
	MaxRetries   int
	RetryBackoff time.Duration
}

type Message struct {
	Topic string
	Key   []byte
	Value []byte
}

// clientOpts builds the options shared by the producer and consumer: seed
// brokers, dial timeout and optional SASL/SCRAM-SHA-512 auth.
func clientOpts(cfg Config) []kgo.Opt {
	opts := []kgo.Opt{
		kgo.SeedBrokers(cfg.Brokers...),
		kgo.DialTimeout(cfg.DialTimeout),
	}

	if cfg.Username != "" && cfg.Password != "" {
		auth := scram.Auth{User: cfg.Username, Pass: cfg.Password}
		opts = append(opts, kgo.SASL(auth.AsSha512Mechanism()))
	}

	return opts
}
