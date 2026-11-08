package producer

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestUnit_observeSend(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		err      error
		errLabel string
	}{
		{name: "delivered", err: nil, errLabel: "false"},
		{name: "failed", err: errors.New("err"), errLabel: "true"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			topic := "observe_send_" + tt.name
			before := sendSamples(t, topic, tt.errLabel)

			observeSend(metric{topic: topic, eventType: "type", timestamp: time.Now()}, tt.err)

			assert.Equal(t, before+1, sendSamples(t, topic, tt.errLabel))
		})
	}
}
