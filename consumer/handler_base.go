package consumer

import (
	"context"
	"fmt"
	"iter"
	"log/slog"
	"maps"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v7"

	"github.com/korableg/bus/codec"
)

type (
	// Message is a received Kafka message.
	Message struct {
		Key       []byte
		Value     []byte
		ID        string // Value of the "id" header
		Group     string // Consumer group
		Topic     string
		TraceID   string // Value of the "trace-id" header
		Partition int32
		Offset    int64
		Timestamp time.Time
		Headers   []codec.Header // All the headers, "id" and "trace-id" included
	}

	// Offset maps a partition to an offset.
	Offset map[int32]int64

	// OffsetSeq is a sequence of partitions and offsets.
	OffsetSeq = iter.Seq2[int32, int64]

	// Handler handles the messages of one topic. HandlerProto and HandlerBatch implement it,
	// implement it yourself for a custom handling.
	Handler interface {
		codec.Meta
		// Handle handles the message. The calls for one partition are sequential, the calls for
		// different partitions are concurrent.
		Handle(context.Context, Message)
		// Offset returns the offset of the last handled message of the partition, 0 if there is none.
		// The group commits a partition up to the minimum offset among the handlers of the topic.
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

// Type returns the event type of the decoder.
func (p *handler[T]) Type() string {
	return p.decoder.Type()
}

// Topic returns the topic of the decoder.
func (p *handler[T]) Topic() string {
	return p.decoder.Topic()
}

// Offset returns the offset of the last handled message of the partition, 0 if there is none.
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

	bo := backoff.NewExponentialBackOff()
	bo.InitialInterval = h.retryInitialInterval
	bo.MaxInterval = h.retryMaxInterval

	// The retries don't stop on the cancellation of ctx, otherwise the message would be dropped
	// on a rebalance. The handler function still gets ctx.
	_, err = backoff.Retry(context.WithoutCancel(ctx), func() (struct{}, error) {
		return struct{}{}, h.handleCtx(ctx, f)
	},
		backoff.WithBackOff(bo),
		backoff.WithMaxElapsedTime(h.retryMaxTime), // zero disables the limit, the backoff default is 15 minutes
		backoff.WithNotify(func(err error, d time.Duration) {
			h.logger.ErrorContext(ctx, "scheduled retry on the consumer handler",
				"interval", d,
				"err", err,
				"topic", h.decoder.Topic())
		}),
	)

	return err
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
