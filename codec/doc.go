// Package codec defines the contracts the producer and the consumer use to
// (de)serialize Kafka messages.
//
// An Encoder wraps one outgoing message: it gives the topic, the key, the event type,
// the payload and the extra headers. A Decoder is bound to one event type and turns
// the payload back into a value; it returns ErrWrongType for a message of another type,
// so several decoders can share a topic.
//
// See the wire package for the protobuf implementation.
package codec
