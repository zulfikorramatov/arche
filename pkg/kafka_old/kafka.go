package kafkaold

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/sasl/scram"
)

type Kafka struct {
	reader *kafka.Reader
	writer *kafka.Writer
}

func New(brokers []string, groupID, topic string, opts ...Option) (*Kafka, error) {
	options := &kafkaOptions{
		readTimeout:  10 * time.Second,
		writeTimeout: 10 * time.Second,
	}

	for _, opt := range opts {
		opt(options)
	}

	config := kafka.ReaderConfig{
		Brokers:          brokers,
		GroupID:          groupID,
		Topic:            topic,
		MinBytes:         10e3,
		MaxBytes:         10e6,
		StartOffset:      kafka.FirstOffset,
		MaxWait:          500 * time.Millisecond,
		ReadBackoffMin:   time.Second,
		ReadBackoffMax:   5 * time.Second,
		CommitInterval:   time.Second,
		SessionTimeout:   30 * time.Second,
		RebalanceTimeout: 30 * time.Second,
		RetentionTime:    24 * time.Hour,
		MaxAttempts:      5,
		GroupBalancers: []kafka.GroupBalancer{
			kafka.RangeGroupBalancer{},
		},
	}

	if options.username != "" && options.password != "" {
		mechanism, err := scram.Mechanism(scram.SHA512, options.username, options.password)
		if err != nil {
			return nil, fmt.Errorf("failed to create SASL mechanism: %w", err)
		}

		dialer := &kafka.Dialer{
			Timeout:       10 * time.Second,
			DualStack:     true,
			SASLMechanism: mechanism,
			TLS:           nil,
			ClientID:      "your-client-id",
		}

		config.Dialer = dialer
	}

	reader := kafka.NewReader(config)

	writer := &kafka.Writer{
		Addr:                   kafka.TCP(brokers...),
		Topic:                  topic,
		Balancer:               &kafka.LeastBytes{},
		BatchSize:              100,
		BatchTimeout:           time.Millisecond,
		ReadTimeout:            options.readTimeout,
		WriteTimeout:           options.writeTimeout,
		RequiredAcks:           kafka.RequireOne,
		Async:                  true,
		AllowAutoTopicCreation: true,
	}

	if options.username != "" && options.password != "" {
		mechanism, err := scram.Mechanism(scram.SHA512, options.username, options.password)
		if err != nil {
			return nil, fmt.Errorf("failed to create SASL mechanism: %w", err)
		}
		transport := &kafka.Transport{
			SASL: mechanism,
			TLS:  nil,
		}
		writer.Transport = transport
	}

	k := &Kafka{
		reader: reader,
		writer: writer,
	}

	return k, nil
}

func (k *Kafka) Consume(
	ctx context.Context,
	handler func(ctx context.Context, key, value []byte, headers []kafka.Header) error,
) error {
	defer func() {
		if err := k.reader.Close(); err != nil {
			fmt.Printf("Error closing reader: %v\n", err)
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			message, err := k.reader.ReadMessage(ctx)
			if err != nil {
				if errors.Is(err, context.Canceled) {
					return nil
				}
				fmt.Printf("Error reading message: %v\n", err)
				continue
			}

			fmt.Printf("Received message: Topic=%s, Partition=%d, Offset=%d, Key=%s\n",
				message.Topic, message.Partition, message.Offset, string(message.Key))

			if err = handler(ctx, message.Key, message.Value, message.Headers); err != nil {
				fmt.Printf("Error handling message: %v\n", err)
				continue
			}

			fmt.Printf("Successfully processed and committed message: Topic=%s, Partition=%d, Offset=%d\n",
				message.Topic, message.Partition, message.Offset)
		}
	}
}

func (k *Kafka) Produce(ctx context.Context, key, value []byte) error {
	message := kafka.Message{
		Key:   key,
		Value: value,
		Time:  time.Now(),
	}

	err := k.writer.WriteMessages(ctx, message)
	if err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	if !k.writer.Async {
		err = k.writer.Close()
		if err != nil {
			return fmt.Errorf("failed to flush messages: %w", err)
		}
	}

	return nil
}

func (k *Kafka) Close() error {
	if err := k.reader.Close(); err != nil {
		return fmt.Errorf("failed to close reader: %w", err)
	}

	if err := k.writer.Close(); err != nil {
		return fmt.Errorf("failed to close writer: %w", err)
	}

	return nil
}
