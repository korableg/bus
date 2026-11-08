package producer

import (
	"context"
	"testing"

	"github.com/IBM/sarama"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/korableg/bus/codec"
)

type encoded struct {
	topic   string
	typ     string
	key     []byte
	payload []byte
	headers []codec.Header
	err     error
}

func validEncoded(topic string) encoded {
	return encoded{
		topic:   topic,
		typ:     "type",
		key:     []byte("key"),
		payload: []byte("payload"),
		headers: []codec.Header{{Key: []byte("type"), Value: []byte("type")}},
	}
}

func newEncoder(t *testing.T, e encoded) func(string) codec.Encoder[string] {
	t.Helper()

	enc := NewMockEncoder[string](gomock.NewController(t))
	enc.EXPECT().Marshal().Return(e.payload, e.err).AnyTimes()
	enc.EXPECT().Topic().Return(e.topic).AnyTimes()
	enc.EXPECT().Type().Return(e.typ).AnyTimes()
	enc.EXPECT().Key().Return(e.key).AnyTimes()
	enc.EXPECT().Headers().Return(e.headers).AnyTimes()

	return func(string) codec.Encoder[string] { return enc }
}

type producerMocks struct {
	cli   *MockClient
	async *MockAsyncProducer
	sync  *MockSyncProducer

	input     chan *sarama.ProducerMessage
	successes chan *sarama.ProducerMessage
	errs      chan *sarama.ProducerError

	syncCloseErr error
	cliCloseErr  error
	flushErr     error
}

// newProducerMocks returns the mocks of an async producer with the input queue of inputSize
// (0 means Send blocks until Close). AsyncClose replies to the queued messages with flushErr
// before closing the channels, as sarama flushes them.
func newProducerMocks(t *testing.T, inputSize int) *producerMocks {
	t.Helper()

	ctrl := gomock.NewController(t)

	m := &producerMocks{
		cli:       NewMockClient(ctrl),
		async:     NewMockAsyncProducer(ctrl),
		sync:      NewMockSyncProducer(ctrl),
		input:     make(chan *sarama.ProducerMessage, inputSize),
		successes: make(chan *sarama.ProducerMessage),
		errs:      make(chan *sarama.ProducerError),
	}

	m.async.EXPECT().Input().Return(m.input).AnyTimes()
	m.async.EXPECT().Successes().Return(m.successes).AnyTimes()
	m.async.EXPECT().Errors().Return(m.errs).AnyTimes()

	m.async.EXPECT().AsyncClose().Do(func() {
		for len(m.input) > 0 {
			m.reply(m.flushErr)
		}

		close(m.successes)
		close(m.errs)
	})
	m.sync.EXPECT().Close().DoAndReturn(func() error { return m.syncCloseErr })
	m.cli.EXPECT().Close().DoAndReturn(func() error { return m.cliCloseErr })

	return m
}

func (m *producerMocks) newProducer(t *testing.T, enc func(string) codec.Encoder[string], opts ...Option) *Producer[string] {
	t.Helper()

	p := newProducer(m.cli, m.async, m.sync, enc, newOptions(opts...))
	t.Cleanup(func() { _ = p.Close() })

	return p
}

func (m *producerMocks) reply(err error) {
	msg := <-m.input

	if err == nil {
		m.successes <- msg
		return
	}

	m.errs <- &sarama.ProducerError{Msg: msg, Err: err}
}

func canceledContext() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	return ctx
}

func sendSamples(t *testing.T, topic, errLabel string) uint64 {
	t.Helper()

	var m dto.Metric

	require.NoError(t, sendHistogram.WithLabelValues(topic, "type", errLabel).(prometheus.Metric).Write(&m))

	return m.GetHistogram().GetSampleCount()
}
