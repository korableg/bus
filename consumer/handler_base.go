package consumer

import (
	"context"
	"fmt"
	"iter"
	"log/slog"
	"maps"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"

	"github.com/korableg/bus/codec"
)

type (
	// Message raw kafka message
	Message struct {
		Key       []byte
		Value     []byte
		ID        string
		Group     string
		Topic     string
		TraceID   string
		Partition int32
		Offset    int64
		Timestamp time.Time
		Headers   []codec.Header
	}

	// Offset kafka offset by partition
	Offset map[int32]int64

	// Offset sequence
	OffsetSeq = iter.Seq2[int32, int64]

	// Handler message handler interface
	Handler interface {
		codec.Meta
		Handle(context.Context, Message)
		Offset(partition int32) int64
	}

	handler[T any] struct {
		logger *slog.Logger

		offset   Offset
		offsetMu sync.Mutex

		timeoutHandler time.Duration

		retry                bool
		retryInitialInterval time.Duration
		retryMaxInterval     time.Duration
		retryMaxTime         time.Duration

		decoder codec.Decoder[T]
	}
)

func newHandler[T any](decoder codec.Decoder[T], o HandlerOptions) handler[T] {
	return handler[T]{
		logger: o.Logger,

		offset: make(Offset),

		timeoutHandler: o.Timeout,

		retry:                o.Retry,
		retryInitialInterval: o.RetryInitialInterval,
		retryMaxInterval:     o.RetryMaxInterval,
		retryMaxTime:         o.RetryMaxTime,

		decoder: decoder,
	}
}

// ProtoReflect protoreflect.Message getter
func (p *handler[T]) Type() string {
	return p.decoder.Type()
}

// Topic getter
func (p *handler[T]) Topic() string {
	return p.decoder.Topic()
}

// Offset gets a handler's offset sequence
func (h *handler[T]) Offset(part int32) (off int64) {
	h.offsetMu.Lock()
	off = h.offset[part]
	h.offsetMu.Unlock()

	return off
}

func (h *handler[T]) updateOffset(offset Offset) {
	h.offsetMu.Lock()

	maps.Copy(h.offset, offset)

	h.offsetMu.Unlock()
}

func (h *handler[T]) handle(ctx context.Context, f func(ctx context.Context) error) (err error) {
	if !h.retry {
		return h.handleCtx(ctx, f)
	}

	return backoff.RetryNotify(func() error {
		return h.handleCtx(ctx, f)
	}, backoff.NewExponentialBackOff(
		backoff.WithInitialInterval(h.retryInitialInterval),
		backoff.WithMaxInterval(h.retryMaxInterval),
		backoff.WithMaxElapsedTime(h.retryMaxTime),
	), func(err error, d time.Duration) {
		h.logger.ErrorContext(ctx, "scheduled retry on the consumer handler",
			"interval", d,
			"err", err,
			"topic", h.decoder.Topic())
	})
}

func (h *handler[T]) handleCtx(ctx context.Context, f func(ctx context.Context) error) (err error) {
	defer func() {
		if p := recover(); p != nil {
			err = fmt.Errorf("panic: %v", p)
		}
	}()

	var cancel context.CancelFunc
	if h.timeoutHandler > 0 {
		ctx, cancel = context.WithTimeout(ctx, h.timeoutHandler)
		defer cancel()
	}

	return observeHandler(h.decoder.Topic(), h.decoder.Type())(f(ctx))
}
