package producer

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type metric struct {
	topic     string
	eventType string
	timestamp time.Time
}

const (
	labelTopic     = "topic"
	labelEventType = "event_type"
	labelError     = "error"
)

var sendHistogram = promauto.NewHistogramVec(prometheus.HistogramOpts{
	Name:    "kafka_producer_sending_seconds",
	Help:    "Histogram of send kafka events latency (seconds).",
	Buckets: prometheus.DefBuckets,
}, []string{labelTopic, labelEventType, labelError})

func observeSend(m metric, err error) {
	sendHistogram.With(prometheus.Labels{
		labelTopic:     m.topic,
		labelEventType: m.eventType,
		labelError:     errToLabel(err),
	}).Observe(time.Since(m.timestamp).Seconds())
}

func errToLabel(err error) string {
	if err == nil {
		return "false"
	}
	return "true"
}
