//go:build integration_test

package test

import (
	"context"
	"encoding/json"
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
	"google.golang.org/protobuf/proto"

	"github.com/korableg/bus/consumer"
	"github.com/korableg/bus/producer"
	"gitlab.services.mts.ru/scs-data-platform/backend/modules/logger"
)

const (
	msgCount     = 10000
	msgSyncCount = 10
)

var (
	traceID   = uuid.NewString()
	bootstrap = "127.0.0.1:9094"
)

func TestIntegration_Producer_Consumer(t *testing.T) {
	defer cleanUp(t)

	ctx := logger.Init(context.Background(), traceID)

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

func testRunProducer(t *testing.T, ctx context.Context, brokers string) error { //nolint: thelper
	prod, err := producer.New(producer.Config{Brokers: brokers})
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
			gErr := prod.SendSync(ctx, &TestEvent1{
				Id:  uuid.NewString(),
				Num: rand.Int64(),
				Nm: &NestedMsg{
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
			gErr := prod.Send(ctx, &TestEvent1{
				Id:  uuid.NewString(),
				Num: rand.Int64(),
				Nm: &NestedMsg{
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
			gErr := prod.Send(ctx, &TestEvent3{
				Id:     uuid.NewString(),
				Result: rand.Int64N(2) == 1,
			})
			assert.NoError(t, gErr)
		}
	}()

	go func() {
		defer wg.Done()
		for range msgCount {
			gErr := prod.Send(ctx, &TestEvent4{
				Id:       rand.Int32(),
				Instance: uuid.NewString(),
			})
			assert.NoError(t, gErr)
		}
	}()

	go func() {
		defer wg.Done()
		for range msgCount {
			gErr := prod.Send(ctx, &TestEvent5{
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

func testRunConsumer(t *testing.T, brokers string) error { //nolint: thelper
	var (
		wg        sync.WaitGroup
		onceErr   sync.Once
		oncePanic sync.Once
	)

	wg.Add(msgCount*4 + msgSyncCount + 1)

	fCfg := consumer.NewFacadeConfig()
	fCfg.Brokers = brokers

	facade := consumer.NewGroupFacade(fCfg,
		consumer.WithEarliestOffsetReset(),
		consumer.WithErrHandler(func(msg consumer.Message, err error) {
			wg.Done()
		}),
		consumer.WithLogger(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}))),
	)

	err := facade.AddHandler("test", consumer.NewHandlerProtoBatch(
		func(ctx context.Context, msgs []*TestEvent1, _ []consumer.Message) error {
			for range msgs {
				wg.Done()
			}
			return nil
		}, consumer.WithHandlerBatchSize(7), consumer.WithHandlerBatchFlushTimeout(1*time.Second)))
	if err != nil {
		return err
	}

	err = facade.AddHandler("test", consumer.NewHandlerProto(
		func(ctx context.Context, msg *TestEvent3, _ consumer.Message) error {
			assert.Equal(t, logger.TraceID(ctx), traceID)
			wg.Done()
			return nil
		}))
	if err != nil {
		return err
	}

	err = facade.AddHandler("test", consumer.NewHandlerProto(
		func(_ context.Context, msg *NonEvent, _ consumer.Message) error { return nil }))
	if !errors.Is(err, consumer.ErrEmptyTopic) {
		if err == nil {
			return errors.New("should be error")
		}
		return err
	}

	err = facade.AddHandler("test", consumer.NewHandlerProto(
		func(_ context.Context, msg *TestEvent4, _ consumer.Message) (err error) {
			wg.Done()
			onceErr.Do(func() {
				err = errors.New("err")
			})
			return err
		}))
	if err != nil {
		return err
	}

	err = facade.AddHandler("test2", consumer.NewHandlerProto(
		func(_ context.Context, msg *TestEvent5, _ consumer.Message) (err error) {
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

		cErr := facade.Stop(context.Background())
		assert.NoError(t, cErr)

		cErr = facade.Stop(context.Background())
		if !errors.Is(cErr, consumer.ErrClosed) {
			assert.NoError(t, cErr)
		}
	}()

	assert.NoError(t, facade.Run(context.Background()))

	return nil
}

func TestIntegration_Producer(t *testing.T) {
	defer cleanUp(t)

	ctx := context.Background()

	k, err := producer.New(
		producer.Config{Brokers: bootstrap})
	require.NoError(t, err)

	err = k.Send(ctx, &TestEvent1{
		Id:  uuid.NewString(),
		Num: 23423,
		Nm:  nil,
	})
	assert.NoError(t, err)

	err = k.Send(ctx, &NonEvent{
		Color: "blue",
	})
	assert.ErrorIs(t, err, producer.ErrEmptyTopic)

	err = k.SendSync(ctx, &TestEvent3{
		Id:     uuid.NewString(),
		Result: true,
	})
	assert.NoError(t, err)

	err = k.Send(ctx, &TestEvent2{
		Solution: "success",
	})
	assert.ErrorIs(t, err, producer.ErrEmptyKey)

	err = k.Close()
	assert.NoError(t, err)

	err = k.Close()
	assert.ErrorIs(t, err, producer.ErrClosed)

	err = k.Send(ctx, &TestEvent1{
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

	p, err := producer.New(producer.Config{Brokers: bootstrap})
	require.NoError(t, err)

	gCfg := consumer.NewGroupConfig()
	gCfg.Brokers = bootstrap
	gCfg.Group = "test"

	c, err := consumer.NewGroup(gCfg,
		consumer.WithEarliestOffsetReset(),
	)
	require.NoError(t, err)

	cCtx, cancel := context.WithCancel(context.Background())

	err = c.AddHandler(consumer.NewHandlerProto(
		func(_ context.Context, msg *TestEvent3, _ consumer.Message) error {
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
		err = p.Send(context.Background(), &TestEvent3{
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
		gErr := c.Stop(context.Background())
		if gErr != nil {
			panic(gErr)
		}
	}()

	assert.NoError(t, c.Run(context.Background()))
	assert.NoError(t, p.Close())
}

func TestIntegration_Producer_Bad_Marshaller(t *testing.T) {
	var (
		msg = &TestEvent1{
			Id:  uuid.NewString(),
			Num: 23423,
			Nm:  nil,
		}
	)

	k, err := producer.New(producer.Config{Brokers: bootstrap},
		producer.WithMarshaller(func(m proto.Message) ([]byte, error) {
			return nil, errors.New("err")
		}))
	require.NoError(t, err)

	assert.Error(t, k.Send(context.Background(), msg))
	assert.NoError(t, k.Close())
}

func cleanUp(t *testing.T) {
	t.Helper()

	admCli, err := sarama.NewClusterAdmin([]string{bootstrap}, sarama.NewConfig())
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

func TestIntegration_Producer_Consumer_Json(t *testing.T) {
	defer cleanUp(t)

	ctx := context.Background()

	k, err := producer.New(
		producer.Config{Brokers: bootstrap},
		producer.WithMarshaller(
			func(m proto.Message) ([]byte, error) {
				return json.Marshal(m)
			}),
	)
	require.NoError(t, err)

	err = k.Send(ctx, &TestEvent1{
		Id:  uuid.NewString(),
		Num: 23423,
		Nm:  nil,
	})
	assert.NoError(t, err)

	err = k.Close()
	assert.NoError(t, err)

	gCfg := consumer.NewGroupConfig()
	gCfg.Brokers = bootstrap
	gCfg.Group = "test"

	c, err := consumer.NewGroup(gCfg,
		consumer.WithEarliestOffsetReset(),
		consumer.WithLogger(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}))),
	)
	require.NoError(t, err)

	c.AddHandler(consumer.NewHandlerProto(
		func(ctx context.Context, msg *TestEvent1, raw consumer.Message) error {
			assert.Equal(t, msg.Num, int64(23423))
			go c.Stop(context.Background()) //nolint: contextcheck
			return nil
		}, consumer.WithHandlerUnmarshal(func(b []byte, m proto.Message) error {
			return json.Unmarshal(b, m)
		})))
	assert.NoError(t, c.Run(ctx))
}
