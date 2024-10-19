package consumer

import (
	"context"
	"sync"

	"go.uber.org/atomic"
	"golang.org/x/sync/errgroup"
)

// GroupFacade is the kafka consumer group facade.
// Facade can orchestrate many Group under the hood.
type GroupFacade struct {
	consumers map[string]*Group
	mu        sync.RWMutex

	cfg  FacadeConfig
	opts []Option

	closed atomic.Bool
}

// NewGroupFacade constructs GroupFacade
func NewGroupFacade(cfg FacadeConfig, opts ...Option) *GroupFacade {
	return &GroupFacade{
		consumers: make(map[string]*Group),
		cfg:       cfg,
		opts:      opts,
	}
}

// AddHandler adds message handler with a group name
func (c *GroupFacade) AddHandler(group string, handler Handler) (err error) {
	if c.isClosed() {
		return ErrClosed
	}

	c.mu.Lock()

	consumer, ok := c.consumers[group]
	if !ok {
		consumer, err = NewGroup(GroupConfig{
			Brokers: c.cfg.Brokers,
			Sasl:    c.cfg.Sasl,
			Group:   group,
		}, c.opts...)
		if err != nil {
			return err
		}
		c.consumers[group] = consumer
	}

	c.mu.Unlock()

	return consumer.AddHandler(handler)
}

// Run starts consume the messages and handle them
func (c *GroupFacade) Run(ctx context.Context) error {
	if c.isClosed() {
		return ErrClosed
	}

	grp, err := c.setup(ctx)
	if err != nil {
		return err
	}

	return grp.Wait()
}

// Close closes the kafka consumers
func (c *GroupFacade) Stop(ctx context.Context) (err error) {
	if c.closed.Swap(true) {
		return ErrClosed
	}

	for _, consumer := range c.consumers {
		err = consumer.Stop(ctx)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *GroupFacade) setup(ctx context.Context) (*errgroup.Group, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	var grp, gCtx = errgroup.WithContext(ctx)
	for _, consumer := range c.consumers {
		grp.Go(func() error {
			c.mu.RLock()
			defer c.mu.RUnlock()

			return consumer.Run(gCtx)
		})
	}

	return grp, nil
}

func (c *GroupFacade) isClosed() bool {
	return c.closed.Load()
}
