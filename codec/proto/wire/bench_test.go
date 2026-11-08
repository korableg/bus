package wire_test

import (
	"testing"

	"github.com/korableg/bus/codec/proto/wire"
	"github.com/korableg/bus/test"
)

func benchEvent() *test.TestEvent1 {
	return &test.TestEvent1{
		Id:  "0199f7d4-5c2e-7b6a-9d1e-3f4a5b6c7d8e",
		Num: 42,
		Nm: &test.NestedMsg{
			Name:     "nested_name",
			Duration: 1000,
		},
	}
}

// BenchmarkEncoder measures what the producer asks the encoder for every message.
func BenchmarkEncoder(b *testing.B) {
	var (
		newEncoder = wire.Encoder()
		msg        = benchEvent()
		err        error
	)

	b.ReportAllocs()

	for b.Loop() {
		enc := newEncoder(msg)

		if _, err = enc.Marshal(); err != nil {
			b.Fatal(err)
		}

		_ = enc.Topic()
		_ = enc.Type()
		_ = enc.Key()
		_ = enc.Headers()
	}
}

func BenchmarkDecoder(b *testing.B) {
	enc := wire.Encoder()(benchEvent())

	payload, err := enc.Marshal()
	if err != nil {
		b.Fatal(err)
	}

	var (
		headers = enc.Headers()
		dec     = wire.Decoder[*test.TestEvent1]()
	)

	b.ReportAllocs()

	for b.Loop() {
		if _, err = dec.Unmarshal(payload, headers); err != nil {
			b.Fatal(err)
		}
	}
}
