package consumer

import (
	"context"
	"errors"
	"maps"
	"slices"
	"sync"
	"time"

	"github.com/korableg/bus/codec"
)

type (
	HandlerProtoFunc[T any] func(ctx context.Context, msg T, raw Message) error

	// HandlerProto handler messages one at a time
	HandlerProto[T any] struct {
		handler[T]

		f HandlerProtoFunc[T]
	}

	HandlerBatchFunc[T any] func(ctx context.Context, msg []T, raw []Message) error

	// HandlerBatch handler batch of messages
	HandlerBatch[T any] struct {
		handler[T]

		batchSize         int
		batchFlushTimeout time.Duration

		buf       []T
		bufRaw    []Message
		bufOffset Offset
		bufMu     sync.Mutex

		ticker *time.Ticker

		f HandlerBatchFunc[T]
	}
)

// NewHandler constructs HandlerProto
func NewHandler[T any](decoder codec.Decoder[T], handlerFunc HandlerProtoFunc[T], opts ...HandlerOption) *HandlerProto[T] {
	o := newHandlerOptions(opts...)

	return &HandlerProto[T]{
		handler: newHandler[T](decoder, o),
		f:       handlerFunc,
	}
}

// Handle received Message handler
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

// NewHandlerBatch constructs HandlerBatch
func NewHandlerBatch[T any](decoder codec.Decoder[T], handlerFunc HandlerBatchFunc[T], opts ...HandlerOption) *HandlerBatch[T] {
	o := newHandlerOptions(opts...)

	var t *time.Ticker
	if o.BatchFlushTimeout > 0 {
		t = time.NewTicker(o.BatchFlushTimeout)
	}

	h := &HandlerBatch[T]{
		handler: newHandler[T](decoder, o),

		batchSize:         o.BatchSize,
		batchFlushTimeout: o.BatchFlushTimeout,

		buf:       make([]T, 0, o.BatchSize),
		bufRaw:    make([]Message, 0, o.BatchSize),
		bufOffset: make(Offset),
		ticker:    t,
		f:         handlerFunc,
	}

	h.startTimeoutTicker()

	return h
}

// Handle received Message handler
func (p *HandlerBatch[T]) Handle(ctx context.Context, raw Message) {
	msg, err := p.decoder.Unmarshal(raw.Value, raw.Headers)
	if err != nil {
		return
	}

	p.bufMu.Lock()
	defer p.bufMu.Unlock()

	p.buf = append(p.buf, msg)
	p.bufRaw = append(p.bufRaw, raw)
	p.bufOffset[raw.Partition] = raw.Offset

	if p.ticker != nil {
		p.ticker.Reset(p.batchFlushTimeout)
	}

	if len(p.buf) < p.batchSize {
		return
	}

	p.flush(ctx)
}

func (p *HandlerBatch[T]) flush(ctx context.Context) {
	if len(p.buf) == 0 {
		return
	}

	var (
		hBuf    = slices.Clone(p.buf)
		hbufRaw = slices.Clone(p.bufRaw)
	)

	defer func() {
		p.buf = p.buf[:0]
		p.bufRaw = p.bufRaw[:0]
		clear(p.bufOffset)
	}()

	err := p.handle(ctx, func(fCtx context.Context) error {
		return p.f(ctx, hBuf, hbufRaw)
	})
	if err != nil {
		p.logger.ErrorContext(ctx, "handle message", "err", err)
	}

	p.updateOffset(maps.Clone(p.bufOffset))
}

func (p *HandlerBatch[T]) flushMu() {
	p.bufMu.Lock()
	defer p.bufMu.Unlock()

	p.flush(context.Background())
}

func (p *HandlerBatch[T]) startTimeoutTicker() {
	if p.ticker == nil {
		return
	}

	go func() {
		for range p.ticker.C {
			p.flushMu()
		}
	}()
}
