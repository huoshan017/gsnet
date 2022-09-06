package common

import (
	"github.com/huoshan017/gsnet/options"
	"github.com/huoshan017/gsnet/packet"
)

type IPacketCodec interface {
	Encode(packet.PacketType, []byte) ([]byte, []byte, error)
	EncodeBytesArray(packet.PacketType, [][]byte) ([]byte, [][]byte, error)
	EncodeBytesPointerArray(pType packet.PacketType, pBytesArray []*[]byte) ([]byte, [][]byte, error)
	Decode(bytesList packet.IBytesList) (packet.IPacket, error)
}

type PacketCodec struct {
	*BasePacketBuilder
	readHeader bool
}

func NewPacketCodec(ops *options.Options) *PacketCodec {
	return &PacketCodec{
		BasePacketBuilder: newBasePacketBuilder(ops),
	}
}

func (pc *PacketCodec) Encode(pType packet.PacketType, buf []byte) ([]byte, []byte, error) {
	return pc.encode(pType, buf)
}

func (pc *PacketCodec) EncodeBytesArray(pType packet.PacketType, bytesArray [][]byte) ([]byte, [][]byte, error) {
	return pc.encodeBytesArray(pType, bytesArray)
}

func (pc *PacketCodec) EncodeBytesPointerArray(pType packet.PacketType, pBytesArray []*[]byte) ([]byte, [][]byte, error) {
	if isBasePacket(pType) || pc.isNoCompressAndEncryption() {
		return pc.encodeBaseBytesPointerArray(pType, pBytesArray)
	} else {
		return pc.encodeBytesPointerArray(pType, pBytesArray)
	}
}

func (pc *PacketCodec) Decode(bytesList packet.IBytesList) (packet.IPacket, error) {
	// read header
	if !pc.readHeader {
		if !bytesList.ReadBytesTo(pc.recvHeaderBuff[:], true) {
			return nil, nil
		}
		pc.readHeader = true
	}

	err := pc.recvHeaderPacket.UnformatFrom(pc.recvHeaderBuff[:])
	if err != nil {
		return nil, err
	}

	dataLen := pc.recvHeaderPacket.GetDataLength()

	// read data
	rbytes, ok := bytesList.ReadBytes(int32(dataLen), true)
	if !ok {
		return nil, nil
	}

	var pak packet.IPacket
	packetType := pc.recvHeaderPacket.GetType()
	if isBasePacket(packetType) || pc.isNoCompressAndEncryption() {
		if rbytes.IsShared() {
			var p = pc.options.GetPacketPool().GetWithType(packet.PoolPacketShared)
			var spak = p.(*packet.SharedPacket)
			spak.Init(packetType, packet.MemoryManagementPoolUserManualFree, &rbytes)
			pak = spak
		} else {
			var p = pc.options.GetPacketPool().Get()
			var ppak = p.(*packet.Packet)
			ppak.Set(packetType, packet.MemoryManagementPoolFrameworkFree, rbytes.PData())
			pak = ppak
		}
	} else {
		// 先解密再解压
		var data []byte
		data, err = pc.decryptAndDecompress(rbytes.Data())
		rbytes.Release(true)
		if err != nil {
			return nil, err
		}
		var p = pc.options.GetPacketPool().Get()
		var ppak = p.(*packet.Packet)
		ppak.SetGCData(packetType, packet.MemoryManagementSystemGC, data)
		pak = ppak
	}

	pc.readHeader = false
	return pak, nil
}
