package consumer

import (
	"bytes"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestUnit_newHandlerOptions(t *testing.T) {
	t.Parallel()

	t.Run("defaults", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, HandlerOptions{
			Logger:               slog.Default(),
			Timeout:              5 * time.Second,
			BatchSize:            100,
			BatchFlushTimeout:    10 * time.Second,
			Retry:                true,
			RetryInitialInterval: 500 * time.Millisecond,
			RetryMaxInterval:     2 * time.Second,
			RetryMaxTime:         0,
		}, newHandlerOptions())
	})

	t.Run("options", func(t *testing.T) {
		t.Parallel()

		l := slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil))

		assert.Equal(t, HandlerOptions{
			Logger:               l,
			Timeout:              17 * time.Hour,
			BatchSize:            3123,
			BatchFlushTimeout:    13 * time.Hour,
			Retry:                false,
			RetryInitialInterval: 111 * time.Millisecond,
			RetryMaxInterval:     2222 * time.Millisecond,
			RetryMaxTime:         88 * time.Hour,
		}, newHandlerOptions(
			WithHandlerTimeout(17*time.Hour),
			WithHandlerBatchFlushTimeout(13*time.Hour),
			WithHandlerBatchSize(3123),
			WithoutHandlerRetry(),
			WithHandlerRetryInitialInterval(111*time.Millisecond),
			WithHandlerRetryMaxInterval(2222*time.Millisecond),
			WithHandlerRetryMaxTime(88*time.Hour),
			WithHandlerLogger(l),
		))
	})

	t.Run("nil and zero values", func(t *testing.T) {
		t.Parallel()

		o := newHandlerOptions(
			WithHandlerLogger(nil),
			WithHandlerRetryInitialInterval(0),
			WithHandlerRetryMaxInterval(-time.Second),
		)

		assert.Equal(t, slog.Default(), o.Logger)
		assert.Equal(t, 500*time.Millisecond, o.RetryInitialInterval)
		assert.Equal(t, 2*time.Second, o.RetryMaxInterval)
	})
}
