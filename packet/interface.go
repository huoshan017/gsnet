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
