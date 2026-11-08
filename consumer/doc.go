// Package consumer provides a Kafka consumer group with single-message and batch handlers,
// retries and offset commits.
//
// A handler is bound to one event type of a topic, several handlers may share a topic:
//
//	g, err := consumer.NewGroup([]string{"localhost:9092"}, "orders-service")
//	if err != nil {
//		return err
//	}
//	defer g.Close()
//
//	err = g.AddHandler(consumer.NewHandler(wire.Decoder[*events.OrderCreated](),
//		func(ctx context.Context, msg *events.OrderCreated, raw consumer.Message) error {
//			return orders.Create(ctx, msg)
//		}))
//	if err != nil {
//		return err
//	}
//
//	err = g.Run(ctx)
//
// With the wire codec, protobus.NewHandler and protobus.NewHandlerBatch infer the decoder
// from the type of the handler function.
//
// # Delivery semantics
//
// Errors and panics of a handler are retried indefinitely by default (RetryMaxTime is zero),
// and a retry is not interrupted by the cancellation of the context.
//
// A message which fails keeps its partition busy until it is handled, and with it the rebalance
// and Group.Close. The lag grows and the errors are logged: it is a signal for a developer to fix
// the handler or to return PermanentError.
//
// If the retries are switched off (WithoutHandlerRetry), the handler returns PermanentError or
// RetryMaxTime has elapsed, the message is considered handled and its offset is committed.
//
// A message which can't be decoded is logged. HandlerProto calls its function with the zero value of T,
// HandlerBatch skips the message.
//
// A message of another type on the same topic is skipped, and for the handler it counts as handled:
// HandlerProto advances its offset at once, HandlerBatch at once or, if it holds unflushed messages
// of that partition, after their flush. The commit of a partition goes up to the minimum offset
// among the handlers of the topic.
//
// The message with the offset 0 is committed only after the next message has been handled.
//
// The buffer of HandlerBatch is not flushed on a rebalance, and the flush by the timer runs
// with context.Background().
package consumer

//go:generate go run go.uber.org/mock/mockgen@v0.6.0 -destination=sarama_mock_test.go -package=consumer github.com/IBM/sarama ConsumerGroup,ConsumerGroupSession,ConsumerGroupClaim,AccessTokenProvider
//go:generate go run go.uber.org/mock/mockgen@v0.6.0 -destination=codec_mock_test.go -package=consumer github.com/korableg/bus/codec Decoder
//go:generate go run go.uber.org/mock/mockgen@v0.6.0 -destination=handler_mock_test.go -package=consumer -self_package=github.com/korableg/bus/consumer github.com/korableg/bus/consumer Handler
