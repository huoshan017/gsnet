package common

import (
	"fmt"
	"io"

	"github.com/huoshan017/gsnet/log"
	"github.com/huoshan017/gsnet/options"
	"github.com/huoshan017/gsnet/packet"
	"github.com/huoshan017/gsnet/pool"
)

// packet构建器
type IPacketBuilder interface {
	EncodeWriteTo(packet.PacketType, []byte, io.Writer) error
	EncodeBytesArrayWriteTo(packet.PacketType, [][]byte, io.Writer) error
	EncodeBytesPointerArrayWriteTo(pType packet.PacketType, pBytesArray []*[]byte, writer io.Writer) error
	DecodeReadFrom(io.Reader) (packet.IPacket, error)
}

type BasePacketBuilder struct {
	options          *options.Options
	sendHeaderPacket packet.IPacketHeader
	sendHeaderBuff   []byte
	recvHeaderPacket packet.IPacketHeader
	recvHeaderBuff   []byte
	compressor       packet.ICompressor
	decompressor     packet.IDecompressor
	encrypter        packet.IEncrypter
	decrypter        packet.IDecrypter
	cryptoKey        []byte
}

func newBasePacketBuilder(ops *options.Options) *BasePacketBuilder {
	pb := &BasePacketBuilder{
		options: ops,
	}
	packetHeaderLength := ops.GetPacketHeaderLength()
	if packetHeaderLength <= 0 {
		packetHeaderLength = packet.DefaultPacketHeaderLen
	}
	encryptionType := ops.GetPacketEncryptionType()
	if pb.cryptoKey == nil {
		ran := ops.GetRand() // options.GetRand() 不是线程安全的，不过这里是在同一goroutine中使用，不存在并发安全问题
		fun := ops.GetGenCryptoKeyFunc()
		if fun != nil {
			pb.cryptoKey = fun(ran)
		} else {
			pb.cryptoKey = packet.GenCryptoKeyDefault(encryptionType, ran)
		}
	}
	if pb.Reset(ops.GetPacketCompressType(), encryptionType, pb.cryptoKey) != nil {
		return nil
	}

	pb.sendHeaderBuff = make([]byte, packetHeaderLength)
	pb.recvHeaderBuff = make([]byte, packetHeaderLength)

	return pb
}

func (pb *BasePacketBuilder) Reset(compressType packet.CompressType, encryptionType packet.EncryptionType, key []byte) error {
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

	if pb.sendHeaderPacket == nil {
		createPacketHeaderFunc := pb.options.GetCreatePacketHeaderFunc()
		if createPacketHeaderFunc == nil {
			pb.sendHeaderPacket = NewDefaultPacketHeader(pb.options)
		} else {
			pb.sendHeaderPacket = createPacketHeaderFunc(pb.options)
		}
	} else {
		pb.sendHeaderPacket.SetCompressType(compressType)
		pb.sendHeaderPacket.SetEncryptionType(encryptionType)
	}
	if pb.recvHeaderPacket == nil {
		createPacketHeaderFunc := pb.options.GetCreatePacketHeaderFunc()
		if createPacketHeaderFunc == nil {
			pb.recvHeaderPacket = NewDefaultPacketHeader(pb.options)
		} else {
			pb.recvHeaderPacket = createPacketHeaderFunc(pb.options)
		}
	} else {
		pb.recvHeaderPacket.SetCompressType(compressType)
		pb.recvHeaderPacket.SetEncryptionType(encryptionType)
	}

	return nil
}

func (pc *BasePacketBuilder) Close() {
	if pc.compressor != nil {
		pc.compressor.Close()
		pc.compressor = nil
	}
	if pc.decompressor != nil {
		pc.decompressor.Close()
		pc.decompressor = nil
	}
}

func (pc *BasePacketBuilder) GetCryptoKey() []byte {
	return pc.cryptoKey
}

