package producer

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/IBM/sarama"
	"github.com/google/uuid"

	"github.com/korableg/bus/codec"
	"github.com/korableg/bus/sasl"
)

// Producer the kafka producer
type Producer[T any] struct {
	asyncProducer sarama.AsyncProducer
	syncProducer  sarama.SyncProducer

	logger *slog.Logger

	isClosed     atomic.Bool
	isIdempotent bool

	encoder func(T) codec.Encoder[T]
	traceID func(context.Context) string
}

// New constructs Producer
func New[T any](encoder func(T) codec.Encoder[T], brokers []string, sasl *sasl.SASL, opts ...Option) (*Producer[T], error) {
	o := NewOptions(opts...)

	cli, err := sarama.NewClient(brokers, newSaramaConfig(sasl, o))
	if err != nil {
		return nil, fmt.Errorf("init sarama client: %w", err)
	}

	asyncProducer, err := sarama.NewAsyncProducerFromClient(cli)
	if err != nil {
		return nil, fmt.Errorf("init async producer: %w", err)
	}

	syncProducer, err := sarama.NewSyncProducerFromClient(cli)
	if err != nil {
		return nil, fmt.Errorf("init sync producer: %w", err)
	}

	pbp := &Producer[T]{
		asyncProducer: asyncProducer,
		syncProducer:  syncProducer,
		encoder:       encoder,

		isIdempotent: o.Idempotence,
		logger:       o.Logger,
		traceID:      o.TraceID,
	}

	go pbp.readEvents()

	return pbp, nil
}

// Send sends the message asynchronously
func (p *Producer[T]) Send(ctx context.Context, msg T) error {
	if p.isClosed.Load() {
		return ErrClosed
	}

	sMsg, err := p.producerMessage(ctx, msg)
	if err != nil {
		return err
	}

	p.asyncProducer.Input() <- sMsg

	return nil
}

// SendSync sends the message synchronously
func (p *Producer[T]) SendSync(ctx context.Context, msg T) error {
	if p.isClosed.Load() {
		return ErrClosed
	}

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

func (p *Producer[T]) producerMessage(ctx context.Context, msg T) (*sarama.ProducerMessage, error) {
	var (
		enc = p.encoder(msg)

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

	payload, err := enc.Marshal()
	if err != nil {
		return nil, fmt.Errorf("marshal message: %w", err)
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
		Key:   codec.HeaderMessageID,
		Value: []byte(id.String()),
	})

	if traceID := p.traceID(ctx); traceID != "" {
		msgHdrs = append(msgHdrs, sarama.RecordHeader{
			Key:   codec.HeaderTraceID,
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
		Metadata: metric{
			topic:     topic,
			eventType: evtType,
			timestamp: ts,
		},
		Timestamp: ts,
	}, nil
}

func (p *Producer[T]) readEvents() {
	var m metric

	for {
		select {
		case success, ok := <-p.asyncProducer.Successes():
			if !ok {
				return
			}

			if m, ok = success.Metadata.(metric); ok {
				observeSend(m, nil)
			}
		case err, ok := <-p.asyncProducer.Errors():
			if !ok {
				return
			}

			p.logger.Error("send kafka message error", "err", err.Err)

			if m, ok = err.Msg.Metadata.(metric); ok {
				observeSend(m, err.Err)
			}
		}
	}
}

// Close closes Producer
func (p *Producer[T]) Close() error {
	if p.isClosed.Swap(true) {
		return ErrClosed
	}

	return errors.Join(
		p.asyncProducer.Close(),
		p.syncProducer.Close(),
	)
}
