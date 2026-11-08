package consumer

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func offsetsOf(values map[int32]int64) map[int32]*atomic.Int64 {
	offsets := make(map[int32]*atomic.Int64, len(values))

	for part, off := range values {
		offsets[part] = &atomic.Int64{}
		offsets[part].Store(off)
	}

	return offsets
}

func TestUnit_handlerController_Add(t *testing.T) {
	t.Parallel()

	var (
		h1   = newOffsetHandler(t, "topic", nil)
		h2   = newOffsetHandler(t, "topic", nil)
		h3   = newOffsetHandler(t, "topic2", nil)
		hCtl = make(handlerController)
	)

	hCtl.Add(h1)
	hCtl.Add(h2)
	hCtl.Add(h3)

	assert.Equal(t, handlerController{"topic": {h1, h2}, "topic2": {h3}}, hCtl)
}

func TestUnit_handlerController_Topics(t *testing.T) {
	t.Parallel()

	hCtl := make(handlerController)
	assert.Empty(t, hCtl.Topics())

	hCtl.Add(newOffsetHandler(t, "topic", nil))
	hCtl.Add(newOffsetHandler(t, "topic", nil))
	hCtl.Add(newOffsetHandler(t, "topic2", nil))

	assert.ElementsMatch(t, []string{"topic", "topic2"}, hCtl.Topics())
}

func TestUnit_handlerController_Handlers(t *testing.T) {
	t.Parallel()

	var (
		h1   = newOffsetHandler(t, "topic", nil)
		h2   = newOffsetHandler(t, "topic", nil)
		hCtl = make(handlerController)
	)

	hCtl.Add(h1)
	hCtl.Add(h2)

	got := hCtl.Handlers("topic")
	assert.Equal(t, []Handler{h1, h2}, got)

	got[0] = nil
	assert.Equal(t, []Handler{h1, h2}, hCtl.Handlers("topic"), "the result is a copy")

	assert.Nil(t, hCtl.Handlers("unknown"))
}

func TestUnit_handlerController_Offset(t *testing.T) {
	t.Parallel()

	collect := func(seq OffsetSeq) map[int32]int64 {
		got := map[int32]int64{}
		for part, off := range seq {
			got[part] = off
		}

		return got
	}

	tests := []struct {
		name       string
		handlers   []map[int32]int64
		partitions []int32
		want       map[int32]int64
	}{
		{
			name:       "one handler",
			handlers:   []map[int32]int64{{0: 5, 1: 7}},
			partitions: []int32{0, 1},
			want:       map[int32]int64{0: 5, 1: 7},
		},
		{
			name:       "minimum among handlers",
			handlers:   []map[int32]int64{{0: 5, 1: 7}, {0: 9, 1: 3}},
			partitions: []int32{0, 1},
			want:       map[int32]int64{0: 5, 1: 3},
		},
		{
			name:       "handler without offset holds partition",
			handlers:   []map[int32]int64{{0: 5, 1: 7}, {1: 8}},
			partitions: []int32{0, 1},
			want:       map[int32]int64{1: 7},
		},
		{
			name:       "only claimed partitions",
			handlers:   []map[int32]int64{{0: 5, 1: 7}},
			partitions: []int32{1},
			want:       map[int32]int64{1: 7},
		},
		{
			name:       "no handlers",
			partitions: []int32{0},
			want:       map[int32]int64{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			hCtl := make(handlerController)
			for _, offsets := range tt.handlers {
				hCtl.Add(newOffsetHandler(t, "topic", offsetsOf(offsets)))
			}

			assert.Equal(t, tt.want, collect(hCtl.Offset("topic", tt.partitions...)))
		})
	}

	t.Run("stops on break", func(t *testing.T) {
		t.Parallel()

		hCtl := make(handlerController)
		hCtl.Add(newOffsetHandler(t, "topic", offsetsOf(map[int32]int64{0: 5, 1: 7})))

		var visited []int32
		for part := range hCtl.Offset("topic", 0, 1) {
			visited = append(visited, part)
			break
		}

		assert.Equal(t, []int32{0}, visited)
	})

	t.Run("handlers skipping other types", func(t *testing.T) {
		t.Parallel()

		var (
			batch = NewHandlerBatch(newDecoder(t, decodePrefix("b")),
				func(context.Context, []string, []Message) error { return nil },
				WithoutHandlerRetry(), WithHandlerBatchSize(10), WithHandlerBatchFlushTimeout(time.Hour))
			proto = NewHandler(newDecoder(t, decodePrefix("p")),
				func(context.Context, string, Message) error { return nil },
				WithoutHandlerRetry())
			hCtl = make(handlerController)
		)

		hCtl.Add(batch)
		hCtl.Add(proto)

		for off := range int64(100) {
			for _, h := range hCtl.Handlers("topic") {
				h.Handle(context.Background(), msgOf(0, off, "p"))
			}
		}

		assert.Equal(t, map[int32]int64{0: 99}, collect(hCtl.Offset("topic", 0)),
			"a partition with the messages of another type for the batch handler must be committed")
	})
}
