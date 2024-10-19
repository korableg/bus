package producer

import (
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestUnit_Options_Producer(t *testing.T) {
	t.Parallel()

	var (
		o = NewOptions()
		l = slog.Default()
	)

	WithoutIdempotence()(&o)
	WithFlushParams(1, 2, 3)(&o)
	WithRetries(3)(&o)
	WithRetryBackOff(4)(&o)
	WithLogger(l)(&o)

	assert.False(t, o.Idempotence)
	assert.Equal(t, o.FlushFrequency, time.Duration(1))
	assert.Equal(t, o.FlushBytes, 2)
	assert.Equal(t, o.FlushMessages, 3)
	assert.Equal(t, o.Retries, 3)
	assert.Equal(t, o.RetryBackOff, time.Duration(4))
	assert.NotNil(t, o.Logger)
	assert.Equal(t, o.Logger, l)
}
