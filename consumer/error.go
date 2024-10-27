package consumer

import (
	"errors"

	"github.com/cenkalti/backoff/v4"
)

var (
	// ErrNoTopics consumer doesn't have topics and can't start
	ErrNoTopics = errors.New("no topics to consume")
	// ErrEmptyTopic the message wrapped by the handler has no topic (not event)
	ErrEmptyTopic = errors.New("empty topic")
	// ErrClosed consumer was closed
	ErrClosed = errors.New("closed")
	// ErrRunned consumer is runned
	ErrRunned = errors.New("runned")

	// ErrMessageUnmarshal the error happens when message tried to unmarshal
	ErrMessageUnmarshal = errors.New("message unmarshal error")
	// ErrMessageHandle the error happens when message tried to handle
	ErrMessageHandle = errors.New("message handle error")
)

func PermanentError(err error) error {
	return backoff.Permanent(err)
}
