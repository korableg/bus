package consumer

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnit_HandlerProto_Handle(t *testing.T) {
	t.Parallel()

	newTestHandler := func(t *testing.T, decode func(string) (string, error), logs *syncBuffer, f HandlerProtoFunc[string]) *HandlerProto[string] {
		t.Helper()

		return NewHandler(newDecoder(t, decode), f, WithoutHandlerRetry(), WithHandlerLogger(newTestLogger(logs)))
	}

	t.Run("message", func(t *testing.T) {
		t.Parallel()

		var (
			got []string
			raw = msgOf(2, 9, "hello")
			h   = newTestHandler(t, decodeString, &syncBuffer{}, func(_ context.Context, msg string, r Message) error {
				got = append(got, msg)
				assert.Equal(t, raw, r)

				return nil
			})
		)

		h.Handle(context.Background(), raw)

		assert.Equal(t, []string{"hello"}, got)
		assert.EqualValues(t, 9, h.Offset(2))
	})

	t.Run("handler error", func(t *testing.T) {
		t.Parallel()

		var (
			logs = &syncBuffer{}
			h    = newTestHandler(t, decodeString, logs, func(context.Context, string, Message) error {
				return errors.New("handler failed")
			})
		)

		h.Handle(context.Background(), msgOf(0, 4, "x"))

		assert.EqualValues(t, 4, h.Offset(0), "the failed message counts as handled without the retries")
		assert.Contains(t, logs.String(), "handler failed")
	})

	t.Run("message of another type", func(t *testing.T) {
		t.Parallel()

		called := false
		h := newTestHandler(t, decodeWrongType, &syncBuffer{}, func(context.Context, string, Message) error {
			called = true
			return nil
		})

		h.Handle(context.Background(), msgOf(0, 1, "x"))

		assert.False(t, called)
		assert.EqualValues(t, 1, h.Offset(0))
	})

	t.Run("broken message", func(t *testing.T) {
		t.Parallel()

		var (
			got  []string
			logs = &syncBuffer{}
			h    = newTestHandler(t, decodePrefix("b"), logs, func(_ context.Context, msg string, _ Message) error {
				got = append(got, msg)
				return nil
			})
		)

		h.Handle(context.Background(), msgOf(0, 3, "broken"))

		assert.Equal(t, []string{""}, got, "the function gets the zero value")
		assert.EqualValues(t, 3, h.Offset(0))
		assert.Contains(t, logs.String(), errBroken.Error())
	})
}

