package consumer

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/IBM/sarama"
	"go.uber.org/mock/gomock"

	"github.com/korableg/bus/codec"
)

type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.buf.String()
}

func newTestLogger(w *syncBuffer) *slog.Logger {
	return slog.New(slog.NewTextHandler(w, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

var errBroken = errors.New("broken payload")

func decodeString(payload string) (string, error) { return payload, nil }

func decodeWrongType(string) (string, error) { return "", codec.ErrWrongType }

// decodePrefix accepts the payloads with prefix, the "broken" payload can't be decoded,
// the other payloads are of another type
func decodePrefix(prefix string) func(string) (string, error) {
	return func(payload string) (string, error) {
		switch {
		case payload == "broken":
			return "", errBroken
		case strings.HasPrefix(payload, prefix):
			return payload, nil
		default:
			return "", codec.ErrWrongType
		}
	}
}

func newDecoder(t *testing.T, decode func(payload string) (string, error)) *MockDecoder[string] {
	t.Helper()

	dec := NewMockDecoder[string](gomock.NewController(t))
	dec.EXPECT().Topic().Return("topic").AnyTimes()
	dec.EXPECT().Type().Return("type").AnyTimes()
	dec.EXPECT().Unmarshal(gomock.Any(), gomock.Any()).DoAndReturn(func(data []byte, _ []codec.Header) (string, error) {
		return decode(string(data))
	}).AnyTimes()

	return dec
}

func newOffsetHandler(t *testing.T, topic string, offsets map[int32]*atomic.Int64) *MockHandler {
	t.Helper()

	h := NewMockHandler(gomock.NewController(t))
	h.EXPECT().Topic().Return(topic).AnyTimes()
	h.EXPECT().Offset(gomock.Any()).DoAndReturn(func(partition int32) int64 {
		if off, ok := offsets[partition]; ok {
			return off.Load()
		}

		return 0
	}).AnyTimes()

	return h
}

func msgOf(partition int32, offset int64, value string) Message {
	return Message{Topic: "topic", Partition: partition, Offset: offset, Value: []byte(value)}
}

type batchCall struct {
	at   time.Time
	msgs []string
}

type batchRecorder struct {
	mu    sync.Mutex
	calls []batchCall

	active, maxActive atomic.Int32
}

func (r *batchRecorder) fn(_ context.Context, msgs []string, _ []Message) error {
	n := r.active.Add(1)
	defer r.active.Add(-1)

	for m := r.maxActive.Load(); n > m && !r.maxActive.CompareAndSwap(m, n); m = r.maxActive.Load() {
	}

	r.mu.Lock()
	r.calls = append(r.calls, batchCall{at: time.Now(), msgs: msgs})
	r.mu.Unlock()

	time.Sleep(time.Millisecond) // widens the window of the overlapping calls

	return nil
}

func (r *batchRecorder) snapshot() []batchCall {
	r.mu.Lock()
	defer r.mu.Unlock()

	return append([]batchCall(nil), r.calls...)
}

func newTestBatchHandler(t *testing.T, decode func(string) (string, error), f HandlerBatchFunc[string], opts ...HandlerOption) *HandlerBatch[string] {
	t.Helper()

	opts = append([]HandlerOption{
		WithoutHandlerRetry(),
		WithHandlerLogger(newTestLogger(&syncBuffer{})),
	}, opts...)

	return NewHandlerBatch(newDecoder(t, decode), f, opts...)
}

func newSessionMock(t *testing.T, ctx context.Context, claims map[string][]int32) *MockConsumerGroupSession {
	t.Helper()

	sess := NewMockConsumerGroupSession(gomock.NewController(t))
	sess.EXPECT().Context().Return(ctx).AnyTimes()
	sess.EXPECT().Claims().Return(claims).AnyTimes()

	return sess
}

func newClaimMock(t *testing.T, topic string, msgs <-chan *sarama.ConsumerMessage) *MockConsumerGroupClaim {
	t.Helper()

	claim := NewMockConsumerGroupClaim(gomock.NewController(t))
	claim.EXPECT().Topic().Return(topic).AnyTimes()
	claim.EXPECT().Messages().Return(msgs).AnyTimes()

	return claim
}

func canceledContext() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	return ctx
}
