package consumer

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/IBM/sarama"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"

	"gitlab.services.mts.ru/scs-data-platform/backend/modules/logger"
	"gitlab.services.mts.ru/scs-data-platform/backend/modules/protobus"
)

// Group is the kafka consumer group
type (
	UnmarshalFunc func([]byte, proto.Message) error
	ResolverFunc  func(protoreflect.FullName) (protoreflect.MessageType, error)

	Group struct {
		cli       sarama.ConsumerGroup
		hdrs      *handlerCollection
		offsetCtl *offsetController

		unmarshal UnmarshalFunc
		resolve   ResolverFunc

		logger     *slog.Logger
		errHandler ErrorHandler
		group      string

		closed chan struct{}
	}
)

// NewGroup constructs Group
func NewGroup(cfg GroupConfig, opts ...Option) (*Group, error) {
	o := NewOptions(opts...)

	cli, err := sarama.NewConsumerGroup(
		strings.Split(cfg.Brokers, ","),
		cfg.Group,
		newSaramaConfig(cfg.Sasl, o))
	if err != nil {
		return nil, fmt.Errorf("init consumer group: %w", err)
	}

	hdrs := newHandlerCollection()

	return &Group{
		cli:       cli,
		hdrs:      hdrs,
		offsetCtl: newOffsetController(hdrs, o),

		unmarshal: o.Unmarshal,
		resolve:   protoregistry.GlobalTypes.FindMessageByName,

		logger:     o.GetLogger(),
		errHandler: o.ErrHandler,
		group:      cfg.Group,

		closed: make(chan struct{}),
	}, nil
}

// AddHandler adds message handler
func (c *Group) AddHandler(h Handler) error {
	if c.isClosed() {
		return ErrClosed
	}

	if h.Topic() == "" {
		return ErrEmptyTopic
	}

	h.SetErrorHandler(func(msg Message, err error) {
		if c.errHandler != nil {
			c.errHandler(msg, fmt.Errorf("%w: %w", ErrMessageHandle, err))
		}
		c.logger.Error(fmt.Sprintf("message handler error: %s", err),
			"name", protobus.GetFullName(msg.Proto),
			"partition", msg.Partition,
			"offset", msg.Offset)
	})

	c.hdrs.Add(h)

	return nil
}

// Run starts consume the messages and handle them
func (c *Group) Run(ctx context.Context) (err error) {
	if c.isClosed() {
		return ErrClosed
	}

	topics := c.hdrs.Topics()
	if len(topics) == 0 {
		return ErrNoTopics
	}

	for {
		select {
		case <-c.closed:
			return nil
		case <-ctx.Done():
			return nil
		default:
		}

		c.logger.Info(fmt.Sprintf("Starting consume by group %s", c.group))

		err = c.cli.Consume(ctx, topics, c)
		if err != nil {
			if errors.Is(err, sarama.ErrClosedConsumerGroup) {
				return err
			}
		}

		c.logger.Info(fmt.Sprintf("Consume by group %s was stopped", c.group), "err", err)
	}
}

// Close closes the kafka consumer
func (c *Group) Stop(context.Context) error {
	close(c.closed)

	c.logger.Info(fmt.Sprintf("Stopping consume by group %s", c.group))

	return c.cli.Close()
}

func (c *Group) Setup(sess sarama.ConsumerGroupSession) error {
	go c.offsetCtl.StartCommitting(sess)

	for topic, partitions := range sess.Claims() {
		c.logger.Info(fmt.Sprintf("rebalance group %s: assigned topic: %s, partitions: %v",
			c.group, topic, partitions))
	}

	return nil
}

func (c *Group) Cleanup(sess sarama.ConsumerGroupSession) error {
	c.offsetCtl.Mark(sess)

	for topic, partitions := range sess.Claims() {
		c.logger.Info(fmt.Sprintf("rebalance group %s: revoked topic: %s, partitions: %v",
			c.group, topic, partitions))
	}

	return nil
}

func (c *Group) ConsumeClaim(sess sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	var handler = c.handlerFunc(sess.Context(), claim.Topic())

	for {
		select {
		case kafkaMsg, ok := <-claim.Messages():
			if !ok {
				return nil
			}

			handler(kafkaMsg)

			c.offsetCtl.Inc()
		case <-sess.Context().Done():
			return nil
		}
	}
}

func (c *Group) handlerFunc(ctx context.Context, topic string) func(*sarama.ConsumerMessage) {
	var (
		handlers = c.hdrs.Handlers(topic)
		msg      Message
	)

	return func(kafkaMsg *sarama.ConsumerMessage) {
		msg = Message{
			Group:     c.group,
			Key:       kafkaMsg.Key,
			Value:     kafkaMsg.Value,
			Topic:     kafkaMsg.Topic,
			Partition: kafkaMsg.Partition,
			Offset:    kafkaMsg.Offset,
			Timestamp: kafkaMsg.Timestamp,
		}

		for _, h := range kafkaMsg.Headers {
			switch string(h.Key) {
			case protobus.HeaderTraceID:
				msg.TraceID = string(h.Value)
			case protobus.HeaderMessageID:
				msg.ID = string(h.Value)
			case protobus.HeaderMessageFullName:
				msg.FullName = protoreflect.FullName(h.Value)
			}
		}

		if msg.FullName != "" {
			mt, err := c.resolve(msg.FullName)
			if err != nil {
				if !errors.Is(err, protoregistry.NotFound) {
					c.logger.Error("resolve message", "err", err)
				}
				return
			}

			msg.Proto = mt.New().Interface()

			err = c.unmarshal(msg.Value, msg.Proto)
			if err != nil {
				msg.Proto = nil

				if c.errHandler != nil {
					c.errHandler(msg, fmt.Errorf("%w: %w", ErrMessageUnmarshal, err))
				}

				c.logger.Debug("unmarshal message", "err", err)
			}
		}

		hCtx := logger.Init(ctx, msg.TraceID)
		for _, h := range handlers {
			h.Handle(hCtx, msg)
		}
	}
}

func (c *Group) isClosed() bool {
	select {
	case <-c.closed:
		return true
	default:
		return false
	}
}
