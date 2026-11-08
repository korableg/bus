package wire

import (
	"google.golang.org/protobuf/proto"

	"github.com/korableg/bus/codec"
)

type encoder struct {
	meta
	msg proto.Message
}

// Marshal reports a typed nil pointer as codec.ErrNilMessage too.
func (e encoder) Marshal() ([]byte, error) {
	if e.msg == nil || !e.msg.ProtoReflect().IsValid() {
		return nil, codec.ErrNilMessage
	}

	return proto.Marshal(e.msg)
}

func (e encoder) Key() []byte {
	if e.msg == nil {
		return nil
	}

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

// Encoder returns the encoder constructor to pass to producer.New. The topic of a message
// is its topic option, the key is the value of the field with the key option (nil without one).
// A nil message is reported by Marshal as codec.ErrNilMessage.
func Encoder() func(msg proto.Message) codec.Encoder[proto.Message] {
	return func(msg proto.Message) codec.Encoder[proto.Message] {
		if msg == nil {
			return encoder{}
		}

		return encoder{msg: msg, meta: newMeta(msg)}
	}
}
