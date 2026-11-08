package producer

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/IBM/sarama"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/korableg/bus/codec"
)

func TestUnit_New(t *testing.T) {
	t.Parallel()

	enc := newEncoder(t, validEncoded("topic"))

	tests := []struct {
		name    string
		enc     func(string) codec.Encoder[string]
		brokers []string
		wantErr string
	}{
		{name: "nil encoder", enc: nil, brokers: []string{"127.0.0.1:1"}, wantErr: "encoder is not defined"},
		{name: "no brokers", enc: enc, brokers: nil, wantErr: "brokers are empty"},
		{name: "unreachable broker", enc: enc, brokers: []string{"127.0.0.1:1"}, wantErr: "init sarama client"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			p, err := New(tt.enc, tt.brokers, WithDialTimeout(50*time.Millisecond))
			require.ErrorContains(t, err, tt.wantErr)
			assert.Nil(t, p)
		})
	}
}

func TestUnit_Producer_Send(t *testing.T) {
	t.Parallel()

	t.Run("queues message", func(t *testing.T) {
		t.Parallel()

		m := newProducerMocks(t, 1)
		p := m.newProducer(t, newEncoder(t, validEncoded("topic")))

		require.NoError(t, p.Send(context.Background(), "msg"))

		msg := <-m.input
		assert.Equal(t, "topic", msg.Topic)
		assert.Equal(t, sarama.ByteEncoder("payload"), msg.Value)
	})

	t.Run("invalid message", func(t *testing.T) {
		t.Parallel()

		e := validEncoded("")

		m := newProducerMocks(t, 1)
		p := m.newProducer(t, newEncoder(t, e))

		require.ErrorIs(t, p.Send(context.Background(), "msg"), ErrEmptyTopic)
		assert.Empty(t, m.input)
	})

	t.Run("closed producer", func(t *testing.T) {
		t.Parallel()

		m := newProducerMocks(t, 1)
		p := m.newProducer(t, newEncoder(t, validEncoded("topic")))

		require.NoError(t, p.Close())
		require.ErrorIs(t, p.Send(context.Background(), "msg"), ErrClosed)
	})

	t.Run("canceled context", func(t *testing.T) {
		t.Parallel()

		m := newProducerMocks(t, 1)
		p := m.newProducer(t, newEncoder(t, validEncoded("topic")))

		require.ErrorIs(t, p.Send(canceledContext(), "msg"), context.Canceled)
		assert.Empty(t, m.input, "a canceled context must not queue the message")
	})

	t.Run("context canceled while queue is full", func(t *testing.T) {
		t.Parallel()

		var (
			m     = newProducerMocks(t, 0)
			p     = m.newProducer(t, newEncoder(t, validEncoded("topic")))
			errCh = make(chan error, 1)
		)

		ctx, cancel := context.WithCancel(context.Background())

		go func() { errCh <- p.Send(ctx, "msg") }()

		select {
		case err := <-errCh:
			t.Fatalf("Send must block, got %v", err)
		case <-time.After(50 * time.Millisecond):
		}

		cancel()

		select {
		case err := <-errCh:
			require.ErrorIs(t, err, context.Canceled)
		case <-time.After(2 * time.Second):
			t.Fatal("Send ignores the context")
		}
	})

	t.Run("close releases send blocked on full queue", func(t *testing.T) {
		t.Parallel()

		var (
			m     = newProducerMocks(t, 0)
			p     = m.newProducer(t, newEncoder(t, validEncoded("topic")))
			errCh = make(chan error, 1)
		)

		go func() { errCh <- p.Send(context.Background(), "msg") }()

		time.Sleep(50 * time.Millisecond)

		require.NoError(t, p.Close())

		select {
		case err := <-errCh:
			require.ErrorIs(t, err, ErrClosed)
		case <-time.After(2 * time.Second):
			t.Fatal("blocked Send was not released by Close")
		}
	})
}

func TestUnit_Producer_SendSync(t *testing.T) {
	t.Parallel()

	t.Run("sends message", func(t *testing.T) {
		t.Parallel()

		m := newProducerMocks(t, 0)
		p := m.newProducer(t, newEncoder(t, validEncoded("topic")))

		m.sync.EXPECT().SendMessage(gomock.Cond(func(msg *sarama.ProducerMessage) bool {
			return msg.Topic == "topic"
		})).Return(int32(0), int64(0), nil)

		require.NoError(t, p.SendSync(context.Background(), "msg"))
	})

	t.Run("send error", func(t *testing.T) {
		t.Parallel()

		var (
			cause = errors.New("broker is unavailable")
			m     = newProducerMocks(t, 0)
			p     = m.newProducer(t, newEncoder(t, validEncoded("topic")))
		)

		m.sync.EXPECT().SendMessage(gomock.Any()).Return(int32(0), int64(0), cause)

		require.ErrorIs(t, p.SendSync(context.Background(), "msg"), cause)
	})

	t.Run("invalid message", func(t *testing.T) {
		t.Parallel()

		e := validEncoded("topic")
		e.key = nil

		m := newProducerMocks(t, 0)
		p := m.newProducer(t, newEncoder(t, e))

		require.ErrorIs(t, p.SendSync(context.Background(), "msg"), ErrEmptyKey)
	})

	t.Run("closed producer", func(t *testing.T) {
		t.Parallel()

		m := newProducerMocks(t, 0)
		p := m.newProducer(t, newEncoder(t, validEncoded("topic")))

		require.NoError(t, p.Close())
		require.ErrorIs(t, p.SendSync(context.Background(), "msg"), ErrClosed)
	})
}

