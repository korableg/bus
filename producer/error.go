package producer

import (
	"errors"
	"fmt"
)

var (
	// ErrEmptyTopic is returned for a message the encoder gives no topic for.
	ErrEmptyTopic = errors.New("empty topic")
	// ErrEmptyKey is returned for a message without a key while the idempotence is on.
	ErrEmptyKey = errors.New("empty key")
	// ErrClosed is returned by the calls of a closed producer.
	ErrClosed = errors.New("closed")
)

// SendError is the delivery error of a message sent by Send. It is passed to the handler set by WithErrorHandler.
type SendError struct {
	Topic string // Topic of the message
	Key   []byte // Key of the message
	ID    string // Message ID (the value of the id header)
	Type  string // Event type of the message
	Msg   any    // Source message passed to Send
	Err   error  // Delivery error
}

// Error returns the description of the error with the topic and the ID of the message.
func (e *SendError) Error() string {
	return fmt.Sprintf("send message (topic %q, id %q): %v", e.Topic, e.ID, e.Err)
}

// Unwrap returns the delivery error.
func (e *SendError) Unwrap() error {
	return e.Err
}
