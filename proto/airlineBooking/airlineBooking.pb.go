// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.35.1
// 	protoc        v3.18.1
// source: airlineBooking/airlineBooking.proto

package airlineBooking

import (
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

type GetSeatsRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	TxId     string `protobuf:"bytes,1,opt,name=tx_id,json=txId,proto3" json:"tx_id,omitempty"`
	FlightId string `protobuf:"bytes,2,opt,name=flight_id,json=flightId,proto3" json:"flight_id,omitempty"`
}

func (x *GetSeatsRequest) Reset() {
	*x = GetSeatsRequest{}
	mi := &file_airlineBooking_airlineBooking_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *GetSeatsRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetSeatsRequest) ProtoMessage() {}

func (x *GetSeatsRequest) ProtoReflect() protoreflect.Message {
	mi := &file_airlineBooking_airlineBooking_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetSeatsRequest.ProtoReflect.Descriptor instead.
func (*GetSeatsRequest) Descriptor() ([]byte, []int) {
	return file_airlineBooking_airlineBooking_proto_rawDescGZIP(), []int{0}
}

func (x *GetSeatsRequest) GetTxId() string {
	if x != nil {
		return x.TxId
	}
	return ""
}

func (x *GetSeatsRequest) GetFlightId() string {
	if x != nil {
		return x.FlightId
	}
	return ""
}

type GetSeatsResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	AvailableSeats int32  `protobuf:"varint,1,opt,name=available_seats,json=availableSeats,proto3" json:"available_seats,omitempty"`
	Message        string `protobuf:"bytes,2,opt,name=message,proto3" json:"message,omitempty"`
}

func (x *GetSeatsResponse) Reset() {
	*x = GetSeatsResponse{}
	mi := &file_airlineBooking_airlineBooking_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *GetSeatsResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetSeatsResponse) ProtoMessage() {}

func (x *GetSeatsResponse) ProtoReflect() protoreflect.Message {
	mi := &file_airlineBooking_airlineBooking_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetSeatsResponse.ProtoReflect.Descriptor instead.
func (*GetSeatsResponse) Descriptor() ([]byte, []int) {
	return file_airlineBooking_airlineBooking_proto_rawDescGZIP(), []int{1}
}

func (x *GetSeatsResponse) GetAvailableSeats() int32 {
	if x != nil {
		return x.AvailableSeats
	}
	return 0
}

func (x *GetSeatsResponse) GetMessage() string {
	if x != nil {
		return x.Message
	}
	return ""
}

type BookSeatsRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	TxId      string `protobuf:"bytes,1,opt,name=tx_id,json=txId,proto3" json:"tx_id,omitempty"`
	FlightId  string `protobuf:"bytes,2,opt,name=flight_id,json=flightId,proto3" json:"flight_id,omitempty"`
	SeatCount int32  `protobuf:"varint,3,opt,name=seat_count,json=seatCount,proto3" json:"seat_count,omitempty"`
}

func (x *BookSeatsRequest) Reset() {
	*x = BookSeatsRequest{}
	mi := &file_airlineBooking_airlineBooking_proto_msgTypes[2]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *BookSeatsRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*BookSeatsRequest) ProtoMessage() {}

func (x *BookSeatsRequest) ProtoReflect() protoreflect.Message {
	mi := &file_airlineBooking_airlineBooking_proto_msgTypes[2]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use BookSeatsRequest.ProtoReflect.Descriptor instead.
func (*BookSeatsRequest) Descriptor() ([]byte, []int) {
	return file_airlineBooking_airlineBooking_proto_rawDescGZIP(), []int{2}
}

func (x *BookSeatsRequest) GetTxId() string {
	if x != nil {
		return x.TxId
	}
	return ""
}

func (x *BookSeatsRequest) GetFlightId() string {
	if x != nil {
		return x.FlightId
	}
	return ""
}

func (x *BookSeatsRequest) GetSeatCount() int32 {
	if x != nil {
		return x.SeatCount
	}
	return 0
}

type BookSeatsResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	AvailableSeats int32  `protobuf:"varint,1,opt,name=available_seats,json=availableSeats,proto3" json:"available_seats,omitempty"`
	Success        bool   `protobuf:"varint,2,opt,name=success,proto3" json:"success,omitempty"`
	Message        string `protobuf:"bytes,3,opt,name=message,proto3" json:"message,omitempty"`
}

func (x *BookSeatsResponse) Reset() {
	*x = BookSeatsResponse{}
	mi := &file_airlineBooking_airlineBooking_proto_msgTypes[3]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *BookSeatsResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*BookSeatsResponse) ProtoMessage() {}

