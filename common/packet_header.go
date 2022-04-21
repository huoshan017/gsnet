package common

import "github.com/huoshan017/gsnet/packet"

type CreateHeaderPacketFunc func(*Options) packet.IPacketHeader

type DefaultPacketHeader struct {
	packet.CommonPacketHeader
}

func NewDefaultPacketHeader(options *Options) packet.IPacketHeader {
	header := &DefaultPacketHeader{
		CommonPacketHeader: packet.CommonPacketHeader{},
	}
	header.SetCompressType(options.GetPacketCompressType())
	header.SetEncryptionType(options.GetPacketEncryptionType())
	return header
}

func (ph *DefaultPacketHeader) Get(key string) any {
	return nil
}

func (ph *DefaultPacketHeader) Set(key string, data any) {
}

func (ph *DefaultPacketHeader) FormatTo(buf []byte) error {
	if len(buf) < packet.DefaultPacketHeaderLen {
		return packet.ErrHeaderLengthTooSmall
	}
	buf[0] = byte(ph.DataLen >> 16 & 0xff)
	buf[1] = byte(ph.DataLen >> 8 & 0xff)
	buf[2] = byte(ph.DataLen & 0xff)
	buf[3] = byte(ph.Type)  // packet type
	buf[4] = byte(ph.CType) // compress type
	buf[5] = byte(ph.EType) // encryption type
	return nil
}

func (ph *DefaultPacketHeader) UnformatFrom(buf []byte) error {
	dataLen := uint32(buf[0]) << 16 & 0xff0000
	dataLen += uint32(buf[1]) << 8 & 0xff00
	dataLen += uint32(buf[2]) & 0xff
	if dataLen > packet.MaxPacketLength {
		return packet.ErrBodyLengthTooLong
	}
	packetType := packet.PacketType(buf[3])
	compressType := packet.CompressType(buf[4])
	encryptionType := packet.EncryptionType(buf[5])
	ph.DataLen = dataLen
	ph.Type = packetType
	ph.CType = compressType
	ph.EType = encryptionType
	return nil
}
