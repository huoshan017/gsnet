package common

import (
	"github.com/huoshan017/gsnet/packet"
	"github.com/huoshan017/gsnet/protocol"
)

type CreatePacketHeaderFunc func(*Options) packet.IPacketHeader

type DefaultPacketHeader struct {
	packet.CommonPacketHeader
}

const (
	usePB = false
)

func NewDefaultPacketHeader(options *Options) packet.IPacketHeader {
	header := &DefaultPacketHeader{
		CommonPacketHeader: packet.CommonPacketHeader{},
	}
	header.SetCompressType(options.GetPacketCompressType())
	header.SetEncryptionType(options.GetPacketEncryptionType())
	return header
}

func (ph *DefaultPacketHeader) GetValue(string) any {
	return nil
}

func (ph *DefaultPacketHeader) SetValue(string, any) {
}

func (ph *DefaultPacketHeader) FormatTo(buf []byte) error {
	if len(buf) < packet.DefaultPacketHeaderLen {
		return packet.ErrHeaderLengthTooSmall
	}
	if usePB {
		var header protocol.PacketHeader
		header.PacketType = int32(ph.Type)
		header.CompressType = int32(ph.CType)
		header.EncryptionType = int32(ph.EType)
		header.PayloadLength = int32(ph.DataLen)
		_, err := header.MarshalToSizedBuffer(buf)
		if err != nil {
			return err
		}
	} else {
		buf[0] = byte(ph.DataLen >> 16 & 0xff)
		buf[1] = byte(ph.DataLen >> 8 & 0xff)
		buf[2] = byte(ph.DataLen & 0xff)
		buf[3] = byte(ph.Type)  // packet type
		buf[4] = byte(ph.CType) // compress type
		buf[5] = byte(ph.EType) // encryption type
	}
	return nil
}

func (ph *DefaultPacketHeader) UnformatFrom(buf []byte) error {
	if usePB {
		var header protocol.PacketHeader
		if err := header.Unmarshal(buf); err != nil {
			return err
		}
		ph.Type = packet.PacketType(header.PacketType)
		ph.CType = packet.CompressType(header.CompressType)
		ph.EType = packet.EncryptionType(header.EncryptionType)
		ph.DataLen = uint32(header.PayloadLength)
	} else {
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
	}

	return nil
}
