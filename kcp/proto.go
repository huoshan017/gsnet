package kcp

import (
	"bytes"

	"github.com/huoshan017/gsnet/common"
)

const (
	FRAME_SYN     = 0
	FRAME_SYN_ACK = 1
	FRAME_ACK     = 2
	FRAME_FIN     = 3
	FRAME_RST     = 4
)

const (
	STATE_CLOSED      int32 = 0
	STATE_LISTENING   int32 = 1
	STATE_SYN_RECV    int32 = 2
	STATE_SYN_SEND    int32 = 3
	STATE_ESTABLISHED int32 = 4
	STATE_FIN_WAIT_1  int32 = 5
	STATE_CLOSE_WAIT  int32 = 6
	STATE_FIN_WAIT_2  int32 = 7
	STATE_LAST_ACK    int32 = 8
	STATE_TIME_WAIT   int32 = 9
)

const (
	defaultMtu           = 1400
	maxFrameHeaderLength = 4 + 1 + 4 + 8 + 4
)

var (
	framePrefix = []byte("0xbf")
	frameSuffix = []byte("0xef")
)

type frameHeader struct {
	frm    uint8
	convId uint32
	token  int64
}

func encodeFrameHeader(header *frameHeader, buf []byte) int32 {
	var count int32
	copy(buf, framePrefix)
	count += int32(len(framePrefix))
	buf[count] = header.frm
	count += 1
	common.Uint32ToBuffer(header.convId, buf[count:])
	count += 4
	common.Int64ToBuffer(header.token, buf[count:])
	count += 8
	copy(buf[count:], frameSuffix)
	count += int32(len(frameSuffix))
	return count
}

func decodeFrameHeader(buf []byte, header *frameHeader) int32 {
	var count int32
	if !bytes.Equal(buf[:len(framePrefix)], framePrefix) {
		return -1
	}
	count += int32(len(framePrefix))
	header.frm = buf[count]
	count += 1
	header.convId = common.BufferToUint32(buf[count:])
	count += 4
	header.token = common.BufferToInt64(buf[count:])
	count += 8
	if !bytes.Equal(buf[count:count+int32(len(frameSuffix))], frameSuffix) {
		return -2
	}
	count += int32(len(frameSuffix))
	return count
}

var (
	timeoutSynAckTimeList []int32 = []int32{3, 7, 15}        // synack超时列表
	timeoutAckTimeList    []int32 = []int32{20, 40, 80, 160} // ack超时列表
)
