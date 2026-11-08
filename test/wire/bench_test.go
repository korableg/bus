//go:build integration_test
// +build integration_test

package test

import (
	"context"
	"log/slog"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	wireCodec "github.com/korableg/bus/codec/proto/wire"
	"github.com/korableg/bus/consumer"
	"github.com/korableg/bus/producer"
	"github.com/korableg/bus/test"
)

const (
	benchSyncWorkers = 32
	benchWaitTimeout = 5 * time.Minute
)

var benchLogger = slog.New(slog.DiscardHandler)

func benchEvent(i int) *test.TestEvent1 {
	return &test.TestEvent1{
		Id:  strconv.Itoa(i),
		Num: int64(i),
		Nm: &test.NestedMsg{
			Name:     "nested_name",
			Duration: uint64(i),
		},
	}
}

func newBenchProducer(b *testing.B, deliveryErrs *atomic.Int64) *producer.Producer[proto.Message] {
	b.Helper()

	p, err := producer.New(wireCodec.Encoder(), bootstrap,
		producer.WithLogger(benchLogger),
		producer.WithErrorHandler(func(error) { deliveryErrs.Add(1) }),
	)
	require.NoError(b, err)

	return p
}

func reportThroughput(b *testing.B) {
	b.Helper()

	if elapsed := b.Elapsed(); elapsed > 0 {
		b.ReportMetric(float64(b.N)/elapsed.Seconds(), "msgs/s")
	}
}

// BenchmarkProducer_Send measures Send until the broker has acknowledged all the messages:
// Close, which waits for the queued messages, is inside the timer.
func BenchmarkProducer_Send(b *testing.B) {
	cleanUp(b)

	var (
		ctx  = context.Background()
		errs atomic.Int64
		p    = newBenchProducer(b, &errs)
	)

	b.ResetTimer()

	for i := range b.N {
		if err := p.Send(ctx, benchEvent(i)); err != nil {
			b.Fatal(err)
		}
	}

	require.NoError(b, p.Close())

	b.StopTimer()

	require.Zero(b, errs.Load(), "delivery errors")
	reportThroughput(b)
}

// BenchmarkProducer_SendSync measures SendSync called one after another: ns/op is the round trip.
func BenchmarkProducer_SendSync(b *testing.B) {
	cleanUp(b)

	var (
		ctx  = context.Background()
		errs atomic.Int64
		p    = newBenchProducer(b, &errs)
	)

	b.ResetTimer()

	for i := range b.N {
		if err := p.SendSync(ctx, benchEvent(i)); err != nil {
			b.Fatal(err)
		}
	}

	b.StopTimer()

	require.NoError(b, p.Close())
	reportThroughput(b)
}

// BenchmarkProducer_SendSyncConcurrent measures SendSync called from benchSyncWorkers goroutines,
// e.g. from the handlers of an HTTP server.
func BenchmarkProducer_SendSyncConcurrent(b *testing.B) {
	cleanUp(b)

	var (
		ctx  = context.Background()
		errs atomic.Int64
		p    = newBenchProducer(b, &errs)
		next atomic.Int64
		wg   sync.WaitGroup
	)

	b.ResetTimer()

	for range benchSyncWorkers {
		wg.Go(func() {
			for i := next.Add(1); i <= int64(b.N); i = next.Add(1) {
				if err := p.SendSync(ctx, benchEvent(int(i))); err != nil {
					b.Error(err)
					return
				}
			}
		})
	}

	wg.Wait()

	b.StopTimer()

	require.NoError(b, p.Close())
	reportThroughput(b)
}

// BenchmarkConsumer_Handler measures a group with one HandlerProto reading 3 partitions.
func BenchmarkConsumer_Handler(b *testing.B) {
	benchmarkConsumer(b, func(handled func(n int)) consumer.Handler {
		return consumer.NewHandler(wireCodec.Decoder[*test.TestEvent1](),
			func(context.Context, *test.TestEvent1, consumer.Message) error {
				handled(1)
				return nil
			})
	})
}

// BenchmarkConsumer_HandlerBatch measures a group with one HandlerBatch (the batch size is 100) reading 3 partitions.
// The short flush timeout keeps the last incomplete batch from adding the default 10s to the result.
func BenchmarkConsumer_HandlerBatch(b *testing.B) {
	benchmarkConsumer(b, func(handled func(n int)) consumer.Handler {
		return consumer.NewHandlerBatch(wireCodec.Decoder[*test.TestEvent1](),
			func(_ context.Context, msgs []*test.TestEvent1, _ []consumer.Message) error {
				handled(len(msgs))
				return nil
			},
			consumer.WithHandlerBatchFlushTimeout(10*time.Millisecond),
		)
	})
}

// benchmarkConsumer produces b.N messages, then measures the group consuming them, from the first handled message
// to the last one. Producing and joining the group are outside the timer.
func benchmarkConsumer(b *testing.B, newHandler func(handled func(n int)) consumer.Handler) {
	b.Helper()

	cleanUp(b)

	var (
		ctx  = context.Background()
		errs atomic.Int64
		p    = newBenchProducer(b, &errs)
	)

	for i := range b.N {
		require.NoError(b, p.Send(ctx, benchEvent(i)))
	}

	require.NoError(b, p.Close())
	require.Zero(b, errs.Load(), "delivery errors")

	var (
		started, done         = make(chan struct{}), make(chan struct{})
		startedOnce, doneOnce sync.Once
		count                 atomic.Int64
	)

	handled := func(n int) {
		startedOnce.Do(func() { close(started) })

		if count.Add(int64(n)) >= int64(b.N) {
			doneOnce.Do(func() { close(done) })
		}
	}

	// a new group starts from the oldest message, cleanUp has deleted the messages of the previous runs
	g, err := consumer.NewGroup(bootstrap, "bench-"+uuid.NewString(),
		consumer.WithEarliestOffsetReset(),
		consumer.WithLogger(benchLogger),
	)
	require.NoError(b, err)
	require.NoError(b, g.AddHandler(newHandler(handled)))

	runErr := make(chan error, 1)
	go func() { runErr <- g.Run(ctx) }()

	waitBench(b, started)
	b.ResetTimer()
	waitBench(b, done)
	b.StopTimer()

	require.NoError(b, g.Close())
	require.NoError(b, <-runErr)
	reportThroughput(b)
}

func waitBench(b *testing.B, ch <-chan struct{}) {
	b.Helper()

	select {
	case <-ch:
	case <-time.After(benchWaitTimeout):
		b.Fatal("timeout")
	}
}
