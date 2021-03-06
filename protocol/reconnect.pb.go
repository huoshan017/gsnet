// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: reconnect.proto

package protocol

import (
	fmt "fmt"
	proto "github.com/golang/protobuf/proto"
	io "io"
	math "math"
	math_bits "math/bits"
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

type ReconnectSyn struct {
	CurrSessionId        uint64   `protobuf:"varint,1,opt,name=CurrSessionId,proto3" json:"CurrSessionId,omitempty"`
	PrevSessionId        uint64   `protobuf:"varint,2,opt,name=PrevSessionId,proto3" json:"PrevSessionId,omitempty"`
	SentNum              int32    `protobuf:"varint,3,opt,name=SentNum,proto3" json:"SentNum,omitempty"`
	RecvNum              int32    `protobuf:"varint,4,opt,name=RecvNum,proto3" json:"RecvNum,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *ReconnectSyn) Reset()         { *m = ReconnectSyn{} }
func (m *ReconnectSyn) String() string { return proto.CompactTextString(m) }
func (*ReconnectSyn) ProtoMessage()    {}
func (*ReconnectSyn) Descriptor() ([]byte, []int) {
	return fileDescriptor_32c32d93e3422bd1, []int{0}
}
func (m *ReconnectSyn) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *ReconnectSyn) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_ReconnectSyn.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *ReconnectSyn) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ReconnectSyn.Merge(m, src)
}
func (m *ReconnectSyn) XXX_Size() int {
	return m.Size()
}
func (m *ReconnectSyn) XXX_DiscardUnknown() {
	xxx_messageInfo_ReconnectSyn.DiscardUnknown(m)
}

var xxx_messageInfo_ReconnectSyn proto.InternalMessageInfo

func (m *ReconnectSyn) GetCurrSessionId() uint64 {
	if m != nil {
		return m.CurrSessionId
	}
	return 0
}

func (m *ReconnectSyn) GetPrevSessionId() uint64 {
	if m != nil {
		return m.PrevSessionId
	}
	return 0
}

func (m *ReconnectSyn) GetSentNum() int32 {
	if m != nil {
		return m.SentNum
	}
	return 0
}

func (m *ReconnectSyn) GetRecvNum() int32 {
	if m != nil {
		return m.RecvNum
	}
	return 0
}

type ReconnectAck struct {
	RecvNum              int32    `protobuf:"varint,1,opt,name=RecvNum,proto3" json:"RecvNum,omitempty"`
	SentNum              int32    `protobuf:"varint,2,opt,name=SentNum,proto3" json:"SentNum,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *ReconnectAck) Reset()         { *m = ReconnectAck{} }
func (m *ReconnectAck) String() string { return proto.CompactTextString(m) }
func (*ReconnectAck) ProtoMessage()    {}
func (*ReconnectAck) Descriptor() ([]byte, []int) {
	return fileDescriptor_32c32d93e3422bd1, []int{1}
}
func (m *ReconnectAck) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *ReconnectAck) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_ReconnectAck.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *ReconnectAck) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ReconnectAck.Merge(m, src)
}
func (m *ReconnectAck) XXX_Size() int {
	return m.Size()
}
func (m *ReconnectAck) XXX_DiscardUnknown() {
	xxx_messageInfo_ReconnectAck.DiscardUnknown(m)
}

var xxx_messageInfo_ReconnectAck proto.InternalMessageInfo

func (m *ReconnectAck) GetRecvNum() int32 {
	if m != nil {
		return m.RecvNum
	}
	return 0
}

func (m *ReconnectAck) GetSentNum() int32 {
	if m != nil {
		return m.SentNum
	}
	return 0
}

