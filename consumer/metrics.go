package consumer

import (
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	labelTopic     = "topic"
	labelEventType = "event_type"
	labelError     = "error"
	labelPartition = "partition"
)

var (
	handlerHistogram = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "kafka_consumer_handler_seconds",
		Help:    "Histogram of the handling kafka events latency (seconds).",
		Buckets: prometheus.DefBuckets,
	}, []string{labelTopic, labelEventType, labelError})

	commitCounter = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "kafka_consumer_committed_total",
		Help: "Total number of committed events.",
	}, []string{labelTopic, labelPartition})
)

func observeHandler(topic, eventType string) func(err error) error {
	start := time.Now()

	return func(err error) error {
		handlerHistogram.With(prometheus.Labels{
			labelTopic:     topic,
			labelEventType: eventType,
			labelError:     errToLabel(err),
		}).Observe(time.Since(start).Seconds())

		return err
	}
}

func incCommitted(topic string, partition int32) {
	commitCounter.With(prometheus.Labels{
		labelTopic:     topic,
		labelPartition: strconv.Itoa(int(partition)),
	}).Inc()
}

func errToLabel(err error) string {
	if err == nil {
		return "false"
	}
	return "true"
}
