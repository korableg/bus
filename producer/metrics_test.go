package producer

import (
	"errors"
	"testing"
	"time"
)

func TestUnit_observeSend(t *testing.T) {
	t.Parallel()

	observeSend(metric{
		topic:     "topic",
		eventType: "event",
		timestamp: time.Now(),
	}, nil)

	observeSend(metric{
		topic:     "topic",
		eventType: "event",
		timestamp: time.Now(),
	}, errors.New("err"))
}
