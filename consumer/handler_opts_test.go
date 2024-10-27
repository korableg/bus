package consumer

import (
	"bytes"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestUnit_HandlerOpts(t *testing.T) {
	t.Parallel()

	l := slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil))

	opts := newHandlerOptions(
		WithHandlerTimeout(17*time.Hour),
		WithHandlerBatchFlushTimeout(13*time.Hour),
		WithHandlerBatchSize(3123),
		WithoutHandlerRetry(),
		WithHandlerRetryInitialInterval(111*time.Millisecond),
		WithHandlerRetryMaxInterval(2222*time.Millisecond),
		WithHandlerRetryMaxTime(88*time.Hour),
		WithHandlerLogger(l),
	)

	assert.Equal(t, opts.Timeout, 17*time.Hour)
	assert.Equal(t, opts.BatchFlushTimeout, 13*time.Hour)
	assert.Equal(t, opts.BatchSize, 3123)
	assert.Equal(t, opts.Retry, false)
	assert.Equal(t, opts.RetryInitialInterval, 111*time.Millisecond)
	assert.Equal(t, opts.RetryMaxInterval, 2222*time.Millisecond)
	assert.Equal(t, opts.RetryMaxTime, 88*time.Hour)
	assert.Equal(t, opts.Logger, l)
}
