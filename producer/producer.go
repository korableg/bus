package producer

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/IBM/sarama"
	"github.com/google/uuid"

	"github.com/korableg/bus/codec"
)

// Producer sends messages of type T to Kafka. It is safe for concurrent use.
type Producer[T any] struct {
	cli           sarama.Client
	asyncProducer sarama.AsyncProducer
	syncProducer  sarama.SyncProducer

	logger *slog.Logger

	mu       sync.RWMutex
	closed   bool
	closing  chan struct{}
	inflight sync.WaitGroup

	done chan struct{}

	errorHandler func(err error)

	isIdempotent bool

	encoder func(T) codec.Encoder[T]
	traceID func(context.Context) string
}

// New connects to brokers and returns a Producer that encodes the messages with encoder,
// e.g. wire.Encoder(). Close it to flush the queued messages and release the connections.
func New[T any](encoder func(T) codec.Encoder[T], brokers []string, opts ...Option) (*Producer[T], error) {
	if encoder == nil {
		return nil, errors.New("encoder is not defined")
	}

	if len(brokers) == 0 {
		return nil, errors.New("brokers are empty")
	}

	o := newOptions(opts...)

	cli, err := sarama.NewClient(brokers, newSaramaConfig(o))
	if err != nil {
		return nil, fmt.Errorf("init sarama client: %w", err)
	}

	asyncProducer, err := sarama.NewAsyncProducerFromClient(cli)
	if err != nil {
		return nil, errors.Join(fmt.Errorf("init async producer: %w", err), cli.Close())
	}

	syncProducer, err := sarama.NewSyncProducerFromClient(cli)
	if err != nil {
		return nil, errors.Join(fmt.Errorf("init sync producer: %w", err), asyncProducer.Close(), cli.Close())
	}

	return newProducer(cli, asyncProducer, syncProducer, encoder, o), nil
}

func newProducer[T any](
	cli sarama.Client,
	asyncProducer sarama.AsyncProducer,
	syncProducer sarama.SyncProducer,
	encoder func(T) codec.Encoder[T],
	o Options,
) *Producer[T] {
	p := &Producer[T]{
		cli:           cli,
		asyncProducer: asyncProducer,
		syncProducer:  syncProducer,

		encoder:      encoder,
		isIdempotent: o.Idempotence,

		logger:  o.Logger,
		traceID: o.TraceID,

		errorHandler: o.ErrorHandler,

		closing: make(chan struct{}),
		done:    make(chan struct{}),
	}

	go p.readEvents()

	return p
}

// Send queues the message for sending and returns without waiting for the broker.
// A delivery error is logged and passed to the handler set by WithErrorHandler.
// Send blocks while the queue is full; it returns ctx.Err() if ctx is done meanwhile
// and ErrClosed if the producer is closed.
func (p *Producer[T]) Send(ctx context.Context, msg T) error {
	if err := p.acquire(); err != nil {
		return err
	}
	defer p.inflight.Done()

	sMsg, err := p.producerMessage(ctx, msg)
	if err != nil {
		return err
	}

	if err = ctx.Err(); err != nil {
		return err
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-p.closing:
		return ErrClosed
	case p.asyncProducer.Input() <- sMsg:
		return nil
	}
}

// SendSync sends the message and waits for the acknowledgment of all in-sync replicas.
func (p *Producer[T]) SendSync(ctx context.Context, msg T) error {
	if err := p.acquire(); err != nil {
		return err
	}
	defer p.inflight.Done()

	sMsg, err := p.producerMessage(ctx, msg)
	if err != nil {
		return err
	}

	_, _, err = p.syncProducer.SendMessage(sMsg)
	if err != nil {
		return fmt.Errorf("send message: %w", err)
	}

	return nil
}

// Close waits for the Send and SendSync calls in progress, flushes the queued messages
// and closes the connections. The delivery errors of the flushed messages reach the error handler
// before Close returns. The next calls return ErrClosed.
func (p *Producer[T]) Close() error {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return ErrClosed
	}

	p.closed = true
	close(p.closing)
	p.mu.Unlock()

	p.inflight.Wait()

	p.asyncProducer.AsyncClose()
	<-p.done

	return errors.Join(
		p.syncProducer.Close(),
		p.cli.Close(),
	)
}

func (p *Producer[T]) acquire() error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.closed {
		return ErrClosed
	}

	p.inflight.Add(1)

	return nil
}

func (p *Producer[T]) producerMessage(ctx context.Context, msg T) (*sarama.ProducerMessage, error) {
	enc := p.encoder(msg)

	payload, err := enc.Marshal()
	if err != nil {
		return nil, fmt.Errorf("marshal message: %w", err)
	}

	var (
		topic   = enc.Topic()
		key     = enc.Key()
		evtType = enc.Type()
		headers = enc.Headers()
	)

	if topic == "" {
		return nil, ErrEmptyTopic
	}

	if p.isIdempotent && len(key) == 0 {
		return nil, ErrEmptyKey
	}

	id, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("generate uuid: %w", err)
	}

	var (
		msgHdrs = make([]sarama.RecordHeader, 0, len(headers)+2)
		ts      = time.Unix(id.Time().UnixTime())
	)

	msgHdrs = append(msgHdrs, sarama.RecordHeader{
		Key:   codec.HeaderMessageIDByte,
		Value: []byte(id.String()),
	})

	if traceID := p.traceID(ctx); traceID != "" {
		msgHdrs = append(msgHdrs, sarama.RecordHeader{
			Key:   codec.HeaderTraceIDByte,
			Value: []byte(traceID),
		})
	}

	for _, h := range headers {
		msgHdrs = append(msgHdrs, sarama.RecordHeader{
			Key:   h.Key,
			Value: h.Value,
		})
	}

	return &sarama.ProducerMessage{
		Topic:   topic,
		Key:     sarama.ByteEncoder(key),
		Value:   sarama.ByteEncoder(payload),
		Headers: msgHdrs,
		Metadata: msgMeta{
			metric: metric{
				topic:     topic,
				eventType: evtType,
				timestamp: ts,
			},
			id:  id.String(),
			key: key,
			msg: msg,
		},
		Timestamp: ts,
	}, nil
}

type msgMeta struct {
	metric

	id  string
	key []byte
	msg any
}

func (p *Producer[T]) handleError(perr *sarama.ProducerError) {
	p.logger.Error("send kafka message error", "err", perr.Err)

	sendErr := &SendError{Err: perr.Err}

	if perr.Msg != nil {
		sendErr.Topic = perr.Msg.Topic

		if m, ok := perr.Msg.Metadata.(msgMeta); ok {
			observeSend(m.metric, perr.Err)

			sendErr.Topic = m.topic
			sendErr.Key = m.key
			sendErr.ID = m.id
			sendErr.Type = m.eventType
			sendErr.Msg = m.msg
		}
	}

	if p.errorHandler != nil {
		p.errorHandler(sendErr)
	}
}

func (p *Producer[T]) readEvents() {
	defer close(p.done)

	var (
		successes = p.asyncProducer.Successes()
		errs      = p.asyncProducer.Errors()
	)

	for successes != nil || errs != nil {
		select {
		case success, ok := <-successes:
			if !ok {
				successes = nil
				continue
			}

			if m, isMeta := success.Metadata.(msgMeta); isMeta {
				observeSend(m.metric, nil)
			}
		case err, ok := <-errs:
			if !ok {
				errs = nil
				continue
			}

			p.handleError(err)
		}
	}
}
