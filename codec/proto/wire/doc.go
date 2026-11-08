// Package wire implements codec.Encoder and codec.Decoder for protobuf messages.
//
// The topic and the key are taken from the options of the proto definition:
//
//	message OrderCreated {
//	  option (korableg.protobuf.event.topic) = "orders";
//
//	  string id = 1 [ (korableg.protobuf.event.key) = true ];
//	}
//
// The encoder puts the full name of the message into the "type" header, and the decoder
// accepts only the messages whose header matches its type.
package wire
