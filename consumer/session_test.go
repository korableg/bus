package consumer

import (
	"context"
	"testing"
	"time"

	"github.com/IBM/sarama"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/korableg/bus/codec"
)

type traceKey struct{}

func withTrace(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, traceKey{}, traceID)
}

func newTestSession(t *testing.T, logs *syncBuffer, opts []Option, handlers ...Handler) *session {
	t.Helper()

	oCtl := newTestOffsetController(t, opts, handlers...)

	return newSession(oCtl.hCtl, oCtl, "group", newTestLogger(logs), withTrace)
}

func TestUnit_session_Setup(t *testing.T) {
	t.Parallel()

	var (
		logs     = &syncBuffer{}
		s        = newTestSession(t, logs, []Option{WithCommitDuration(0)})
		ctxCalls = make(chan struct{}, 1)
		sess     = NewMockConsumerGroupSession(gomock.NewController(t))
	)

	sess.EXPECT().Claims().Return(map[string][]int32{"topic": {0, 1}})
	sess.EXPECT().Context().DoAndReturn(func() context.Context {
		select {
		case ctxCalls <- struct{}{}:
		default:
		}

		return canceledContext()
	}).AnyTimes()

	require.NoError(t, s.Setup(sess))

	select {
	case <-ctxCalls:
	case <-time.After(2 * time.Second):
		t.Fatal("Setup must start committing")
	}

	assert.Contains(t, logs.String(), "rebalance group group: assigned topic: topic, partitions: [0 1]")
}

func TestUnit_session_Cleanup(t *testing.T) {
	t.Parallel()

	var (
		logs = &syncBuffer{}
		h    = newOffsetHandler(t, "cleanup_topic", offsetsOf(map[int32]int64{0: 5}))
		s    = newTestSession(t, logs, nil, h)
		sess = newSessionMock(t, context.Background(), map[string][]int32{"cleanup_topic": {0}})
	)

	gomock.InOrder(
		sess.EXPECT().MarkOffset("cleanup_topic", int32(0), int64(6), "ok"),
		sess.EXPECT().Commit(),
	)

	require.NoError(t, s.Cleanup(sess))
	assert.Contains(t, logs.String(), "rebalance group group: revoked topic: cleanup_topic, partitions: [0]")
}

func TestUnit_session_ConsumeClaim(t *testing.T) {
	t.Parallel()

	t.Run("messages", func(t *testing.T) {
		t.Parallel()

		var (
			ts   = time.Now()
			msgs = make(chan *sarama.ConsumerMessage, 2)
			want = []Message{
				{
					Key:       []byte("key"),
					Value:     []byte("value"),
					ID:        "id",
					Group:     "group",
					Topic:     "topic",
					TraceID:   "trace",
					Partition: 1,
					Offset:    10,
					Timestamp: ts,
					Headers: []codec.Header{
						{Key: []byte(codec.HeaderMessageID), Value: []byte("id")},
						{Key: []byte(codec.HeaderTraceID), Value: []byte("trace")},
						{Key: []byte("type"), Value: []byte("type")},
					},
				},
				{Group: "group", Topic: "topic", Partition: 1, Offset: 11, Headers: []codec.Header{}},
			}
			h1   = NewMockHandler(gomock.NewController(t))
			h2   = NewMockHandler(gomock.NewController(t))
			sess = newSessionMock(t, context.Background(), nil)
		)

		msgs <- &sarama.ConsumerMessage{
			Key:       []byte("key"),
			Value:     []byte("value"),
			Topic:     "topic",
			Partition: 1,
			Offset:    10,
			Timestamp: ts,
			Headers: []*sarama.RecordHeader{
				{Key: []byte(codec.HeaderMessageID), Value: []byte("id")},
				{Key: []byte(codec.HeaderTraceID), Value: []byte("trace")},
				{Key: []byte("type"), Value: []byte("type")},
			},
		}
		msgs <- &sarama.ConsumerMessage{Topic: "topic", Partition: 1, Offset: 11}
		close(msgs)

		for _, h := range []*MockHandler{h1, h2} {
			h.EXPECT().Topic().Return("topic").AnyTimes()
			gomock.InOrder(
				h.EXPECT().Handle(gomock.Any(), want[0]).Do(func(ctx context.Context, _ Message) {
					assert.Equal(t, "trace", ctx.Value(traceKey{}))
				}),
				h.EXPECT().Handle(gomock.Any(), want[1]).Do(func(ctx context.Context, _ Message) {
					assert.Nil(t, ctx.Value(traceKey{}))
				}),
			)
		}

		s := newTestSession(t, &syncBuffer{}, []Option{WithCommitCount(2)}, h1, h2)

		require.NoError(t, s.ConsumeClaim(sess, newClaimMock(t, "topic", msgs)))
		assert.Len(t, s.oCtl.commitSig, 1, "every handled message counts for the commit")
	})

	t.Run("session done", func(t *testing.T) {
		t.Parallel()

		var (
			s    = newTestSession(t, &syncBuffer{}, nil)
			sess = newSessionMock(t, canceledContext(), nil)
			msgs = make(chan *sarama.ConsumerMessage)
			done = make(chan error, 1)
		)

		go func() { done <- s.ConsumeClaim(sess, newClaimMock(t, "topic", msgs)) }()

		select {
		case err := <-done:
			require.NoError(t, err)
		case <-time.After(2 * time.Second):
			t.Fatal("ConsumeClaim must return when the session is done")
		}
	})
}
