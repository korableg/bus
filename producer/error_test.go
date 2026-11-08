package producer

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUnit_SendError_Error(t *testing.T) {
	t.Parallel()

	err := &SendError{Topic: "topic", ID: "id", Err: errors.New("delivery failed")}

	assert.Equal(t, `send message (topic "topic", id "id"): delivery failed`, err.Error())
}

func TestUnit_SendError_Unwrap(t *testing.T) {
	t.Parallel()

	cause := errors.New("delivery failed")

	var err error = &SendError{Err: cause}

	assert.Same(t, cause, errors.Unwrap(err))
	assert.ErrorIs(t, err, cause)
}
