package packet

import "io"

// 包接口
type IPacket interface {
	Data() *[]byte
	SetData(*[]byte)
}

// packet池
type IPacketPool interface {
	Get() IPacket
	Put(IPacket)
}

// packet构建器
type IPacketBuilder interface {
	EncodeWriteTo(PacketType, []byte, io.Writer) error
	DecodeReadFrom(io.Reader) (IPacket, error)
}
