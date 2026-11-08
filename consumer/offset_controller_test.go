package consumer

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func newTestOffsetController(t *testing.T, opts []Option, handlers ...Handler) *offsetController {
	t.Helper()

	hCtl := make(handlerController)
	for _, h := range handlers {
		hCtl.Add(h)
	}

	opts = append([]Option{WithLogger(newTestLogger(&syncBuffer{}))}, opts...)

	return newOffsetController(hCtl, newOptions(opts...))
}

func committed(topic, partition string) float64 {
	return testutil.ToFloat64(committedOffset.WithLabelValues(topic, partition))
}

func TestUnit_offsetController_Inc(t *testing.T) {
	t.Parallel()

	t.Run("signal every commit count", func(t *testing.T) {
		t.Parallel()

		oCtl := newTestOffsetController(t, []Option{WithCommitCount(3)})

		oCtl.Inc()
		oCtl.Inc()
		assert.Empty(t, oCtl.commitSig)

		oCtl.Inc()
		assert.Len(t, oCtl.commitSig, 1)

		for range 6 {
			oCtl.Inc()
		}

		assert.Len(t, oCtl.commitSig, 1, "a pending signal is not duplicated")
	})

	t.Run("zero commit count", func(t *testing.T) {
		t.Parallel()

		oCtl := newTestOffsetController(t, []Option{WithCommitCount(0)})

		for range 100 {
			oCtl.Inc()
		}

		assert.Empty(t, oCtl.commitSig)
	})
}

func TestUnit_offsetController_Mark(t *testing.T) {
	t.Parallel()

	t.Run("handled offsets", func(t *testing.T) {
		t.Parallel()

		const topic = "oc_mark_topic"

		var (
			offsets = offsetsOf(map[int32]int64{0: 0, 1: 0})
			oCtl    = newTestOffsetController(t, nil, newOffsetHandler(t, topic, offsets))
			sess    = newSessionMock(t, context.Background(), map[string][]int32{topic: {0, 1}})
		)

		assert.False(t, oCtl.Mark(sess), "nothing is handled yet")

		offsets[0].Store(14)
		sess.EXPECT().MarkOffset(topic, int32(0), int64(15), "ok")
		assert.True(t, oCtl.Mark(sess))
		assert.Equal(t, 15.0, committed(topic, "0"))

		offsets[0].Store(15)
		offsets[1].Store(2)
		sess.EXPECT().MarkOffset(topic, int32(0), int64(16), "ok")
		sess.EXPECT().MarkOffset(topic, int32(1), int64(3), "ok")
		assert.True(t, oCtl.Mark(sess))
		assert.Equal(t, 16.0, committed(topic, "0"))
		assert.Equal(t, 3.0, committed(topic, "1"))
	})

	t.Run("concurrent calls", func(t *testing.T) {
		t.Parallel()

		const topic = "oc_concurrent_topic"

		var (
			offsets = offsetsOf(map[int32]int64{0: 0})
			oCtl    = newTestOffsetController(t, nil, newOffsetHandler(t, topic, offsets))
			sess    = newSessionMock(t, context.Background(), map[string][]int32{topic: {0}})
			done    = make(chan struct{})
		)

		sess.EXPECT().MarkOffset(topic, int32(0), gomock.Any(), "ok").AnyTimes()

		go func() {
			defer close(done)

			for i := range 200 {
				offsets[0].Store(int64(i + 1))
				oCtl.Mark(sess)
			}
		}()

		for range 200 {
			oCtl.Mark(sess)
		}

		<-done
		oCtl.Mark(sess)

		assert.Equal(t, 201.0, committed(topic, "0"))
	})
}

func TestUnit_offsetController_StartCommitting(t *testing.T) {
	t.Parallel()

	const topic = "oc_start_committing_topic"

	run := func(oCtl *offsetController, sess *MockConsumerGroupSession) <-chan struct{} {
		done := make(chan struct{})

		go func() {
			defer close(done)

			oCtl.StartCommitting(sess)
		}()

		return done
	}

	t.Run("commit on signal", func(t *testing.T) {
		t.Parallel()

		var (
			ctx, cancel = context.WithCancel(context.Background())
			commits     = make(chan struct{}, 1)
			oCtl        = newTestOffsetController(t, []Option{WithCommitDuration(0)},
				newOffsetHandler(t, topic, offsetsOf(map[int32]int64{0: 4})))
			sess = newSessionMock(t, ctx, map[string][]int32{topic: {0}})
		)

		sess.EXPECT().MarkOffset(topic, int32(0), int64(5), "ok")
		sess.EXPECT().Commit().Do(func() { commits <- struct{}{} })

		done := run(oCtl, sess)
		oCtl.sendSig()

		select {
		case <-commits:
		case <-time.After(2 * time.Second):
			t.Fatal("the signal did not commit the offsets")
		}

		cancel()
		<-done
	})

	t.Run("no commit without handled offsets", func(t *testing.T) {
		t.Parallel()

		var (
			ctx, cancel = context.WithCancel(context.Background())
			oCtl        = newTestOffsetController(t, []Option{WithCommitDuration(0)},
				newOffsetHandler(t, topic, nil))
			sess = newSessionMock(t, ctx, map[string][]int32{topic: {0}})
		)

		done := run(oCtl, sess)
		oCtl.sendSig()

		assert.Eventually(t, func() bool { return len(oCtl.commitSig) == 0 }, 2*time.Second, time.Millisecond)

		cancel()
		<-done
	})

	t.Run("commit by duration", func(t *testing.T) {
		t.Parallel()

		var (
			ctx, cancel = context.WithCancel(context.Background())
			commits     atomic.Int64
			oCtl        = newTestOffsetController(t, []Option{WithCommitDuration(10 * time.Millisecond)},
				newOffsetHandler(t, topic, offsetsOf(map[int32]int64{0: 4})))
			sess = newSessionMock(t, ctx, map[string][]int32{topic: {0}})
		)

		sess.EXPECT().MarkOffset(topic, int32(0), int64(5), "ok").AnyTimes()
		sess.EXPECT().Commit().Do(func() { commits.Add(1) }).AnyTimes()

		done := run(oCtl, sess)

		assert.Eventually(t, func() bool { return commits.Load() >= 2 }, 2*time.Second, time.Millisecond)

		cancel()
		<-done
	})

	t.Run("session done", func(t *testing.T) {
		t.Parallel()

		var (
			oCtl = newTestOffsetController(t, nil)
			sess = newSessionMock(t, canceledContext(), nil)
		)

		select {
		case <-run(oCtl, sess):
		case <-time.After(2 * time.Second):
			t.Fatal("StartCommitting must return when the session is done")
		}
	})
}
