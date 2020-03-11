// Code generated by protoc-gen-go. DO NOT EDIT.
// source: messages.proto

package cosipbft

import (
	fmt "fmt"
	proto "github.com/golang/protobuf/proto"
	any "github.com/golang/protobuf/ptypes/any"
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

// ForwardLinkProto is the message representing a forward link between
// two proposals. It contains both hash and the prepare and commit
// signatures.
type ForwardLinkProto struct {
	From                 []byte   `protobuf:"bytes,1,opt,name=from,proto3" json:"from,omitempty"`
	To                   []byte   `protobuf:"bytes,2,opt,name=to,proto3" json:"to,omitempty"`
	Prepare              *any.Any `protobuf:"bytes,3,opt,name=prepare,proto3" json:"prepare,omitempty"`
	Commit               *any.Any `protobuf:"bytes,4,opt,name=commit,proto3" json:"commit,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *ForwardLinkProto) Reset()         { *m = ForwardLinkProto{} }
func (m *ForwardLinkProto) String() string { return proto.CompactTextString(m) }
func (*ForwardLinkProto) ProtoMessage()    {}
func (*ForwardLinkProto) Descriptor() ([]byte, []int) {
	return fileDescriptor_4dc296cbfe5ffcd5, []int{0}
}

func (m *ForwardLinkProto) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ForwardLinkProto.Unmarshal(m, b)
}
func (m *ForwardLinkProto) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ForwardLinkProto.Marshal(b, m, deterministic)
}
func (m *ForwardLinkProto) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ForwardLinkProto.Merge(m, src)
}
func (m *ForwardLinkProto) XXX_Size() int {
	return xxx_messageInfo_ForwardLinkProto.Size(m)
}
func (m *ForwardLinkProto) XXX_DiscardUnknown() {
	xxx_messageInfo_ForwardLinkProto.DiscardUnknown(m)
}

var xxx_messageInfo_ForwardLinkProto proto.InternalMessageInfo

func (m *ForwardLinkProto) GetFrom() []byte {
	if m != nil {
		return m.From
	}
	return nil
}

func (m *ForwardLinkProto) GetTo() []byte {
	if m != nil {
		return m.To
	}
	return nil
}

func (m *ForwardLinkProto) GetPrepare() *any.Any {
	if m != nil {
		return m.Prepare
	}
	return nil
}

func (m *ForwardLinkProto) GetCommit() *any.Any {
	if m != nil {
		return m.Commit
	}
	return nil
}

// ChainProto is the message representing a list of forward links that creates
// a verifiable chain.
type ChainProto struct {
	Links                []*ForwardLinkProto `protobuf:"bytes,1,rep,name=links,proto3" json:"links,omitempty"`
	XXX_NoUnkeyedLiteral struct{}            `json:"-"`
	XXX_unrecognized     []byte              `json:"-"`
	XXX_sizecache        int32               `json:"-"`
}

func (m *ChainProto) Reset()         { *m = ChainProto{} }
func (m *ChainProto) String() string { return proto.CompactTextString(m) }
func (*ChainProto) ProtoMessage()    {}
func (*ChainProto) Descriptor() ([]byte, []int) {
	return fileDescriptor_4dc296cbfe5ffcd5, []int{1}
}

func (m *ChainProto) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ChainProto.Unmarshal(m, b)
}
func (m *ChainProto) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ChainProto.Marshal(b, m, deterministic)
}
func (m *ChainProto) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ChainProto.Merge(m, src)
}
func (m *ChainProto) XXX_Size() int {
	return xxx_messageInfo_ChainProto.Size(m)
}
func (m *ChainProto) XXX_DiscardUnknown() {
	xxx_messageInfo_ChainProto.DiscardUnknown(m)
}

var xxx_messageInfo_ChainProto proto.InternalMessageInfo

func (m *ChainProto) GetLinks() []*ForwardLinkProto {
	if m != nil {
		return m.Links
	}
	return nil
}

// Prepare is the message sent to start a consensus for a proposal.
type Prepare struct {
	Proposal             *any.Any `protobuf:"bytes,1,opt,name=proposal,proto3" json:"proposal,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *Prepare) Reset()         { *m = Prepare{} }
func (m *Prepare) String() string { return proto.CompactTextString(m) }
func (*Prepare) ProtoMessage()    {}
func (*Prepare) Descriptor() ([]byte, []int) {
	return fileDescriptor_4dc296cbfe5ffcd5, []int{2}
}

func (m *Prepare) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Prepare.Unmarshal(m, b)
}
func (m *Prepare) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Prepare.Marshal(b, m, deterministic)
}
func (m *Prepare) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Prepare.Merge(m, src)
}
func (m *Prepare) XXX_Size() int {
	return xxx_messageInfo_Prepare.Size(m)
}
func (m *Prepare) XXX_DiscardUnknown() {
	xxx_messageInfo_Prepare.DiscardUnknown(m)
}

