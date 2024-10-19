package codec

var (
	HeaderMessageID = []byte("id")
	HeaderTraceID   = []byte("trace-id")
)

type (
	Header struct {
		Key   []byte
		Value []byte
	}

	Meta interface {
		Topic() string
		Key() []byte
		Headers() []Header
		Type() string
	}

	Encoder[T any] interface {
		Meta
		Marshal() ([]byte, error)
	}
)
