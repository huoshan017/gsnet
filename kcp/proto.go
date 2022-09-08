package kcp

import "github.com/huoshan017/gsnet/common"

const (
	defaultMtu           = 1400
	maxFrameHeaderLength = 13
)

type frameHeader struct {
	frm    uint8
	convId uint32
	token  int64
}

func encodeFrameHeader(header *frameHeader, buf []byte) int32 {
	var count int32
	length := int32(len(buf))
	if length > 0 {
		buf[0] = header.frm
		count += 1
		if length > 4 {
			common.Uint32ToBuffer(header.convId, buf[1:])
			count += 4
			if length > 4+8 {
				common.Int64ToBuffer(header.token, buf[5:])
				count += 8
			}
		}
	}
	return count
}

func decodeFrameHeader(buf []byte, header *frameHeader) int32 {
	var count int32
	length := int32(len(buf))
	if length > 0 {
		header.frm = buf[0]
		count += 1
		if length > 4 {
			header.convId = common.BufferToUint32(buf[1:])
			count += 4
			if length > 8+4 {
				header.token = common.BufferToInt64(buf[5:])
				count += 8
			}
		}
	}
	return count
}
