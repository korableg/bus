package consumer

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/IBM/sarama"
	"github.com/cenkalti/backoff/v7"
)

const (
	consumeRetryInitialInterval = 100 * time.Millisecond
	consumeRetryMaxInterval     = 10 * time.Second
)

// Group is a Kafka consumer group that passes the messages of the topics of its handlers to them
// and commits the handled offsets.
type Group struct {
	cli     sarama.ConsumerGroup
	hCtl    handlerController
	session *session

	logger *slog.Logger

	closed  chan struct{}
	closing atomic.Bool
	runned  atomic.Bool
}

// NewGroup returns the consumer group named group. Add the handlers with AddHandler, then call Run.
func NewGroup(brokers []string, group string, opts ...Option) (*Group, error) {
	o := newOptions(opts...)

	cli, err := sarama.NewConsumerGroup(brokers, group, newSaramaConfig(o))
	if err != nil {
		return nil, fmt.Errorf("init consumer group: %w", err)
	}

	return newGroup(cli, group, o), nil
}

func newGroup(cli sarama.ConsumerGroup, group string, o Options) *Group {
	var (
		hCtl = make(handlerController)
		oCtl = newOffsetController(hCtl, o)
	)

	g := &Group{
		cli:     cli,
		hCtl:    hCtl,
		session: newSession(hCtl, oCtl, group, o.Logger, o.TraceIDCtx),

		logger: o.Logger.With("group", group),

		closed: make(chan struct{}),
	}

	go g.logErrors()

	return g
}

func (c *Group) logErrors() {
	for err := range c.cli.Errors() {
		c.logger.Error("consumer group error", "err", err)
	}
}

// AddHandler subscribes h to its topic. It must be called before Run: it returns ErrRunned after Run
// has started, ErrClosed after Close and ErrEmptyTopic for a handler without a topic.
func (c *Group) AddHandler(h Handler) error {
	if c.isClosed() {
		return ErrClosed
	}

	if c.isRunned() {
		return ErrRunned
	}

	if h.Topic() == "" {
		return ErrEmptyTopic
	}

	c.hCtl.Add(h)

	return nil
}

// Run consumes the topics of the handlers until ctx is done or the group is closed, then returns nil.
// It rejoins the group after a rebalance and retries the connection errors with a backoff.
// It returns ErrNoTopics if no handler has been added.
func (c *Group) Run(ctx context.Context) (err error) {
	if c.isClosed() {
		return ErrClosed
	}

	topics := c.hCtl.Topics()
	if len(topics) == 0 {
		return ErrNoTopics
	}

	c.runned.Store(true)

	bo := backoff.NewExponentialBackOff()
	bo.InitialInterval = consumeRetryInitialInterval
	bo.MaxInterval = consumeRetryMaxInterval
	bo.Reset()

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
		if err == nil {
			// a rebalance ends the session without an error
			c.logger.InfoContext(ctx, "consumption was finished")
			bo.Reset()

			continue
		}

		if errors.Is(err, sarama.ErrClosedConsumerGroup) {
			c.logger.InfoContext(ctx, "consumption was finished", "err", err)
			return nil
		}

		delay := bo.NextBackOff()
		c.logger.ErrorContext(ctx, "consumption was finished with error", "err", err, "retry_in", delay)

		if !c.wait(ctx, delay) {
			return nil
		}
	}
}

func (c *Group) wait(ctx context.Context, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()

	select {
	case <-t.C:
		return true
	case <-ctx.Done():
		return false
	case <-c.closed:
		return false
	}
}

// Close leaves the group and releases the connections. It waits for the handlers to finish their
// current messages. The next calls return ErrClosed.
func (c *Group) Close() error {
	if c.closing.Swap(true) {
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

func (c *Group) isRunned() bool {
	return c.runned.Load()
}
