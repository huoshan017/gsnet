package packet

import "errors"

const (
	MaxPacketLength = 128 * 1024
)

var (
	ErrBodyLenInvalid = errors.New("gsnet: receive body length too long")
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
