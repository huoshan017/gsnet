package common

import (
	"fmt"
	"io"

	"github.com/huoshan017/gsnet/log"
	"github.com/huoshan017/gsnet/packet"
	"github.com/huoshan017/gsnet/pool"
)

// packet构建器
type IPacketBuilder interface {
	EncodeWriteTo(packet.PacketType, []byte, io.Writer) error
	EncodeBytesArrayWriteTo(packet.PacketType, [][]byte, io.Writer) error
	EncodeBytesPointerArrayWriteTo(pType packet.PacketType, pBytesArray []*[]byte, writer io.Writer) error
	DecodeReadFrom(io.Reader) (packet.IPacket, error)
	Close()
}

type IPacketBuilderArgsGetter interface {
	Get() []any
}

const (
	DefaultPacketHeaderLen = 6
)

// builder for type `Packet`
type DefaultPacketBuilder struct {
	options      *Options
	compressor   packet.ICompressor
	decompressor packet.IDecompressor
	encrypter    packet.IEncrypter
	decrypter    packet.IDecrypter
	cryptoKey    []byte
}

func NewDefaultPacketBuilder(options *Options) *DefaultPacketBuilder {
	pb := &DefaultPacketBuilder{
		options: options,
	}
	if pb.cryptoKey == nil {
		pb.cryptoKey = packet.GenCryptoKey(options.GetPacketEncryptionType(), options.GetRand())
	}
	if pb.Reset(options.GetPacketCompressType(), options.GetPacketEncryptionType(), pb.cryptoKey) != nil {
		return nil
	}
	return pb
}

func (pb *DefaultPacketBuilder) Reset(compressType packet.CompressType, encryptionType packet.EncryptionType, key []byte) error {
	var err error
	err = pb.createEncrypter(encryptionType, key)
	if err != nil {
		return err
	}

	err = pb.createDecrypter(encryptionType, key)
	if err != nil {
		return err
	}

	err = pb.createCompressor(compressType)
	if err != nil {
		return err
	}

	err = pb.createDecompressor(compressType)
	if err != nil {
		return err
	}

	return nil
}

func (pc *DefaultPacketBuilder) Close() {
	if pc.compressor != nil {
		pc.compressor.Close()
		pc.compressor = nil
	}
	if pc.decompressor != nil {
		pc.decompressor.Close()
		pc.decompressor = nil
	}
}

func (pc *DefaultPacketBuilder) GetCryptoKey() []byte {
	return pc.cryptoKey
}

func (pc *DefaultPacketBuilder) createEncrypter(encryptionType packet.EncryptionType, key []byte) error {
	var (
		encrypter packet.IEncrypter
		err       error
	)
	switch encryptionType {
	case packet.EncryptionNone:
	case packet.EncryptionAes:
		encrypter, err = packet.NewAesEncrypter(key)
		if err != nil {
			log.Infof("gsnet: create aes encrypter with key %v err %v", key, err)
		}
	case packet.EncryptionDes:
		encrypter, err = packet.NewDesEncrypter(key)
		if err != nil {
			log.Infof("gsnet: create des encrypter with key %v err %v", key, err)
		}
	default:
		err = fmt.Errorf("gsnet: invalid encryption type %v", key)
	}
	pc.encrypter = encrypter
	return err
}

func (pc *DefaultPacketBuilder) createDecrypter(encryptionType packet.EncryptionType, key []byte) error {
	var (
		decrypter packet.IDecrypter
		err       error
	)
	switch encryptionType {
	case packet.EncryptionNone:
	case packet.EncryptionAes:
		decrypter, err = packet.NewAesDecrypter(key)
		if err != nil {
			log.Infof("gsnet: create aes decrypter with key %v err %v", key, err)
		}
	case packet.EncryptionDes:
		decrypter, err = packet.NewDesDecrypter(key)
		if err != nil {
			log.Infof("gsnet: create des decrypter with key %v err %v", key, err)
		}
	default:
		err = fmt.Errorf("gsnet: invalid encryption type %v", key)
	}
	pc.decrypter = decrypter
	return err
}

