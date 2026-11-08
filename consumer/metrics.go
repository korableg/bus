package consumer

import (
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/korableg/bus/internal/metrics"
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

	// committedOffset is the committed offset by partition. After a rebalance the previous owner keeps
	// exporting the last value of the partition, so use max by (topic, partition) in the queries.
	committedOffset = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "kafka_consumer_committed_offset",
		Help: "Committed offset (the next offset to consume) by partition.",
	}, []string{labelTopic, labelPartition})
)

func observeHandler(topic, eventType string) func(err error) error {
	start := time.Now()

	return func(err error) error {
		handlerHistogram.With(prometheus.Labels{
			labelTopic:     topic,
			labelEventType: eventType,
			labelError:     metrics.ErrToLabel(err),
		}).Observe(time.Since(start).Seconds())

		return err
	}
}

func setCommitted(topic string, partition int32, offset int64) {
	committedOffset.With(prometheus.Labels{
		labelTopic:     topic,
		labelPartition: strconv.Itoa(int(partition)),
	}).Set(float64(offset))
}
