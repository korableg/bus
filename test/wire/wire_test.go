//go:build integration_test
// +build integration_test

package test

import (
	"context"
	"errors"
	"log/slog"
	"math/rand/v2"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/IBM/sarama"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"

	wireCodec "github.com/korableg/bus/codec/proto/wire"
	"github.com/korableg/bus/consumer"
	"github.com/korableg/bus/producer"
	"github.com/korableg/bus/test"
)

const (
	msgCount     = 10000
	msgSyncCount = 10
)

var (
	traceID   = uuid.NewString()
	bootstrap = []string{os.Getenv("KAFKA_BOOTSTRAP_SERVERS")}
)

func TestIntegration_Producer_Consumer(t *testing.T) {
	defer cleanUp(t)

	ctx := context.Background()

	grp, _ := errgroup.WithContext(context.Background())
	grp.Go(func() error {
		return testRunProducer(t, ctx, bootstrap)
	})
	grp.Go(func() error {
		return testRunConsumer(t, bootstrap)
	})

	assert.Eventually(t, func() bool {
		assert.NoError(t, grp.Wait())
		return true
	}, 20*time.Second, time.Second)
}

func testRunProducer(t *testing.T, ctx context.Context, brokers []string) error { //nolint: thelper
	prod, err := producer.New(wireCodec.Encoder(), brokers, producer.WithTraceID(func(ctx context.Context) string { return traceID }))
	if err != nil {
		return err
	}

	defer func() {
		cErr := prod.Close()
		assert.NoError(t, cErr)
	}()

	var wg sync.WaitGroup
	wg.Add(5)

	go func() {
		defer wg.Done()
		for range msgSyncCount {
			gErr := prod.SendSync(ctx, &test.TestEvent1{
				Id:  uuid.NewString(),
				Num: rand.Int64(),
				Nm: &test.NestedMsg{
					Name:     "nested_name",
					Duration: rand.Uint64(),
				},
			})
			assert.NoError(t, gErr)
		}
	}()

	go func() {
		defer wg.Done()
		for range msgCount {
			gErr := prod.Send(ctx, &test.TestEvent1{
				Id:  uuid.NewString(),
				Num: rand.Int64(),
				Nm: &test.NestedMsg{
					Name:     "nested_name",
					Duration: rand.Uint64(),
				},
			})
			assert.NoError(t, gErr)
		}
	}()

	go func() {
		defer wg.Done()
		for range msgCount {
			gErr := prod.Send(ctx, &test.TestEvent3{
				Id:     uuid.NewString(),
				Result: rand.Int64N(2) == 1,
			})
			assert.NoError(t, gErr)
		}
	}()

	go func() {
		defer wg.Done()
		for range msgCount {
			gErr := prod.Send(ctx, &test.TestEvent4{
				Id:       rand.Int32(),
				Instance: uuid.NewString(),
			})
			assert.NoError(t, gErr)
		}
	}()

	go func() {
		defer wg.Done()
		for range msgCount {
			gErr := prod.Send(ctx, &test.TestEvent5{
				Id:       rand.Int32(),
				Division: "main",
				Office:   rand.Uint64(),
			})
			assert.NoError(t, gErr)
		}
	}()

	wg.Wait()

	return nil
}

func testRunConsumer(t *testing.T, brokers []string) error { //nolint: thelper
	type ctxKey struct{}

	var (
		wg        sync.WaitGroup
		onceErr   sync.Once
		oncePanic sync.Once
	)

	wg.Add(msgCount*4 + msgSyncCount + 1)

	cons, err := consumer.NewGroup(brokers, "test",
		consumer.WithEarliestOffsetReset(),
		consumer.WithLogger(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}))),
		consumer.WithTraceIDCtxFunc(func(ctx context.Context, s string) context.Context {
			return context.WithValue(ctx, ctxKey{}, s)
		}))

	if err != nil {
		return err
	}

	err = cons.AddHandler(consumer.NewHandlerBatch(wireCodec.Decoder[*test.TestEvent1](),
		func(ctx context.Context, msgs []*test.TestEvent1, _ []consumer.Message) error {
			for range msgs {
				wg.Done()
			}
			return nil
		}, consumer.WithHandlerBatchSize(7), consumer.WithHandlerBatchFlushTimeout(1*time.Second)))
	if err != nil {
		return err
	}

	err = cons.AddHandler(consumer.NewHandler(wireCodec.Decoder[*test.TestEvent3](),
		func(ctx context.Context, msg *test.TestEvent3, _ consumer.Message) error {
			tid := ctx.Value(ctxKey{})
			assert.Equal(t, tid.(string), traceID)
			wg.Done()
			return nil
		}))
	if err != nil {
		return err
	}

	err = cons.AddHandler(consumer.NewHandler(wireCodec.Decoder[*test.NonEvent](),
		func(_ context.Context, msg *test.NonEvent, _ consumer.Message) error { return nil }))
	if !errors.Is(err, consumer.ErrEmptyTopic) {
		if err == nil {
			return errors.New("should be error")
		}
		return err
	}

	err = cons.AddHandler(consumer.NewHandler(wireCodec.Decoder[*test.TestEvent4](),
		func(_ context.Context, msg *test.TestEvent4, _ consumer.Message) (err error) {
			wg.Done()
			onceErr.Do(func() {
				err = errors.New("err")
			})
			return err
		}))
	if err != nil {
		return err
	}

	err = cons.AddHandler(consumer.NewHandler(wireCodec.Decoder[*test.TestEvent5](),
		func(_ context.Context, msg *test.TestEvent5, _ consumer.Message) (err error) {
			oncePanic.Do(func() {
				panic(errors.New("panic err"))
			})
			wg.Done()
			return err
		}))
	if err != nil {
		return err
	}

	go func() {
		wg.Wait()

		cErr := cons.Close()
		assert.NoError(t, cErr)

		cErr = cons.Close()
		if !errors.Is(cErr, consumer.ErrClosed) {
			assert.NoError(t, cErr)
		}
	}()

	assert.NoError(t, cons.Run(context.Background()))

	return nil
}

