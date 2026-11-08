package consumer

import (
	"context"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/IBM/sarama"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

type testGroup struct {
	*Group

	cli          *MockConsumerGroup
	errs         chan error
	consumeCalls atomic.Int64
}

func newTestGroup(t *testing.T, logs *syncBuffer, consume func(ctx context.Context) error) *testGroup {
	t.Helper()

	g := &testGroup{
		cli:  NewMockConsumerGroup(gomock.NewController(t)),
		errs: make(chan error, 8),
	}

	g.cli.EXPECT().Errors().Return(g.errs).AnyTimes()
	g.cli.EXPECT().Close().DoAndReturn(func() error {
		close(g.errs)
		return nil
	})
	g.cli.EXPECT().Consume(gomock.Any(), []string{"topic"}, gomock.Any()).
		DoAndReturn(func(ctx context.Context, _ []string, _ sarama.ConsumerGroupHandler) error {
			g.consumeCalls.Add(1)
			return consume(ctx)
		}).AnyTimes()

	g.Group = newGroup(g.cli, "test-group", newOptions(WithLogger(newTestLogger(logs))))
	t.Cleanup(func() { _ = g.Close() })

	return g
}

func consumeErr(err error) func(context.Context) error {
	return func(context.Context) error { return err }
}

func (g *testGroup) addHandler(t *testing.T) {
	t.Helper()

	h := newOffsetHandler(t, "topic", nil)
	require.NoError(t, g.AddHandler(h))
}

func TestUnit_NewGroup(t *testing.T) {
	t.Parallel()

	g, err := NewGroup([]string{"127.0.0.1:1"}, "group", WithDialTimeout(50*time.Millisecond))
	require.ErrorContains(t, err, "init consumer group")
	assert.Nil(t, g)
}

func TestUnit_Group_AddHandler(t *testing.T) {
	t.Parallel()

	t.Run("handler", func(t *testing.T) {
		t.Parallel()

		var (
			g = newTestGroup(t, &syncBuffer{}, consumeErr(nil))
			h = newOffsetHandler(t, "topic", nil)
		)

		require.NoError(t, g.AddHandler(h))
		assert.Equal(t, []Handler{h}, g.hCtl.Handlers("topic"))
	})

	t.Run("handler without topic", func(t *testing.T) {
		t.Parallel()

		g := newTestGroup(t, &syncBuffer{}, consumeErr(nil))

		require.ErrorIs(t, g.AddHandler(newOffsetHandler(t, "", nil)), ErrEmptyTopic)
	})

	t.Run("closed group", func(t *testing.T) {
		t.Parallel()

		g := newTestGroup(t, &syncBuffer{}, consumeErr(nil))

		require.NoError(t, g.Close())
		require.ErrorIs(t, g.AddHandler(newOffsetHandler(t, "topic", nil)), ErrClosed)
	})

	t.Run("after run", func(t *testing.T) {
		t.Parallel()

		g := newTestGroup(t, &syncBuffer{}, consumeErr(nil))
		g.addHandler(t)

		require.NoError(t, g.Run(canceledContext()))
		require.ErrorIs(t, g.AddHandler(newOffsetHandler(t, "topic2", nil)), ErrRunned)
	})
}

func TestUnit_Group_Run(t *testing.T) {
	t.Parallel()

	t.Run("closed group", func(t *testing.T) {
		t.Parallel()

		g := newTestGroup(t, &syncBuffer{}, consumeErr(nil))
		g.addHandler(t)

		require.NoError(t, g.Close())
		require.ErrorIs(t, g.Run(context.Background()), ErrClosed)
	})

	t.Run("without handlers", func(t *testing.T) {
		t.Parallel()

		g := newTestGroup(t, &syncBuffer{}, consumeErr(nil))

		require.ErrorIs(t, g.Run(context.Background()), ErrNoTopics)
		require.NoError(t, g.AddHandler(newOffsetHandler(t, "topic", nil)), "the failed Run must not forbid adding handlers")
	})

	t.Run("backoff after error", func(t *testing.T) {
		t.Parallel()

		var (
			logs = &syncBuffer{}
			g    = newTestGroup(t, logs, consumeErr(errors.New("broker is unavailable")))
			done = make(chan error, 1)
		)

		g.addHandler(t)

		ctx, cancel := context.WithCancel(context.Background())

		go func() { done <- g.Run(ctx) }()

		time.Sleep(300 * time.Millisecond)

		calls := g.consumeCalls.Load()
		assert.GreaterOrEqual(t, calls, int64(1))
		assert.LessOrEqual(t, calls, int64(5), "Consume must be paused with a backoff after errors")

		start := time.Now()

		cancel()

		select {
		case err := <-done:
			require.NoError(t, err)
			assert.Less(t, time.Since(start), time.Second, "the backoff must be interrupted by the context")
		case <-time.After(5 * time.Second):
			t.Fatal("Run was not finished after the context cancel")
		}

		assert.Contains(t, logs.String(), "broker is unavailable")
	})

	t.Run("close interrupts backoff", func(t *testing.T) {
		t.Parallel()

		var (
			g    = newTestGroup(t, &syncBuffer{}, consumeErr(errors.New("boom")))
			done = make(chan error, 1)
		)

		g.addHandler(t)

		go func() { done <- g.Run(context.Background()) }()

		time.Sleep(50 * time.Millisecond)

		require.NoError(t, g.Close())

		select {
		case err := <-done:
			require.NoError(t, err)
		case <-time.After(5 * time.Second):
			t.Fatal("Run was not finished after Close")
		}
	})

	t.Run("rebalance without pause", func(t *testing.T) {
		t.Parallel()

		g := newTestGroup(t, &syncBuffer{}, func(context.Context) error {
			time.Sleep(time.Millisecond)
			return nil
		})
		g.addHandler(t)

		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()

		require.NoError(t, g.Run(ctx))
		assert.Greater(t, g.consumeCalls.Load(), int64(10))
	})

	t.Run("closed consumer group", func(t *testing.T) {
		t.Parallel()

		g := newTestGroup(t, &syncBuffer{}, consumeErr(sarama.ErrClosedConsumerGroup))
		g.addHandler(t)

		require.NoError(t, g.Run(context.Background()))
		assert.EqualValues(t, 1, g.consumeCalls.Load())
	})
}

func TestUnit_Group_Close(t *testing.T) {
	t.Parallel()

	var (
		g = newTestGroup(t, &syncBuffer{}, consumeErr(nil))

		wg      sync.WaitGroup
		success atomic.Int64
		closed  atomic.Int64
	)

	for range 16 {
		wg.Go(func() {
			switch err := g.Close(); {
			case err == nil:
				success.Add(1)
			case errors.Is(err, ErrClosed):
				closed.Add(1)
			}
		})
	}

	wg.Wait()

	assert.EqualValues(t, 1, success.Load(), "the client must be closed once")
	assert.EqualValues(t, 15, closed.Load())
	assert.ErrorIs(t, g.Close(), ErrClosed)
}

func TestUnit_Group_logErrors(t *testing.T) {
	t.Parallel()

	var (
		logs = &syncBuffer{}
		g    = newTestGroup(t, logs, consumeErr(nil))
	)

	g.errs <- errors.New("offset commit failed")

	require.Eventually(t, func() bool {
		out := logs.String()

		return strings.Contains(out, "consumer group error") &&
			strings.Contains(out, "offset commit failed") &&
			strings.Contains(out, "group=test-group")
	}, 2*time.Second, 5*time.Millisecond)
}
