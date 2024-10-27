package consumer

import (
	"testing"

	"github.com/cenkalti/backoff/v4"
	"github.com/stretchr/testify/assert"
)

func TestUnit_PermanentError(t *testing.T) {
	t.Parallel()

	err := PermanentError(ErrClosed)
	var permErr *backoff.PermanentError

	assert.ErrorAs(t, err, &permErr)
}