func TestUnit_Producer_Close(t *testing.T) {
	t.Parallel()

	t.Run("second call", func(t *testing.T) {
		t.Parallel()

		m := newProducerMocks(t, 0)
		p := m.newProducer(t, newEncoder(t, validEncoded("topic")))

		require.NoError(t, p.Close())
		require.ErrorIs(t, p.Close(), ErrClosed)
	})

	t.Run("concurrent calls", func(t *testing.T) {
		t.Parallel()

		var (
			m = newProducerMocks(t, 0)
			p = m.newProducer(t, newEncoder(t, validEncoded("topic")))

			wg      sync.WaitGroup
			success atomic.Int64
		)

		for range 8 {
			wg.Go(func() {
				if p.Close() == nil {
					success.Add(1)
				}
			})
		}

		wg.Wait()

		assert.EqualValues(t, 1, success.Load())
	})

	t.Run("close errors", func(t *testing.T) {
		t.Parallel()

		var (
			errSync   = errors.New("sync producer close failed")
			errClient = errors.New("client close failed")
			m         = newProducerMocks(t, 0)
			p         = m.newProducer(t, newEncoder(t, validEncoded("topic")))
		)

		m.syncCloseErr = errSync
		m.cliCloseErr = errClient

		err := p.Close()
		require.ErrorIs(t, err, errSync)
		require.ErrorIs(t, err, errClient)
	})

	t.Run("flushed messages reach error handler", func(t *testing.T) {
		t.Parallel()

		const count = 50

		var (
			cause  = errors.New("delivery failed on close")
			failed atomic.Int64
			m      = newProducerMocks(t, count)
			p      = m.newProducer(t, newEncoder(t, validEncoded("topic")),
				WithErrorHandler(func(err error) {
					if errors.Is(err, cause) {
						failed.Add(1)
					}
				}))
		)

		m.flushErr = cause

		for range count {
			require.NoError(t, p.Send(context.Background(), "msg"))
		}

		require.NoError(t, p.Close())
		assert.EqualValues(t, count, failed.Load())
	})

	t.Run("concurrent with sends", func(t *testing.T) {
		t.Parallel()

		var (
			m = newProducerMocks(t, 0)
			p = m.newProducer(t, newEncoder(t, validEncoded("topic")))

			wg      sync.WaitGroup
			started = make(chan struct{})
		)

		m.sync.EXPECT().SendMessage(gomock.Any()).Return(int32(0), int64(0), nil).AnyTimes()

		for i := range 32 {
			wg.Go(func() {
				<-started

				for range 200 {
					var err error
					if i%2 == 0 {
						err = p.Send(context.Background(), "msg")
					} else {
						err = p.SendSync(context.Background(), "msg")
					}

					if errors.Is(err, ErrClosed) {
						return
					}
				}
			})
		}

		close(started)
		time.Sleep(5 * time.Millisecond)

		require.NoError(t, p.Close())
		wg.Wait()
	})
}

