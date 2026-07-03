package postgres

import "time"

const (
	defaultMaxConns    int32         = 10
	defaultMinConns    int32         = 1
	defaultConnTimeout time.Duration = 5 * time.Second
	defaultRetryDelay  time.Duration = 5 * time.Second
	defaultMaxAttempts int           = 5
)

type options struct {
	maxConns    int32
	minConns    int32
	connTimeout time.Duration
	retryDelay  time.Duration
	maxAttempts int
}

type Option func(*options)

func defaultOptions() options {
	return options{
		maxConns:    defaultMaxConns,
		minConns:    defaultMinConns,
		connTimeout: defaultConnTimeout,
		retryDelay:  defaultRetryDelay,
		maxAttempts: defaultMaxAttempts,
	}
}

func WithMaxConns(n int32) Option {
	return func(o *options) {
		o.maxConns = n
	}
}

func WithMinConns(n int32) Option {
	return func(o *options) {
		o.minConns = n
	}
}

func WithConnTimeout(d time.Duration) Option {
	return func(o *options) {
		o.connTimeout = d
	}
}

func WithRetryDelay(d time.Duration) Option {
	return func(o *options) {
		o.retryDelay = d
	}
}

func WithMaxAttempts(n int) Option {
	return func(o *options) {
		o.maxAttempts = n
	}
}