var xxx_messageInfo_Prepare proto.InternalMessageInfo

func (m *Prepare) GetProposal() *any.Any {
	if m != nil {
		return m.Proposal
	}
	return nil
}

// Commit is the message sent to commit to a proposal.
type Commit struct {
	To                   []byte   `protobuf:"bytes,1,opt,name=to,proto3" json:"to,omitempty"`
	Prepare              *any.Any `protobuf:"bytes,2,opt,name=prepare,proto3" json:"prepare,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *Commit) Reset()         { *m = Commit{} }
func (m *Commit) String() string { return proto.CompactTextString(m) }
func (*Commit) ProtoMessage()    {}
func (*Commit) Descriptor() ([]byte, []int) {
	return fileDescriptor_4dc296cbfe5ffcd5, []int{3}
}

func (m *Commit) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Commit.Unmarshal(m, b)
}
func (m *Commit) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Commit.Marshal(b, m, deterministic)
}
func (m *Commit) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Commit.Merge(m, src)
}
func (m *Commit) XXX_Size() int {
	return xxx_messageInfo_Commit.Size(m)
}
func (m *Commit) XXX_DiscardUnknown() {
	xxx_messageInfo_Commit.DiscardUnknown(m)
}

var xxx_messageInfo_Commit proto.InternalMessageInfo

func (m *Commit) GetTo() []byte {
	if m != nil {
		return m.To
	}
	return nil
}

func (m *Commit) GetPrepare() *any.Any {
	if m != nil {
		return m.Prepare
	}
	return nil
}

// Propagate is the last message of a consensus process to send the valid
// forward link to participants.
type Propagate struct {
	To                   []byte   `protobuf:"bytes,1,opt,name=to,proto3" json:"to,omitempty"`
	Commit               *any.Any `protobuf:"bytes,2,opt,name=commit,proto3" json:"commit,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *Propagate) Reset()         { *m = Propagate{} }
func (m *Propagate) String() string { return proto.CompactTextString(m) }
func (*Propagate) ProtoMessage()    {}
func (*Propagate) Descriptor() ([]byte, []int) {
	return fileDescriptor_4dc296cbfe5ffcd5, []int{4}
}

func (m *Propagate) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Propagate.Unmarshal(m, b)
}
func (m *Propagate) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Propagate.Marshal(b, m, deterministic)
}
func (m *Propagate) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Propagate.Merge(m, src)
}
func (m *Propagate) XXX_Size() int {
	return xxx_messageInfo_Propagate.Size(m)
}
func (m *Propagate) XXX_DiscardUnknown() {
	xxx_messageInfo_Propagate.DiscardUnknown(m)
}

var xxx_messageInfo_Propagate proto.InternalMessageInfo

func (m *Propagate) GetTo() []byte {
	if m != nil {
		return m.To
	}
	return nil
}

func (m *Propagate) GetCommit() *any.Any {
	if m != nil {
		return m.Commit
	}
	return nil
}

func init() {
	proto.RegisterType((*ForwardLinkProto)(nil), "cosipbft.ForwardLinkProto")
	proto.RegisterType((*ChainProto)(nil), "cosipbft.ChainProto")
	proto.RegisterType((*Prepare)(nil), "cosipbft.Prepare")
	proto.RegisterType((*Commit)(nil), "cosipbft.Commit")
	proto.RegisterType((*Propagate)(nil), "cosipbft.Propagate")
}

