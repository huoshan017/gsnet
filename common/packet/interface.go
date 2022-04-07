package packet

import "io"

// 包接口
type IPacket interface {
	Type() PacketType
	Data() *[]byte
	SetData(*[]byte)
	MMType() MemoryManagementType
}

// packet池
type IPacketPool interface {
	Get() IPacket
	Put(IPacket)
}

// packet构建器
type IPacketBuilder interface {
	EncodeWriteTo(PacketType, []byte, io.Writer) error
	EncodeBytesArrayWriteTo(PacketType, [][]byte, io.Writer) error
	EncodeBytesPointerArrayWriteTo(pType PacketType, pBytesArray []*[]byte, writer io.Writer) error
	DecodeReadFrom(io.Reader) (IPacket, error)
}
