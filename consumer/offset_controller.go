package consumer

import (
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/IBM/sarama"
)

type offsetController struct {
	handlers *handlerCollection

	commitCount    int64
	commitDuration time.Duration

	msgs atomic.Int64

	commitSig chan struct{}

	logger *slog.Logger
}

func newOffsetController(handlers *handlerCollection, o Options) *offsetController {
	return &offsetController{
		handlers: handlers,

		commitCount:    int64(o.CommitCount),
		commitDuration: o.CommitDuration,

		commitSig: make(chan struct{}, 1),

		logger: o.GetLogger(),
	}
}

func (o *offsetController) Inc() {
	if o.commitCount > 0 && o.msgs.Add(1)%o.commitCount == 0 {
		o.sendSig()
	}
}

func (o *offsetController) Mark(sess sarama.ConsumerGroupSession) (marked bool) {
	for topic, parts := range sess.Claims() {
		offset := o.getOffset(topic, parts...)
		for part, off := range offset {
			sess.MarkOffset(topic, part, off+1, "ok")
			incCommitted(topic, part)
			o.logger.Debug("mark offset", "topic", topic, "partition", part, "offset", off)
			marked = true
		}
	}

	return marked
}

func (o *offsetController) StartCommitting(sess sarama.ConsumerGroupSession) {
	if o.commitDuration > 0 {
		go o.startDurationCommitting(sess)
	}

	for {
		select {
		case <-o.commitSig:
			if o.Mark(sess) {
				sess.Commit()
			}
		case <-sess.Context().Done():
			return
		}
	}
}

func (o *offsetController) startDurationCommitting(sess sarama.ConsumerGroupSession) {
	ticker := time.NewTicker(o.commitDuration)
	for {
		select {
		case <-ticker.C:
			o.sendSig()
		case <-sess.Context().Done():
			ticker.Stop()
			return
		}
	}
}

func (o *offsetController) getOffset(topic string, partitions ...int32) Offset {
	var (
		dstOffset  = make(Offset)
		srcOffsets = o.handlers.Offsets(topic)
		srcOff     int64
		dstOff     int64
	)

	if len(srcOffsets) == 0 {
		return nil
	}

	for _, part := range partitions {
		dstOff = dstOffset[part]

		for _, src := range srcOffsets {
			srcOff = src[part]

			if srcOff == 0 {
				delete(dstOffset, part)
				break
			}

			if dstOff == 0 || dstOff > srcOff {
				dstOffset[part] = srcOff
				dstOff = srcOff
			}
		}
	}

	if len(dstOffset) == 0 {
		return nil
	}

	return dstOffset
}

func (o *offsetController) sendSig() {
	select {
	case o.commitSig <- struct{}{}:
	default:
	}
}
