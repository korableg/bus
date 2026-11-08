package consumer

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnit_handler_handle(t *testing.T) {
	t.Parallel()

	newTestHandler := func(t *testing.T, opts ...HandlerOption) *handler[string] {
		t.Helper()

		opts = append([]HandlerOption{
			WithHandlerLogger(newTestLogger(&syncBuffer{})),
			WithHandlerRetryInitialInterval(time.Millisecond),
			WithHandlerRetryMaxInterval(time.Millisecond),
		}, opts...)

		h := newHandler(newDecoder(t, decodeString), newHandlerOptions(opts...))

		return &h
	}

	failTimes := func(n int, calls *int) func(context.Context) error {
		return func(context.Context) error {
			*calls++
			if *calls <= n {
				return errors.New("handler failed")
			}

			return nil
		}
	}

	t.Run("without retry", func(t *testing.T) {
		t.Parallel()

		var (
			calls int
			h     = newTestHandler(t, WithoutHandlerRetry())
		)

		require.Error(t, h.handle(context.Background(), failTimes(1, &calls)))
		assert.Equal(t, 1, calls)
	})

	t.Run("retries until success", func(t *testing.T) {
		t.Parallel()

		var (
			calls int
			h     = newTestHandler(t)
		)

		require.NoError(t, h.handle(context.Background(), failTimes(2, &calls)))
		assert.Equal(t, 3, calls)
	})

	t.Run("retries go on after cancellation", func(t *testing.T) {
		t.Parallel()

		var (
			calls int
			errs  []error
			h     = newTestHandler(t)
		)

		err := h.handle(canceledContext(), func(ctx context.Context) error {
			errs = append(errs, ctx.Err())
			return failTimes(2, &calls)(ctx)
		})

		require.NoError(t, err)
		assert.Equal(t, 3, calls)
		assert.Equal(t, []error{context.Canceled, context.Canceled, context.Canceled}, errs,
			"the function must get the canceled context")
	})

	t.Run("permanent error", func(t *testing.T) {
		t.Parallel()

		var (
			calls int
			cause = errors.New("broken forever")
			h     = newTestHandler(t)
		)

		err := h.handle(context.Background(), func(context.Context) error {
			calls++
			return PermanentError(cause)
		})

		require.ErrorIs(t, err, cause)
		assert.Equal(t, 1, calls)
	})

	t.Run("retry max time", func(t *testing.T) {
		t.Parallel()

		var (
			calls int
			h     = newTestHandler(t, WithHandlerRetryMaxTime(50*time.Millisecond))
		)

		require.Error(t, h.handle(context.Background(), failTimes(1<<30, &calls)))
		assert.Greater(t, calls, 1)
	})

	t.Run("panic", func(t *testing.T) {
		t.Parallel()

		h := newTestHandler(t, WithoutHandlerRetry())

		err := h.handle(context.Background(), func(context.Context) error { panic("boom") })
		require.ErrorContains(t, err, "panic: boom")
	})

	t.Run("timeout", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			timeout      time.Duration
			wantDeadline bool
		}{
			{timeout: time.Minute, wantDeadline: true},
			{timeout: 0, wantDeadline: false},
		}

		for _, tt := range tests {
			h := newTestHandler(t, WithoutHandlerRetry(), WithHandlerTimeout(tt.timeout))

			err := h.handle(context.Background(), func(ctx context.Context) error {
				_, ok := ctx.Deadline()
				assert.Equal(t, tt.wantDeadline, ok, "timeout %v", tt.timeout)

				return nil
			})
			require.NoError(t, err)
		}
	})
}
