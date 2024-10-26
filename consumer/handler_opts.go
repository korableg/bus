package consumer

import (
	"log/slog"
	"time"
)

type (
	HandlerOptions struct {
		Logger *slog.Logger

		Timeout time.Duration

		BatchSize         int
		BatchFlushTimeout time.Duration

		Retry                bool
		RetryInitialInterval time.Duration
		RetryMaxInterval     time.Duration
		RetryMaxTime         time.Duration
	}

	// HandlerOption handler option
	HandlerOption func(*HandlerOptions)
)

func newHandlerOptions(opts ...HandlerOption) HandlerOptions {
	o := HandlerOptions{
		Logger: slog.Default(),

		Timeout: 5 * time.Second,

		BatchSize:         100,
		BatchFlushTimeout: 10 * time.Second,

		Retry:                true,
		RetryInitialInterval: 500 * time.Millisecond,
		RetryMaxInterval:     2 * time.Second,
		RetryMaxTime:         0, // if zero, retry indefinitely
	}

	for _, op := range opts {
		op(&o)
	}

	return o
}

// WithHandlerTimeout sets Timeout option
func WithHandlerTimeout(timeout time.Duration) HandlerOption {
	return func(o *HandlerOptions) {
		o.Timeout = timeout
	}
}

// WithHandlerBatchSize sets BatchSize option
func WithHandlerBatchSize(size int) HandlerOption {
	return func(o *HandlerOptions) {
		o.BatchSize = size
	}
}

// WithHandlerBatchFlushTimeout sets BatchFlushTimeout option
func WithHandlerBatchFlushTimeout(timeout time.Duration) HandlerOption {
	return func(o *HandlerOptions) {
		o.BatchFlushTimeout = timeout
	}
}

// WithoutHandlerRetry sets Retry option to false
func WithoutHandlerRetry() HandlerOption {
	return func(o *HandlerOptions) {
		o.Retry = false
	}
}

// WithHandlerRetryInitialInterval sets RetryInitialInterval option
func WithHandlerRetryInitialInterval(d time.Duration) HandlerOption {
	return func(o *HandlerOptions) {
		o.RetryInitialInterval = d
	}
}

// WithHandlerRetryMaxInterval sets RetryMaxInterval option
func WithHandlerRetryMaxInterval(d time.Duration) HandlerOption {
	return func(o *HandlerOptions) {
		o.RetryMaxInterval = d
	}
}

// WithHandlerRetryMaxTime sets RetryMaxTime option
func WithHandlerRetryMaxTime(d time.Duration) HandlerOption {
	return func(o *HandlerOptions) {
		o.RetryMaxTime = d
	}
}

// WithHandlerLogger sets Logger option
func WithHandlerLogger(l *slog.Logger) HandlerOption {
	return func(o *HandlerOptions) {
		o.Logger = l
	}
}
