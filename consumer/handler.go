package consumer

import (
	"context"
	"fmt"
	"log/slog"
	"maps"
	"slices"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	"gitlab.services.mts.ru/scs-data-platform/backend/modules/protobus"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
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
		FullName  protoreflect.FullName
		Proto     proto.Message
	}

	// Offset kafka offset by partition
	Offset map[int32]int64

	// ErrorHandler handler when an error occurs (on unmarshal or during handling)
	ErrorHandler func(Message, error)

	// Handler message handler interface
	Handler interface {
		Type() protoreflect.MessageType
		Topic() string
		Handle(context.Context, Message)
		Offset(diff bool) Offset
		SetErrorHandler(ErrorHandler)
	}

	HandlerRegistrar interface {
		AddHandler(Handler) error
	}

	handler[T proto.Message] struct {
		logger *slog.Logger

		topic       string
		msgType     protoreflect.MessageType
		msgFullName protoreflect.FullName

		unmarshal UnmarshalFunc

		offsetUpdated bool
		offset        Offset
		offsetMu      sync.Mutex

		errHandler     ErrorHandler
		timeoutHandler time.Duration

		retry                bool
		retryInitialInterval time.Duration
		retryMaxInterval     time.Duration
		retryMaxTime         time.Duration
	}
)

func newHandler[T proto.Message](o HandlerOptions) handler[T] {
	var zero T
	return handler[T]{
		logger: o.Logger,

		topic:       protobus.GetTopic(zero),
		msgType:     zero.ProtoReflect().Type(),
		msgFullName: protobus.GetFullName(zero),

		offset: make(Offset),

		timeoutHandler: o.Timeout,

		retry:                o.Retry,
		retryInitialInterval: o.RetryInitialInterval,
		retryMaxInterval:     o.RetryMaxInterval,
		retryMaxTime:         o.RetryMaxTime,

		unmarshal: o.Unmarshal,
	}
}

// ProtoReflect protoreflect.Message getter
func (p *handler[T]) Type() protoreflect.MessageType {
	return p.msgType
}

// Topic getter
func (p *handler[T]) Topic() string {
	return p.topic
}

// Offset gets a handler's offset
func (h *handler[T]) Offset(diff bool) Offset {
	h.offsetMu.Lock()
	defer h.offsetMu.Unlock()

	if diff && !h.offsetUpdated {
		return nil
	}

	h.offsetUpdated = false

	return maps.Clone(h.offset)
}

func (h *handler[T]) SetErrorHandler(errHandler ErrorHandler) {
	h.errHandler = errHandler
}

func (h *handler[T]) updateOffset(offset Offset) {
	h.offsetMu.Lock()
	defer h.offsetMu.Unlock()

	for part, off := range offset {
		var curOffset = h.offset[part]
		if curOffset < off {
			h.offset[part] = off
			h.offsetUpdated = true
		}
	}
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
			"topic", h.topic)
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

	return observeHandler(h.topic, string(h.msgFullName))(f(ctx))
}

func (h *handler[T]) resolveMsg(src proto.Message, raw []byte) (T, bool) {
	if h.unmarshal == nil {
		msg, ok := src.(T)
		return msg, ok
	}

	var msg = h.msgType.New().Interface()
	return msg.(T), h.unmarshal(raw, msg) == nil
}

type handlerCollection struct {
	h  map[string][]Handler
	mu sync.RWMutex
}

func newHandlerCollection() *handlerCollection {
	return &handlerCollection{
		h: make(map[string][]Handler),
	}
}

func (hc *handlerCollection) Add(h Handler) {
	hc.mu.Lock()
	hc.h[h.Topic()] = append(hc.h[h.Topic()], h)
	hc.mu.Unlock()
}

func (hc *handlerCollection) Topics() []string {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	topics := make([]string, 0, len(hc.h))
	for t := range hc.h {
		topics = append(topics, t)
	}

	return topics
}

func (hc *handlerCollection) Handlers(topic string) []Handler {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	src := hc.h[topic]
	if len(src) == 0 {
		return nil
	}

	return slices.Clone(src)
}

func (hc *handlerCollection) Offsets(topic string) []Offset {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	offsets := make([]Offset, 0, len(hc.h[topic]))
	for _, h := range hc.h[topic] {
		off := h.Offset(true)
		if off != nil {
			offsets = append(offsets, off)
		}
	}

	return offsets
}
