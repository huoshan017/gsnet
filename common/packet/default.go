package packet

import (
	"io"
	"sync"

	"github.com/huoshan017/gsnet/common/pool"
)

const (
	DefaultPacketHeaderLen = 3
)

// pool for type `Packet`
type DefaultPacketPool struct {
	pool         *sync.Pool
	usingPackets sync.Map // 保存分配的Packet地址
}

func NewDefaultPacketPool() *DefaultPacketPool {
	return &DefaultPacketPool{
		pool: &sync.Pool{
			New: func() any {
				return &Packet{}
			},
		},
	}
}

func (p *DefaultPacketPool) Get() IPacket {
	pak := p.pool.Get().(IPacket)
	p.usingPackets.Store(pak, true)
	return pak
}

func (p *DefaultPacketPool) Put(pak IPacket) {
	// 不是*Packet类型
	if _, o := any(pak).(*Packet); !o {
		return
	}
	if _, o := p.usingPackets.LoadAndDelete(pak); !o {
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

func (pc *DefaultPacketBuilder) EncodeWriteTo(pType PacketType, data []byte, writer io.Writer) error {
	dataLen := 3 + len(data)
	var d = [DefaultPacketHeaderLen + 3]byte{}
	pc.formatHeader(d[:], pType, dataLen)
	// todo 压缩处理和加密处理
	_, err := writer.Write(d[:])
	if err == nil {
		_, err = writer.Write(data)
	}
	return err
}

func (pc *DefaultPacketBuilder) EncodeBytesArrayWriteTo(pType PacketType, datas [][]byte, writer io.Writer) error {
	var d = [DefaultPacketHeaderLen + 3]byte{}
	var dataLen int
	for i := 0; i < len(datas); i++ {
		dataLen += len(datas[i])
	}
	dataLen += 3
	pc.formatHeader(d[:], pType, dataLen)
	// todo 压缩处理和加密处理
	_, err := writer.Write(d[:])
	if err == nil {
		for i := 0; i < len(datas); i++ {
			_, err = writer.Write(datas[i])
			if err != nil {
				break
			}
		}
	}
	return err
}

func (pc *DefaultPacketBuilder) EncodeBytesPointerArrayWriteTo(pType PacketType, pBytesArray []*[]byte, writer io.Writer) error {
	var d = [DefaultPacketHeaderLen + 3]byte{}
	var dataLen int
	for i := 0; i < len(pBytesArray); i++ {
		dataLen += len(*pBytesArray[i])
	}
	dataLen += 3
	pc.formatHeader(d[:], pType, dataLen)
	// todo 压缩处理和加密处理
	_, err := writer.Write(d[:])
	if err == nil {
		for i := 0; i < len(*pBytesArray[i]); i++ {
			_, err = writer.Write(*pBytesArray[i])
			if err != nil {
				break
			}
		}
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
	// 压缩类型
	ctype := CompressType(header[4])
	switch ctype {
	// todo 增加压缩类型的处理
	}
	// 加密类型
	etype := EncryptionType(header[5])
	switch etype {
	// todo 增加加密类型的处理
	}
	p := pc.options.PacketPool.Get()
	pak := any(p).(*Packet)
	pak.typ = PacketType(header[3])
	pak.mType = MemoryManagementPoolFrameworkFree
	pak.SetData(pData)
	return pak, nil
}

func (pc DefaultPacketBuilder) formatHeader(header []byte, pType PacketType, dataLen int) {
	// data length
	header[0] = byte(dataLen >> 16 & 0xff)
	header[1] = byte(dataLen >> 8 & 0xff)
	header[2] = byte(dataLen & 0xff)

	header[3] = byte(pType)            // packet type
	header[4] = byte(pc.options.CType) // compress type
	header[5] = byte(pc.options.EType) // encryption type
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
