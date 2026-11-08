package consumer

import (
	"errors"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func handlerSamples(t *testing.T, topic, errLabel string) uint64 {
	t.Helper()

	var m dto.Metric

	require.NoError(t, handlerHistogram.WithLabelValues(topic, "type", errLabel).(prometheus.Metric).Write(&m))

	return m.GetHistogram().GetSampleCount()
}

func TestUnit_observeHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		err      error
		errLabel string
	}{
		{name: "handled", err: nil, errLabel: "false"},
		{name: "failed", err: errors.New("err"), errLabel: "true"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			topic := "observe_handler_" + tt.name
			before := handlerSamples(t, topic, tt.errLabel)

			err := observeHandler(topic, "type")(tt.err)

			assert.Equal(t, tt.err, err, "the error is passed through")
			assert.Equal(t, before+1, handlerSamples(t, topic, tt.errLabel))
		})
	}
}

func TestUnit_setCommitted(t *testing.T) {
	t.Parallel()

	setCommitted("set_committed_topic", 3, 42)
	assert.Equal(t, 42.0, committed("set_committed_topic", "3"))

	setCommitted("set_committed_topic", 3, 43)
	assert.Equal(t, 43.0, committed("set_committed_topic", "3"))
}
