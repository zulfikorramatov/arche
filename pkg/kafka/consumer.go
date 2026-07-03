package kafka

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/twmb/franz-go/pkg/kgo"
)

type Handler func(ctx context.Context, msg Message) error

type Consumer struct {
	client       *kgo.Client
	maxRetries   int
	retryBackoff time.Duration
}

func NewConsumer(ctx context.Context, cfg Config, opts ...Option) (*Consumer, error) {
	o := defaultOptions()
	for _, opt := range opts {
		opt(&o)
	}

	clientOptions := append(
		clientOpts(cfg, o),
		kgo.ConsumerGroup(cfg.GroupID),
		kgo.ConsumeTopics(cfg.Topics...),
		kgo.DisableAutoCommit(),
	)

	client, err := kgo.NewClient(clientOptions...)
	if err != nil {
		return nil, fmt.Errorf("new consumer client: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, o.dialTimeout)
	defer cancel()

	if err := client.Ping(pingCtx); err != nil {
		client.Close()
		return nil, fmt.Errorf("ping: %w", err)
	}

	return &Consumer{client: client, maxRetries: o.maxRetries, retryBackoff: o.retryBackoff}, nil
}

func (c *Consumer) Run(ctx context.Context, handler Handler) error {
	for {
		fetches := c.client.PollFetches(ctx)
		if fetches.IsClientClosed() {
			return nil
		}

		if ctx.Err() != nil {
			return nil
		}

		if errs := fetches.Errors(); len(errs) > 0 {
			if errors.Is(errs[0].Err, context.Canceled) {
				return nil
			}
			return fmt.Errorf("poll fetches: %w", errs[0].Err)
		}

		iter := fetches.RecordIter()
		for !iter.Done() {
			record := iter.Next()
			msg := Message{Topic: record.Topic, Key: record.Key, Value: record.Value}

			if err := c.handle(ctx, handler, msg); err != nil {
				if ctx.Err() != nil {
					return nil
				}
				return fmt.Errorf("handle message: %w", err)
			}

			if err := c.commit(ctx, record); err != nil {
				return err
			}
		}
	}
}

func (c *Consumer) handle(ctx context.Context, handler Handler, msg Message) error {
	var err error
	for attempt := 0; ; attempt++ {
		if err = handler(ctx, msg); err == nil {
			return nil
		}
		if attempt >= c.maxRetries {
			return fmt.Errorf("after %d retries: %w", c.maxRetries, err)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(c.retryBackoff):
		}
	}
}

func (c *Consumer) commit(ctx context.Context, record *kgo.Record) error {
	if err := c.client.CommitRecords(ctx, record); err != nil {
		return fmt.Errorf("commit record: %w", err)
	}
	return nil
}

func (c *Consumer) Close() {
	c.client.Close()
}
