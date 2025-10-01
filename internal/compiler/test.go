package compiler

import (
	"encoding/base64"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/runtime/protoimpl"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
)

type User struct {
	state             protoimpl.MessageState `protogen:"open.v1"`
	id                *int64                 `protobuf:"varint,1,opt,name=id,proto3,oneof" json:"id"`
	first_name        string                 `protobuf:"bytes,2,opt,name=first_name,proto3" json:"first_name"`
	last_name         string                 `protobuf:"bytes,3,opt,name=last_name,proto3" json:"last_name"`
	email             string                 `protobuf:"bytes,4,opt,name=email,proto3" json:"email"`
	phone             string                 `protobuf:"bytes,5,opt,name=phone,proto3" json:"phone"`
	emergency_contact *string                `protobuf:"bytes,6,opt,name=emergency_contact,proto3,oneof" json:"emergency_contact"`
	date_of_birth     string                 `protobuf:"bytes,7,opt,name=date_of_birth,proto3" json:"date_of_birth"`
	profile_picture   *string                `protobuf:"bytes,8,opt,name=profile_picture,proto3,oneof" json:"profile_picture"`
	password          string                 `protobuf:"bytes,9,opt,name=password,proto3" json:"password"`
	unknownFields     protoimpl.UnknownFields
	sizeCache         protoimpl.SizeCache
}

func (x *User) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*User) ProtoMessage() {}

func (x *User) ProtoReflect() protoreflect.Message {
	dec, err := base64.StdEncoding.DecodeString("eyJuYW1lIjoiVXNlciIsImZpZWxkIjpbeyJuYW1lIjoiaWQiLCJudW1iZXIiOjEsImxhYmVsIjoiTEFCRUxfT1BUSU9OQUwiLCJ0eXBlIjoiVFlQRV9JTlQ2NCIsIm9uZW9mSW5kZXgiOjAsImpzb25OYW1lIjoiaWQiLCJwcm90bzNPcHRpb25hbCI6dHJ1ZX0seyJuYW1lIjoiZmlyc3RfbmFtZSIsIm51bWJlciI6MiwibGFiZWwiOiJMQUJFTF9PUFRJT05BTCIsInR5cGUiOiJUWVBFX1NUUklORyIsImpzb25OYW1lIjoiZmlyc3RfbmFtZSJ9LHsibmFtZSI6Imxhc3RfbmFtZSIsIm51bWJlciI6MywibGFiZWwiOiJMQUJFTF9PUFRJT05BTCIsInR5cGUiOiJUWVBFX1NUUklORyIsImpzb25OYW1lIjoibGFzdF9uYW1lIn0seyJuYW1lIjoiZW1haWwiLCJudW1iZXIiOjQsImxhYmVsIjoiTEFCRUxfT1BUSU9OQUwiLCJ0eXBlIjoiVFlQRV9TVFJJTkciLCJqc29uTmFtZSI6ImVtYWlsIn0seyJuYW1lIjoicGhvbmUiLCJudW1iZXIiOjUsImxhYmVsIjoiTEFCRUxfT1BUSU9OQUwiLCJ0eXBlIjoiVFlQRV9TVFJJTkciLCJqc29uTmFtZSI6InBob25lIn0seyJuYW1lIjoiZW1lcmdlbmN5X2NvbnRhY3QiLCJudW1iZXIiOjYsImxhYmVsIjoiTEFCRUxfT1BUSU9OQUwiLCJ0eXBlIjoiVFlQRV9TVFJJTkciLCJvbmVvZkluZGV4IjoxLCJqc29uTmFtZSI6ImVtZXJnZW5jeV9jb250YWN0IiwicHJvdG8zT3B0aW9uYWwiOnRydWV9LHsibmFtZSI6ImRhdGVfb2ZfYmlydGgiLCJudW1iZXIiOjcsImxhYmVsIjoiTEFCRUxfT1BUSU9OQUwiLCJ0eXBlIjoiVFlQRV9TVFJJTkciLCJqc29uTmFtZSI6ImRhdGVfb2ZfYmlydGgifSx7Im5hbWUiOiJwcm9maWxlX3BpY3R1cmUiLCJudW1iZXIiOjgsImxhYmVsIjoiTEFCRUxfT1BUSU9OQUwiLCJ0eXBlIjoiVFlQRV9TVFJJTkciLCJvbmVvZkluZGV4IjoyLCJqc29uTmFtZSI6InByb2ZpbGVfcGljdHVyZSIsInByb3RvM09wdGlvbmFsIjp0cnVlfSx7Im5hbWUiOiJwYXNzd29yZCIsIm51bWJlciI6OSwibGFiZWwiOiJMQUJFTF9PUFRJT05BTCIsInR5cGUiOiJUWVBFX1NUUklORyIsImpzb25OYW1lIjoicGFzc3dvcmQifV0sIm9uZW9mRGVjbCI6W3sibmFtZSI6Il9pZCJ9LHsibmFtZSI6Il9lbWVyZ2VuY3lfY29udGFjdCJ9LHsibmFtZSI6Il9wcm9maWxlX3BpY3R1cmUifV19")
	if err != nil {
		panic(err)
	}
	md := new(descriptorpb.DescriptorProto)
	if err := protojson.Unmarshal(dec, md); err != nil {
		panic(err)
	}
	zz := dynamicpb.NewMessage(md.ProtoReflect().Descriptor())
	return zz
}

