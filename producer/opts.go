package producer

import (
	"context"
	"crypto/tls"
	"log/slog"
	"time"

	"github.com/korableg/bus/sasl"
)

type (
	// Options are the producer settings, set them with the With* options.
	Options struct {
		Idempotence     bool // Idempotent producer mode, every message must have a key (default true)
		MaxOpenRequests int  // How many outstanding requests a connection may have before sending on it blocks (default 1, required by the idempotence)

		FlushFrequency time.Duration // How long to buffer the messages before sending them (default 0, send at once)
		FlushBytes     int           // How many bytes to buffer before sending them (default 0)
		FlushMessages  int           // How many messages to buffer before sending them (default 0)

		Retries      int           // How many times to retry a failed send (default 10)
		RetryBackOff time.Duration // How long to wait before the next retry (default 300ms)

		DialTimeout  time.Duration // How long to wait for the initial connection (default 5s)
		ReadTimeout  time.Duration // How long to wait for a response (default 10s)
		WriteTimeout time.Duration // How long to wait for a transmit (default 5s)

		Logger  *slog.Logger                 // Logger (default slog.Default)
		TraceID func(context.Context) string // Gets the trace ID from the context for the "trace-id" header (default none)

		SASL *sasl.SASL  // SASL authentication with the broker (default none)
		TLS  *tls.Config // TLS settings of the broker connection (default nil, TLS is disabled)

		// ErrorHandler is called with *SendError for each message Send failed to deliver (default nil, the error is only logged).
		ErrorHandler func(err error)
	}

	// Option changes Options, pass it to New.
	Option func(*Options)
)

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

		TraceID: noTraceID,
		Logger:  slog.Default(),
	}

	for _, opt := range opts {
		opt(&o)
	}

	if o.Logger == nil {
		o.Logger = slog.Default()
	}

	if o.TraceID == nil {
		o.TraceID = noTraceID
	}

	return o
}

func noTraceID(context.Context) string { return "" }

// WithoutIdempotence switches the idempotence off, so the messages may have no key.
func WithoutIdempotence() Option {
	return func(o *Options) {
		o.Idempotence = false
	}
}

// WithMaxOpenRequests sets how many outstanding requests a connection may have before sending on it blocks.
// The idempotent producer requires 1, so raise it only together with WithoutIdempotence.
func WithMaxOpenRequests(val int) Option {
	return func(o *Options) {
		o.MaxOpenRequests = val
	}
}

// WithFlushParams makes the producer buffer the messages and send them when any of the limits is reached:
// the time since the first buffered message, the size in bytes or the count of the messages.
func WithFlushParams(frequency time.Duration, bytes, messages int) Option {
	return func(o *Options) {
		o.FlushFrequency = frequency
		o.FlushBytes = bytes
		o.FlushMessages = messages
	}
}

// WithRetries sets how many times a failed send is retried.
func WithRetries(value int) Option {
	return func(o *Options) {
		o.Retries = value
	}
}

// WithRetryBackOff sets how long to wait before the next retry of a failed send.
func WithRetryBackOff(d time.Duration) Option {
	return func(o *Options) {
		o.RetryBackOff = d
	}
}

// WithDialTimeout sets how long to wait for the initial connection.
func WithDialTimeout(d time.Duration) Option {
	return func(o *Options) {
		o.DialTimeout = d
	}
}

// WithReadTimeout sets how long to wait for a response.
func WithReadTimeout(d time.Duration) Option {
	return func(o *Options) {
		o.ReadTimeout = d
	}
}

// WithWriteTimeout sets how long to wait for a transmit.
func WithWriteTimeout(d time.Duration) Option {
	return func(o *Options) {
		o.WriteTimeout = d
	}
}

// WithLogger sets the logger. A nil logger means slog.Default.
func WithLogger(logger *slog.Logger) Option {
	return func(o *Options) {
		o.Logger = logger
	}
}

// WithTraceID sets the function getting the trace ID from the context of Send and SendSync.
// A non-empty trace ID is sent in the "trace-id" header.
func WithTraceID(traceID func(context.Context) string) Option {
	return func(o *Options) {
		o.TraceID = traceID
	}
}

// WithSASL sets the authentication with the user name and the password
// (sasl.TypePlaintext, sasl.TypeSCRAMSHA256 or sasl.TypeSCRAMSHA512).
func WithSASL(mechanism sasl.Mechanism, username, password string) Option {
	return func(o *Options) {
		o.SASL = &sasl.SASL{
			Mechanism: mechanism,
			UserName:  username,
			Password:  password,
		}
	}
}

// WithSASLOAuth sets the OAUTHBEARER authentication with the tokens of provider.
func WithSASLOAuth(provider sasl.TokenProvider) Option {
	return func(o *Options) {
		o.SASL = &sasl.SASL{
			Mechanism:     sasl.TypeOAuth,
			TokenProvider: provider,
		}
	}
}

// WithTLS enables TLS for the broker connection. A nil config means the default one.
func WithTLS(cfg *tls.Config) Option {
	return func(o *Options) {
		if cfg == nil {
			cfg = &tls.Config{MinVersion: tls.VersionTLS12}
		}

		o.TLS = cfg
	}
}

// WithErrorHandler sets the handler of the delivery errors of Send. The handler receives *SendError
// (use errors.As to get it) for each message Send accepted but Kafka failed to deliver,
// including the messages flushed by Close.
//
// The handler is called from the producer's event loop, so it must not block:
// otherwise the delivery of the other messages stalls and Close hangs.
func WithErrorHandler(handler func(err error)) Option {
	return func(o *Options) {
		o.ErrorHandler = handler
	}
}