type ReconnectTransport struct {
	DataList             [][]byte `protobuf:"bytes,1,rep,name=DataList,proto3" json:"DataList,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *ReconnectTransport) Reset()         { *m = ReconnectTransport{} }
func (m *ReconnectTransport) String() string { return proto.CompactTextString(m) }
func (*ReconnectTransport) ProtoMessage()    {}
func (*ReconnectTransport) Descriptor() ([]byte, []int) {
	return fileDescriptor_32c32d93e3422bd1, []int{2}
}
func (m *ReconnectTransport) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *ReconnectTransport) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_ReconnectTransport.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *ReconnectTransport) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ReconnectTransport.Merge(m, src)
}
func (m *ReconnectTransport) XXX_Size() int {
	return m.Size()
}
func (m *ReconnectTransport) XXX_DiscardUnknown() {
	xxx_messageInfo_ReconnectTransport.DiscardUnknown(m)
}

var xxx_messageInfo_ReconnectTransport proto.InternalMessageInfo

func (m *ReconnectTransport) GetDataList() [][]byte {
	if m != nil {
		return m.DataList
	}
	return nil
}

type ReconnectEnd struct {
	IsSuccess            bool     `protobuf:"varint,1,opt,name=IsSuccess,proto3" json:"IsSuccess,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *ReconnectEnd) Reset()         { *m = ReconnectEnd{} }
func (m *ReconnectEnd) String() string { return proto.CompactTextString(m) }
func (*ReconnectEnd) ProtoMessage()    {}
func (*ReconnectEnd) Descriptor() ([]byte, []int) {
	return fileDescriptor_32c32d93e3422bd1, []int{3}
}
func (m *ReconnectEnd) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *ReconnectEnd) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_ReconnectEnd.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *ReconnectEnd) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ReconnectEnd.Merge(m, src)
}
func (m *ReconnectEnd) XXX_Size() int {
	return m.Size()
}
func (m *ReconnectEnd) XXX_DiscardUnknown() {
	xxx_messageInfo_ReconnectEnd.DiscardUnknown(m)
}

var xxx_messageInfo_ReconnectEnd proto.InternalMessageInfo

func (m *ReconnectEnd) GetIsSuccess() bool {
	if m != nil {
		return m.IsSuccess
	}
	return false
}

func init() {
	proto.RegisterType((*ReconnectSyn)(nil), "protocol.ReconnectSyn")
	proto.RegisterType((*ReconnectAck)(nil), "protocol.ReconnectAck")
	proto.RegisterType((*ReconnectTransport)(nil), "protocol.ReconnectTransport")
	proto.RegisterType((*ReconnectEnd)(nil), "protocol.ReconnectEnd")
}

func init() { proto.RegisterFile("reconnect.proto", fileDescriptor_32c32d93e3422bd1) }

var fileDescriptor_32c32d93e3422bd1 = []byte{
	// 241 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0xe2, 0x2f, 0x4a, 0x4d, 0xce,
	0xcf, 0xcb, 0x4b, 0x4d, 0x2e, 0xd1, 0x2b, 0x28, 0xca, 0x2f, 0xc9, 0x17, 0xe2, 0x00, 0x53, 0xc9,
	0xf9, 0x39, 0x4a, 0x7d, 0x8c, 0x5c, 0x3c, 0x41, 0x30, 0xd9, 0xe0, 0xca, 0x3c, 0x21, 0x15, 0x2e,
	0x5e, 0xe7, 0xd2, 0xa2, 0xa2, 0xe0, 0xd4, 0xe2, 0xe2, 0xcc, 0xfc, 0x3c, 0xcf, 0x14, 0x09, 0x46,
	0x05, 0x46, 0x0d, 0x96, 0x20, 0x54, 0x41, 0x90, 0xaa, 0x80, 0xa2, 0xd4, 0x32, 0x84, 0x2a, 0x26,
	0x88, 0x2a, 0x14, 0x41, 0x21, 0x09, 0x2e, 0xf6, 0xe0, 0xd4, 0xbc, 0x12, 0xbf, 0xd2, 0x5c, 0x09,
	0x66, 0x05, 0x46, 0x0d, 0xd6, 0x20, 0x18, 0x17, 0x24, 0x13, 0x94, 0x9a, 0x5c, 0x06, 0x92, 0x61,
	0x81, 0xc8, 0x40, 0xb9, 0x4a, 0x4e, 0x48, 0xee, 0x71, 0x4c, 0xce, 0x46, 0x56, 0xc9, 0x88, 0xa2,
	0x12, 0xd9, 0x74, 0x26, 0x14, 0xd3, 0x95, 0x0c, 0xb8, 0x84, 0xe0, 0x66, 0x84, 0x14, 0x25, 0xe6,
	0x15, 0x17, 0xe4, 0x17, 0x95, 0x08, 0x49, 0x71, 0x71, 0xb8, 0x24, 0x96, 0x24, 0xfa, 0x64, 0x16,
	0x97, 0x48, 0x30, 0x2a, 0x30, 0x6b, 0xf0, 0x04, 0xc1, 0xf9, 0x4a, 0x3a, 0x48, 0xb6, 0xba, 0xe6,
	0xa5, 0x08, 0xc9, 0x70, 0x71, 0x7a, 0x16, 0x07, 0x97, 0x26, 0x27, 0xa7, 0x16, 0x17, 0x83, 0xed,
	0xe5, 0x08, 0x42, 0x08, 0x38, 0xc9, 0x9e, 0x78, 0x24, 0xc7, 0x78, 0xe1, 0x91, 0x1c, 0xe3, 0x83,
	0x47, 0x72, 0x8c, 0x33, 0x1e, 0xcb, 0x31, 0x44, 0x71, 0xeb, 0xe9, 0xe9, 0xc3, 0xc2, 0x34, 0x89,
	0x0d, 0xcc, 0x32, 0x06, 0x04, 0x00, 0x00, 0xff, 0xff, 0x7c, 0x47, 0x43, 0x47, 0x77, 0x01, 0x00,
	0x00,
}

