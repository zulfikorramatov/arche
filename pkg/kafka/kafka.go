// Package kafka wraps franz-go behind Config-driven constructors for a
// durable Producer and an at-least-once Consumer. The package is independent
// of the rest of the project: it has no knowledge of env vars or .env files —
// callers populate Config and pass it in, so it stays copy-pasteable.
//
// Producer guarantees delivery: acks=all + idempotent writes + synchronous
// Produce. Consumer never loses messages: auto-commit is disabled and offsets
// are committed only after the handler succeeds (at-least-once). See README.md.
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
	Username     string // optional SASL/SCRAM-SHA-512
	Password     string
	MaxRetries   int           // consumer: per-message retries before the error is treated as fatal
	RetryBackoff time.Duration // consumer: delay between retries
}

// Message is the decoded record handed to a consumer handler and the payload
// accepted by the producer.
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
