package codec

// Names of the headers the producer sets on every message.
const (
	// HeaderMessageID is the header with the unique ID of the message (UUIDv7).
	HeaderMessageID = "id"
	// HeaderTraceID is the header with the trace ID taken from the context of the sender.
	HeaderTraceID = "trace-id"
)

// HeaderMessageIDByte and HeaderTraceIDByte are the header names as byte slices
// to avoid the conversion on every message. Don't modify them.
var (
	HeaderMessageIDByte = []byte(HeaderMessageID)
	HeaderTraceIDByte   = []byte(HeaderTraceID)
)

type (
	// Header is a Kafka record header.
	Header struct {
		Key   []byte
		Value []byte
	}

	// Meta describes where an event is sent and how it is identified.
	Meta interface {
		// Topic returns the Kafka topic of the event.
		Topic() string
		// Type returns the name of the event type; it distinguishes the events sharing a topic.
		Type() string
	}

	// Encoder wraps one outgoing message of type T.
	Encoder[T any] interface {
		Meta
		// Marshal returns the payload. It returns an error (e.g. ErrNilMessage) for an invalid message,
		// and the producer reports it before checking the topic and the key.
		Marshal() ([]byte, error)
		// Key returns the partitioning key, nil if the message has no key.
		Key() []byte
		// Headers returns the extra headers the decoder needs to recognize the message.
		Headers() []Header
	}

	// Decoder turns the payloads of one event type into values of type T.
	Decoder[T any] interface {
		Meta
		// Unmarshal decodes the payload. It returns ErrWrongType if the headers point to another event type.
		Unmarshal(data []byte, headers []Header) (T, error)
	}
)
