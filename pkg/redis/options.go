package redis

import "time"

const (
	defaultPoolSize     int           = 10
	defaultDialTimeout  time.Duration = 1 * time.Second
	defaultReadTimeout  time.Duration = 3 * time.Second
	defaultWriteTimeout time.Duration = 3 * time.Second
	defaultRetryDelay   time.Duration = 3 * time.Second
	defaultMaxAttempts  int           = 5
)

type options struct {
	poolSize     int
	dialTimeout  time.Duration
	readTimeout  time.Duration
	writeTimeout time.Duration
	retryDelay   time.Duration
	maxAttempts  int
}

type Option func(*options)

func defaultOptions() options {
	return options{
		poolSize:     defaultPoolSize,
		dialTimeout:  defaultDialTimeout,
		readTimeout:  defaultReadTimeout,
		writeTimeout: defaultWriteTimeout,
		retryDelay:   defaultRetryDelay,
		maxAttempts:  defaultMaxAttempts,
	}
}

func WithPoolSize(n int) Option {
	return func(o *options) {
		o.poolSize = n
	}
}

func WithDialTimeout(d time.Duration) Option {
	return func(o *options) {
		o.dialTimeout = d
	}
}

func WithReadTimeout(d time.Duration) Option {
	return func(o *options) {
		o.readTimeout = d
	}
}

func WithWriteTimeout(d time.Duration) Option {
	return func(o *options) {
		o.writeTimeout = d
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
