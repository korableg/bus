package wire

import (
	"bytes"
	"errors"
	"fmt"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"

	"github.com/korableg/bus/codec"
)

type decoder[T proto.Message] struct {
	meta
}

func (d *decoder[T]) Unmarshal(data []byte, headers []codec.Header) (msg T, err error) {
	var (
		fn string
		ok bool
	)
	for _, h := range headers {
		if bytes.Equal(headerType, h.Key) {
			fn = string(h.Value)
			break
		}
	}

	if fn != d.Type() {
		return msg, codec.ErrWrongType
	}

	msgType, err := protoregistry.GlobalTypes.FindMessageByName(protoreflect.FullName(fn))
	if err != nil {
		return msg, fmt.Errorf("can't resolve message type: %w", err)
	}

	protoMsg := msgType.New().Interface()

	err = proto.Unmarshal(data, protoMsg)
	if err != nil {
		return msg, fmt.Errorf("unmarshal message: %w", err)
	}

	msg, ok = protoMsg.(T)
	if !ok {
		return msg, errors.New("casting message type error")
	}

	return msg, nil
}

func Decoder[T proto.Message]() codec.Decoder[T] {
	var zero T
	return &decoder[T]{
		meta: newMeta(zero),
	}
}
