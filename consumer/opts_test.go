package consumer

import (
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestUnit_Options(t *testing.T) {
	t.Parallel()

	var (
		o = NewOptions()
		l = slog.Default()
	)

	WithLogger(l)(&o)
	WithCommitQueueSize(10)(&o)
	WithCommitCount(5)(&o)
	WithCommitDuration(6)(&o)
	WithEarliestOffsetReset()(&o)
	WithFetchMaxWait(7)(&o)
	WithFetchMinBytes(8)(&o)
	WithErrHandler(func(message Message, err error) {})(&o)

	assert.NotNil(t, o.Logger)
	assert.NotNil(t, o.Unmarshal)
	assert.Equal(t, o.CommitDuration, time.Duration(6))
	assert.Equal(t, o.InitialOffset, InitialOffsetEarliest)
	assert.Equal(t, o.FetchMinBytes, int32(8))
	assert.NotNil(t, o.ErrHandler)
	assert.Equal(t, o.CommitQueueSize, 10)

	assert.Equal(t, o.GetLogger(), l)
}
