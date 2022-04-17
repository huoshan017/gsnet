package common

import (
	"fmt"
	"io"

	"github.com/huoshan017/gsnet/log"
	"github.com/huoshan017/gsnet/packet"
	"github.com/huoshan017/gsnet/pool"
)

// builder for type `Packet`
type DefaultPacketBuilder struct {
	options      packet.PacketOptions
	compressor   packet.ICompressor
	decompressor packet.IDecompressor
	encrypter    packet.IEncrypter
	decrypter    packet.IDecrypter
}

func NewDefaultPacketBuilder(options packet.PacketOptions) *DefaultPacketBuilder {
	pb := &DefaultPacketBuilder{
		options: options,
	}

	var err error

	err = pb.createEncrypter()
	if err != nil {
		return nil
	}

	err = pb.createDecrypter()
	if err != nil {
		return nil
	}

	err = pb.createCompressor()
	if err != nil {
		return nil
	}

	err = pb.createDecompressor()
	if err != nil {
		return nil
	}

	return pb
}

func (pc *DefaultPacketBuilder) createEncrypter() error {
	var (
		encrypter packet.IEncrypter
		err       error
	)
	switch pc.options.EType {
	case packet.EncryptionNone:
	case packet.EncryptionAes:
		pc.options.CryptoKey = packet.GenAesKey()
		encrypter, err = packet.NewAesEncrypter(pc.options.CryptoKey)
		if err != nil {
			log.Infof("gsnet: create aes encrypter err %v", err)
		}
	case packet.EncryptionDes:
		pc.options.CryptoKey = packet.GenDesKey()
		encrypter, err = packet.NewDesEncrypter(pc.options.CryptoKey)
		if err != nil {
			log.Infof("gsnet: create des encrypter err %v", err)
		}
	default:
		log.Infof("gsnet: invalid encryption type %v", pc.options.EType)
	}
	pc.encrypter = encrypter
	return err
}

func (pc *DefaultPacketBuilder) createDecrypter() error {
	var (
		decrypter packet.IDecrypter
		err       error
	)
	switch pc.options.EType {
	case packet.EncryptionNone:
	case packet.EncryptionAes:
		decrypter, err = packet.NewAesDecrypter(pc.options.CryptoKey)
		if err != nil {
			log.Infof("gsnet: create aes decrypter err %v", err)
		}
	case packet.EncryptionDes:
		decrypter, err = packet.NewDesDecrypter(pc.options.CryptoKey)
		if err != nil {
			log.Infof("gsnet: create des decrypter err %v", err)
		}
	default:
		err = fmt.Errorf("gsnet: invalid encryption type %v", pc.options.EType)
	}
	pc.decrypter = decrypter
	return err
}

func (pc *DefaultPacketBuilder) createCompressor() error {
	var (
		compressor packet.ICompressor
		err        error
	)
	switch pc.options.CType {
	case packet.CompressNone:
	case packet.CompressZip:
		compressor = packet.NewZlibCompressor()
	case packet.CompressGzip:
		compressor = packet.NewGzipCompressor()
	case packet.CompressSnappy:
		compressor = packet.NewSnappyCompressor()
	default:
		err = fmt.Errorf("gsnet: invalid compress type %v", pc.options.CType)
	}
	pc.compressor = compressor
	return err
}

func (pc *DefaultPacketBuilder) createDecompressor() error {
	var (
		decompressor packet.IDecompressor
		err          error
	)
	switch pc.options.CType {
	case packet.CompressNone:
	case packet.CompressZip:
		decompressor = packet.NewZlibDecompressor()
	case packet.CompressGzip:
		decompressor = packet.NewGzipDecompressor()
	case packet.CompressSnappy:
		decompressor = packet.NewSnappyDecompressor()
	default:
		log.Infof("gsnet: invalid compress type %v", pc.options.CType)
	}
	pc.decompressor = decompressor
	return err
}

func (pc *DefaultPacketBuilder) EncodeWriteTo(pType packet.PacketType, data []byte, writer io.Writer) error {
	pc.checkAndCreateCompressorAndEncrypter()

	dataLen := 3 + len(data)
	var d = [packet.DefaultPacketHeaderLen + 3]byte{}
	pc.formatHeader(d[:], pType, dataLen)

	var err error
	// 不需要压缩和加密
	if pType == packet.PacketHandshake || pType == packet.PacketHandshakeAck || (pc.compressor == nil && pc.decompressor == nil && pc.encrypter == nil && pc.decrypter == nil) {
		_, err = writer.Write(d[:])
		if err == nil {
			_, err = writer.Write(data)
		}
		return err
	}

	temp := pool.GetBuffPool().Alloc(int32(len(d) + dataLen))
	copy(*temp, d[:])
	copy((*temp)[len(d):], data)

	// todo 压缩处理和加密处理
	err = pc.compressAndEncryptWriteData(*temp, writer)
	pool.GetBuffPool().Free(temp)
	return err
}

