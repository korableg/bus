// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.34.2
// 	protoc        v5.26.1
// source: test/test.proto

package test

import (
	_ "github.com/korableg/bus/codec/proto/event"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type NestedMsg struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Name     string `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	Duration uint64 `protobuf:"varint,2,opt,name=duration,proto3" json:"duration,omitempty"`
}

func (x *NestedMsg) Reset() {
	*x = NestedMsg{}
	if protoimpl.UnsafeEnabled {
		mi := &file_test_test_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *NestedMsg) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*NestedMsg) ProtoMessage() {}

func (x *NestedMsg) ProtoReflect() protoreflect.Message {
	mi := &file_test_test_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use NestedMsg.ProtoReflect.Descriptor instead.
func (*NestedMsg) Descriptor() ([]byte, []int) {
	return file_test_test_proto_rawDescGZIP(), []int{0}
}

func (x *NestedMsg) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *NestedMsg) GetDuration() uint64 {
	if x != nil {
		return x.Duration
	}
	return 0
}

type TestEvent1 struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Id  string     `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	Num int64      `protobuf:"varint,2,opt,name=num,proto3" json:"num,omitempty"`
	Nm  *NestedMsg `protobuf:"bytes,3,opt,name=nm,proto3" json:"nm,omitempty"`
}

func (x *TestEvent1) Reset() {
	*x = TestEvent1{}
	if protoimpl.UnsafeEnabled {
		mi := &file_test_test_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *TestEvent1) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*TestEvent1) ProtoMessage() {}

func (x *TestEvent1) ProtoReflect() protoreflect.Message {
	mi := &file_test_test_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use TestEvent1.ProtoReflect.Descriptor instead.
func (*TestEvent1) Descriptor() ([]byte, []int) {
	return file_test_test_proto_rawDescGZIP(), []int{1}
}

func (x *TestEvent1) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *TestEvent1) GetNum() int64 {
	if x != nil {
		return x.Num
	}
	return 0
}

func (x *TestEvent1) GetNm() *NestedMsg {
	if x != nil {
		return x.Nm
	}
	return nil
}

type TestEvent2 struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Solution string `protobuf:"bytes,1,opt,name=solution,proto3" json:"solution,omitempty"`
}

func (x *TestEvent2) Reset() {
	*x = TestEvent2{}
	if protoimpl.UnsafeEnabled {
		mi := &file_test_test_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *TestEvent2) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*TestEvent2) ProtoMessage() {}

