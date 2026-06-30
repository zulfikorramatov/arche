package kafkaold

import "time"

type kafkaOptions struct {
	username     string
	password     string
	readTimeout  time.Duration
	writeTimeout time.Duration
}

type Option func(*kafkaOptions)

func SASLAuth(username, password string) Option {
	return func(o *kafkaOptions) {
		o.username = username
		o.password = password
	}
}

func ReadTimeout(timeout time.Duration) Option {
	return func(o *kafkaOptions) {
		o.readTimeout = timeout
	}
}

func WriteTimeout(timeout time.Duration) Option {
	return func(o *kafkaOptions) {
		o.writeTimeout = timeout
	}
}
