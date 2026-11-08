package consumer

import (
	"errors"

	"github.com/cenkalti/backoff/v7"
)

var (
	// ErrNoTopics is returned by Group.Run if no handler has been added.
	ErrNoTopics = errors.New("no topics to consume")
	// ErrEmptyTopic is returned by Group.AddHandler for a handler without a topic.
	ErrEmptyTopic = errors.New("empty topic")
	// ErrClosed is returned by the calls of a closed group.
	ErrClosed = errors.New("closed")
	// ErrRunned is returned by Group.AddHandler after Group.Run has started.
	ErrRunned = errors.New("runned")
)

// PermanentError marks err as permanent: a handler function returning it is not retried,
// the message is considered handled and its offset is committed. Use it for the messages
// a retry can't fix, e.g. invalid ones.
func PermanentError(err error) error {
	return backoff.Permanent(err)
}
