// Code generated by protoc-gen-go. DO NOT EDIT.
// source: messages.proto

package darc

import (
	fmt "fmt"
	proto "github.com/golang/protobuf/proto"
	math "math"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion3 // please upgrade the proto package

type Expression struct {
	Matches              []string `protobuf:"bytes,1,rep,name=matches,proto3" json:"matches,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *Expression) Reset()         { *m = Expression{} }
func (m *Expression) String() string { return proto.CompactTextString(m) }
func (*Expression) ProtoMessage()    {}
func (*Expression) Descriptor() ([]byte, []int) {
	return fileDescriptor_4dc296cbfe5ffcd5, []int{0}
}

func (m *Expression) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Expression.Unmarshal(m, b)
}
func (m *Expression) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Expression.Marshal(b, m, deterministic)
}
func (m *Expression) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Expression.Merge(m, src)
}
func (m *Expression) XXX_Size() int {
	return xxx_messageInfo_Expression.Size(m)
}
func (m *Expression) XXX_DiscardUnknown() {
	xxx_messageInfo_Expression.DiscardUnknown(m)
}

var xxx_messageInfo_Expression proto.InternalMessageInfo

func (m *Expression) GetMatches() []string {
	if m != nil {
		return m.Matches
	}
	return nil
}

type AccessProto struct {
	Rules                map[string]*Expression `protobuf:"bytes,1,rep,name=rules,proto3" json:"rules,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	XXX_NoUnkeyedLiteral struct{}               `json:"-"`
	XXX_unrecognized     []byte                 `json:"-"`
	XXX_sizecache        int32                  `json:"-"`
}

func (m *AccessProto) Reset()         { *m = AccessProto{} }
func (m *AccessProto) String() string { return proto.CompactTextString(m) }
func (*AccessProto) ProtoMessage()    {}
func (*AccessProto) Descriptor() ([]byte, []int) {
	return fileDescriptor_4dc296cbfe5ffcd5, []int{1}
}

func (m *AccessProto) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_AccessProto.Unmarshal(m, b)
}
func (m *AccessProto) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_AccessProto.Marshal(b, m, deterministic)
}
func (m *AccessProto) XXX_Merge(src proto.Message) {
	xxx_messageInfo_AccessProto.Merge(m, src)
}
func (m *AccessProto) XXX_Size() int {
	return xxx_messageInfo_AccessProto.Size(m)
}
func (m *AccessProto) XXX_DiscardUnknown() {
	xxx_messageInfo_AccessProto.DiscardUnknown(m)
}

var xxx_messageInfo_AccessProto proto.InternalMessageInfo

func (m *AccessProto) GetRules() map[string]*Expression {
	if m != nil {
		return m.Rules
	}
	return nil
}

type Task struct {
	Key                  []byte       `protobuf:"bytes,1,opt,name=key,proto3" json:"key,omitempty"`
	Access               *AccessProto `protobuf:"bytes,2,opt,name=access,proto3" json:"access,omitempty"`
	XXX_NoUnkeyedLiteral struct{}     `json:"-"`
	XXX_unrecognized     []byte       `json:"-"`
	XXX_sizecache        int32        `json:"-"`
}

func (m *Task) Reset()         { *m = Task{} }
func (m *Task) String() string { return proto.CompactTextString(m) }
func (*Task) ProtoMessage()    {}
func (*Task) Descriptor() ([]byte, []int) {
	return fileDescriptor_4dc296cbfe5ffcd5, []int{2}
}

func (m *Task) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Task.Unmarshal(m, b)
}
func (m *Task) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Task.Marshal(b, m, deterministic)
}
func (m *Task) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Task.Merge(m, src)
}
func (m *Task) XXX_Size() int {
	return xxx_messageInfo_Task.Size(m)
}
func (m *Task) XXX_DiscardUnknown() {
	xxx_messageInfo_Task.DiscardUnknown(m)
}

var xxx_messageInfo_Task proto.InternalMessageInfo

func (m *Task) GetKey() []byte {
	if m != nil {
		return m.Key
	}
	return nil
}

func (m *Task) GetAccess() *AccessProto {
	if m != nil {
		return m.Access
	}
	return nil
}

func init() {
	proto.RegisterType((*Expression)(nil), "darc.Expression")
	proto.RegisterType((*AccessProto)(nil), "darc.AccessProto")
	proto.RegisterMapType((map[string]*Expression)(nil), "darc.AccessProto.RulesEntry")
	proto.RegisterType((*Task)(nil), "darc.Task")
}

func init() {
	proto.RegisterFile("messages.proto", fileDescriptor_4dc296cbfe5ffcd5)
}

var fileDescriptor_4dc296cbfe5ffcd5 = []byte{
	// 206 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0xe2, 0xcb, 0x4d, 0x2d, 0x2e,
	0x4e, 0x4c, 0x4f, 0x2d, 0xd6, 0x2b, 0x28, 0xca, 0x2f, 0xc9, 0x17, 0x62, 0x49, 0x49, 0x2c, 0x4a,
	0x56, 0x52, 0xe3, 0xe2, 0x72, 0xad, 0x28, 0x28, 0x4a, 0x2d, 0x2e, 0xce, 0xcc, 0xcf, 0x13, 0x92,
	0xe0, 0x62, 0xcf, 0x4d, 0x2c, 0x49, 0xce, 0x48, 0x2d, 0x96, 0x60, 0x54, 0x60, 0xd6, 0xe0, 0x0c,
	0x82, 0x71, 0x95, 0x7a, 0x19, 0xb9, 0xb8, 0x1d, 0x93, 0x93, 0x53, 0x8b, 0x8b, 0x03, 0xc0, 0xba,
	0x8d, 0xb8, 0x58, 0x8b, 0x4a, 0x73, 0xa0, 0xea, 0xb8, 0x8d, 0x64, 0xf4, 0x40, 0xa6, 0xe9, 0x21,
	0xa9, 0xd0, 0x0b, 0x02, 0x49, 0xbb, 0xe6, 0x95, 0x14, 0x55, 0x06, 0x41, 0x94, 0x4a, 0x79, 0x71,
	0x71, 0x21, 0x04, 0x85, 0x04, 0xb8, 0x98, 0xb3, 0x53, 0x2b, 0x25, 0x18, 0x15, 0x18, 0x35, 0x38,
	0x83, 0x40, 0x4c, 0x21, 0x35, 0x2e, 0xd6, 0xb2, 0xc4, 0x9c, 0xd2, 0x54, 0x09, 0x26, 0x05, 0x46,
	0x0d, 0x6e, 0x23, 0x01, 0x88, 0x99, 0x08, 0xe7, 0x05, 0x41, 0xa4, 0xad, 0x98, 0x2c, 0x18, 0x95,
	0x9c, 0xb9, 0x58, 0x42, 0x12, 0x8b, 0xb3, 0x91, 0x4d, 0xe1, 0x81, 0x98, 0xa2, 0xc9, 0xc5, 0x96,
	0x08, 0x76, 0x06, 0xd4, 0x18, 0x41, 0x0c, 0xa7, 0x05, 0x41, 0x15, 0x24, 0xb1, 0x81, 0x43, 0xc2,
	0x18, 0x10, 0x00, 0x00, 0xff, 0xff, 0x8d, 0x34, 0x9a, 0x08, 0x1b, 0x01, 0x00, 0x00,
}