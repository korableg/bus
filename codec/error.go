package codec

import "errors"

var (
	// ErrWrongType is returned by Decoder.Unmarshal for a message of another event type.
	ErrWrongType = errors.New("wrong type")
	// ErrNilMessage is returned by Encoder.Marshal for a nil message.
	ErrNilMessage = errors.New("nil message")
)