func TestUnit_Producer_producerMessage(t *testing.T) {
	t.Parallel()

	newTestProducer := func(t *testing.T, e encoded, idempotent bool, traceID string) *Producer[string] {
		t.Helper()

		return &Producer[string]{
			encoder:      newEncoder(t, e),
			isIdempotent: idempotent,
			traceID:      func(context.Context) string { return traceID },
		}
	}

	t.Run("message", func(t *testing.T) {
		t.Parallel()

		p := newTestProducer(t, validEncoded("topic"), true, "trace")

		msg, err := p.producerMessage(context.Background(), "source")
		require.NoError(t, err)

		require.Len(t, msg.Headers, 3)

		assert.Equal(t, codec.HeaderMessageIDByte, msg.Headers[0].Key)
		id, err := uuid.Parse(string(msg.Headers[0].Value))
		require.NoError(t, err)
		assert.EqualValues(t, 7, id.Version())

		assert.Equal(t, sarama.RecordHeader{Key: codec.HeaderTraceIDByte, Value: []byte("trace")}, msg.Headers[1])
		assert.Equal(t, sarama.RecordHeader{Key: []byte("type"), Value: []byte("type")}, msg.Headers[2])

		assert.Equal(t, "topic", msg.Topic)
		assert.Equal(t, sarama.ByteEncoder("key"), msg.Key)
		assert.Equal(t, sarama.ByteEncoder("payload"), msg.Value)
		assert.Equal(t, time.Unix(id.Time().UnixTime()), msg.Timestamp)

		meta, ok := msg.Metadata.(msgMeta)
		require.True(t, ok)
		assert.Equal(t, msgMeta{
			metric: metric{topic: "topic", eventType: "type", timestamp: msg.Timestamp},
			id:     id.String(),
			key:    []byte("key"),
			msg:    "source",
		}, meta)
	})

	t.Run("without trace id", func(t *testing.T) {
		t.Parallel()

		p := newTestProducer(t, validEncoded("topic"), true, "")

		msg, err := p.producerMessage(context.Background(), "source")
		require.NoError(t, err)

		for _, h := range msg.Headers {
			assert.NotEqual(t, codec.HeaderTraceIDByte, h.Key)
		}
	})

	t.Run("marshal error goes before validation", func(t *testing.T) {
		t.Parallel()

		errInvalid := errors.New("invalid message")

		p := newTestProducer(t, encoded{err: errInvalid}, true, "")

		_, err := p.producerMessage(context.Background(), "source")
		require.ErrorIs(t, err, errInvalid)
	})

	t.Run("empty topic", func(t *testing.T) {
		t.Parallel()

		p := newTestProducer(t, validEncoded(""), true, "")

		_, err := p.producerMessage(context.Background(), "source")
		require.ErrorIs(t, err, ErrEmptyTopic)
	})

	t.Run("empty key", func(t *testing.T) {
		t.Parallel()

		e := validEncoded("topic")
		e.key = nil

		_, err := newTestProducer(t, e, true, "").producerMessage(context.Background(), "source")
		require.ErrorIs(t, err, ErrEmptyKey)

		_, err = newTestProducer(t, e, false, "").producerMessage(context.Background(), "source")
		require.NoError(t, err, "the key is optional without the idempotence")
	})
}

func TestUnit_Producer_handleError(t *testing.T) {
	t.Parallel()

	var (
		cause = errors.New("delivery failed")
		ts    = time.Now()
	)

	tests := []struct {
		name       string
		perr       *sarama.ProducerError
		want       *SendError
		wantFailed uint64
	}{
		{
			name: "message of producer",
			perr: &sarama.ProducerError{Err: cause, Msg: &sarama.ProducerMessage{
				Topic: "handle_error_topic",
				Metadata: msgMeta{
					metric: metric{topic: "handle_error_topic", eventType: "type", timestamp: ts},
					id:     "id",
					key:    []byte("key"),
					msg:    "source",
				},
			}},
			want: &SendError{
				Topic: "handle_error_topic",
				Key:   []byte("key"),
				ID:    "id",
				Type:  "type",
				Msg:   "source",
				Err:   cause,
			},
			wantFailed: 1,
		},
		{
			name: "message without metadata",
			perr: &sarama.ProducerError{Err: cause, Msg: &sarama.ProducerMessage{Topic: "handle_error_raw_topic"}},
			want: &SendError{Topic: "handle_error_raw_topic", Err: cause},
		},
		{
			name: "without message",
			perr: &sarama.ProducerError{Err: cause},
			want: &SendError{Err: cause},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var (
				got    error
				before = sendSamples(t, tt.want.Topic, "true")
				p      = &Producer[string]{
					logger:       newOptions().Logger,
					errorHandler: func(err error) { got = err },
				}
			)

			p.handleError(tt.perr)

			var sendErr *SendError

			require.ErrorAs(t, got, &sendErr)
			assert.Equal(t, tt.want, sendErr)
			assert.Equal(t, before+tt.wantFailed, sendSamples(t, tt.want.Topic, "true"))
		})
	}

	t.Run("without error handler", func(t *testing.T) {
		t.Parallel()

		p := &Producer[string]{logger: newOptions().Logger}

		assert.NotPanics(t, func() { p.handleError(&sarama.ProducerError{Err: cause}) })
	})
}

func TestUnit_Producer_readEvents(t *testing.T) {
	t.Parallel()

	const topic = "read_events_topic"

	var (
		m      = newProducerMocks(t, 2)
		cause  = errors.New("delivery failed")
		errCh  = make(chan error, 1)
		okFrom = sendSamples(t, topic, "false")
		p      = m.newProducer(t, newEncoder(t, validEncoded(topic)),
			WithErrorHandler(func(err error) { errCh <- err }))
	)

	require.NoError(t, p.Send(context.Background(), "msg"))
	require.NoError(t, p.Send(context.Background(), "msg"))

	m.reply(nil)
	m.reply(cause)

	select {
	case err := <-errCh:
		require.ErrorIs(t, err, cause)
	case <-time.After(2 * time.Second):
		t.Fatal("the error handler was not called")
	}

	require.NoError(t, p.Close())

	select {
	case <-p.done:
	default:
		t.Fatal("readEvents must finish after the channels are closed")
	}

	assert.Equal(t, okFrom+1, sendSamples(t, topic, "false"))
}