func (m *ReconnectSyn) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *ReconnectSyn) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *ReconnectSyn) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.XXX_unrecognized != nil {
		i -= len(m.XXX_unrecognized)
		copy(dAtA[i:], m.XXX_unrecognized)
	}
	if m.RecvNum != 0 {
		i = encodeVarintReconnect(dAtA, i, uint64(m.RecvNum))
		i--
		dAtA[i] = 0x20
	}
	if m.SentNum != 0 {
		i = encodeVarintReconnect(dAtA, i, uint64(m.SentNum))
		i--
		dAtA[i] = 0x18
	}
	if m.PrevSessionId != 0 {
		i = encodeVarintReconnect(dAtA, i, uint64(m.PrevSessionId))
		i--
		dAtA[i] = 0x10
	}
	if m.CurrSessionId != 0 {
		i = encodeVarintReconnect(dAtA, i, uint64(m.CurrSessionId))
		i--
		dAtA[i] = 0x8
	}
	return len(dAtA) - i, nil
}

func (m *ReconnectAck) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *ReconnectAck) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *ReconnectAck) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.XXX_unrecognized != nil {
		i -= len(m.XXX_unrecognized)
		copy(dAtA[i:], m.XXX_unrecognized)
	}
	if m.SentNum != 0 {
		i = encodeVarintReconnect(dAtA, i, uint64(m.SentNum))
		i--
		dAtA[i] = 0x10
	}
	if m.RecvNum != 0 {
		i = encodeVarintReconnect(dAtA, i, uint64(m.RecvNum))
		i--
		dAtA[i] = 0x8
	}
	return len(dAtA) - i, nil
}

func (m *ReconnectTransport) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *ReconnectTransport) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *ReconnectTransport) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.XXX_unrecognized != nil {
		i -= len(m.XXX_unrecognized)
		copy(dAtA[i:], m.XXX_unrecognized)
	}
	if len(m.DataList) > 0 {
		for iNdEx := len(m.DataList) - 1; iNdEx >= 0; iNdEx-- {
			i -= len(m.DataList[iNdEx])
			copy(dAtA[i:], m.DataList[iNdEx])
			i = encodeVarintReconnect(dAtA, i, uint64(len(m.DataList[iNdEx])))
			i--
			dAtA[i] = 0xa
		}
	}
	return len(dAtA) - i, nil
}

func (m *ReconnectEnd) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *ReconnectEnd) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *ReconnectEnd) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.XXX_unrecognized != nil {
		i -= len(m.XXX_unrecognized)
		copy(dAtA[i:], m.XXX_unrecognized)
	}
	if m.IsSuccess {
		i--
		if m.IsSuccess {
			dAtA[i] = 1
		} else {
			dAtA[i] = 0
		}
		i--
		dAtA[i] = 0x8
	}
	return len(dAtA) - i, nil
}