func TestIntegration_Producer(t *testing.T) {
	defer cleanUp(t)

	ctx := context.Background()

	k, err := producer.New(wireCodec.Encoder(), bootstrap)
	require.NoError(t, err)

	err = k.Send(ctx, &test.TestEvent1{
		Id:  uuid.NewString(),
		Num: 23423,
		Nm:  nil,
	})
	assert.NoError(t, err)

	err = k.Send(ctx, &test.NonEvent{
		Color: "blue",
	})
	assert.ErrorIs(t, err, producer.ErrEmptyTopic)

	err = k.SendSync(ctx, &test.TestEvent3{
		Id:     uuid.NewString(),
		Result: true,
	})
	assert.NoError(t, err)

	err = k.Send(ctx, &test.TestEvent2{
		Solution: "success",
	})
	assert.ErrorIs(t, err, producer.ErrEmptyKey)

	err = k.Close()
	assert.NoError(t, err)

	err = k.Close()
	assert.ErrorIs(t, err, producer.ErrClosed)

	err = k.Send(ctx, &test.TestEvent1{
		Id:  uuid.NewString(),
		Num: 23423,
		Nm:  nil,
	})
	assert.ErrorIs(t, err, producer.ErrClosed)
}

func TestIntegration_Consumer_ContextCanceled(t *testing.T) {
	defer cleanUp(t)

	const (
		msgs                = 1000
		receiveBeforeCancel = 100
		receiveAfterCancel  = 20
	)
	var (
		wgBefore sync.WaitGroup
		wgAfter  sync.WaitGroup

		countBefore atomic.Int64
		countAfter  atomic.Int64
	)

	wgBefore.Add(receiveBeforeCancel)
	wgAfter.Add(receiveAfterCancel)

	p, err := producer.New(wireCodec.Encoder(), bootstrap)
	require.NoError(t, err)

	c, err := consumer.NewGroup(bootstrap, "test",
		consumer.WithEarliestOffsetReset(),
	)
	require.NoError(t, err)

	cCtx, cancel := context.WithCancel(context.Background())

	err = c.AddHandler(consumer.NewHandler(wireCodec.Decoder[*test.TestEvent3](),
		func(_ context.Context, msg *test.TestEvent3, _ consumer.Message) error {
			switch {
			case countBefore.Add(1) <= receiveBeforeCancel:
				wgBefore.Done()
			case countAfter.Add(1) <= receiveAfterCancel:
				wgAfter.Done()
			}

			return nil
		}))
	require.NoError(t, err)

	for range msgs {
		err = p.Send(context.Background(), &test.TestEvent3{
			Id: uuid.NewString(),
		})
		assert.NoError(t, err)
	}

	var wgCancel sync.WaitGroup
	wgCancel.Add(1)
	go func() {
		gErr := c.Run(cCtx)
		wgCancel.Done()
		if gErr != nil {
			panic(gErr)
		}
	}()

	wgBefore.Wait()

	cancel()

	wgCancel.Wait()

	go func() {
		wgAfter.Wait()
		gErr := c.Close()
		if gErr != nil {
			panic(gErr)
		}
	}()

	assert.NoError(t, c.Run(context.Background()))
	assert.NoError(t, p.Close())
}

func cleanUp(t *testing.T) {
	t.Helper()

	admCli, err := sarama.NewClusterAdmin(bootstrap, sarama.NewConfig())
	require.NoError(t, err)

	defer admCli.Close()

	err = admCli.DeleteRecords("test_topic", map[int32]int64{
		0: -1,
		1: -1,
		2: -1,
	})
	assert.NoError(t, err)

	err = admCli.DeleteRecords("test_topic_2", map[int32]int64{
		0: -1,
	})
	assert.NoError(t, err)
}
