package consumer

import (
	"maps"
	"slices"
)

type handlerController map[string][]Handler

func (hc handlerController) Add(h Handler) {
	hc[h.Topic()] = append(hc[h.Topic()], h)
}

func (hc handlerController) Topics() (topics []string) {
	topics = make([]string, 0, len(hc))

	slices.AppendSeq(topics, maps.Keys(hc))

	return topics
}

func (hc handlerController) Handlers(topic string) []Handler {
	src := hc[topic]
	if len(src) == 0 {
		return nil
	}

	return slices.Clone(src)
}

func (hc handlerController) Offset(topic string, partitions ...int32) OffsetSeq {
	return func(yield func(int32, int64) bool) {
		var srcOff, dstOff int64

		for _, part := range partitions {
			dstOff = 0

			for _, h := range hc[topic] {
				srcOff = h.Offset(part)

				if srcOff <= 0 {
					dstOff = 0
					break
				}

				if dstOff == 0 || dstOff > srcOff {
					dstOff = srcOff
				}
			}

			if dstOff == 0 {
				continue
			}

			if !yield(part, dstOff) {
				break
			}
		}
	}
}
