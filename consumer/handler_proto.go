package consumer

import (
	"context"
	"maps"
	"slices"
	"sync"
	"time"

	"google.golang.org/protobuf/proto"
)

type (
	HandlerProtoFunc[T proto.Message] func(ctx context.Context, msg T, raw Message) error

	// HandlerProto handler messages one at a time
	HandlerProto[T proto.Message] struct {
		handler[T]

		f HandlerProtoFunc[T]
	}

	HandlerProtoBatchFunc[T proto.Message] func(ctx context.Context, msg []T, raw []Message) error

	// HandlerProtoBatch handler batch of messages
	HandlerProtoBatch[T proto.Message] struct {
		handler[T]

		batchSize         int
		batchFlushTimeout time.Duration

		buf       []T
		bufRaw    []Message
		bufOffset Offset
		bufMu     sync.Mutex

		ticker *time.Ticker

		f HandlerProtoBatchFunc[T]
	}
)

// NewHandlerProto constructs HandlerProto
func NewHandlerProto[T proto.Message](handlerFunc HandlerProtoFunc[T], opts ...HandlerOption) *HandlerProto[T] {
	o := newHandlerOptions(opts...)

	return &HandlerProto[T]{
		handler: newHandler[T](o),
		f:       handlerFunc,
	}
}

// Handle received Message handler
func (p *HandlerProto[T]) Handle(ctx context.Context, raw Message) {
	msg, ok := p.resolveMsg(raw.Proto, raw.Value)
	if !ok {
		return
	}

	err := p.handle(ctx, func(fCtx context.Context) error {
		return p.f(fCtx, msg, raw)
	})

	if err != nil {
		if p.errHandler != nil {
			p.errHandler(raw, err)
		}
	}

	p.updateOffset(Offset{raw.Partition: raw.Offset})
}

// NewHandlerProtoBatch constructs HandlerProtoBatch
func NewHandlerProtoBatch[T proto.Message](handlerFunc HandlerProtoBatchFunc[T], opts ...HandlerOption) *HandlerProtoBatch[T] {
	o := newHandlerOptions(opts...)

	var t *time.Ticker
	if o.BatchFlushTimeout > 0 {
		t = time.NewTicker(o.BatchFlushTimeout)
	}

	h := &HandlerProtoBatch[T]{
		handler: newHandler[T](o),

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
func (p *HandlerProtoBatch[T]) Handle(ctx context.Context, raw Message) {
	msg, ok := p.resolveMsg(raw.Proto, raw.Value)
	if !ok {
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

func (p *HandlerProtoBatch[T]) flush(ctx context.Context) {
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
		if p.errHandler != nil {
			p.errHandler(Message{}, err)
		}
	}

	p.updateOffset(maps.Clone(p.bufOffset))
}

func (p *HandlerProtoBatch[T]) flushMu() {
	p.bufMu.Lock()
	defer p.bufMu.Unlock()

	p.flush(context.Background())
}

func (p *HandlerProtoBatch[T]) startTimeoutTicker() {
	if p.ticker == nil {
		return
	}

	go func() {
		for range p.ticker.C {
			p.flushMu()
		}
	}()
}
