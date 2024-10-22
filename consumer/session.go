package consumer

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/IBM/sarama"

	"github.com/korableg/bus/codec"
)

type session struct {
	oCtl *offsetController
	hCtl *handlerController

	group string

	logger     *slog.Logger
	traceIDCtx TraceIDCtxFunc
}

func newSession(hCtl *handlerController, oCtl *offsetController, group string, logger *slog.Logger, traceIDCtx TraceIDCtxFunc) *session {
	return &session{
		oCtl:  oCtl,
		hCtl:  hCtl,
		group: group,

		logger:     logger,
		traceIDCtx: traceIDCtx,
	}
}

func (s *session) Setup(sess sarama.ConsumerGroupSession) error {
	go s.oCtl.StartCommitting(sess)

	for topic, partitions := range sess.Claims() {
		s.logger.Info(fmt.Sprintf("rebalance group %s: assigned topic: %s, partitions: %v",
			s.group, topic, partitions))
	}

	return nil
}

func (s *session) Cleanup(sess sarama.ConsumerGroupSession) error {
	s.oCtl.Mark(sess)

	sess.Commit()

	for topic, partitions := range sess.Claims() {
		s.logger.Info(fmt.Sprintf("rebalance group %s: revoked topic: %s, partitions: %v",
			s.group, topic, partitions))
	}

	return nil
}

func (s *session) ConsumeClaim(sess sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	var handler = s.handlerFunc(sess.Context(), claim.Topic())

	for {
		select {
		case kafkaMsg, ok := <-claim.Messages():
			if !ok {
				return nil
			}

			handler(kafkaMsg)

			s.oCtl.Inc()
		case <-sess.Context().Done():
			// s.logger.Info(fmt.Sprintf("Consume by group %s was stopped", s.group), "err", err)
			return nil
		}
	}
}

func (s *session) handlerFunc(ctx context.Context, topic string) func(*sarama.ConsumerMessage) {
	var handlers = s.hCtl.Handlers(topic)

	return func(kafkaMsg *sarama.ConsumerMessage) {
		raw := Message{
			Group:     s.group,
			Key:       kafkaMsg.Key,
			Value:     kafkaMsg.Value,
			Topic:     kafkaMsg.Topic,
			Partition: kafkaMsg.Partition,
			Offset:    kafkaMsg.Offset,
			Timestamp: kafkaMsg.Timestamp,
			Headers:   make([]codec.Header, 0, len(kafkaMsg.Headers)),
		}

		for _, h := range kafkaMsg.Headers {
			switch string(h.Key) {
			case codec.HeaderTraceID:
				raw.TraceID = string(h.Value)
			case codec.HeaderMessageID:
				raw.ID = string(h.Value)
			}

			raw.Headers = append(raw.Headers, codec.Header{
				Key:   h.Key,
				Value: h.Value,
			})
		}

		hCtx, hCancel := context.WithCancel(ctx)
		if raw.TraceID != "" {
			hCtx = s.traceIDCtx(hCtx, raw.TraceID)
		}

		for _, h := range handlers {
			h.Handle(hCtx, raw)
		}

		hCancel()
	}
}
