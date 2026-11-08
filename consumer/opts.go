package consumer

import (
	"context"
	"crypto/tls"
	"log/slog"
	"time"

	"github.com/IBM/sarama"

	"github.com/korableg/bus/sasl"
)

// Where a group without a committed offset starts consuming a partition.
const (
	InitialOffsetLatest   = sarama.OffsetNewest // From the next produced message
	InitialOffsetEarliest = sarama.OffsetOldest // From the oldest retained message
)

type (
	// TraceIDCtxFunc returns a copy of ctx carrying traceID, the value of the "trace-id" header.
	TraceIDCtxFunc func(ctx context.Context, traceID string) context.Context

	// Options are the group settings, set them with the With* options.
	Options struct {
		InitialOffset int64 // Where to start without a committed offset (default InitialOffsetLatest)

		FetchMaxWait  time.Duration // How long the broker waits for FetchMinBytes before responding (default 500ms)
		FetchMinBytes int32         // How many bytes the broker collects before responding, more means higher throughput and latency (default 1)

		CommitCount    int           // Commit after this many messages, zero disables it (default 20)
		CommitDuration time.Duration // Commit at this interval, zero disables it (default 1s)

		DialTimeout  time.Duration // How long to wait for the initial connection (default 5s)
		ReadTimeout  time.Duration // How long to wait for a response (default 10s)
		WriteTimeout time.Duration // How long to wait for a transmit (default 5s)

		Logger     *slog.Logger   // Logger (default slog.Default)
		TraceIDCtx TraceIDCtxFunc // Puts the trace ID of a message into the context of the handlers (default none)
		SASL       *sasl.SASL     // SASL authentication with the broker (default none)
		TLS        *tls.Config    // TLS settings of the broker connection (default nil, TLS is disabled)
	}

	// Option changes Options, pass it to NewGroup.
	Option func(*Options)
)

func newOptions(opts ...Option) Options {
	o := Options{
		InitialOffset: InitialOffsetLatest,

		FetchMaxWait:  500 * time.Millisecond,
		FetchMinBytes: 1,

		CommitCount:    20,
		CommitDuration: time.Second,

		DialTimeout:  5 * time.Second,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 5 * time.Second,

		Logger:     slog.Default(),
		TraceIDCtx: noTraceIDCtx,
	}

	for _, opt := range opts {
		opt(&o)
	}

	if o.Logger == nil {
		o.Logger = slog.Default()
	}

	if o.TraceIDCtx == nil {
		o.TraceIDCtx = noTraceIDCtx
	}

	return o
}

func noTraceIDCtx(ctx context.Context, _ string) context.Context { return ctx }

// WithEarliestOffsetReset makes a group without a committed offset start from the oldest retained message.
func WithEarliestOffsetReset() Option {
	return func(o *Options) {
		o.InitialOffset = InitialOffsetEarliest
	}
}

// WithFetchMaxWait sets how long the broker waits for the minimum fetch size before responding.
func WithFetchMaxWait(d time.Duration) Option {
	return func(o *Options) {
		o.FetchMaxWait = d
	}
}

// WithFetchMinBytes sets how many bytes the broker collects before responding to a fetch.
func WithFetchMinBytes(value int32) Option {
	return func(o *Options) {
		o.FetchMinBytes = value
	}
}

// WithCommitCount makes the group commit after every value messages, zero disables it.
func WithCommitCount(value int) Option {
	return func(o *Options) {
		o.CommitCount = value
	}
}

// WithCommitDuration makes the group commit at the interval, zero disables it.
func WithCommitDuration(value time.Duration) Option {
	return func(o *Options) {
		o.CommitDuration = value
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

// WithTraceIDCtxFunc sets the function putting the trace ID of a message into the context of the handlers.
// It is called only for the messages with the "trace-id" header.
func WithTraceIDCtxFunc(f TraceIDCtxFunc) Option {
	return func(o *Options) {
		o.TraceIDCtx = f
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
