package wire_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/korableg/bus/codec"
	"github.com/korableg/bus/codec/proto/wire"
	"github.com/korableg/bus/test"
)

func TestUnit_Decoder(t *testing.T) {
	t.Parallel()

	dec := wire.Decoder[*test.TestEvent1]()

	assert.Equal(t, "test_topic", dec.Topic())
	assert.Equal(t, "event.TestEvent1", dec.Type())

	encode := func(t *testing.T, msg proto.Message) ([]byte, []codec.Header) {
		t.Helper()

		enc := wire.Encoder()(msg)

		payload, err := enc.Marshal()
		require.NoError(t, err)

		return payload, enc.Headers()
	}

	t.Run("message of the type", func(t *testing.T) {
		t.Parallel()

		want := &test.TestEvent1{Id: "42", Num: 7, Nm: &test.NestedMsg{Name: "n"}}

		got, err := dec.Unmarshal(encode(t, want))
		require.NoError(t, err)
		assert.True(t, proto.Equal(want, got))
	})

	t.Run("message of another type", func(t *testing.T) {
		t.Parallel()

		_, err := dec.Unmarshal(encode(t, &test.TestEvent2{Solution: "s"}))
		require.ErrorIs(t, err, codec.ErrWrongType)
	})

	t.Run("message without type header", func(t *testing.T) {
		t.Parallel()

		payload, _ := encode(t, &test.TestEvent1{Id: "42"})

		_, err := dec.Unmarshal(payload, nil)
		require.ErrorIs(t, err, codec.ErrWrongType)
	})

	t.Run("broken payload", func(t *testing.T) {
		t.Parallel()

		_, headers := encode(t, &test.TestEvent1{Id: "42"})

		_, err := dec.Unmarshal([]byte{0xff, 0xff, 0xff}, headers)
		require.Error(t, err)
		assert.NotErrorIs(t, err, codec.ErrWrongType)
	})
}
