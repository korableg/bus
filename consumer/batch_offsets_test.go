package consumer

import (
	"maps"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

type offsetReports struct {
	reported []Offset
}

func (r *offsetReports) report(o Offset) {
	r.reported = append(r.reported, maps.Clone(o))
}

func TestUnit_batchOffsets_add(t *testing.T) {
	t.Parallel()

	var (
		r = &offsetReports{}
		b = newBatchOffsets(r.report)
	)

	b.add(0, 5)
	b.add(0, 6)
	b.add(1, 3)

	assert.Empty(t, r.reported, "the offsets are held back till the flush")
	assert.Equal(t, Offset{0: 6, 1: 3}, b.pending)
}

func TestUnit_batchOffsets_skip(t *testing.T) {
	t.Parallel()

	t.Run("partition without buffered messages", func(t *testing.T) {
		t.Parallel()

		var (
			r = &offsetReports{}
			b = newBatchOffsets(r.report)
		)

		b.add(0, 5)
		b.skip(1, 3)

		assert.Equal(t, []Offset{{1: 3}}, r.reported)
	})

	t.Run("partition with buffered messages", func(t *testing.T) {
		t.Parallel()

		var (
			r = &offsetReports{}
			b = newBatchOffsets(r.report)
		)

		b.add(0, 5)
		b.skip(0, 7)

		assert.Empty(t, r.reported)
		assert.Equal(t, Offset{0: 7}, b.pending)
	})
}

func TestUnit_batchOffsets_flush(t *testing.T) {
	t.Parallel()

	t.Run("reports held back offsets", func(t *testing.T) {
		t.Parallel()

		var (
			r = &offsetReports{}
			b = newBatchOffsets(r.report)
		)

		b.add(0, 5)
		b.skip(0, 7)
		b.flush()

		assert.Equal(t, []Offset{{0: 7}}, r.reported)

		b.skip(0, 8)
		assert.Equal(t, []Offset{{0: 7}, {0: 8}}, r.reported, "after the flush a skip is reported at once")
	})

	t.Run("reported offset never goes back", func(t *testing.T) {
		t.Parallel()

		const count = 10000

		var (
			last int64 = -1
			back bool
			b    = newBatchOffsets(func(o Offset) {
				if off, ok := o[0]; ok {
					back = back || off < last
					last = off
				}
			})
			stop = make(chan struct{})
			wg   sync.WaitGroup
		)

		wg.Go(func() {
			for {
				select {
				case <-stop:
					return
				default:
					b.flush()
				}
			}
		})

		for i := range int64(count) {
			if i%2 == 0 {
				b.add(0, i)
			} else {
				b.skip(0, i)
			}
		}

		close(stop)
		wg.Wait()
		b.flush()

		assert.False(t, back, "the offset of the partition must never move back")
		assert.EqualValues(t, count-1, last)
	})
}
