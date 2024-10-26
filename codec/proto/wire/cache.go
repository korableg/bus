package wire

import (
	"sync"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/korableg/bus/codec/proto/event"
)

var (
	keyDescr   = make(map[protoreflect.FullName]protoreflect.FieldDescriptor)
	keyDescrMu sync.RWMutex
)

func keyField(msg proto.Message) protoreflect.FieldDescriptor {
	var fn = fullName(msg)

	keyDescrMu.RLock()
	field, ok := keyDescr[fn]
	keyDescrMu.RUnlock()

	if ok {
		return field
	}

	field = keyFieldLong(msg)

	keyDescrMu.Lock()
	keyDescr[fn] = field
	keyDescrMu.Unlock()

	return field
}

func keyFieldLong(msg proto.Message) protoreflect.FieldDescriptor {
	if msg == nil {
		return nil
	}

	var (
		fields  = msg.ProtoReflect().Descriptor().Fields()
		field   protoreflect.FieldDescriptor
		keyExt  any
		key, ok bool
	)

	for i := 0; i < fields.Len(); i++ {
		field = fields.Get(i)
		keyExt = proto.GetExtension(field.Options(), event.E_Key)

		if key, ok = keyExt.(bool); ok && key {
			return field
		}
	}

	return nil
}
