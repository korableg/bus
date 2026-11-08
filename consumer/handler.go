package consumer

import (
	"context"
	"errors"
	"slices"
	"sync"
	"time"

	"github.com/korableg/bus/codec"
)

type (
	// HandlerProtoFunc handles one message: msg is the decoded value, raw is the Kafka message.
	HandlerProtoFunc[T any] func(ctx context.Context, msg T, raw Message) error

	// HandlerProto handles messages one at a time.
	//
	// The handler function may be called concurrently for the messages of different partitions
	// (the consumer group handles every claimed partition in its own goroutine),
	// calls for one partition are sequential.
	HandlerProto[T any] struct {
		handler[T]

		f HandlerProtoFunc[T]
	}

	// HandlerBatchFunc handles a batch: msg[i] is the decoded value of raw[i].
	HandlerBatchFunc[T any] func(ctx context.Context, msg []T, raw []Message) error

	// HandlerBatch accumulates messages and handles them in batches.
	//
	// The handler has one buffer for all the partitions of the topic, so a batch may contain
	// the messages of several partitions. The calls of the handler function are sequential.
	// A buffer is flushed when it reaches the batch size (WithHandlerBatchSize) or when no message
	// has arrived for the flush timeout (WithHandlerBatchFlushTimeout): while the messages keep coming
	// the buffer waits for the batch size, and the timeout keeps them from getting stale when the flow slows down.
	// A flush by the size is done on the goroutine of the partition which has filled the buffer,
	// a flush by the timeout is done on a separate goroutine with context.Background().
	//
	// The offsets are committed only after the batch has been handled.
	// A message the handler doesn't handle (of another type or broken) advances the offset of its partition
	// at once, or after the flush if the buffer holds messages of that partition.
	// The buffer is not flushed on a rebalance.
	HandlerBatch[T any] struct {
		handler[T]

		batchSize         int
		batchFlushTimeout time.Duration

		buf    []T
		bufRaw []Message
		bufMu  sync.Mutex

		offsets *batchOffsets

		timer   *time.Timer
		flushAt time.Time // a timer call before it is stale

		f HandlerBatchFunc[T]
	}
)

// NewHandler returns the handler calling handlerFunc for every message decoder accepts.
// The topic of the handler is the topic of decoder.
func NewHandler[T any](decoder codec.Decoder[T], handlerFunc HandlerProtoFunc[T], opts ...HandlerOption) *HandlerProto[T] {
	o := newHandlerOptions(opts...)

	return &HandlerProto[T]{
		handler: newHandler[T](decoder, o),
		f:       handlerFunc,
	}
}

// Handle decodes raw and calls the handler function, retrying it according to the handler options.
// It is called by Group.
func (p *HandlerProto[T]) Handle(ctx context.Context, raw Message) {
	defer p.updateOffset(Offset{raw.Partition: raw.Offset})

	msg, err := p.decoder.Unmarshal(raw.Value, raw.Headers)
	if err != nil {
		if errors.Is(err, codec.ErrWrongType) {
			return
		}
		p.logger.ErrorContext(ctx, "unmarshal message", "err", err)
	}

	err = p.handle(ctx, func(fCtx context.Context) error {
		return p.f(fCtx, msg, raw)
	})

	if err != nil {
		p.logger.ErrorContext(ctx, "handle message",
			"err", err, "topic", raw.Topic, "partition", raw.Partition, "offset", raw.Offset)
	}
}

// NewHandlerBatch returns the handler calling handlerFunc for the batches of the messages decoder accepts.
// The topic of the handler is the topic of decoder.
func NewHandlerBatch[T any](decoder codec.Decoder[T], handlerFunc HandlerBatchFunc[T], opts ...HandlerOption) *HandlerBatch[T] {
	o := newHandlerOptions(opts...)

	h := &HandlerBatch[T]{
		handler: newHandler[T](decoder, o),

		batchSize:         o.BatchSize,
		batchFlushTimeout: o.BatchFlushTimeout,

		buf:    make([]T, 0, o.BatchSize),
		bufRaw: make([]Message, 0, o.BatchSize),

		f: handlerFunc,
	}

	h.offsets = newBatchOffsets(h.updateOffset)

	return h
}

// Handle decodes raw and adds it to the buffer, flushing the full buffer. It is called by Group.
func (p *HandlerBatch[T]) Handle(ctx context.Context, raw Message) {
	msg, err := p.decoder.Unmarshal(raw.Value, raw.Headers)
	if err != nil {
		if !errors.Is(err, codec.ErrWrongType) {
			p.logger.ErrorContext(ctx, "unmarshal message",
				"err", err, "topic", raw.Topic, "partition", raw.Partition, "offset", raw.Offset)
		}

		p.offsets.skip(raw.Partition, raw.Offset)

		return
	}

	p.bufMu.Lock()
	defer p.bufMu.Unlock()

	p.buf = append(p.buf, msg)
	p.bufRaw = append(p.bufRaw, raw)
	p.offsets.add(raw.Partition, raw.Offset)

	if len(p.buf) < p.batchSize {
		p.resetTimer()
		return
	}

	p.flush(ctx)
}

// resetTimer must be called with p.bufMu held.
func (p *HandlerBatch[T]) resetTimer() {
	if p.batchFlushTimeout <= 0 {
		return
	}

	// taken before the reset, so the timer never fires before flushAt
	p.flushAt = time.Now().Add(p.batchFlushTimeout)

	if p.timer == nil {
		p.timer = time.AfterFunc(p.batchFlushTimeout, p.flushByTimeout)
		return
	}

	p.timer.Reset(p.batchFlushTimeout)
}

// flush must be called with p.bufMu held.
func (p *HandlerBatch[T]) flush(ctx context.Context) {
	if len(p.buf) == 0 {
		return
	}

	if p.timer != nil {
		p.timer.Stop()
	}

	var (
		hBuf    = slices.Clone(p.buf)
		hbufRaw = slices.Clone(p.bufRaw)
	)

	defer func() {
		clear(p.buf)
		p.buf = p.buf[:0]
		clear(p.bufRaw)
		p.bufRaw = p.bufRaw[:0]
	}()

	err := p.handle(ctx, func(fCtx context.Context) error {
		return p.f(fCtx, hBuf, hbufRaw)
	})
	if err != nil {
		p.logger.ErrorContext(ctx, "handle message", "err", err)
	}

	p.offsets.flush()
}

// flushByTimeout is called by the timer. The timer may fire while a message holds p.bufMu: then its call
// is stale, as the message has reset the timer or flushed the buffer by the size.
func (p *HandlerBatch[T]) flushByTimeout() {
	p.bufMu.Lock()
	defer p.bufMu.Unlock()

	if time.Now().Before(p.flushAt) {
		return
	}

	p.flush(context.Background())
}