func init() { proto.RegisterFile("messages.proto", fileDescriptor_4dc296cbfe5ffcd5) }

var fileDescriptor_4dc296cbfe5ffcd5 = []byte{
	// 271 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x8c, 0x90, 0xcf, 0x4b, 0xc3, 0x30,
	0x14, 0xc7, 0x49, 0x37, 0xb7, 0xf9, 0x36, 0x86, 0x14, 0x0f, 0x71, 0xa7, 0x92, 0x53, 0x0f, 0x92,
	0x8d, 0x79, 0x1c, 0x08, 0x3a, 0x10, 0x05, 0x0f, 0xa5, 0x47, 0x6f, 0xe9, 0x4c, 0x6b, 0x58, 0x9b,
	0x17, 0x92, 0x88, 0xec, 0xff, 0xf0, 0x0f, 0x16, 0xd2, 0x56, 0xc1, 0xdf, 0xb7, 0xf0, 0xf8, 0xe4,
	0x7d, 0xbf, 0x9f, 0x07, 0xf3, 0x46, 0x3a, 0x27, 0x2a, 0xe9, 0xb8, 0xb1, 0xe8, 0x31, 0x9e, 0xec,
	0xd0, 0x29, 0x53, 0x94, 0x7e, 0x71, 0x56, 0x21, 0x56, 0xb5, 0x5c, 0x86, 0x79, 0xf1, 0x5c, 0x2e,
	0x85, 0x3e, 0xb4, 0x10, 0x7b, 0x25, 0x70, 0x72, 0x83, 0xf6, 0x45, 0xd8, 0xc7, 0x7b, 0xa5, 0xf7,
	0x59, 0xf8, 0x19, 0xc3, 0xb0, 0xb4, 0xd8, 0x50, 0x92, 0x90, 0x74, 0x96, 0x87, 0x77, 0x3c, 0x87,
	0xc8, 0x23, 0x8d, 0xc2, 0x24, 0xf2, 0x18, 0x73, 0x18, 0x1b, 0x2b, 0x8d, 0xb0, 0x92, 0x0e, 0x12,
	0x92, 0x4e, 0xd7, 0xa7, 0xbc, 0x4d, 0xe1, 0x7d, 0x0a, 0xbf, 0xd2, 0x87, 0xbc, 0x87, 0xe2, 0x73,
	0x18, 0xed, 0xb0, 0x69, 0x94, 0xa7, 0xc3, 0x5f, 0xf0, 0x8e, 0x61, 0x97, 0x00, 0xdb, 0x27, 0xa1,
	0x74, 0xdb, 0x67, 0x05, 0x47, 0xb5, 0xd2, 0x7b, 0x47, 0x49, 0x32, 0x48, 0xa7, 0xeb, 0x05, 0xef,
	0xcd, 0xf8, 0xe7, 0xea, 0x79, 0x0b, 0xb2, 0x0d, 0x8c, 0xb3, 0x2e, 0x78, 0x05, 0x13, 0x63, 0xd1,
	0xa0, 0x13, 0x75, 0x10, 0xfa, 0x29, 0xfa, 0x9d, 0x62, 0xb7, 0x30, 0xda, 0x86, 0x1a, 0x9d, 0x34,
	0xf9, 0x4e, 0x3a, 0xfa, 0x87, 0x34, 0xbb, 0x83, 0xe3, 0xcc, 0xa2, 0x11, 0x95, 0xf0, 0xf2, 0xcb,
	0xb2, 0x8f, 0x8b, 0x44, 0x7f, 0x5f, 0xe4, 0x7a, 0xf6, 0x00, 0x7c, 0xd3, 0x7b, 0x17, 0xa3, 0xc0,
	0x5c, 0xbc, 0x05, 0x00, 0x00, 0xff, 0xff, 0xdb, 0x25, 0x17, 0xb2, 0xf4, 0x01, 0x00, 0x00,
}