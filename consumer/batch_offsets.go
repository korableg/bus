package consumer

import "sync"

// batchOffsets holds back the offsets of HandlerBatch till the flush, by partition:
// the offset of the last buffered message or of a later skipped one.
//
// It has its own mutex, not the one of the buffer, so skipping a message never waits for a flush.
// The offsets are reported under the mutex: otherwise a skip could report its offset after a flush
// has taken the pending ones, and the flush would move the offset back.
type batchOffsets struct {
	mu      sync.Mutex
	pending Offset
	report  func(Offset) // advances the offsets of the handler, it must not keep the map
}

func newBatchOffsets(report func(Offset)) *batchOffsets {
	return &batchOffsets{
		pending: make(Offset),
		report:  report,
	}
}

func (b *batchOffsets) add(partition int32, offset int64) {
	b.mu.Lock()
	b.pending[partition] = offset
	b.mu.Unlock()
}

// skip advances the offset over a message the handler doesn't handle: after the flush if the buffer
// holds messages of the partition, otherwise at once. The messages of one partition come in order,
// so the skipped offset is above the buffered ones.
func (b *batchOffsets) skip(partition int32, offset int64) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if _, ok := b.pending[partition]; ok {
		b.pending[partition] = offset
		return
	}

	b.report(Offset{partition: offset})
}

func (b *batchOffsets) flush() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.report(b.pending)
	clear(b.pending)
}
