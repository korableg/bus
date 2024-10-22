package consumer

import (
	"context"
	"log/slog"
	"time"

	"github.com/IBM/sarama"

	"github.com/korableg/bus/sasl"
)

const (
	InitialOffsetLatest   = sarama.OffsetNewest
	InitialOffsetEarliest = sarama.OffsetOldest
)

type (
	TraceIDCtxFunc func(context.Context, string) context.Context

	// Options consumer options
	Options struct {
		InitialOffset int64 // Determine when the group should to start consuming (default: "latest")

		FetchMaxWait  time.Duration // Waiting time before returning messages to the client (relevant if FetchMinBytes > 1) (default 500)
		FetchMinBytes int32         // Determine how many bytes kafka library have to receive before returning to the client (latency << FetchMinBytes << throughput) (default 1)

		CommitCount    int           // After how many messages do client have to call the commit (default 20)
		CommitDuration time.Duration // After how much time do client have to call the commit (default 1 second)

		DialTimeout  time.Duration // How long to wait for the initial connection
		ReadTimeout  time.Duration // How long to wait for a response
		WriteTimeout time.Duration // How long to wait for a transmit

		Logger     *slog.Logger   // Logger (default: slog.Default)
		TraceIDCtx TraceIDCtxFunc // Func which inserts trace id into the context
		SASL       *sasl.SASL     // SASL based authentication with broker
	}

	// Option consumer option
	Option func(*Options)
)

// NewOptions constructs Options
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

		Logger: slog.Default(),
		TraceIDCtx: func(ctx context.Context, _ string) context.Context {
			return ctx
		},
	}

	for _, opt := range opts {
		opt(&o)
	}

	return o
}

// WithEarliestOffsetReset sets AutoOffsetReset to "earliest" mode
func WithEarliestOffsetReset() Option {
	return func(o *Options) {
		o.InitialOffset = InitialOffsetEarliest
	}
}

// WithFetchMaxWait sets FetchMaxWait option
func WithFetchMaxWait(d time.Duration) Option {
	return func(o *Options) {
		o.FetchMaxWait = d
	}
}

// WithFetchMinBytes sets FetchMinBytes option
func WithFetchMinBytes(value int32) Option {
	return func(o *Options) {
		o.FetchMinBytes = value
	}
}

// WithCommitCount sets CommitCount option
func WithCommitCount(value int) Option {
	return func(o *Options) {
		o.CommitCount = value
	}
}

// WithCommitDuration sets CommitDuration option
func WithCommitDuration(value time.Duration) Option {
	return func(o *Options) {
		o.CommitDuration = value
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

// WithTraceIDCtxFunc sets func which inserts trace id to the context
func WithTraceIDCtxFunc(f TraceIDCtxFunc) Option {
	return func(o *Options) {
		o.TraceIDCtx = f
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
