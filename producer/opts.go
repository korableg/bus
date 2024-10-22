package producer

import (
	"context"
	"log/slog"
	"time"

	"github.com/korableg/bus/sasl"
)

type (
	// Options producer options
	Options struct {
		Idempotence     bool // Idempotence producer mode (default true)
		MaxOpenRequests int  // How many outstanding requests a connection is allowed to have before sending on it blocks (default 1 for the idempotence support)

		FlushFrequency time.Duration // How much time the kafka library have to waiting for before send the message (default 0 (instant sending))
		FlushBytes     int           // How many bytes the kafka library have to save up in the buffer before send the message (default 0)
		FlushMessages  int           // How many messages the kafka library have to save up in the buffer before send the message (default 0)

		Retries      int           // How many retries the kafka library have to do if message send was failed (default 10)
		RetryBackOff time.Duration // How much time the kafka library have to waiting for before next attempt (default 300ms)

		DialTimeout  time.Duration // How long to wait for the initial connection
		ReadTimeout  time.Duration // How long to wait for a response
		WriteTimeout time.Duration // How long to wait for a transmit

		Logger  *slog.Logger                 // Logger (default writer is the os.Stdout)
		TraceID func(context.Context) string // Your func to get the trace ID from context

		SASL *sasl.SASL // SASL based authentication with broker
	}

	// Option producer option
	Option func(*Options)
)

// NewOptions constructs Options
func newOptions(opts ...Option) Options {
	o := Options{
		Idempotence:     true,
		MaxOpenRequests: 1,

		FlushFrequency: 0,
		FlushBytes:     0,
		FlushMessages:  0,

		Retries:      10,
		RetryBackOff: 300 * time.Millisecond,

		DialTimeout:  5 * time.Second,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 5 * time.Second,

		TraceID: func(context.Context) string { return "" },
		Logger:  slog.Default(),
	}

	for _, opt := range opts {
		opt(&o)
	}

	return o
}

// WithoutIdempotence switch off idempotence mode
func WithoutIdempotence() Option {
	return func(o *Options) {
		o.Idempotence = false
	}
}

// WithMaxOpenRequests sets how many outstanding requests a connection is allowed to have before sending on it blocks
func WithMaxOpenRequests(val int) Option {
	return func(o *Options) {
		o.MaxOpenRequests = val
	}
}

// WithFlushParams sets flush parameters
func WithFlushParams(frequency time.Duration, bytes, messages int) Option {
	return func(o *Options) {
		o.FlushFrequency = frequency
		o.FlushBytes = bytes
		o.FlushMessages = messages
	}
}

// WithRetries sets Retries option
func WithRetries(value int) Option {
	return func(o *Options) {
		o.Retries = value
	}
}

// WithRetryBackOff sets RetryBackOffMs option
func WithRetryBackOff(d time.Duration) Option {
	return func(o *Options) {
		o.RetryBackOff = d
	}
}

// WithDialTimeout sets how long to wait for the initial connection
func WithDialTimeout(d time.Duration) Option {
	return func(o *Options) {
		o.DialTimeout = d
	}
}

// WithReadTimeout sets how long to wait for a response
func WithReadTimeout(d time.Duration) Option {
	return func(o *Options) {
		o.ReadTimeout = d
	}
}

// WithWriteTimeout sets how long to wait for a transmit
func WithWriteTimeout(d time.Duration) Option {
	return func(o *Options) {
		o.WriteTimeout = d
	}
}

// WithLogger sets Logger option
func WithLogger(logger *slog.Logger) Option {
	return func(o *Options) {
		o.Logger = logger
	}
}

// WithTraceID sets your trace ID getter from context
func WithTraceID(traceID func(context.Context) string) Option {
	return func(o *Options) {
		o.TraceID = traceID
	}
}

// WithSASL sets SASL based authentication with broker
func WithSASL(mechanism sasl.Mechanism, username, password string) Option {
	return func(o *Options) {
		o.SASL = &sasl.SASL{
			Mechanism: mechanism,
			UserName:  username,
			Password:  password,
		}
	}
}
