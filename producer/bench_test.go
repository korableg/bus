package producer

import (
	"context"
	"testing"

	"google.golang.org/protobuf/proto"

	"github.com/korableg/bus/codec/proto/wire"
	"github.com/korableg/bus/test"
)

// BenchmarkProducerMessage measures the work Send and SendSync do before passing the message to sarama:
// encoding, the key, the message ID and the headers.
func BenchmarkProducerMessage(b *testing.B) {
	var (
		p = &Producer[proto.Message]{
			encoder:      wire.Encoder(),
			isIdempotent: true,
			traceID:      func(context.Context) string { return "0199f7d4-5c2e-7b6a-9d1e-3f4a5b6c7d8e" },
		}
		msg = &test.TestEvent1{
			Id:  "0199f7d4-5c2e-7b6a-9d1e-3f4a5b6c7d8e",
			Num: 42,
			Nm: &test.NestedMsg{
				Name:     "nested_name",
				Duration: 1000,
			},
		}
		ctx = context.Background()
		err error
	)

	b.ReportAllocs()

	for b.Loop() {
		if _, err = p.producerMessage(ctx, msg); err != nil {
			b.Fatal(err)
		}
	}
}