func (x *TestEvent2) ProtoReflect() protoreflect.Message {
	mi := &file_test_test_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use TestEvent2.ProtoReflect.Descriptor instead.
func (*TestEvent2) Descriptor() ([]byte, []int) {
	return file_test_test_proto_rawDescGZIP(), []int{2}
}

func (x *TestEvent2) GetSolution() string {
	if x != nil {
		return x.Solution
	}
	return ""
}

type TestEvent3 struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Id     string `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	Result bool   `protobuf:"varint,2,opt,name=result,proto3" json:"result,omitempty"`
}

func (x *TestEvent3) Reset() {
	*x = TestEvent3{}
	if protoimpl.UnsafeEnabled {
		mi := &file_test_test_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *TestEvent3) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*TestEvent3) ProtoMessage() {}

func (x *TestEvent3) ProtoReflect() protoreflect.Message {
	mi := &file_test_test_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use TestEvent3.ProtoReflect.Descriptor instead.
func (*TestEvent3) Descriptor() ([]byte, []int) {
	return file_test_test_proto_rawDescGZIP(), []int{3}
}

func (x *TestEvent3) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *TestEvent3) GetResult() bool {
	if x != nil {
		return x.Result
	}
	return false
}

type TestEvent4 struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Id       int32  `protobuf:"varint,1,opt,name=id,proto3" json:"id,omitempty"`
	Instance string `protobuf:"bytes,2,opt,name=instance,proto3" json:"instance,omitempty"`
}

func (x *TestEvent4) Reset() {
	*x = TestEvent4{}
	if protoimpl.UnsafeEnabled {
		mi := &file_test_test_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *TestEvent4) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*TestEvent4) ProtoMessage() {}

func (x *TestEvent4) ProtoReflect() protoreflect.Message {
	mi := &file_test_test_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use TestEvent4.ProtoReflect.Descriptor instead.
func (*TestEvent4) Descriptor() ([]byte, []int) {
	return file_test_test_proto_rawDescGZIP(), []int{4}
}

func (x *TestEvent4) GetId() int32 {
	if x != nil {
		return x.Id
	}
	return 0
}

func (x *TestEvent4) GetInstance() string {
	if x != nil {
		return x.Instance
	}
	return ""
}

type TestEvent5 struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Id       int32  `protobuf:"varint,1,opt,name=id,proto3" json:"id,omitempty"`
	Division string `protobuf:"bytes,2,opt,name=division,proto3" json:"division,omitempty"`
	Office   uint64 `protobuf:"varint,3,opt,name=office,proto3" json:"office,omitempty"`
}

func (x *TestEvent5) Reset() {
	*x = TestEvent5{}
	if protoimpl.UnsafeEnabled {
		mi := &file_test_test_proto_msgTypes[5]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *TestEvent5) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*TestEvent5) ProtoMessage() {}

func (x *TestEvent5) ProtoReflect() protoreflect.Message {
	mi := &file_test_test_proto_msgTypes[5]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use TestEvent5.ProtoReflect.Descriptor instead.
func (*TestEvent5) Descriptor() ([]byte, []int) {
	return file_test_test_proto_rawDescGZIP(), []int{5}
}

func (x *TestEvent5) GetId() int32 {
	if x != nil {
		return x.Id
	}
	return 0
}

func (x *TestEvent5) GetDivision() string {
	if x != nil {
		return x.Division
	}
	return ""
}

func (x *TestEvent5) GetOffice() uint64 {
	if x != nil {
		return x.Office
	}
	return 0
}

type NonEvent struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Color string `protobuf:"bytes,1,opt,name=color,proto3" json:"color,omitempty"`
}

func (x *NonEvent) Reset() {
	*x = NonEvent{}
	if protoimpl.UnsafeEnabled {
		mi := &file_test_test_proto_msgTypes[6]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *NonEvent) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*NonEvent) ProtoMessage() {}

func (x *NonEvent) ProtoReflect() protoreflect.Message {
	mi := &file_test_test_proto_msgTypes[6]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use NonEvent.ProtoReflect.Descriptor instead.
func (*NonEvent) Descriptor() ([]byte, []int) {
	return file_test_test_proto_rawDescGZIP(), []int{6}
}

func (x *NonEvent) GetColor() string {
	if x != nil {
		return x.Color
	}
	return ""
}

var File_test_test_proto protoreflect.FileDescriptor

var file_test_test_proto_rawDesc = []byte{
	0x0a, 0x0f, 0x74, 0x65, 0x73, 0x74, 0x2f, 0x74, 0x65, 0x73, 0x74, 0x2e, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x12, 0x05, 0x65, 0x76, 0x65, 0x6e, 0x74, 0x1a, 0x1d, 0x63, 0x6f, 0x64, 0x65, 0x63, 0x2f,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x65, 0x76, 0x65, 0x6e, 0x74, 0x2f, 0x65, 0x76, 0x65, 0x6e,
	0x74, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x3b, 0x0a, 0x09, 0x4e, 0x65, 0x73, 0x74, 0x65,
	0x64, 0x4d, 0x73, 0x67, 0x12, 0x12, 0x0a, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x01, 0x20, 0x01,
	0x28, 0x09, 0x52, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x12, 0x1a, 0x0a, 0x08, 0x64, 0x75, 0x72, 0x61,
	0x74, 0x69, 0x6f, 0x6e, 0x18, 0x02, 0x20, 0x01, 0x28, 0x04, 0x52, 0x08, 0x64, 0x75, 0x72, 0x61,
	0x74, 0x69, 0x6f, 0x6e, 0x22, 0x64, 0x0a, 0x0a, 0x54, 0x65, 0x73, 0x74, 0x45, 0x76, 0x65, 0x6e,
	0x74, 0x31, 0x12, 0x13, 0x0a, 0x02, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x42, 0x03,
	0x80, 0x7d, 0x01, 0x52, 0x02, 0x69, 0x64, 0x12, 0x10, 0x0a, 0x03, 0x6e, 0x75, 0x6d, 0x18, 0x02,
	0x20, 0x01, 0x28, 0x03, 0x52, 0x03, 0x6e, 0x75, 0x6d, 0x12, 0x20, 0x0a, 0x02, 0x6e, 0x6d, 0x18,
	0x03, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x10, 0x2e, 0x65, 0x76, 0x65, 0x6e, 0x74, 0x2e, 0x4e, 0x65,
	0x73, 0x74, 0x65, 0x64, 0x4d, 0x73, 0x67, 0x52, 0x02, 0x6e, 0x6d, 0x3a, 0x0d, 0xc2, 0x3e, 0x0a,
	0x74, 0x65, 0x73, 0x74, 0x5f, 0x74, 0x6f, 0x70, 0x69, 0x63, 0x22, 0x37, 0x0a, 0x0a, 0x54, 0x65,
	0x73, 0x74, 0x45, 0x76, 0x65, 0x6e, 0x74, 0x32, 0x12, 0x1a, 0x0a, 0x08, 0x73, 0x6f, 0x6c, 0x75,
	0x74, 0x69, 0x6f, 0x6e, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x08, 0x73, 0x6f, 0x6c, 0x75,
	0x74, 0x69, 0x6f, 0x6e, 0x3a, 0x0d, 0xc2, 0x3e, 0x0a, 0x74, 0x65, 0x73, 0x74, 0x5f, 0x74, 0x6f,
	0x70, 0x69, 0x63, 0x22, 0x48, 0x0a, 0x0a, 0x54, 0x65, 0x73, 0x74, 0x45, 0x76, 0x65, 0x6e, 0x74,
	0x33, 0x12, 0x13, 0x0a, 0x02, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x42, 0x03, 0x80,
	0x7d, 0x01, 0x52, 0x02, 0x69, 0x64, 0x12, 0x16, 0x0a, 0x06, 0x72, 0x65, 0x73, 0x75, 0x6c, 0x74,
	0x18, 0x02, 0x20, 0x01, 0x28, 0x08, 0x52, 0x06, 0x72, 0x65, 0x73, 0x75, 0x6c, 0x74, 0x3a, 0x0d,
	0xc2, 0x3e, 0x0a, 0x74, 0x65, 0x73, 0x74, 0x5f, 0x74, 0x6f, 0x70, 0x69, 0x63, 0x22, 0x4e, 0x0a,
	0x0a, 0x54, 0x65, 0x73, 0x74, 0x45, 0x76, 0x65, 0x6e, 0x74, 0x34, 0x12, 0x13, 0x0a, 0x02, 0x69,
	0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x05, 0x42, 0x03, 0x80, 0x7d, 0x01, 0x52, 0x02, 0x69, 0x64,
	0x12, 0x1a, 0x0a, 0x08, 0x69, 0x6e, 0x73, 0x74, 0x61, 0x6e, 0x63, 0x65, 0x18, 0x02, 0x20, 0x01,
	0x28, 0x09, 0x52, 0x08, 0x69, 0x6e, 0x73, 0x74, 0x61, 0x6e, 0x63, 0x65, 0x3a, 0x0f, 0xc2, 0x3e,
	0x0c, 0x74, 0x65, 0x73, 0x74, 0x5f, 0x74, 0x6f, 0x70, 0x69, 0x63, 0x5f, 0x32, 0x22, 0x66, 0x0a,
	0x0a, 0x54, 0x65, 0x73, 0x74, 0x45, 0x76, 0x65, 0x6e, 0x74, 0x35, 0x12, 0x13, 0x0a, 0x02, 0x69,
	0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x05, 0x42, 0x03, 0x80, 0x7d, 0x01, 0x52, 0x02, 0x69, 0x64,
	0x12, 0x1a, 0x0a, 0x08, 0x64, 0x69, 0x76, 0x69, 0x73, 0x69, 0x6f, 0x6e, 0x18, 0x02, 0x20, 0x01,
	0x28, 0x09, 0x52, 0x08, 0x64, 0x69, 0x76, 0x69, 0x73, 0x69, 0x6f, 0x6e, 0x12, 0x16, 0x0a, 0x06,
	0x6f, 0x66, 0x66, 0x69, 0x63, 0x65, 0x18, 0x03, 0x20, 0x01, 0x28, 0x04, 0x52, 0x06, 0x6f, 0x66,
	0x66, 0x69, 0x63, 0x65, 0x3a, 0x0f, 0xc2, 0x3e, 0x0c, 0x74, 0x65, 0x73, 0x74, 0x5f, 0x74, 0x6f,
	0x70, 0x69, 0x63, 0x5f, 0x32, 0x22, 0x20, 0x0a, 0x08, 0x4e, 0x6f, 0x6e, 0x45, 0x76, 0x65, 0x6e,
	0x74, 0x12, 0x14, 0x0a, 0x05, 0x63, 0x6f, 0x6c, 0x6f, 0x72, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09,
	0x52, 0x05, 0x63, 0x6f, 0x6c, 0x6f, 0x72, 0x42, 0x1e, 0x5a, 0x1c, 0x67, 0x69, 0x74, 0x68, 0x75,
	0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x6b, 0x6f, 0x72, 0x61, 0x62, 0x6c, 0x65, 0x67, 0x2f, 0x62,
	0x75, 0x73, 0x2f, 0x74, 0x65, 0x73, 0x74, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_test_test_proto_rawDescOnce sync.Once
	file_test_test_proto_rawDescData = file_test_test_proto_rawDesc
)

func file_test_test_proto_rawDescGZIP() []byte {
	file_test_test_proto_rawDescOnce.Do(func() {
		file_test_test_proto_rawDescData = protoimpl.X.CompressGZIP(file_test_test_proto_rawDescData)
	})
	return file_test_test_proto_rawDescData
}

var file_test_test_proto_msgTypes = make([]protoimpl.MessageInfo, 7)
var file_test_test_proto_goTypes = []any{
	(*NestedMsg)(nil),  // 0: event.NestedMsg
	(*TestEvent1)(nil), // 1: event.TestEvent1
	(*TestEvent2)(nil), // 2: event.TestEvent2
	(*TestEvent3)(nil), // 3: event.TestEvent3
	(*TestEvent4)(nil), // 4: event.TestEvent4
	(*TestEvent5)(nil), // 5: event.TestEvent5
	(*NonEvent)(nil),   // 6: event.NonEvent
}
var file_test_test_proto_depIdxs = []int32{
	0, // 0: event.TestEvent1.nm:type_name -> event.NestedMsg
	1, // [1:1] is the sub-list for method output_type
	1, // [1:1] is the sub-list for method input_type
	1, // [1:1] is the sub-list for extension type_name
	1, // [1:1] is the sub-list for extension extendee
	0, // [0:1] is the sub-list for field type_name
}

func init() { file_test_test_proto_init() }
func file_test_test_proto_init() {
	if File_test_test_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_test_test_proto_msgTypes[0].Exporter = func(v any, i int) any {
			switch v := v.(*NestedMsg); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_test_test_proto_msgTypes[1].Exporter = func(v any, i int) any {
			switch v := v.(*TestEvent1); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_test_test_proto_msgTypes[2].Exporter = func(v any, i int) any {
			switch v := v.(*TestEvent2); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_test_test_proto_msgTypes[3].Exporter = func(v any, i int) any {
			switch v := v.(*TestEvent3); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_test_test_proto_msgTypes[4].Exporter = func(v any, i int) any {
			switch v := v.(*TestEvent4); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_test_test_proto_msgTypes[5].Exporter = func(v any, i int) any {
			switch v := v.(*TestEvent5); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_test_test_proto_msgTypes[6].Exporter = func(v any, i int) any {
			switch v := v.(*NonEvent); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_test_test_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   7,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_test_test_proto_goTypes,
		DependencyIndexes: file_test_test_proto_depIdxs,
		MessageInfos:      file_test_test_proto_msgTypes,
	}.Build()
	File_test_test_proto = out.File
	file_test_test_proto_rawDesc = nil
	file_test_test_proto_goTypes = nil
	file_test_test_proto_depIdxs = nil
}