func (pc *DefaultPacketBuilder) EncodeBytesArrayWriteTo(pType packet.PacketType, datas [][]byte, writer io.Writer) error {
	pc.checkAndCreateCompressorAndEncrypter()

	var d = [packet.DefaultPacketHeaderLen + 3]byte{}
	var dataLen int
	for i := 0; i < len(datas); i++ {
		dataLen += len(datas[i])
	}
	dataLen += 3
	pc.formatHeader(d[:], pType, dataLen)

	var err error
	if pType == packet.PacketHandshake || pType == packet.PacketHandshakeAck || (pc.compressor == nil && pc.decompressor == nil && pc.encrypter == nil && pc.decrypter == nil) {
		_, err = writer.Write(d[:])
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

	temp := pool.GetBuffPool().Alloc(int32(len(d) + dataLen))
	copy(*temp, d[:])
	offset := len(d)
	for i := 0; i < len(datas); i++ {
		copy((*temp)[offset:], datas[i])
		offset += len(datas[i])
	}
	// todo 压缩处理和加密处理
	err = pc.compressAndEncryptWriteData(*temp, writer)
	pool.GetBuffPool().Free(temp)

	return err
}

func (pc *DefaultPacketBuilder) EncodeBytesPointerArrayWriteTo(pType packet.PacketType, pBytesArray []*[]byte, writer io.Writer) error {
	pc.checkAndCreateCompressorAndEncrypter()

	var d = [packet.DefaultPacketHeaderLen + 3]byte{}
	var dataLen int
	for i := 0; i < len(pBytesArray); i++ {
		dataLen += len(*pBytesArray[i])
	}
	dataLen += 3
	pc.formatHeader(d[:], pType, dataLen)

	var err error
	if pType == packet.PacketHandshake || pType == packet.PacketHandshakeAck || (pc.compressor == nil && pc.decompressor == nil && pc.encrypter == nil && pc.decrypter == nil) {
		_, err = writer.Write(d[:])
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
	temp := pool.GetBuffPool().Alloc(int32(len(d) + dataLen))
	copy(*temp, d[:])
	offset := len(d)
	for i := 0; i < len(pBytesArray); i++ {
		copy((*temp)[offset:], *pBytesArray[i])
		offset += len(*pBytesArray[i])
	}
	// todo 压缩处理和加密处理
	err = pc.compressAndEncryptWriteData(*temp, writer)
	pool.GetBuffPool().Free(temp)
	return err
}

func (pc *DefaultPacketBuilder) DecodeReadFrom(reader io.Reader) (packet.IPacket, error) {
	pc.checkAndCreateDecompressorAndDecrypter()

	var header = [packet.DefaultPacketHeaderLen + 3]byte{}
	// read header
	_, err := io.ReadFull(reader, header[:])
	if err != nil {
		return nil, err
	}
	dataLen := uint32(header[0]) << 16 & 0xff0000
	dataLen += uint32(header[1]) << 8 & 0xff00
	dataLen += uint32(header[2]) & 0xff
	if dataLen > packet.MaxPacketLength {
		return nil, packet.ErrBodyLenInvalid
	}
	pData := pool.GetBuffPool().Alloc(int32(dataLen) - 3)

	// read data
	_, err = io.ReadFull(reader, *pData)
	if err != nil {
		return nil, err
	}

	packetType := packet.PacketType(header[3])
	var pak *packet.Packet
	if packetType == packet.PacketHandshake || packetType == packet.PacketHandshakeAck || (pc.compressor == nil && pc.decompressor == nil && pc.encrypter == nil && pc.decrypter == nil) {
		p := pc.options.PacketPool.Get()
		pak = any(p).(*packet.Packet)
		pak.Set(packetType, packet.MemoryManagementPoolFrameworkFree, pData)
	} else {
		// 先解密再解压
		var nd []byte
		nd, err = pc.decryptAndDecompress(*pData)
		if err != nil {
			return nil, err
		}
		p := pc.options.PacketPool.Get()
		pak = any(p).(*packet.Packet)
		pak.Set(packetType, packet.MemoryManagementSystemGC, &nd)
	}
	return pak, nil
}

func (pc *DefaultPacketBuilder) checkAndCreateCompressorAndEncrypter() {
	if pc.compressor == nil && packet.IsValidCompressType(pc.options.CType) {
		pc.createCompressor()
	}
	if pc.encrypter == nil && packet.IsValidEncryptionType(pc.options.EType) {
		pc.createEncrypter()
	}
}

func (pc *DefaultPacketBuilder) checkAndCreateDecompressorAndDecrypter() {
	if pc.decompressor == nil && packet.IsValidCompressType(pc.options.CType) {
		pc.createDecompressor()
	}
	if pc.decrypter == nil && packet.IsValidEncryptionType(pc.options.EType) {
		pc.createDecrypter()
	}
}

func (pc *DefaultPacketBuilder) compressAndEncryptWriteData(data []byte, writer io.Writer) error {
	var (
		err error
	)
	if pc.compressor != nil {
		data, err = pc.compressor.Compress(data)
		if err != nil {
			return err
		}
	}
	if pc.encrypter != nil {
		data, err = pc.encrypter.Encrypt(data)
		if err != nil {
			return err
		}
	}
	_, err = writer.Write(data)
	return err
}

func (pc *DefaultPacketBuilder) decryptAndDecompress(data []byte) ([]byte, error) {
	var (
		err error
	)
	if pc.decrypter != nil {
		data, err = pc.decrypter.Decrypt(data)
		if err != nil {
			return nil, err
		}
	}
	if pc.compressor != nil {
		data, err = pc.decompressor.Decompress(data)
		if err != nil {
			return nil, err
		}
	}
	return data, nil
}

func (pc DefaultPacketBuilder) formatHeader(header []byte, pType packet.PacketType, dataLen int) {
	// data length
	header[0] = byte(dataLen >> 16 & 0xff)
	header[1] = byte(dataLen >> 8 & 0xff)
	header[2] = byte(dataLen & 0xff)

	header[3] = byte(pType)            // packet type
	header[4] = byte(pc.options.CType) // compress type
	header[5] = byte(pc.options.EType) // encryption type
}
