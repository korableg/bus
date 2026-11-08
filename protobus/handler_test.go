package protobus_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/korableg/bus/codec/proto/wire"
	"github.com/korableg/bus/consumer"
	"github.com/korableg/bus/protobus"
	"github.com/korableg/bus/test"
)

func message(t *testing.T, msg proto.Message, partition int32, offset int64) consumer.Message {
	t.Helper()

	enc := wire.Encoder()(msg)

	payload, err := enc.Marshal()
	require.NoError(t, err)

	return consumer.Message{
		Value:     payload,
		Headers:   enc.Headers(),
		Topic:     enc.Topic(),
		Partition: partition,
		Offset:    offset,
	}
}

func TestUnit_NewHandler(t *testing.T) {
	t.Parallel()

	var (
		want = &test.TestEvent1{Id: "42", Num: 7}
		raw  = message(t, want, 1, 5)
		got  []*test.TestEvent1
	)

	// The type of the event is inferred from the function.
	h := protobus.NewHandler(func(_ context.Context, msg *test.TestEvent1, r consumer.Message) error {
		got = append(got, msg)
		assert.Equal(t, raw, r)

		return nil
	}, consumer.WithoutHandlerRetry())

	assert.Equal(t, "test_topic", h.Topic())
	assert.Equal(t, "event.TestEvent1", h.Type())

	h.Handle(context.Background(), raw)
	h.Handle(context.Background(), message(t, &test.TestEvent2{Solution: "s"}, 1, 6))

	require.Len(t, got, 1, "the message of another type is skipped")
	assert.True(t, proto.Equal(want, got[0]))
	assert.EqualValues(t, 6, h.Offset(1))
}

func TestUnit_NewHandlerBatch(t *testing.T) {
	t.Parallel()

	var (
		want = []*test.TestEvent1{{Id: "1"}, {Id: "2"}}
		raws = []consumer.Message{message(t, want[0], 0, 3), message(t, want[1], 0, 4)}
		got  [][]*test.TestEvent1
	)

	h := protobus.NewHandlerBatch(func(_ context.Context, msgs []*test.TestEvent1, r []consumer.Message) error {
		got = append(got, msgs)
		assert.Equal(t, raws, r)

		return nil
	}, consumer.WithoutHandlerRetry(), consumer.WithHandlerBatchSize(2), consumer.WithHandlerBatchFlushTimeout(0))

	assert.Equal(t, "test_topic", h.Topic())
	assert.Equal(t, "event.TestEvent1", h.Type())

	for _, raw := range raws {
		h.Handle(context.Background(), raw)
	}

	require.Len(t, got, 1, "the options are passed: the batch is flushed by its size")
	require.Len(t, got[0], 2)
	assert.True(t, proto.Equal(want[0], got[0][0]))
	assert.True(t, proto.Equal(want[1], got[0][1]))
	assert.EqualValues(t, 4, h.Offset(0))
}