func encodeVarintReconnect(dAtA []byte, offset int, v uint64) int {
	offset -= sovReconnect(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *ReconnectSyn) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if m.CurrSessionId != 0 {
		n += 1 + sovReconnect(uint64(m.CurrSessionId))
	}
	if m.PrevSessionId != 0 {
		n += 1 + sovReconnect(uint64(m.PrevSessionId))
	}
	if m.SentNum != 0 {
		n += 1 + sovReconnect(uint64(m.SentNum))
	}
	if m.RecvNum != 0 {
		n += 1 + sovReconnect(uint64(m.RecvNum))
	}
	if m.XXX_unrecognized != nil {
		n += len(m.XXX_unrecognized)
	}
	return n
}

func (m *ReconnectAck) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if m.RecvNum != 0 {
		n += 1 + sovReconnect(uint64(m.RecvNum))
	}
	if m.SentNum != 0 {
		n += 1 + sovReconnect(uint64(m.SentNum))
	}
	if m.XXX_unrecognized != nil {
		n += len(m.XXX_unrecognized)
	}
	return n
}

func (m *ReconnectTransport) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if len(m.DataList) > 0 {
		for _, b := range m.DataList {
			l = len(b)
			n += 1 + l + sovReconnect(uint64(l))
		}
	}
	if m.XXX_unrecognized != nil {
		n += len(m.XXX_unrecognized)
	}
	return n
}

func (m *ReconnectEnd) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if m.IsSuccess {
		n += 2
	}
	if m.XXX_unrecognized != nil {
		n += len(m.XXX_unrecognized)
	}
	return n
}

func sovReconnect(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozReconnect(x uint64) (n int) {
	return sovReconnect(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *ReconnectSyn) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowReconnect
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: ReconnectSyn: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: ReconnectSyn: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field CurrSessionId", wireType)
			}
			m.CurrSessionId = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowReconnect
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.CurrSessionId |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 2:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field PrevSessionId", wireType)
			}
			m.PrevSessionId = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowReconnect
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.PrevSessionId |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 3:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field SentNum", wireType)
			}
			m.SentNum = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowReconnect
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.SentNum |= int32(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 4:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field RecvNum", wireType)
			}
			m.RecvNum = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowReconnect
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.RecvNum |= int32(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		default:
			iNdEx = preIndex
			skippy, err := skipReconnect(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthReconnect
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			m.XXX_unrecognized = append(m.XXX_unrecognized, dAtA[iNdEx:iNdEx+skippy]...)
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *ReconnectAck) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowReconnect
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: ReconnectAck: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: ReconnectAck: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field RecvNum", wireType)
			}
			m.RecvNum = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowReconnect
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.RecvNum |= int32(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 2:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field SentNum", wireType)
			}
			m.SentNum = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowReconnect
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.SentNum |= int32(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		default:
			iNdEx = preIndex
			skippy, err := skipReconnect(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthReconnect
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			m.XXX_unrecognized = append(m.XXX_unrecognized, dAtA[iNdEx:iNdEx+skippy]...)
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *ReconnectTransport) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowReconnect
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: ReconnectTransport: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: ReconnectTransport: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field DataList", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowReconnect
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				byteLen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if byteLen < 0 {
				return ErrInvalidLengthReconnect
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthReconnect
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.DataList = append(m.DataList, make([]byte, postIndex-iNdEx))
			copy(m.DataList[len(m.DataList)-1], dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipReconnect(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthReconnect
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			m.XXX_unrecognized = append(m.XXX_unrecognized, dAtA[iNdEx:iNdEx+skippy]...)
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *ReconnectEnd) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowReconnect
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: ReconnectEnd: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: ReconnectEnd: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field IsSuccess", wireType)
			}
			var v int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowReconnect
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				v |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			m.IsSuccess = bool(v != 0)
		default:
			iNdEx = preIndex
			skippy, err := skipReconnect(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthReconnect
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			m.XXX_unrecognized = append(m.XXX_unrecognized, dAtA[iNdEx:iNdEx+skippy]...)
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func skipReconnect(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowReconnect
			}
			if iNdEx >= l {
				return 0, io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		wireType := int(wire & 0x7)
		switch wireType {
		case 0:
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowReconnect
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				iNdEx++
				if dAtA[iNdEx-1] < 0x80 {
					break
				}
			}
		case 1:
			iNdEx += 8
		case 2:
			var length int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowReconnect
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				length |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if length < 0 {
				return 0, ErrInvalidLengthReconnect
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupReconnect
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthReconnect
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthReconnect        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowReconnect          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupReconnect = fmt.Errorf("proto: unexpected end of group")
)
