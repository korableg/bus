package producer

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/korableg/bus/internal/metrics"
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
		labelError:     metrics.ErrToLabel(err),
	}).Observe(time.Since(m.timestamp).Seconds())
}
