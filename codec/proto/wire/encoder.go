package wire

import (
	"github.com/korableg/bus/codec"
	"google.golang.org/protobuf/proto"
)

type encoder struct {
	meta
}

func (e encoder) Marshal() ([]byte, error) {
	return proto.Marshal(e.msg)
}

func Encoder() func(msg proto.Message) codec.Encoder[proto.Message] {
	return func(msg proto.Message) codec.Encoder[proto.Message] {
		return encoder{meta: meta{msg: msg}}
	}
}