func (x *BookSeatsResponse) ProtoReflect() protoreflect.Message {
	mi := &file_airlineBooking_airlineBooking_proto_msgTypes[3]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use BookSeatsResponse.ProtoReflect.Descriptor instead.
func (*BookSeatsResponse) Descriptor() ([]byte, []int) {
	return file_airlineBooking_airlineBooking_proto_rawDescGZIP(), []int{3}
}

func (x *BookSeatsResponse) GetAvailableSeats() int32 {
	if x != nil {
		return x.AvailableSeats
	}
	return 0
}

func (x *BookSeatsResponse) GetSuccess() bool {
	if x != nil {
		return x.Success
	}
	return false
}

func (x *BookSeatsResponse) GetMessage() string {
	if x != nil {
		return x.Message
	}
	return ""
}

var File_airlineBooking_airlineBooking_proto protoreflect.FileDescriptor

var file_airlineBooking_airlineBooking_proto_rawDesc = []byte{
	0x0a, 0x23, 0x61, 0x69, 0x72, 0x6c, 0x69, 0x6e, 0x65, 0x42, 0x6f, 0x6f, 0x6b, 0x69, 0x6e, 0x67,
	0x2f, 0x61, 0x69, 0x72, 0x6c, 0x69, 0x6e, 0x65, 0x42, 0x6f, 0x6f, 0x6b, 0x69, 0x6e, 0x67, 0x2e,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x05, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x43, 0x0a, 0x0f,
	0x47, 0x65, 0x74, 0x53, 0x65, 0x61, 0x74, 0x73, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12,
	0x13, 0x0a, 0x05, 0x74, 0x78, 0x5f, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04,
	0x74, 0x78, 0x49, 0x64, 0x12, 0x1b, 0x0a, 0x09, 0x66, 0x6c, 0x69, 0x67, 0x68, 0x74, 0x5f, 0x69,
	0x64, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x08, 0x66, 0x6c, 0x69, 0x67, 0x68, 0x74, 0x49,
	0x64, 0x22, 0x55, 0x0a, 0x10, 0x47, 0x65, 0x74, 0x53, 0x65, 0x61, 0x74, 0x73, 0x52, 0x65, 0x73,
	0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x27, 0x0a, 0x0f, 0x61, 0x76, 0x61, 0x69, 0x6c, 0x61, 0x62,
	0x6c, 0x65, 0x5f, 0x73, 0x65, 0x61, 0x74, 0x73, 0x18, 0x01, 0x20, 0x01, 0x28, 0x05, 0x52, 0x0e,
	0x61, 0x76, 0x61, 0x69, 0x6c, 0x61, 0x62, 0x6c, 0x65, 0x53, 0x65, 0x61, 0x74, 0x73, 0x12, 0x18,
	0x0a, 0x07, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x07, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x22, 0x63, 0x0a, 0x10, 0x42, 0x6f, 0x6f, 0x6b,
	0x53, 0x65, 0x61, 0x74, 0x73, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x13, 0x0a, 0x05,
	0x74, 0x78, 0x5f, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x74, 0x78, 0x49,
	0x64, 0x12, 0x1b, 0x0a, 0x09, 0x66, 0x6c, 0x69, 0x67, 0x68, 0x74, 0x5f, 0x69, 0x64, 0x18, 0x02,
	0x20, 0x01, 0x28, 0x09, 0x52, 0x08, 0x66, 0x6c, 0x69, 0x67, 0x68, 0x74, 0x49, 0x64, 0x12, 0x1d,
	0x0a, 0x0a, 0x73, 0x65, 0x61, 0x74, 0x5f, 0x63, 0x6f, 0x75, 0x6e, 0x74, 0x18, 0x03, 0x20, 0x01,
	0x28, 0x05, 0x52, 0x09, 0x73, 0x65, 0x61, 0x74, 0x43, 0x6f, 0x75, 0x6e, 0x74, 0x22, 0x70, 0x0a,
	0x11, 0x42, 0x6f, 0x6f, 0x6b, 0x53, 0x65, 0x61, 0x74, 0x73, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e,
	0x73, 0x65, 0x12, 0x27, 0x0a, 0x0f, 0x61, 0x76, 0x61, 0x69, 0x6c, 0x61, 0x62, 0x6c, 0x65, 0x5f,
	0x73, 0x65, 0x61, 0x74, 0x73, 0x18, 0x01, 0x20, 0x01, 0x28, 0x05, 0x52, 0x0e, 0x61, 0x76, 0x61,
	0x69, 0x6c, 0x61, 0x62, 0x6c, 0x65, 0x53, 0x65, 0x61, 0x74, 0x73, 0x12, 0x18, 0x0a, 0x07, 0x73,
	0x75, 0x63, 0x63, 0x65, 0x73, 0x73, 0x18, 0x02, 0x20, 0x01, 0x28, 0x08, 0x52, 0x07, 0x73, 0x75,
	0x63, 0x63, 0x65, 0x73, 0x73, 0x12, 0x18, 0x0a, 0x07, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65,
	0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x07, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x32,
	0x9f, 0x02, 0x0a, 0x0d, 0x46, 0x6c, 0x69, 0x67, 0x68, 0x74, 0x42, 0x6f, 0x6f, 0x6b, 0x69, 0x6e,
	0x67, 0x12, 0x3b, 0x0a, 0x08, 0x47, 0x65, 0x74, 0x53, 0x65, 0x61, 0x74, 0x73, 0x12, 0x16, 0x2e,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x47, 0x65, 0x74, 0x53, 0x65, 0x61, 0x74, 0x73, 0x52, 0x65,
	0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x17, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x47, 0x65,
	0x74, 0x53, 0x65, 0x61, 0x74, 0x73, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x45,
	0x0a, 0x10, 0x50, 0x72, 0x6f, 0x70, 0x6f, 0x73, 0x65, 0x42, 0x6f, 0x6f, 0x6b, 0x53, 0x65, 0x61,
	0x74, 0x73, 0x12, 0x17, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x42, 0x6f, 0x6f, 0x6b, 0x53,
	0x65, 0x61, 0x74, 0x73, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x18, 0x2e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x2e, 0x42, 0x6f, 0x6f, 0x6b, 0x53, 0x65, 0x61, 0x74, 0x73, 0x52, 0x65, 0x73,
	0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x44, 0x0a, 0x0f, 0x43, 0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x42,
	0x6f, 0x6f, 0x6b, 0x53, 0x65, 0x61, 0x74, 0x73, 0x12, 0x17, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x2e, 0x42, 0x6f, 0x6f, 0x6b, 0x53, 0x65, 0x61, 0x74, 0x73, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73,
	0x74, 0x1a, 0x18, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x42, 0x6f, 0x6f, 0x6b, 0x53, 0x65,
	0x61, 0x74, 0x73, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x44, 0x0a, 0x0f, 0x43,
	0x61, 0x6e, 0x63, 0x65, 0x6c, 0x42, 0x6f, 0x6f, 0x6b, 0x53, 0x65, 0x61, 0x74, 0x73, 0x12, 0x17,
	0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x42, 0x6f, 0x6f, 0x6b, 0x53, 0x65, 0x61, 0x74, 0x73,
	0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x18, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e,
	0x42, 0x6f, 0x6f, 0x6b, 0x53, 0x65, 0x61, 0x74, 0x73, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73,
	0x65, 0x42, 0x11, 0x5a, 0x0f, 0x2f, 0x61, 0x69, 0x72, 0x6c, 0x69, 0x6e, 0x65, 0x42, 0x6f, 0x6f,
	0x6b, 0x69, 0x6e, 0x67, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_airlineBooking_airlineBooking_proto_rawDescOnce sync.Once
	file_airlineBooking_airlineBooking_proto_rawDescData = file_airlineBooking_airlineBooking_proto_rawDesc
)

func file_airlineBooking_airlineBooking_proto_rawDescGZIP() []byte {
	file_airlineBooking_airlineBooking_proto_rawDescOnce.Do(func() {
		file_airlineBooking_airlineBooking_proto_rawDescData = protoimpl.X.CompressGZIP(file_airlineBooking_airlineBooking_proto_rawDescData)
	})
	return file_airlineBooking_airlineBooking_proto_rawDescData
}

var file_airlineBooking_airlineBooking_proto_msgTypes = make([]protoimpl.MessageInfo, 4)
var file_airlineBooking_airlineBooking_proto_goTypes = []any{
	(*GetSeatsRequest)(nil),   // 0: proto.GetSeatsRequest
	(*GetSeatsResponse)(nil),  // 1: proto.GetSeatsResponse
	(*BookSeatsRequest)(nil),  // 2: proto.BookSeatsRequest
	(*BookSeatsResponse)(nil), // 3: proto.BookSeatsResponse
}
var file_airlineBooking_airlineBooking_proto_depIdxs = []int32{
	0, // 0: proto.FlightBooking.GetSeats:input_type -> proto.GetSeatsRequest
	2, // 1: proto.FlightBooking.ProposeBookSeats:input_type -> proto.BookSeatsRequest
	2, // 2: proto.FlightBooking.CommitBookSeats:input_type -> proto.BookSeatsRequest
	2, // 3: proto.FlightBooking.CancelBookSeats:input_type -> proto.BookSeatsRequest
	1, // 4: proto.FlightBooking.GetSeats:output_type -> proto.GetSeatsResponse
	3, // 5: proto.FlightBooking.ProposeBookSeats:output_type -> proto.BookSeatsResponse
	3, // 6: proto.FlightBooking.CommitBookSeats:output_type -> proto.BookSeatsResponse
	3, // 7: proto.FlightBooking.CancelBookSeats:output_type -> proto.BookSeatsResponse
	4, // [4:8] is the sub-list for method output_type
	0, // [0:4] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_airlineBooking_airlineBooking_proto_init() }
func file_airlineBooking_airlineBooking_proto_init() {
	if File_airlineBooking_airlineBooking_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_airlineBooking_airlineBooking_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   4,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_airlineBooking_airlineBooking_proto_goTypes,
		DependencyIndexes: file_airlineBooking_airlineBooking_proto_depIdxs,
		MessageInfos:      file_airlineBooking_airlineBooking_proto_msgTypes,
	}.Build()
	File_airlineBooking_airlineBooking_proto = out.File
	file_airlineBooking_airlineBooking_proto_rawDesc = nil
	file_airlineBooking_airlineBooking_proto_goTypes = nil
	file_airlineBooking_airlineBooking_proto_depIdxs = nil
}
