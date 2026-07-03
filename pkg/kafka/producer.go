package kafka

import (
	"context"
	"fmt"

	"github.com/twmb/franz-go/pkg/kgo"
)

type Producer struct {
	client *kgo.Client
}

func NewProducer(ctx context.Context, cfg Config, opts ...Option) (*Producer, error) {
	o := defaultOptions()
	for _, opt := range opts {
		opt(&o)
	}

	clientOptions := append(
		clientOpts(cfg, o),
		kgo.RequiredAcks(kgo.AllISRAcks()),
		kgo.AllowAutoTopicCreation(),
	)

	client, err := kgo.NewClient(clientOptions...)
	if err != nil {
		return nil, fmt.Errorf("new producer client: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, o.dialTimeout)
	defer cancel()

	if err := client.Ping(pingCtx); err != nil {
		client.Close()
		return nil, fmt.Errorf("ping: %w", err)
	}

	return &Producer{client: client}, nil
}

func (p *Producer) Produce(ctx context.Context, topic string, key, value []byte) error {
	record := &kgo.Record{Topic: topic, Key: key, Value: value}
	if err := p.client.ProduceSync(ctx, record).FirstErr(); err != nil {
		return fmt.Errorf("produce: %w", err)
	}
	return nil
}

func (p *Producer) Close() {
	p.client.Close()
}
