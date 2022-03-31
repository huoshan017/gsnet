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
	PacketNormal PacketType = iota
)

type CompressType int8

const (
	CompressNone CompressType = iota
	CompressZip  CompressType = 1
)

type EncryptionType int8

const (
	EncryptionNone EncryptionType = iota
)

// 内存管理类型
type MemoryManagementType int8

const (
	MemoryManagementSystemGC           = iota // 系统GC，默认管理方式
	MemoryManagementPoolFrameworkFree  = 1    // 内存池分配由框架释放
	MemoryManagementPoolUserManualFree = 2    // 内存池分配使用者手动释放
)

type PacketOptions struct {
	CType      CompressType
	EType      EncryptionType
	PacketPool IPacketPool
}

// 基础包结构
type Packet struct {
	typ PacketType // 类型
	//cType CompressType   // 是否被压缩
	//eType EncryptionType // 是否加密加密
	mType MemoryManagementType // 内存管理类型
	data  *[]byte
}

func (p Packet) Data() *[]byte {
	return p.data
}

func (p *Packet) SetData(data *[]byte) {
	p.data = data
}

func (p Packet) MMType() MemoryManagementType {
	return p.mType
}

// bytes包定义
type BytesPacket []byte

func (p *BytesPacket) Data() *[]byte {
	return (*[]byte)(p)
}

func (p *BytesPacket) SetData(*[]byte) {

}

func (p BytesPacket) MMType() MemoryManagementType {
	return MemoryManagementSystemGC
}
