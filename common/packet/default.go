package packet

import (
	"errors"
	"io"
	"sync"

	"github.com/huoshan017/gsnet/common/pool"
)

const (
	DefaultPacketHeaderLen = 3
)

var (
	ErrBodyLenInvalid = errors.New("gsnet: receive body length too long")
)

// pool for type `Packet`
type DefaultPacketPool struct {
	pool *sync.Pool
}

func NewDefaultPacketPool() *DefaultPacketPool {
	return &DefaultPacketPool{
		pool: &sync.Pool{
			New: func() interface{} {
				return &Packet{}
			},
		},
	}
}

func (p *DefaultPacketPool) Get() IPacket {
	pak := p.pool.Get().(IPacket)
	return pak
}

func (p *DefaultPacketPool) Put(pak IPacket) {
	// 不是*Packet类型
	if _, o := interface{}(pak).(*Packet); !o {
		return
	}
	pool.GetBuffPool().Free(pak.Data())
	p.pool.Put(pak)
}

// builder for type `Packet`
type DefaultPacketBuilder struct {
	options PacketOptions
}

func NewDefaultPacketBuilder(options PacketOptions) *DefaultPacketBuilder {
	pb := &DefaultPacketBuilder{
		options: options,
	}
	return pb
}

func (pc *DefaultPacketBuilder) EncodeWriteTo(pType PacketType, rawData []byte, writer io.Writer) error {
	dataLen := 3 + len(rawData)
	var d = [DefaultPacketHeaderLen + 3]byte{}
	// data length
	d[0] = byte(dataLen >> 16 & 0xff)
	d[1] = byte(dataLen >> 8 & 0xff)
	d[2] = byte(dataLen & 0xff)

	d[3] = byte(pType)            // packet type
	d[4] = byte(pc.options.CType) // compress type
	d[5] = byte(pc.options.EType) // encryption type

	_, err := writer.Write(d[:])
	if err == nil {
		_, err = writer.Write(rawData)
	}
	return err
}

func (pc *DefaultPacketBuilder) DecodeReadFrom(reader io.Reader) (IPacket, error) {
	var header = [DefaultPacketHeaderLen + 3]byte{}
	// read header
	_, err := io.ReadFull(reader, header[:])
	if err != nil {
		return nil, err
	}
	dataLen := uint32(header[0]) << 16 & 0xff0000
	dataLen += uint32(header[1]) << 8 & 0xff00
	dataLen += uint32(header[2]) & 0xff
	if dataLen > MaxPacketLength {
		return nil, ErrBodyLenInvalid
	}
	//data := make([]byte, dataLen-3) // 内存池优化
	//pData := &data
	pData := pool.GetBuffPool().Alloc(int32(dataLen) - 3)

	// read data
	_, err = io.ReadFull(reader, *pData)
	if err != nil {
		return nil, err
	}
	p := pc.options.PacketPool.Get()
	pak := interface{}(p).(*Packet)
	pak.typ = PacketType(header[3])
	pak.cType = CompressType(header[4])
	pak.eType = EncryptionType(header[5])
	pak.SetData(pData)
	return pak, nil
}

// 默认的packet池和packet构建器
var (
	defaultPacketPool    *DefaultPacketPool
	defaultPacketBuilder *DefaultPacketBuilder
)

func init() {
	defaultPacketPool = NewDefaultPacketPool()
	defaultPacketBuilder = NewDefaultPacketBuilder(PacketOptions{
		PacketPool: defaultPacketPool,
	})
}

func GetDefaultPacketPool() *DefaultPacketPool {
	return defaultPacketPool
}

func GetDefaultPacketBuilder() *DefaultPacketBuilder {
	return defaultPacketBuilder
}
