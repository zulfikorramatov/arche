package kafka

import "time"

const (
	defaultDialTimeout  time.Duration = 10 * time.Second
	defaultMaxRetries   int           = 3
	defaultRetryBackoff time.Duration = time.Second
)

type options struct {
	dialTimeout  time.Duration
	maxRetries   int
	retryBackoff time.Duration
}

type Option func(*options)

func defaultOptions() options {
	return options{
		dialTimeout:  defaultDialTimeout,
		maxRetries:   defaultMaxRetries,
		retryBackoff: defaultRetryBackoff,
	}
}

func WithDialTimeout(d time.Duration) Option {
	return func(o *options) {
		o.dialTimeout = d
	}
}

func WithMaxRetries(n int) Option {
	return func(o *options) {
		o.maxRetries = n
	}
}

func WithRetryBackoff(d time.Duration) Option {
	return func(o *options) {
		o.retryBackoff = d
	}
}
