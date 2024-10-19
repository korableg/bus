package wire

import (
	"github.com/korableg/bus/codec"
	"github.com/korableg/bus/codec/proto/event"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

var headerFullName = []byte("full-name")

type meta struct {
	msg proto.Message
}

func (m meta) Topic() string {
	if m.msg == nil {
		return ""
	}

	var (
		opts     = m.msg.ProtoReflect().Descriptor().Options()
		topicRaw = proto.GetExtension(opts, event.E_Topic)
		topic, _ = topicRaw.(string)
	)

	return topic
}

func (m meta) Key() []byte {
	var field = keyField(m.msg)
	if field == nil {
		return nil
	}

	return []byte(m.msg.ProtoReflect().Get(field).String())
}

func (m meta) Headers() []codec.Header {
	if m.msg == nil {
		return nil
	}

	return []codec.Header{
		{
			Key:   headerFullName,
			Value: []byte(fullName(m.msg)),
		},
	}
}

func (m meta) Type() string {
	if m.msg == nil {
		return ""
	}

	return string(fullName(m.msg))
}

func fullName(msg proto.Message) protoreflect.FullName {
	if msg == nil {
		return ""
	}

	return msg.ProtoReflect().Descriptor().FullName()
}