func (x *User) Marshal() ([]byte, error) {
	return proto.Marshal(x)
}

func (x *User) Unmarshal(bytes []byte) error {
	return proto.Unmarshal(bytes, x)
}

type PostReq struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	user          User                   `protobuf:"bytes,1,opt,name=user,proto3" json:"user"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *PostReq) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*PostReq) ProtoMessage() {}

func (x *PostReq) ProtoReflect() protoreflect.Message {
	dec, err := base64.StdEncoding.DecodeString("eyJuYW1lIjoiUG9zdFJlcSIsImZpZWxkIjpbeyJuYW1lIjoidXNlciIsIm51bWJlciI6MSwibGFiZWwiOiJMQUJFTF9PUFRJT05BTCIsInR5cGUiOiJUWVBFX01FU1NBR0UiLCJ0eXBlTmFtZSI6Ii5kYXNoYm9hcmQudXNlcnMuVXNlciIsImpzb25OYW1lIjoidXNlciJ9XX0=")
	if err != nil {
		panic(err)
	}
	md := new(descriptorpb.DescriptorProto)
	if err := protojson.Unmarshal(dec, md); err != nil {
		panic(err)
	}
	return md.ProtoReflect()
}

func (x *PostReq) Marshal() ([]byte, error) {
	return proto.Marshal(x)
}

func (x *PostReq) Unmarshal(bytes []byte) error {
	return proto.Unmarshal(bytes, x)
}

type PostRes struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	id            int64                  `protobuf:"varint,1,opt,name=id,proto3" json:"id"`
	first_name    string                 `protobuf:"bytes,2,opt,name=first_name,proto3" json:"first_name"`
	last_name     string                 `protobuf:"bytes,3,opt,name=last_name,proto3" json:"last_name"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *PostRes) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*PostRes) ProtoMessage() {}

func (x *PostRes) ProtoReflect() protoreflect.Message {
	dec, err := base64.StdEncoding.DecodeString("eyJuYW1lIjoiUG9zdFJlcyIsImZpZWxkIjpbeyJuYW1lIjoiaWQiLCJudW1iZXIiOjEsImxhYmVsIjoiTEFCRUxfT1BUSU9OQUwiLCJ0eXBlIjoiVFlQRV9JTlQ2NCIsImpzb25OYW1lIjoiaWQifSx7Im5hbWUiOiJmaXJzdF9uYW1lIiwibnVtYmVyIjoyLCJsYWJlbCI6IkxBQkVMX09QVElPTkFMIiwidHlwZSI6IlRZUEVfU1RSSU5HIiwianNvbk5hbWUiOiJmaXJzdF9uYW1lIn0seyJuYW1lIjoibGFzdF9uYW1lIiwibnVtYmVyIjozLCJsYWJlbCI6IkxBQkVMX09QVElPTkFMIiwidHlwZSI6IlRZUEVfU1RSSU5HIiwianNvbk5hbWUiOiJsYXN0X25hbWUifV19")
	if err != nil {
		panic(err)
	}
	md := new(descriptorpb.DescriptorProto)
	if err := protojson.Unmarshal(dec, md); err != nil {
		panic(err)
	}
	return md.ProtoReflect()
}

func (x *PostRes) Marshal() ([]byte, error) {
	return proto.Marshal(x)
}

func (x *PostRes) Unmarshal(bytes []byte) error {
	return proto.Unmarshal(bytes, x)
}