func (pc *DefaultPacketBuilder) createCompressor(compressType packet.CompressType) error {
	var (
		compressor packet.ICompressor
		err        error
	)
	switch compressType {
	case packet.CompressNone:
	case packet.CompressZlib:
		compressor = packet.NewZlibCompressor()
	case packet.CompressGzip:
		compressor = packet.NewGzipCompressor()
	case packet.CompressSnappy:
		compressor = packet.NewSnappyCompressor()
	default:
		err = fmt.Errorf("gsnet: invalid compress type %v", compressType)
	}
	pc.compressor = compressor
	return err
}

func (pc *DefaultPacketBuilder) createDecompressor(compressType packet.CompressType) error {
	var (
		decompressor packet.IDecompressor
		err          error
	)
	switch compressType {
	case packet.CompressNone:
	case packet.CompressZlib:
		decompressor = packet.NewZlibDecompressor()
	case packet.CompressGzip:
		decompressor = packet.NewGzipDecompressor()
	case packet.CompressSnappy:
		decompressor = packet.NewSnappyDecompressor()
	default:
		err = fmt.Errorf("gsnet: invalid compress type %v", compressType)
	}
	pc.decompressor = decompressor
	return err
}

func (pc *DefaultPacketBuilder) EncodeWriteTo(pType packet.PacketType, data []byte, writer io.Writer) error {
	//pc.checkAndCreateCompressorAndEncrypter()

	var err error

	// 压缩处理和加密处理
	if !(isBasePacket(pType) || pc.isNoCompressAndEncryption()) {
		data, err = pc.compressAndEncrypt(data)
		if err != nil {
			return err
		}
	}

	dataLen := len(data)
	var header = [DefaultPacketHeaderLen]byte{}
	pc.formatHeader(header[:], dataLen, pType)

	_, err = writer.Write(header[:])
	if err == nil {
		_, err = writer.Write(data)
	}

	return err
}

func (pc *DefaultPacketBuilder) EncodeBytesArrayWriteTo(pType packet.PacketType, datas [][]byte, writer io.Writer) error {
	//pc.checkAndCreateCompressorAndEncrypter()

	var (
		header  = [DefaultPacketHeaderLen]byte{}
		dataLen int
		err     error
	)

	for i := 0; i < len(datas); i++ {
		dataLen += len(datas[i])
	}

	if isBasePacket(pType) || pc.isNoCompressAndEncryption() {
		pc.formatHeader(header[:], dataLen, pType)
		_, err = writer.Write(header[:])
		if err == nil {
			for i := 0; i < len(datas); i++ {
				_, err = writer.Write(datas[i])
				if err != nil {
					break
				}
			}
		}
	} else {
		temp := pool.GetBuffPool().Alloc(int32(dataLen))
		var offset int
		for i := 0; i < len(datas); i++ {
			copy((*temp)[offset:], datas[i])
			offset += len(datas[i])
		}
		// 压缩处理和加密处理
		var data []byte
		data, err = pc.compressAndEncrypt(*temp)
		pool.GetBuffPool().Free(temp)
		if err == nil {
			pc.formatHeader(header[:], len(data), pType)
			_, err = writer.Write(header[:])
			if err == nil {
				_, err = writer.Write(data)
			}
		}
	}

	return err
}

func (pc *DefaultPacketBuilder) EncodeBytesPointerArrayWriteTo(pType packet.PacketType, pBytesArray []*[]byte, writer io.Writer) error {
	//pc.checkAndCreateCompressorAndEncrypter()

	var (
		dataLen int
		header  = [DefaultPacketHeaderLen]byte{}
		err     error
	)

	for i := 0; i < len(pBytesArray); i++ {
		dataLen += len(*pBytesArray[i])
	}

	if isBasePacket(pType) || pc.isNoCompressAndEncryption() {
		pc.formatHeader(header[:], dataLen, pType)
		_, err = writer.Write(header[:])
		if err == nil {
			for i := 0; i < len(*pBytesArray[i]); i++ {
				_, err = writer.Write(*pBytesArray[i])
				if err != nil {
					break
				}
			}
		}
	} else {
		temp := pool.GetBuffPool().Alloc(int32(dataLen))
		var offset int
		for i := 0; i < len(pBytesArray); i++ {
			copy((*temp)[offset:], *pBytesArray[i])
			offset += len(*pBytesArray[i])
		}
		// 压缩处理和加密处理
		var data []byte
		data, err = pc.compressAndEncrypt(*temp)
		pool.GetBuffPool().Free(temp)
		if err == nil {
			pc.formatHeader(header[:], len(data), pType)
			_, err = writer.Write(header[:])
			if err == nil {
				_, err = writer.Write(data)
			}
		}
	}
	return err
}

