package consumer

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/IBM/sarama"
)

// Group is the kafka consumer group
type (
	Group struct {
		cli     sarama.ConsumerGroup
		hCtl    *handlerController
		session *session

		logger *slog.Logger

		closed chan struct{}
	}
)

// NewGroup constructs Group
func NewGroup(brokers []string, group string, opts ...Option) (*Group, error) {
	o := newOptions(opts...)

	cli, err := sarama.NewConsumerGroup(brokers, group, newSaramaConfig(o))
	if err != nil {
		return nil, fmt.Errorf("init consumer group: %w", err)
	}

	var (
		hCtl = newHandlerCollection()
		oCtl = newOffsetController(hCtl, o)
	)

	return &Group{
		cli:     cli,
		hCtl:    hCtl,
		session: newSession(hCtl, oCtl, group, o.Logger, o.TraceIDCtx),

		logger: o.Logger.With("group", group),

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

	c.hCtl.Add(h)

	return nil
}

// Run starts consume the messages and handle them
func (c *Group) Run(ctx context.Context) (err error) {
	if c.isClosed() {
		return ErrClosed
	}

	topics := c.hCtl.Topics()
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

		c.logger.InfoContext(ctx, "starting of consumption")

		err = c.cli.Consume(ctx, topics, c.session)

		c.logger.InfoContext(ctx, "consumption was finished", "err", err)

		if err != nil && errors.Is(err, sarama.ErrClosedConsumerGroup) {
			return nil
		}
	}
}

// Close closes the kafka consumer
func (c *Group) Close() error {
	if c.isClosed() {
		return ErrClosed
	}

	close(c.closed)
	return c.cli.Close()
}

func (c *Group) isClosed() bool {
	select {
	case <-c.closed:
		return true
	default:
		return false
	}
}
