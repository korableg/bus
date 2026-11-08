// Package producer provides a Kafka producer that sends messages through a pluggable codec.
//
// The producer sends every message with a generated UUIDv7 in the "id" header and,
// if WithTraceID is set, the trace ID of the context in the "trace-id" header:
//
//	p, err := producer.New(wire.Encoder(), []string{"localhost:9092"})
//	if err != nil {
//		return err
//	}
//	defer p.Close()
//
//	err = p.SendSync(ctx, &events.OrderCreated{Id: "42"})
//
// Send returns as soon as the message is queued, and the delivery errors are reported to
// the handler set by WithErrorHandler. SendSync waits for the acknowledgment of the broker.
//
// The producer is idempotent by default, so every message must have a key (see WithoutIdempotence).
package producer

//go:generate go run go.uber.org/mock/mockgen@v0.6.0 -destination=sarama_mock_test.go -package=producer github.com/IBM/sarama Client,AsyncProducer,SyncProducer,AccessTokenProvider
//go:generate go run go.uber.org/mock/mockgen@v0.6.0 -destination=codec_mock_test.go -package=producer github.com/korableg/bus/codec Encoder
