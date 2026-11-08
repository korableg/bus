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

func TestUnit_Encoder(t *testing.T) {
	t.Parallel()

	t.Run("message", func(t *testing.T) {
		t.Parallel()

		msg := &test.TestEvent1{Id: "42", Num: 7}
		enc := wire.Encoder()(msg)

		payload, err := enc.Marshal()
		require.NoError(t, err)

		var got test.TestEvent1
		require.NoError(t, proto.Unmarshal(payload, &got))
		assert.True(t, proto.Equal(msg, &got))

		assert.Equal(t, "test_topic", enc.Topic())
		assert.Equal(t, "event.TestEvent1", enc.Type())
		assert.Equal(t, []byte("42"), enc.Key())
		assert.Equal(t, []codec.Header{{Key: []byte("type"), Value: []byte("event.TestEvent1")}}, enc.Headers())
	})

	t.Run("non-string key", func(t *testing.T) {
		t.Parallel()

		enc := wire.Encoder()(&test.TestEvent4{Id: 17})

		assert.Equal(t, "test_topic_2", enc.Topic())
		assert.Equal(t, []byte("17"), enc.Key())
	})

	t.Run("message without event options", func(t *testing.T) {
		t.Parallel()

		enc := wire.Encoder()(&test.NonEvent{Color: "red"})

		_, err := enc.Marshal()
		require.NoError(t, err)
		assert.Empty(t, enc.Topic())
		assert.Nil(t, enc.Key())
	})

	var typedNil *test.TestEvent1

	for name, msg := range map[string]proto.Message{
		"nil interface":     nil,
		"typed nil pointer": typedNil,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			enc := wire.Encoder()(msg)

			_, err := enc.Marshal()
			require.ErrorIs(t, err, codec.ErrNilMessage)

			assert.NotPanics(t, func() {
				enc.Topic()
				enc.Type()
				enc.Key()
				enc.Headers()
			})
		})
	}
}
