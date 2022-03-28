package packet

const (
	MaxPacketLength = 128 * 1024
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

type PacketOptions struct {
	CType      CompressType
	EType      EncryptionType
	PacketPool IPacketPool
}

// 基础包结构
type Packet struct {
	typ   PacketType     // 类型
	cType CompressType   // 是否被压缩
	eType EncryptionType // 是否加密加密
	data  *[]byte
}

func (p Packet) Data() *[]byte {
	return p.data
}

func (p *Packet) SetData(data *[]byte) {
	p.data = data
}

// bytes包定义
type BytesPacket []byte

func (p *BytesPacket) Data() *[]byte {
	return (*[]byte)(p)
}

func (p *BytesPacket) SetData(*[]byte) {

}
