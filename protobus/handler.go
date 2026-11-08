package protobus

import (
	"google.golang.org/protobuf/proto"

	"github.com/korableg/bus/codec/proto/wire"
	"github.com/korableg/bus/consumer"
)

// NewHandler returns consumer.NewHandler with wire.Decoder of T, the type of the messages handlerFunc takes.
func NewHandler[T proto.Message](
	handlerFunc consumer.HandlerProtoFunc[T], opts ...consumer.HandlerOption,
) *consumer.HandlerProto[T] {
	return consumer.NewHandler(wire.Decoder[T](), handlerFunc, opts...)
}

// NewHandlerBatch returns consumer.NewHandlerBatch with wire.Decoder of T, the type of the messages handlerFunc takes.
func NewHandlerBatch[T proto.Message](
	handlerFunc consumer.HandlerBatchFunc[T], opts ...consumer.HandlerOption,
) *consumer.HandlerBatch[T] {
	return consumer.NewHandlerBatch(wire.Decoder[T](), handlerFunc, opts...)
}
