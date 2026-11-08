package consumer

import (
	"log/slog"
	"time"
)

type (
	// HandlerOptions are the handler settings, set them with the WithHandler* options.
	HandlerOptions struct {
		Logger *slog.Logger // Logger (default slog.Default)

		Timeout time.Duration // Timeout of one call of the handler function (default 5s, zero disables it)

		BatchSize         int           // Size of the batch of HandlerBatch (default 100)
		BatchFlushTimeout time.Duration // How long HandlerBatch waits for the next message before flushing an incomplete batch (default 10s)

		Retry                bool          // Retry the failed calls (default true)
		RetryInitialInterval time.Duration // Interval before the first retry (default 500ms)
		RetryMaxInterval     time.Duration // Maximum interval between the retries (default 2s)
		RetryMaxTime         time.Duration // How long to retry one message (default 0, indefinitely)
	}

	// HandlerOption changes HandlerOptions, pass it to NewHandler or NewHandlerBatch.
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
		RetryMaxTime:         0,
	}

	for _, op := range opts {
		op(&o)
	}

	// zero intervals would turn the retries into a busy loop
	if o.Logger == nil {
		o.Logger = slog.Default()
	}

	if o.RetryInitialInterval <= 0 {
		o.RetryInitialInterval = 500 * time.Millisecond
	}

	if o.RetryMaxInterval <= 0 {
		o.RetryMaxInterval = 2 * time.Second
	}

	return o
}

// WithHandlerTimeout sets the timeout of one call of the handler function, zero disables it.
func WithHandlerTimeout(timeout time.Duration) HandlerOption {
	return func(o *HandlerOptions) {
		o.Timeout = timeout
	}
}

// WithHandlerBatchSize sets how many messages HandlerBatch collects before calling its function.
func WithHandlerBatchSize(size int) HandlerOption {
	return func(o *HandlerOptions) {
		o.BatchSize = size
	}
}

// WithHandlerBatchFlushTimeout sets how long HandlerBatch waits for the next message: if none arrives,
// the incomplete batch is flushed. Zero disables the flush by the timeout.
func WithHandlerBatchFlushTimeout(timeout time.Duration) HandlerOption {
	return func(o *HandlerOptions) {
		o.BatchFlushTimeout = timeout
	}
}

// WithoutHandlerRetry switches the retries off: the handler function is called once per message,
// and after an error or a panic the message is considered handled and its offset is committed.
func WithoutHandlerRetry() HandlerOption {
	return func(o *HandlerOptions) {
		o.Retry = false
	}
}

// WithHandlerRetryInitialInterval sets the interval before the first retry. The interval grows exponentially.
func WithHandlerRetryInitialInterval(d time.Duration) HandlerOption {
	return func(o *HandlerOptions) {
		o.RetryInitialInterval = d
	}
}

// WithHandlerRetryMaxInterval sets the maximum interval between the retries.
func WithHandlerRetryMaxInterval(d time.Duration) HandlerOption {
	return func(o *HandlerOptions) {
		o.RetryMaxInterval = d
	}
}

// WithHandlerRetryMaxTime sets how long one message is retried. After that the message is considered
// handled and its offset is committed. Zero (default) means to retry indefinitely.
func WithHandlerRetryMaxTime(d time.Duration) HandlerOption {
	return func(o *HandlerOptions) {
		o.RetryMaxTime = d
	}
}

// WithHandlerLogger sets the logger. A nil logger means slog.Default.
func WithHandlerLogger(l *slog.Logger) HandlerOption {
	return func(o *HandlerOptions) {
		o.Logger = l
	}
}
