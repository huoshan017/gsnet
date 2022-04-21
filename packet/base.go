package packet

import "errors"

const (
	MaxPacketLength = 128 * 1024
)

var (
	ErrBodyLengthTooLong    = errors.New("gsnet: receive body length too long")
	ErrHeaderLengthTooSmall = errors.New("gsnet: default packet header length too small")
)

type PacketType int8

const (
	PacketNormalData   PacketType = iota
	PacketHandshake    PacketType = 1
	PacketHandshakeAck PacketType = 2
	PacketHeartbeat    PacketType = 3
	PacketHeartbeatAck PacketType = 4
	PacketSentAck      PacketType = 5
)

// 内存管理类型
type MemoryManagementType int8

const (
	MemoryManagementSystemGC           = iota // 系统GC，默认管理方式
	MemoryManagementPoolFrameworkFree  = 1    // 内存池分配由框架释放
	MemoryManagementPoolUserManualFree = 2    // 内存池分配使用者手动释放
)

const (
	DefaultPacketHeaderLen = 6
)

type CommonPacketHeader struct {
	Type    PacketType
	CType   CompressType
	EType   EncryptionType
	DataLen uint32
}

func (ph *CommonPacketHeader) SetType(typ PacketType) {
	ph.Type = typ
}

func (ph *CommonPacketHeader) GetType() PacketType {
	return ph.Type
}

func (ph *CommonPacketHeader) SetCompressType(typ CompressType) {
	ph.CType = typ
}

func (ph *CommonPacketHeader) GetCompressType() CompressType {
	return ph.CType
}

func (ph *CommonPacketHeader) SetEncryptionType(et EncryptionType) {
	ph.EType = et
}

func (ph *CommonPacketHeader) GetEncryptionType() EncryptionType {
	return ph.EType
}

func (ph *CommonPacketHeader) SetDataLength(length uint32) {
	ph.DataLen = length
}

func (ph *CommonPacketHeader) GetDataLength() uint32 {
	return ph.DataLen
}

// bytes包定义
type BytesPacket []byte

func (p BytesPacket) Type() PacketType {
	return PacketNormalData
}

func (p BytesPacket) Data() []byte {
	return p
}

func (p *BytesPacket) PData() *[]byte {
	return (*[]byte)(p)
}

func (p BytesPacket) MMType() MemoryManagementType {
	return MemoryManagementSystemGC
}

// 基础包结构
type Packet struct {
	typ   PacketType           // 类型
	mType MemoryManagementType // 内存管理类型
	data  *[]byte
	data2 []byte
}

func (p Packet) Type() PacketType {
	return p.typ
}

func (p Packet) PData() *[]byte {
	return p.data
}

func (p Packet) Data() []byte {
	if p.mType == MemoryManagementSystemGC {
		return p.data2
	} else {
		return *p.data
	}
}

func (p Packet) MMType() MemoryManagementType {
	return p.mType
}

func (p *Packet) Set(typ PacketType, mType MemoryManagementType, data *[]byte) {
	p.typ = typ
	p.mType = mType
	p.data = data
}

func (p *Packet) Set2(typ PacketType, mType MemoryManagementType, data []byte) {
	p.typ = typ
	p.mType = MemoryManagementSystemGC
	p.data2 = data
}