func (pc *DefaultPacketBuilder) DecodeReadFrom(reader io.Reader) (packet.IPacket, error) {
	//pc.checkAndCreateDecompressorAndDecrypter()

	var header = [DefaultPacketHeaderLen]byte{}
	// read header
	_, err := io.ReadFull(reader, header[:])
	if err != nil {
		return nil, err
	}

	dataLen, packetType, _, _ := pc.unformatHeader(header[:])

	if dataLen > packet.MaxPacketLength {
		return nil, packet.ErrBodyLenInvalid
	}

	pData := pool.GetBuffPool().Alloc(int32(dataLen))

	// read data
	_, err = io.ReadFull(reader, *pData)
	if err != nil {
		return nil, err
	}

	var pak *packet.Packet
	if isBasePacket(packetType) || pc.isNoCompressAndEncryption() {
		p := pc.options.GetPacketPool().Get()
		pak = any(p).(*packet.Packet)
		pak.Set(packetType, packet.MemoryManagementPoolFrameworkFree, pData)
	} else {
		// 先解密再解压
		var data []byte
		data, err = pc.decryptAndDecompress(*pData)
		pool.GetBuffPool().Free(pData) // !!!!! free pool data
		if err != nil {
			return nil, err
		}
		p := pc.options.GetPacketPool().Get()
		pak = any(p).(*packet.Packet)
		pak.Set2(packetType, packet.MemoryManagementSystemGC, data)
	}
	return pak, nil
}

func (pc *DefaultPacketBuilder) compressAndEncrypt(data []byte) ([]byte, error) {
	var (
		err error
	)
	if pc.compressor != nil {
		data, err = pc.compressor.Compress(data)
		if err != nil {
			return nil, err
		}
	}
	if pc.encrypter != nil {
		data, err = pc.encrypter.Encrypt(data)
		if err != nil {
			return nil, err
		}
	}

	return data, err
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
	if pc.decompressor != nil {
		data, err = pc.decompressor.Decompress(data)
		if err != nil {
			return nil, err
		}
	}
	return data, nil
}

func (pc DefaultPacketBuilder) isNoCompressAndEncryption() bool {
	return pc.compressor == nil && pc.decompressor == nil && pc.encrypter == nil && pc.decrypter == nil
}

func (pc DefaultPacketBuilder) formatHeader(header []byte, dataLen int, pType packet.PacketType) {
	// data length
	header[0] = byte(dataLen >> 16 & 0xff)
	header[1] = byte(dataLen >> 8 & 0xff)
	header[2] = byte(dataLen & 0xff)
	header[3] = byte(pType)                                // packet type
	header[4] = byte(pc.options.GetPacketCompressType())   // compress type
	header[5] = byte(pc.options.GetPacketEncryptionType()) // encryption type
}

func (pc DefaultPacketBuilder) unformatHeader(header []byte) (uint32, packet.PacketType, packet.CompressType, packet.EncryptionType) {
	dataLen := uint32(header[0]) << 16 & 0xff0000
	dataLen += uint32(header[1]) << 8 & 0xff00
	dataLen += uint32(header[2]) & 0xff
	packetType := packet.PacketType(header[3])
	compressType := packet.CompressType(header[4])
	encryptionType := packet.EncryptionType(header[5])
	return dataLen, packetType, compressType, encryptionType
}

func isBasePacket(pakType packet.PacketType) bool {
	return pakType >= packet.PacketHandshake && pakType <= packet.PacketSentAck
}