func TestUnit_HandlerBatch_Handle(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("one buffer for all partitions", func(t *testing.T) {
		t.Parallel()

		var (
			r = &batchRecorder{}
			h = newTestBatchHandler(t, decodeString, r.fn, WithHandlerBatchSize(3), WithHandlerBatchFlushTimeout(time.Hour))
		)

		h.Handle(ctx, msgOf(0, 10, "a"))
		h.Handle(ctx, msgOf(1, 20, "x"))

		assert.Empty(t, r.snapshot())
		assert.EqualValues(t, 0, h.Offset(0), "no offset before the flush")
		assert.EqualValues(t, 0, h.Offset(1), "no offset before the flush")

		h.Handle(ctx, msgOf(0, 11, "b"))

		calls := r.snapshot()
		require.Len(t, calls, 1)
		assert.Equal(t, []string{"a", "x", "b"}, calls[0].msgs)
		assert.EqualValues(t, 11, h.Offset(0))
		assert.EqualValues(t, 20, h.Offset(1))
	})

	t.Run("flush by size", func(t *testing.T) {
		t.Parallel()

		var (
			r = &batchRecorder{}
			h = newTestBatchHandler(t, decodeString, r.fn, WithHandlerBatchSize(3), WithHandlerBatchFlushTimeout(time.Hour))
		)

		for i := range 7 {
			h.Handle(ctx, msgOf(0, int64(i+1), "m"))
		}

		calls := r.snapshot()
		require.Len(t, calls, 2)
		assert.Len(t, calls[0].msgs, 3)
		assert.Len(t, calls[1].msgs, 3)
		assert.EqualValues(t, 6, h.Offset(0), "the seventh message is still in the buffer")
	})

	t.Run("flush timeout is counted from the last message", func(t *testing.T) {
		t.Parallel()

		const (
			timeout = 300 * time.Millisecond
			count   = 15
		)

		var (
			r      = &batchRecorder{}
			h      = newTestBatchHandler(t, decodeString, r.fn, WithHandlerBatchSize(1000), WithHandlerBatchFlushTimeout(timeout))
			lastAt time.Time
		)

		for i := range count {
			h.Handle(ctx, msgOf(0, int64(i+1), "m"))
			lastAt = time.Now()
			time.Sleep(30 * time.Millisecond)
		}

		assert.Empty(t, r.snapshot(), "the buffer must not be flushed by the timeout while the messages keep coming")

		require.Eventually(t, func() bool { return h.Offset(0) == count }, 2*time.Second, 10*time.Millisecond)

		calls := r.snapshot()
		require.Len(t, calls, 1)
		assert.Len(t, calls[0].msgs, count)
		assert.GreaterOrEqual(t, calls[0].at.Sub(lastAt), timeout-20*time.Millisecond)
	})

	t.Run("timer is reused", func(t *testing.T) {
		t.Parallel()

		var (
			r = &batchRecorder{}
			h = newTestBatchHandler(t, decodeString, r.fn, WithHandlerBatchSize(2), WithHandlerBatchFlushTimeout(time.Hour))
		)

		h.Handle(ctx, msgOf(0, 1, "a"))
		timer := h.timer
		require.NotNil(t, timer)

		h.Handle(ctx, msgOf(0, 2, "b"))
		h.Handle(ctx, msgOf(0, 3, "c"))

		assert.Len(t, r.snapshot(), 1)
		assert.Same(t, timer, h.timer)
	})

	t.Run("message of another type", func(t *testing.T) {
		t.Parallel()

		var (
			r = &batchRecorder{}
			h = newTestBatchHandler(t, decodeWrongType, r.fn, WithHandlerBatchSize(1))
		)

		h.Handle(ctx, msgOf(0, 5, "x"))

		assert.Empty(t, r.snapshot())
		assert.EqualValues(t, 5, h.Offset(0))
	})

	t.Run("skipped offset waits for flush of partition", func(t *testing.T) {
		t.Parallel()

		var (
			r = &batchRecorder{}
			h = newTestBatchHandler(t, decodePrefix("b"), r.fn,
				WithHandlerBatchSize(10), WithHandlerBatchFlushTimeout(100*time.Millisecond))
		)

		h.Handle(ctx, msgOf(0, 1, "b1"))
		h.Handle(ctx, msgOf(0, 2, "other"))
		h.Handle(ctx, msgOf(1, 7, "other"))

		assert.EqualValues(t, 0, h.Offset(0), "the buffered message of the partition holds the offset")
		assert.EqualValues(t, 7, h.Offset(1), "nothing is buffered for the partition, the offset moves at once")

		require.Eventually(t, func() bool { return h.Offset(0) == 2 }, 2*time.Second, 5*time.Millisecond)
		require.Len(t, r.snapshot(), 1)
		assert.Equal(t, []string{"b1"}, r.snapshot()[0].msgs)
	})

	t.Run("skip does not wait for flush", func(t *testing.T) {
		t.Parallel()

		var (
			release = make(chan struct{})
			started = make(chan struct{})
			flushed = make(chan struct{})
			skipped = make(chan struct{})
			h       = newTestBatchHandler(t, decodePrefix("b"), func(context.Context, []string, []Message) error {
				close(started)
				<-release

				return nil
			}, WithHandlerBatchSize(1))
		)

		go func() {
			defer close(flushed)

			h.Handle(ctx, msgOf(0, 1, "b1"))
		}()

		<-started

		go func() {
			defer close(skipped)

			h.Handle(ctx, msgOf(0, 2, "other"))
			h.Handle(ctx, msgOf(1, 5, "other"))
		}()

		select {
		case <-skipped:
		case <-time.After(2 * time.Second):
			t.Fatal("skipping a message waited for the flush")
		}

		assert.EqualValues(t, 5, h.Offset(1))
		assert.EqualValues(t, 0, h.Offset(0), "the batch of the partition is still being handled")

		close(release)
		<-flushed

		assert.EqualValues(t, 2, h.Offset(0))
	})

	t.Run("broken message", func(t *testing.T) {
		t.Parallel()

		var (
			logs = &syncBuffer{}
			r    = &batchRecorder{}
			h    = newTestBatchHandler(t, decodePrefix("b"), r.fn,
				WithHandlerBatchSize(1), WithHandlerLogger(newTestLogger(logs)))
		)

		h.Handle(ctx, msgOf(0, 3, "broken"))
		h.Handle(ctx, msgOf(0, 4, "other"))

		assert.Empty(t, r.snapshot())
		assert.EqualValues(t, 4, h.Offset(0))
		assert.Contains(t, logs.String(), errBroken.Error())
		assert.Equal(t, 1, strings.Count(logs.String(), "unmarshal message"), "a message of another type is not logged")
	})

	t.Run("calls from partitions are sequential", func(t *testing.T) {
		t.Parallel()

		var (
			r  = &batchRecorder{}
			h  = newTestBatchHandler(t, decodeString, r.fn, WithHandlerBatchSize(5), WithHandlerBatchFlushTimeout(5*time.Millisecond))
			wg sync.WaitGroup
		)

		for part := range int32(4) {
			wg.Go(func() {
				for i := range 100 {
					h.Handle(ctx, msgOf(part, int64(i+1), "m"))

					if i%7 == 0 {
						time.Sleep(time.Millisecond)
					}
				}
			})
		}

		wg.Wait()

		require.Eventually(t, func() bool {
			total := 0
			for _, call := range r.snapshot() {
				total += len(call.msgs)
			}

			return total == 400
		}, 5*time.Second, 10*time.Millisecond)

		assert.EqualValues(t, 1, r.maxActive.Load(), "the batch function must not be called concurrently")

		for part := range int32(4) {
			assert.EqualValues(t, 100, h.Offset(part))
		}
	})
}

