package bus

import (
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/korableg/bus/event"
)

// GetTopic message topic getter
func GetTopic(msg proto.Message) string {
	if msg == nil {
		return ""
	}

	var (
		opts     = msg.ProtoReflect().Descriptor().Options()
		topicRaw = proto.GetExtension(opts, event.E_Topic)
	)

	if topic, ok := topicRaw.(string); ok {
		return topic
	}

	return ""
}

// GetFullName message fullname getter
func GetFullName(msg proto.Message) protoreflect.FullName {
	if msg == nil {
		return ""
	}

	return msg.ProtoReflect().Type().Descriptor().FullName()
}
