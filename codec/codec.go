package codec

const (
	HeaderMessageID = "id"
	HeaderTraceID   = "trace-id"
)

var (
	HeaderMessageIDByte = []byte(HeaderMessageID)
	HeaderTraceIDByte   = []byte(HeaderTraceID)
)

type (
	Header struct {
		Key   []byte
		Value []byte
	}

	Meta interface {
		Topic() string
		Type() string
	}

	Encoder[T any] interface {
		Meta
		Marshal() ([]byte, error)
		Key() []byte
		Headers() []Header
	}

	Decoder[T any] interface {
		Meta
		Unmarshal(data []byte, headers []Header) (T, error)
	}
)
