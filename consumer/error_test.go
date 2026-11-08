package consumer

import (
	"testing"

	"github.com/cenkalti/backoff/v7"
	"github.com/stretchr/testify/assert"
)

func TestUnit_PermanentError(t *testing.T) {
	t.Parallel()

	err := PermanentError(ErrClosed)

	assert.ErrorIs(t, err, backoff.ErrPermanent)
	assert.ErrorIs(t, err, ErrClosed)
}