func (pc *BasePacketBuilder) createEncrypter(encryptionType packet.EncryptionType, key []byte) error {
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

func (pc *BasePacketBuilder) createDecrypter(encryptionType packet.EncryptionType, key []byte) error {
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

func (pc *BasePacketBuilder) createCompressor(compressType packet.CompressType) error {
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

func (pc *BasePacketBuilder) createDecompressor(compressType packet.CompressType) error {
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

func (pc *BasePacketBuilder) compressAndEncrypt(data []byte) ([]byte, error) {
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

func (pc *BasePacketBuilder) decryptAndDecompress(data []byte) ([]byte, error) {
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

func (pc BasePacketBuilder) isNoCompressAndEncryption() bool {
	return pc.compressor == nil && pc.decompressor == nil && pc.encrypter == nil && pc.decrypter == nil
}

func (pc *BasePacketBuilder) encode(ptype packet.PacketType, data []byte) ([]byte, []byte, error) {
	var err error

	// 压缩处理和加密处理
	if !(isBasePacket(ptype) || pc.isNoCompressAndEncryption()) {
		data, err = pc.compressAndEncrypt(data)
		if err != nil {
			return nil, nil, err
		}
	}

	dataLen := len(data)
	pc.sendHeaderPacket.SetType(ptype)
	pc.sendHeaderPacket.SetDataLength(uint32(dataLen))
	err = pc.sendHeaderPacket.FormatTo(pc.sendHeaderBuff[:])
	if err != nil {
		return nil, nil, err
	}

	return pc.sendHeaderBuff[:], data, err
}

func (pc *BasePacketBuilder) encodeBytesArray(ptype packet.PacketType, datas [][]byte) ([]byte, [][]byte, error) {
	var (
		dataLen int
		err     error
	)

	for i := 0; i < len(datas); i++ {
		dataLen += len(datas[i])
	}

	if isBasePacket(ptype) || pc.isNoCompressAndEncryption() {
		pc.sendHeaderPacket.SetType(ptype)
		pc.sendHeaderPacket.SetDataLength(uint32(dataLen))
		err = pc.sendHeaderPacket.FormatTo(pc.sendHeaderBuff[:])
		if err == nil {
			return pc.sendHeaderBuff[:], datas, nil
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
			dataLen = len(data)
			pc.sendHeaderPacket.SetType(ptype)
			pc.sendHeaderPacket.SetDataLength(uint32(dataLen))
			err = pc.sendHeaderPacket.FormatTo(pc.sendHeaderBuff[:])
			if err == nil {
				//_, err = writer.Write(pc.sendHeaderBuff[:])
				//if err == nil {
				//	_, err = writer.Write(data)
				//}
				return pc.sendHeaderBuff[:], [][]byte{data}, nil
			}
		}
	}
	return nil, nil, err
}

func (pc *BasePacketBuilder) encodeBaseBytesPointerArray(ptype packet.PacketType, pBytesArray []*[]byte) ([]byte, [][]byte, error) {
	var (
		dataLen int
		datas   [][]byte
		err     error
	)

	for i := 0; i < len(pBytesArray); i++ {
		dataLen += len(*pBytesArray[i])
		datas = append(datas, *pBytesArray[i])
	}

	pc.sendHeaderPacket.SetType(ptype)
	pc.sendHeaderPacket.SetDataLength(uint32(dataLen))
	err = pc.sendHeaderPacket.FormatTo(pc.sendHeaderBuff[:])
	if err == nil {
		return pc.sendHeaderBuff[:], datas, nil
	}

	return nil, nil, err
}

func (pc *BasePacketBuilder) encodeBytesPointerArray(ptype packet.PacketType, pBytesArray []*[]byte) ([]byte, [][]byte, error) {
	var (
		dataLen int
		err     error
	)

	for i := 0; i < len(pBytesArray); i++ {
		dataLen += len(*pBytesArray[i])
	}
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
		dataLen = len(data)
		pc.sendHeaderPacket.SetType(ptype)
		pc.sendHeaderPacket.SetDataLength(uint32(dataLen))
		err = pc.sendHeaderPacket.FormatTo(pc.sendHeaderBuff[:])
		if err == nil {
			return pc.sendHeaderBuff[:], [][]byte{data}, nil
		}
	}
	return nil, nil, err
}

// builder for type `Packet`
type PacketBuilder struct {
	*BasePacketBuilder
}

func NewPacketBuilder(ops *options.Options) *PacketBuilder {
	return &PacketBuilder{newBasePacketBuilder(ops)}
}

func (pc *PacketBuilder) EncodeWriteTo(pType packet.PacketType, data []byte, writer io.Writer) error {
	var (
		header []byte
		err    error
	)
	header, data, err = pc.encode(pType, data)
	if err == nil {
		_, err = writer.Write(header)
		if err == nil {
			_, err = writer.Write(data)
		}
	}
	return err
}

func (pc *PacketBuilder) EncodeBytesArrayWriteTo(pType packet.PacketType, datas [][]byte, writer io.Writer) error {
	header, datas, err := pc.encodeBytesArray(pType, datas)
	if err != nil {
		return err
	}
	_, err = writer.Write(header)
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

func (pc *PacketBuilder) EncodeBytesPointerArrayWriteTo(pType packet.PacketType, pBytesArray []*[]byte, writer io.Writer) error {
	var (
		header []byte
		datas  [][]byte
		err    error
	)
	if isBasePacket(pType) || pc.isNoCompressAndEncryption() {
		header, datas, err = pc.encodeBaseBytesPointerArray(pType, pBytesArray)
	} else {
		header, datas, err = pc.encodeBytesPointerArray(pType, pBytesArray)
	}
	if err == nil {
		_, err = writer.Write(header)
		if err == nil {
			for i := 0; i < len(datas); i++ {
				_, err = writer.Write(datas[i])
				if err != nil {
					break
				}
			}
		}
	}
	return err
}

func (pc *PacketBuilder) DecodeReadFrom(reader io.Reader) (packet.IPacket, error) {
	// read header
	_, err := io.ReadFull(reader, pc.recvHeaderBuff[:])
	if err != nil {
		return nil, err
	}

	err = pc.recvHeaderPacket.UnformatFrom(pc.recvHeaderBuff[:])
	if err != nil {
		return nil, err
	}

	dataLen := pc.recvHeaderPacket.GetDataLength()

	pData := pool.GetBuffPool().Alloc(int32(dataLen))

	// read data
	_, err = io.ReadFull(reader, *pData)
	if err != nil {
		return nil, err
	}

	var pak *packet.Packet
	packetType := pc.recvHeaderPacket.GetType()
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
		pak.SetGCData(packetType, packet.MemoryManagementSystemGC, data)
	}
	return pak, nil
}

func isBasePacket(pakType packet.PacketType) bool {
	return pakType >= packet.PacketHandshake && pakType <= packet.PacketSentAck
}
