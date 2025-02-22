// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.2
// 	protoc        v5.29.3
// source: v1/crypto.proto

package v1

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

type SignedMessage struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Keyid         string                 `protobuf:"bytes,1,opt,name=keyid,proto3" json:"keyid,omitempty"`         // a unique identifier generated as the SHA256 of the public key used to sign the message.
	Payload       []byte                 `protobuf:"bytes,2,opt,name=payload,proto3" json:"payload,omitempty"`     // the payload
	Signature     []byte                 `protobuf:"bytes,3,opt,name=signature,proto3" json:"signature,omitempty"` // the signature of the payload
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *SignedMessage) Reset() {
	*x = SignedMessage{}
	mi := &file_v1_crypto_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *SignedMessage) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*SignedMessage) ProtoMessage() {}

func (x *SignedMessage) ProtoReflect() protoreflect.Message {
	mi := &file_v1_crypto_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use SignedMessage.ProtoReflect.Descriptor instead.
func (*SignedMessage) Descriptor() ([]byte, []int) {
	return file_v1_crypto_proto_rawDescGZIP(), []int{0}
}

func (x *SignedMessage) GetKeyid() string {
	if x != nil {
		return x.Keyid
	}
	return ""
}

func (x *SignedMessage) GetPayload() []byte {
	if x != nil {
		return x.Payload
	}
	return nil
}

func (x *SignedMessage) GetSignature() []byte {
	if x != nil {
		return x.Signature
	}
	return nil
}

type EncryptedMessage struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Payload       []byte                 `protobuf:"bytes,1,opt,name=payload,proto3" json:"payload,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *EncryptedMessage) Reset() {
	*x = EncryptedMessage{}
	mi := &file_v1_crypto_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *EncryptedMessage) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*EncryptedMessage) ProtoMessage() {}

func (x *EncryptedMessage) ProtoReflect() protoreflect.Message {
	mi := &file_v1_crypto_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use EncryptedMessage.ProtoReflect.Descriptor instead.
func (*EncryptedMessage) Descriptor() ([]byte, []int) {
	return file_v1_crypto_proto_rawDescGZIP(), []int{1}
}

func (x *EncryptedMessage) GetPayload() []byte {
	if x != nil {
		return x.Payload
	}
	return nil
}

type PublicKey struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Keyid         string                 `protobuf:"bytes,1,opt,name=keyid,proto3" json:"keyid,omitempty"`                     // a unique identifier generated as the SHA256 of the public key.
	Ed25519       string                 `protobuf:"bytes,2,opt,name=ed25519,json=ed25519pub,proto3" json:"ed25519,omitempty"` // base64 encoded public key
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *PublicKey) Reset() {
	*x = PublicKey{}
	mi := &file_v1_crypto_proto_msgTypes[2]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *PublicKey) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*PublicKey) ProtoMessage() {}

func (x *PublicKey) ProtoReflect() protoreflect.Message {
	mi := &file_v1_crypto_proto_msgTypes[2]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use PublicKey.ProtoReflect.Descriptor instead.
func (*PublicKey) Descriptor() ([]byte, []int) {
	return file_v1_crypto_proto_rawDescGZIP(), []int{2}
}

func (x *PublicKey) GetKeyid() string {
	if x != nil {
		return x.Keyid
	}
	return ""
}

func (x *PublicKey) GetEd25519() string {
	if x != nil {
		return x.Ed25519
	}
	return ""
}

type PrivateKey struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Keyid         string                 `protobuf:"bytes,1,opt,name=keyid,proto3" json:"keyid,omitempty"`                      // a unique identifier generated as the SHA256 of the public key.
	Ed25519       string                 `protobuf:"bytes,2,opt,name=ed25519,json=ed25519priv,proto3" json:"ed25519,omitempty"` // base64 encoded private key
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *PrivateKey) Reset() {
	*x = PrivateKey{}
	mi := &file_v1_crypto_proto_msgTypes[3]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *PrivateKey) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*PrivateKey) ProtoMessage() {}

func (x *PrivateKey) ProtoReflect() protoreflect.Message {
	mi := &file_v1_crypto_proto_msgTypes[3]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use PrivateKey.ProtoReflect.Descriptor instead.
func (*PrivateKey) Descriptor() ([]byte, []int) {
	return file_v1_crypto_proto_rawDescGZIP(), []int{3}
}

func (x *PrivateKey) GetKeyid() string {
	if x != nil {
		return x.Keyid
	}
	return ""
}

func (x *PrivateKey) GetEd25519() string {
	if x != nil {
		return x.Ed25519
	}
	return ""
}

var File_v1_crypto_proto protoreflect.FileDescriptor

var file_v1_crypto_proto_rawDesc = []byte{
	0x0a, 0x0f, 0x76, 0x31, 0x2f, 0x63, 0x72, 0x79, 0x70, 0x74, 0x6f, 0x2e, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x12, 0x02, 0x76, 0x31, 0x22, 0x5d, 0x0a, 0x0d, 0x53, 0x69, 0x67, 0x6e, 0x65, 0x64, 0x4d,
	0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x12, 0x14, 0x0a, 0x05, 0x6b, 0x65, 0x79, 0x69, 0x64, 0x18,
	0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x6b, 0x65, 0x79, 0x69, 0x64, 0x12, 0x18, 0x0a, 0x07,
	0x70, 0x61, 0x79, 0x6c, 0x6f, 0x61, 0x64, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x07, 0x70,
	0x61, 0x79, 0x6c, 0x6f, 0x61, 0x64, 0x12, 0x1c, 0x0a, 0x09, 0x73, 0x69, 0x67, 0x6e, 0x61, 0x74,
	0x75, 0x72, 0x65, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x09, 0x73, 0x69, 0x67, 0x6e, 0x61,
	0x74, 0x75, 0x72, 0x65, 0x22, 0x2c, 0x0a, 0x10, 0x45, 0x6e, 0x63, 0x72, 0x79, 0x70, 0x74, 0x65,
	0x64, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x12, 0x18, 0x0a, 0x07, 0x70, 0x61, 0x79, 0x6c,
	0x6f, 0x61, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x07, 0x70, 0x61, 0x79, 0x6c, 0x6f,
	0x61, 0x64, 0x22, 0x3e, 0x0a, 0x09, 0x50, 0x75, 0x62, 0x6c, 0x69, 0x63, 0x4b, 0x65, 0x79, 0x12,
	0x14, 0x0a, 0x05, 0x6b, 0x65, 0x79, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05,
	0x6b, 0x65, 0x79, 0x69, 0x64, 0x12, 0x1b, 0x0a, 0x07, 0x65, 0x64, 0x32, 0x35, 0x35, 0x31, 0x39,
	0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0a, 0x65, 0x64, 0x32, 0x35, 0x35, 0x31, 0x39, 0x70,
	0x75, 0x62, 0x22, 0x40, 0x0a, 0x0a, 0x50, 0x72, 0x69, 0x76, 0x61, 0x74, 0x65, 0x4b, 0x65, 0x79,
	0x12, 0x14, 0x0a, 0x05, 0x6b, 0x65, 0x79, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x05, 0x6b, 0x65, 0x79, 0x69, 0x64, 0x12, 0x1c, 0x0a, 0x07, 0x65, 0x64, 0x32, 0x35, 0x35, 0x31,
	0x39, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0b, 0x65, 0x64, 0x32, 0x35, 0x35, 0x31, 0x39,
	0x70, 0x72, 0x69, 0x76, 0x42, 0x2c, 0x5a, 0x2a, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63,
	0x6f, 0x6d, 0x2f, 0x67, 0x61, 0x72, 0x65, 0x74, 0x68, 0x67, 0x65, 0x6f, 0x72, 0x67, 0x65, 0x2f,
	0x62, 0x61, 0x63, 0x6b, 0x72, 0x65, 0x73, 0x74, 0x2f, 0x67, 0x65, 0x6e, 0x2f, 0x67, 0x6f, 0x2f,
	0x76, 0x31, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_v1_crypto_proto_rawDescOnce sync.Once
	file_v1_crypto_proto_rawDescData = file_v1_crypto_proto_rawDesc
)

func file_v1_crypto_proto_rawDescGZIP() []byte {
	file_v1_crypto_proto_rawDescOnce.Do(func() {
		file_v1_crypto_proto_rawDescData = protoimpl.X.CompressGZIP(file_v1_crypto_proto_rawDescData)
	})
	return file_v1_crypto_proto_rawDescData
}

var file_v1_crypto_proto_msgTypes = make([]protoimpl.MessageInfo, 4)
var file_v1_crypto_proto_goTypes = []any{
	(*SignedMessage)(nil),    // 0: v1.SignedMessage
	(*EncryptedMessage)(nil), // 1: v1.EncryptedMessage
	(*PublicKey)(nil),        // 2: v1.PublicKey
	(*PrivateKey)(nil),       // 3: v1.PrivateKey
}
var file_v1_crypto_proto_depIdxs = []int32{
	0, // [0:0] is the sub-list for method output_type
	0, // [0:0] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_v1_crypto_proto_init() }
func file_v1_crypto_proto_init() {
	if File_v1_crypto_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_v1_crypto_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   4,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_v1_crypto_proto_goTypes,
		DependencyIndexes: file_v1_crypto_proto_depIdxs,
		MessageInfos:      file_v1_crypto_proto_msgTypes,
	}.Build()
	File_v1_crypto_proto = out.File
	file_v1_crypto_proto_rawDesc = nil
	file_v1_crypto_proto_goTypes = nil
	file_v1_crypto_proto_depIdxs = nil
}
