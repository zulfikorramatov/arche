package kafka

import (
	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/pkg/sasl/scram"
)

type Config struct {
	Brokers  []string
	GroupID  string
	Topics   []string
	Username string
	Password string
}

type Message struct {
	Topic string
	Key   []byte
	Value []byte
}

func clientOpts(cfg Config, o options) []kgo.Opt {
	opts := []kgo.Opt{
		kgo.SeedBrokers(cfg.Brokers...),
		kgo.DialTimeout(o.dialTimeout),
	}

	if cfg.Username != "" && cfg.Password != "" {
		auth := scram.Auth{User: cfg.Username, Pass: cfg.Password}
		opts = append(opts, kgo.SASL(auth.AsSha512Mechanism()))
	}

	return opts
}
