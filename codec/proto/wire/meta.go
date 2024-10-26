package wire

import (
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/korableg/bus/codec/proto/event"
)

var headerType = []byte("type")

type meta struct {
	topic string
	typ   string
}

func newMeta(msg proto.Message) meta {
	return meta{
		topic: topic(msg),
		typ:   string(fullName(msg)),
	}
}

func (m meta) Topic() string {
	return m.topic
}

func (m meta) Type() string {
	return m.typ
}

func fullName(msg proto.Message) protoreflect.FullName {
	return msg.ProtoReflect().Descriptor().FullName()
}

func topic(msg proto.Message) string {
	var (
		opts     = msg.ProtoReflect().Descriptor().Options()
		topicRaw = proto.GetExtension(opts, event.E_Topic)
		topic, _ = topicRaw.(string) //nolint: errcheck
	)

	return topic
}
