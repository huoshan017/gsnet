package packet

// 包接口
type IPacket interface {
	Type() PacketType
	MMType() MemoryManagementType
	Data() []byte
	PData() *[]byte
}

// packet池
type IPacketPool interface {
	Get() IPacket
	Put(IPacket)
}

type IPacketHeader interface {
	SetType(PacketType)
	GetType() PacketType
	SetCompressType(CompressType)
	GetCompressType() CompressType
	SetEncryptionType(EncryptionType)
	GetEncryptionType() EncryptionType
	SetDataLength(uint32)
	GetDataLength() uint32
	Get(key string) any
	Set(key string, data any)
	FormatTo(buf []byte) error
	UnformatFrom(buf []byte) error
}
