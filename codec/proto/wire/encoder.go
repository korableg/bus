package wire

import (
	"google.golang.org/protobuf/proto"

	"github.com/korableg/bus/codec"
)

type encoder struct {
	meta
	msg proto.Message
}

func (e encoder) Marshal() ([]byte, error) {
	return proto.Marshal(e.msg)
}

func (e encoder) Key() []byte {
	var field = keyField(e.msg)
	if field == nil {
		return nil
	}

	return []byte(e.msg.ProtoReflect().Get(field).String())
}

func (e encoder) Headers() []codec.Header {
	if e.msg == nil {
		return nil
	}

	return []codec.Header{
		{
			Key:   headerType,
			Value: []byte(fullName(e.msg)),
		},
	}
}

func Encoder() func(msg proto.Message) codec.Encoder[proto.Message] {
	return func(msg proto.Message) codec.Encoder[proto.Message] {
		return encoder{msg: msg, meta: newMeta(msg)}
	}
}
