package common

import (
	"io"
	"sync"
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
	CType CompressType
	EType EncryptionType
}

type Packet struct {
	typ   PacketType     // 类型
	cType CompressType   // 是否被压缩
	eType EncryptionType // 是否加密加密
	data  []byte
}

type IPacketBuilder interface {
	EncodeWriteTo(pType PacketType, rawData []byte, writer io.Writer) error
	DecodeReadFrom(io.Reader) (*Packet, error)
	RecyclePacket(*Packet)
}

const (
	DefaultPacketHeaderLen = 3
)

type DefaultPacketBuilder struct {
	options PacketOptions
}

func NewDefaultPacketBuilder(options PacketOptions) *DefaultPacketBuilder {
	return &DefaultPacketBuilder{
		options: options,
	}
}

func (pc *DefaultPacketBuilder) EncodeWriteTo(pType PacketType, rawData []byte, writer io.Writer) error {
	dataLen := 3 + len(rawData)
	var d = [DefaultPacketHeaderLen + 3]byte{}
	d[0] = byte(dataLen >> 16 & 0xff)
	d[1] = byte(dataLen >> 8 & 0xff)
	d[2] = byte(dataLen & 0xff)
	d[3] = byte(pType)
	d[4] = byte(pc.options.CType)
	d[5] = byte(pc.options.EType)
	_, err := writer.Write(d[:])
	if err == nil {
		_, err = writer.Write(rawData)
	}
	return err
}

func (pc *DefaultPacketBuilder) DecodeReadFrom(reader io.Reader) (*Packet, error) {
	var header = [DefaultPacketHeaderLen + 3]byte{}
	// read header
	_, err := io.ReadFull(reader, header[:])
	if err != nil {
		return nil, err
	}
	dataLen := uint32(header[0]) << 16 & 0xff0000
	dataLen += uint32(header[1]) << 8 & 0xff00
	dataLen += uint32(header[2]) & 0xff
	data := make([]byte, dataLen-3) // 内存池优化
	// read data
	_, err = io.ReadFull(reader, data)
	if err != nil {
		return nil, err
	}
	packet := PacketGet()
	packet.typ = PacketType(header[3])
	packet.cType = CompressType(header[4])
	packet.eType = EncryptionType(header[5])
	packet.data = data
	return packet, nil
}

func (pc *DefaultPacketBuilder) RecyclePacket(packet *Packet) {

}

// packet池
var packetPool *sync.Pool

func init() {
	packetPool = &sync.Pool{
		New: func() interface{} {
			return &Packet{}
		},
	}
}

func PacketGet() *Packet {
	return packetPool.Get().(*Packet)
}

func PacketPut(packet *Packet) {
	packetPool.Put(packet)
}
