package packet

// 包接口
type IPacket interface {
	Type() PacketType
	MMType() MemoryManagementType
	Data() []byte
	PData() *[]byte
	Release() bool
}

// packet池
type IPacketPool interface {
	Get() IPacket
	GetWithType(PoolPacketType) IPacket
	Put(IPacket)
}

type IPacketSharedPool interface {
	GetShared() *SharedPacket
}

type IPacketBaseHeader interface {
	SetType(PacketType)
	GetType() PacketType
	SetMagicNumber(int32)
	GetMagicNumber() int32
	SetVersion(int32)
	GetVersion() int32
	SetCompressType(CompressType)
	GetCompressType() CompressType
	SetEncryptionType(EncryptionType)
	GetEncryptionType() EncryptionType
	SetDataLength(uint32)
	GetDataLength() uint32
}

type IPacketHeader interface {
	IPacketBaseHeader
	GetValue(string) any
	SetValue(string, any)
	FormatTo(buf []byte) error
	UnformatFrom(buf []byte) error
}