func TestUnit_HandlerBatch_flushByTimeout(t *testing.T) {
	t.Parallel()

	t.Run("stale call", func(t *testing.T) {
		t.Parallel()

		var (
			r = &batchRecorder{}
			h = newTestBatchHandler(t, decodeString, r.fn, WithHandlerBatchSize(10), WithHandlerBatchFlushTimeout(time.Hour))
		)

		h.Handle(context.Background(), msgOf(0, 1, "a"))

		// the timer fired before the message reset it, and its call waited for the buffer lock
		h.flushByTimeout()

		assert.Empty(t, r.snapshot())
		assert.EqualValues(t, 0, h.Offset(0))
	})

	t.Run("due call", func(t *testing.T) {
		t.Parallel()

		var (
			r = &batchRecorder{}
			h = newTestBatchHandler(t, decodeString, r.fn, WithHandlerBatchSize(10), WithHandlerBatchFlushTimeout(time.Hour))
		)

		h.Handle(context.Background(), msgOf(0, 1, "a"))

		h.bufMu.Lock()
		h.flushAt = time.Now().Add(-time.Second)
		h.bufMu.Unlock()

		h.flushByTimeout()

		calls := r.snapshot()
		require.Len(t, calls, 1)
		assert.Equal(t, []string{"a"}, calls[0].msgs)
		assert.EqualValues(t, 1, h.Offset(0))
	})

	t.Run("empty buffer", func(t *testing.T) {
		t.Parallel()

		var (
			r = &batchRecorder{}
			h = newTestBatchHandler(t, decodeString, r.fn)
		)

		h.flushByTimeout()

		assert.Empty(t, r.snapshot())
	})
}
