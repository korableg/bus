package consumer

import (
	"slices"
	"sync"
)

type handlerController struct {
	h  map[string][]Handler
	mu sync.RWMutex
}

func newHandlerCollection() *handlerController {
	return &handlerController{
		h: make(map[string][]Handler),
	}
}

func (hc *handlerController) Add(h Handler) {
	hc.mu.Lock()
	hc.h[h.Topic()] = append(hc.h[h.Topic()], h)
	hc.mu.Unlock()
}

func (hc *handlerController) Topics() []string {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	topics := make([]string, 0, len(hc.h))
	for t := range hc.h {
		topics = append(topics, t)
	}

	return topics
}

func (hc *handlerController) Handlers(topic string) []Handler {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	src := hc.h[topic]
	if len(src) == 0 {
		return nil
	}

	return slices.Clone(src)
}

func (hc *handlerController) Offsets(topic string) []Offset {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	offsets := make([]Offset, 0, len(hc.h[topic]))
	for _, h := range hc.h[topic] {
		off := h.Offset(true)
		if off != nil {
			offsets = append(offsets, off)
		}
	}

	return offsets
}
