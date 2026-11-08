// Package protobus binds the wire codec to the consumer handlers, so the type of the event
// is inferred from the handler function and isn't written twice:
//
//	err = g.AddHandler(protobus.NewHandler(
//		func(ctx context.Context, msg *events.OrderCreated, raw consumer.Message) error {
//			return orders.Create(ctx, msg)
//		}))
//
// is the same as
//
//	err = g.AddHandler(consumer.NewHandler(wire.Decoder[*events.OrderCreated](),
//		func(ctx context.Context, msg *events.OrderCreated, raw consumer.Message) error {
//			return orders.Create(ctx, msg)
//		}))
//
// The package is separate from wire so that the codec doesn't depend on the consumer and Kafka.
package protobus
