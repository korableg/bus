package producer

import (
	"errors"
)

var (
	// ErrEmptyTopic message (proto.Message) has no topic (no event)
	ErrEmptyTopic = errors.New("empty topic")
	// ErrEmptyKey message (proto.Message) has no key
	ErrEmptyKey = errors.New("empty key")
	// ErrClosed producer is closed
	ErrClosed = errors.New("closed")
)