type GetReq struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	id            int64                  `protobuf:"varint,1,opt,name=id,proto3" json:"id"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *GetReq) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetReq) ProtoMessage() {}

func (x *GetReq) ProtoReflect() protoreflect.Message {
	dec, err := base64.StdEncoding.DecodeString("eyJuYW1lIjoiR2V0UmVxIiwiZmllbGQiOlt7Im5hbWUiOiJpZCIsIm51bWJlciI6MSwibGFiZWwiOiJMQUJFTF9PUFRJT05BTCIsInR5cGUiOiJUWVBFX0lOVDY0IiwianNvbk5hbWUiOiJpZCJ9XX0=")
	if err != nil {
		panic(err)
	}
	md := new(descriptorpb.DescriptorProto)
	if err := protojson.Unmarshal(dec, md); err != nil {
		panic(err)
	}
	return md.ProtoReflect()
}

func (x *GetReq) Marshal() ([]byte, error) {
	return proto.Marshal(x)
}

func (x *GetReq) Unmarshal(bytes []byte) error {
	return proto.Unmarshal(bytes, x)
}

type GetRes struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	user          User                   `protobuf:"bytes,1,opt,name=user,proto3" json:"user"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *GetRes) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetRes) ProtoMessage() {}

func (x *GetRes) ProtoReflect() protoreflect.Message {
	dec, err := base64.StdEncoding.DecodeString("eyJuYW1lIjoiR2V0UmVzIiwiZmllbGQiOlt7Im5hbWUiOiJ1c2VyIiwibnVtYmVyIjoxLCJsYWJlbCI6IkxBQkVMX09QVElPTkFMIiwidHlwZSI6IlRZUEVfTUVTU0FHRSIsInR5cGVOYW1lIjoiLmRhc2hib2FyZC51c2Vycy5Vc2VyIiwianNvbk5hbWUiOiJ1c2VyIn1dfQ==")
	if err != nil {
		panic(err)
	}
	md := new(descriptorpb.DescriptorProto)
	if err := protojson.Unmarshal(dec, md); err != nil {
		panic(err)
	}
	return md.ProtoReflect()
}

func (x *GetRes) Marshal() ([]byte, error) {
	return proto.Marshal(x)
}

func (x *GetRes) Unmarshal(bytes []byte) error {
	return proto.Unmarshal(bytes, x)
}

type GetByEmailReq struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	email         string                 `protobuf:"bytes,1,opt,name=email,proto3" json:"email"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *GetByEmailReq) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetByEmailReq) ProtoMessage() {}

func (x *GetByEmailReq) ProtoReflect() protoreflect.Message {
	dec, err := base64.StdEncoding.DecodeString("eyJuYW1lIjoiR2V0QnlFbWFpbFJlcSIsImZpZWxkIjpbeyJuYW1lIjoiZW1haWwiLCJudW1iZXIiOjEsImxhYmVsIjoiTEFCRUxfT1BUSU9OQUwiLCJ0eXBlIjoiVFlQRV9TVFJJTkciLCJqc29uTmFtZSI6ImVtYWlsIn1dfQ==")
	if err != nil {
		panic(err)
	}
	md := new(descriptorpb.DescriptorProto)
	if err := protojson.Unmarshal(dec, md); err != nil {
		panic(err)
	}
	return md.ProtoReflect()
}

func (x *GetByEmailReq) Marshal() ([]byte, error) {
	return proto.Marshal(x)
}

func (x *GetByEmailReq) Unmarshal(bytes []byte) error {
	return proto.Unmarshal(bytes, x)
}

type GetByEmailRes struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	user          User                   `protobuf:"bytes,1,opt,name=user,proto3" json:"user"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *GetByEmailRes) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetByEmailRes) ProtoMessage() {}

func (x *GetByEmailRes) ProtoReflect() protoreflect.Message {
	dec, err := base64.StdEncoding.DecodeString("eyJuYW1lIjoiR2V0QnlFbWFpbFJlcyIsImZpZWxkIjpbeyJuYW1lIjoidXNlciIsIm51bWJlciI6MSwibGFiZWwiOiJMQUJFTF9PUFRJT05BTCIsInR5cGUiOiJUWVBFX01FU1NBR0UiLCJ0eXBlTmFtZSI6Ii5kYXNoYm9hcmQudXNlcnMuVXNlciIsImpzb25OYW1lIjoidXNlciJ9XX0=")
	if err != nil {
		panic(err)
	}
	md := new(descriptorpb.DescriptorProto)
	if err := protojson.Unmarshal(dec, md); err != nil {
		panic(err)
	}
	return md.ProtoReflect()
}

func (x *GetByEmailRes) Marshal() ([]byte, error) {
	return proto.Marshal(x)
}

func (x *GetByEmailRes) Unmarshal(bytes []byte) error {
	return proto.Unmarshal(bytes, x)
}
