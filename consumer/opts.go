package consumer

import (
	"log/slog"
	"time"

	"github.com/IBM/sarama"
	"google.golang.org/protobuf/proto"
)

const (
	InitialOffsetLatest   = sarama.OffsetNewest
	InitialOffsetEarliest = sarama.OffsetOldest
)

type (
	// Options consumer options
	Options struct {
		InitialOffset   int64         // Determine when the group should to start consuming (default "latest")
		FetchMaxWait    time.Duration // Waiting time before returning messages to the client (relevant if FetchMinBytes > 1) (default 500)
		FetchMinBytes   int32         // Determine how many bytes kafka library have to receive before returning to the client (latency << FetchMinBytes << throughput) (default 1)
		CommitCount     int           // After how many messages do client have to call the commit (default 20)
		CommitDuration  time.Duration // After how much time do client have to call the commit (default 1 second)
		CommitQueueSize int           // Commit channel size (default 50)
		Unmarshal       UnmarshalFunc // Message unmarshaller (default proto.Unmarshal)
		ErrHandler      ErrorHandler  // Handler when an error occurs (on unmarshal or during handling) (default nil)
		Logger          *slog.Logger  // Logger (default writer is the os.Stdout)
	}

	// Option consumer option
	Option func(*Options)
)

// NewOptions constructs Options
func NewOptions(opts ...Option) Options {
	o := Options{
		InitialOffset:   InitialOffsetLatest,
		FetchMaxWait:    500 * time.Millisecond,
		FetchMinBytes:   1,
		CommitCount:     20,
		CommitDuration:  time.Second,
		CommitQueueSize: 10,
		Unmarshal:       proto.Unmarshal,
	}

	for _, opt := range opts {
		opt(&o)
	}

	return o
}

// GetLogger logger getter
func (o *Options) GetLogger() *slog.Logger {
	if o.Logger == nil {
		return slog.Default()
	}

	return o.Logger
}

// WithErrHandler adds error handler
func WithErrHandler(f ErrorHandler) Option {
	return func(o *Options) {
		o.ErrHandler = f
	}
}

// WithEarliestOffsetReset sets AutoOffsetReset to "earliest" mode
func WithEarliestOffsetReset() Option {
	return func(o *Options) {
		o.InitialOffset = InitialOffsetEarliest
	}
}

// WithFetchMaxWait sets FetchMaxWait option
func WithFetchMaxWait(d time.Duration) Option {
	return func(o *Options) {
		o.FetchMaxWait = d
	}
}

// WithFetchMinBytes sets FetchMinBytes option
func WithFetchMinBytes(value int32) Option {
	return func(o *Options) {
		o.FetchMinBytes = value
	}
}

// WithCommitQueueSize sets CommitQueueSize option
func WithCommitQueueSize(value int) Option {
	return func(o *Options) {
		o.CommitQueueSize = value
	}
}

// WithCommitCount sets CommitCount option
func WithCommitCount(value int) Option {
	return func(o *Options) {
		o.CommitCount = value
	}
}

// WithCommitDuration sets CommitDuration option
func WithCommitDuration(value time.Duration) Option {
	return func(o *Options) {
		o.CommitDuration = value
	}
}

// WithLogger sets Logger option
func WithLogger(logger *slog.Logger) Option {
	return func(o *Options) {
		o.Logger = logger
	}
}

// WithUnmarshaller sets Unmarshal option
func WithUnmarshaller(unmarshal UnmarshalFunc) Option {
	return func(o *Options) {
		o.Unmarshal = unmarshal
	}
}
