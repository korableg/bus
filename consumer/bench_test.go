package consumer

import (
	"context"
	"testing"
	"time"

	"github.com/korableg/bus/codec"
	"github.com/korableg/bus/codec/proto/wire"
	"github.com/korableg/bus/test"
)

func benchMessage(b *testing.B) Message {
	b.Helper()

	enc := wire.Encoder()(&test.TestEvent1{
		Id:  "0199f7d4-5c2e-7b6a-9d1e-3f4a5b6c7d8e",
		Num: 42,
		Nm: &test.NestedMsg{
			Name:     "nested_name",
			Duration: 1000,
		},
	})

	payload, err := enc.Marshal()
	if err != nil {
		b.Fatal(err)
	}

	return Message{
		Key:       enc.Key(),
		Value:     payload,
		ID:        "0199f7d4-5c2e-7b6a-9d1e-3f4a5b6c7d8f",
		Group:     "bench",
		Topic:     enc.Topic(),
		Timestamp: time.Now(),
		Headers: append([]codec.Header{
			{Key: codec.HeaderMessageIDByte, Value: []byte("0199f7d4-5c2e-7b6a-9d1e-3f4a5b6c7d8f")},
		}, enc.Headers()...),
	}
}

// BenchmarkHandler measures the handler with the default options: decoding, the call with the timeout
// and the retry wrapper, the metrics and the offset update.
func BenchmarkHandler(b *testing.B) {
	var (
		raw = benchMessage(b)
		ctx = context.Background()
		h   = NewHandler(wire.Decoder[*test.TestEvent1](),
			func(context.Context, *test.TestEvent1, Message) error { return nil })
	)

	b.ReportAllocs()

	for b.Loop() {
		raw.Offset++
		h.Handle(ctx, raw)
	}
}

// BenchmarkHandlerBatch measures the batch handler with the default options (the batch size is 100),
// the cost of one message.
func BenchmarkHandlerBatch(b *testing.B) {
	var (
		raw = benchMessage(b)
		ctx = context.Background()
		h   = NewHandlerBatch(wire.Decoder[*test.TestEvent1](),
			func(context.Context, []*test.TestEvent1, []Message) error { return nil })
	)

	b.ReportAllocs()

	for b.Loop() {
		raw.Offset++
		h.Handle(ctx, raw)
	}

	b.StopTimer()

	h.bufMu.Lock()
	if h.timer != nil {
		h.timer.Stop()
	}
	h.bufMu.Unlock()
}
