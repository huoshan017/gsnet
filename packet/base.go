package packet

import (
	"errors"
)

const (
	MaxPacketLength = 128 * 1024
)

var (
	ErrBodyLengthTooLong    = errors.New("gsnet: receive body length too long")
	ErrHeaderLengthTooSmall = errors.New("gsnet: default packet header length too small")
)

type PacketType uint8

const (
	PacketNormalData         PacketType = iota // 数据包
	PacketHandshake          PacketType = 1    // 握手请求
	PacketHandshakeAck       PacketType = 2    // 握手回应
	PacketHeartbeat          PacketType = 3    // 心跳请求
	PacketHeartbeatAck       PacketType = 4    // 心跳回应
	PacketReconnectSyn       PacketType = 5    // 重连
	PacketReconnectAck       PacketType = 6    // 重连回应
	PacketReconnectTransport PacketType = 7    // 重连数据传输
	PacketReconnectEnd       PacketType = 8    // 重连结束
	PacketSentAck            PacketType = 100  // 发送回应
)

// 内存管理类型
type MemoryManagementType uint8

const (
	MemoryManagementSystemGC           MemoryManagementType = iota // 系统GC，默认管理方式
	MemoryManagementPoolFrameworkFree  MemoryManagementType = 1    // 内存池分配由框架释放
	MemoryManagementPoolUserManualFree MemoryManagementType = 2    // 内存池分配使用者手动释放
	MemoryManagementNone               MemoryManagementType = 255
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

func (ph *CommonPacketHeader) SetVersion(version int32) {

}

func (ph *CommonPacketHeader) GetVersion() int32 {
	return 0
}

func (ph *CommonPacketHeader) SetMagicNumber(number int32) {

}

func (ph *CommonPacketHeader) GetMagicNumber() int32 {
	return 0
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
	typ    PacketType           // 类型
	mType  MemoryManagementType // 内存管理类型
	data   *[]byte              // 数据缓冲地址
	offset int32                // 有效数据偏移
	data2  []byte               // GC对应的数据
}

func (p *Packet) Reset() {
	p.typ = PacketNormalData
	p.mType = MemoryManagementSystemGC
	p.data = nil
	p.offset = 0
	p.data2 = nil
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
		return (*p.data)[p.offset:]
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

func (p *Packet) SetGCData(typ PacketType, mType MemoryManagementType, data []byte) {
	p.typ = typ
	p.mType = MemoryManagementSystemGC
	p.data2 = data
}

func (p *Packet) ChangeDataOwnership(newPak *Packet, dataOffset int32, toMMType MemoryManagementType) {
	newPak.typ = p.typ
	newPak.offset = dataOffset
	if p.mType == MemoryManagementSystemGC {
		newPak.data2 = p.data2
		p.data2 = nil
		p.mType = MemoryManagementSystemGC
	} else {
		newPak.data = p.data
		p.data = nil
		if toMMType != MemoryManagementSystemGC {
			newPak.mType = toMMType
		} else {
			newPak.mType = p.mType
		}
	}
}

func (p *Packet) Offset(offset int32) {
	p.offset = offset
}
