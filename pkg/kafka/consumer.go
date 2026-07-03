package kafka

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/twmb/franz-go/pkg/kgo"
)

const defaultRetryBackoff = time.Second

// Handler processes a single consumed message. Returning an error triggers a
// bounded retry (see Config.MaxRetries); if every attempt fails the error is
// fatal and Run returns it. Handlers must be idempotent — at-least-once means
// a message can be redelivered (on retry, or after a pod restart).
type Handler func(ctx context.Context, msg Message) error

// Consumer is an at-least-once Kafka consumer group reader. Auto-commit is
// disabled; offsets are committed only after the handler succeeds, so a crash
// mid-processing redelivers the message instead of losing it.
type Consumer struct {
	client       *kgo.Client
	maxRetries   int
	retryBackoff time.Duration
}

func NewConsumer(ctx context.Context, cfg Config) (*Consumer, error) {
	opts := append(
		clientOpts(cfg),
		kgo.ConsumerGroup(cfg.GroupID),
		kgo.ConsumeTopics(cfg.Topics...),
		kgo.DisableAutoCommit(),
	)

	client, err := kgo.NewClient(opts...)
	if err != nil {
		return nil, fmt.Errorf("new consumer client: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, cfg.DialTimeout)
	defer cancel()

	if err := client.Ping(pingCtx); err != nil {
		client.Close()
		return nil, fmt.Errorf("ping: %w", err)
	}

	backoff := cfg.RetryBackoff
	if backoff <= 0 {
		backoff = defaultRetryBackoff
	}

	return &Consumer{client: client, maxRetries: cfg.MaxRetries, retryBackoff: backoff}, nil
}

// Run polls and dispatches messages until ctx is canceled (clean shutdown,
// returns nil). Each record is committed only after the handler succeeds. A
// message that still fails after MaxRetries — or an unrecoverable client/poll
// error — is returned as a fatal error so the caller can fail the pod and let
// Kubernetes restart it; the surviving pods cover the partitions via rebalance.
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

		var processed []*kgo.Record
		iter := fetches.RecordIter()
		for !iter.Done() {
			record := iter.Next()
			msg := Message{Topic: record.Topic, Key: record.Key, Value: record.Value}

			if err := c.handle(ctx, handler, msg); err != nil {
				if ctx.Err() != nil {
					return nil // canceled mid-retry: clean shutdown, not fatal
				}
				_ = c.commit(ctx, processed) // best-effort: don't reprocess what already succeeded
				return fmt.Errorf("handle message: %w", err)
			}
			processed = append(processed, record)
		}

		if err := c.commit(ctx, processed); err != nil {
			return err
		}
	}
}

// handle runs the handler with bounded retries and backoff. It returns nil on
// success, ctx.Err() if cancelled while waiting, or the last handler error once
// the retries are exhausted.
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

func (c *Consumer) commit(ctx context.Context, records []*kgo.Record) error {
	if len(records) == 0 {
		return nil
	}
	if err := c.client.CommitRecords(ctx, records...); err != nil {
		return fmt.Errorf("commit records: %w", err)
	}
	return nil
}

func (c *Consumer) Close() {
	c.client.Close()
}
