package kcp

import (
	"bytes"
	"fmt"
	"reflect"
	"sync"
	"unsafe"

	"github.com/huoshan017/gsnet/common"
	"github.com/huoshan017/gsnet/log"
)

const (
	FRAME_SYN     = 0
	FRAME_SYN_ACK = 1
	FRAME_ACK     = 2
	FRAME_FIN     = 3
	FRAME_RST     = 4
	FRAME_DATA    = 10
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
	framePrefix                   = "0xbf"
	frameSuffix                   = "0xef"
	framePrefixAndHeaderLength    = len(framePrefix) + 1 + 4 + 8 + 2
	frameSuffixLength             = len(frameSuffix)
	framePrefixHeaderSuffixLength = len(framePrefix) + 1 + 4 + 8 + 2 + len(frameSuffix)
	defaultMtu                    = 1400 - framePrefixHeaderSuffixLength // 減掉的長度是幀頭去掉會話ID
)

var (
	framePrefixBytes = []byte(framePrefix)
	frameSuffixBytes = []byte(frameSuffix)
)

var (
	timeoutSynAckTimeList []int32 = []int32{3, 7, 15}        // synack超时列表
	timeoutAckTimeList    []int32 = []int32{20, 40, 80, 160} // ack超时列表
)

type frameHeader struct {
	frm     uint8
	dataLen uint16
	convId  uint32
	token   int64
}

func checkFrameValid(frm int) bool {
	return frm == FRAME_SYN || frm == FRAME_ACK || frm == FRAME_SYN_ACK || frm == FRAME_FIN || frm == FRAME_RST || frm == FRAME_DATA
}

func isHandshakeFrame(frm int) bool {
	return frm == FRAME_SYN || frm == FRAME_ACK || frm == FRAME_SYN_ACK
}

func isDisconnectFrame(frm int) bool {
	return frm == FRAME_FIN
}

func isDataFrame(frm int) bool {
	return frm == FRAME_DATA
}

func encodeFramePreffixHeader(header *frameHeader, buf []byte) {
	var count int32
	copy(buf, framePrefixBytes)
	count += int32(len(framePrefixBytes))
	buf[count] = header.frm
	count += 1
	common.Uint32ToBuffer(header.convId, buf[count:])
	count += 4
	common.Int64ToBuffer(header.token, buf[count:])
	count += 8
	if header.dataLen > 0 {
		common.Uint16ToBuffer(header.dataLen, buf[count:])
		count += 2
	}
}

func encodeFrameSuffix(buf []byte) {
	copy(buf, frameSuffixBytes)
}

func encodeFrameHeader(header *frameHeader, buf []byte) int32 {
	encodeFramePreffixHeader(header, buf)
	copy(buf[framePrefixAndHeaderLength:], frameSuffixBytes)
	return int32(framePrefixAndHeaderLength + len(frameSuffixBytes))
}

func decodeFramePreffixHeader(buf []byte, header *frameHeader) int32 {
	var count int32
	if !bytes.Equal(buf[:len(framePrefixBytes)], framePrefixBytes) {
		return -1
	}
	count += int32(len(framePrefixBytes))
	header.frm = buf[count]
	if !checkFrameValid(int(header.frm)) {
		return -2
	}
	count += 1
	header.convId = common.BufferToUint32(buf[count:])
	count += 4
	header.token = common.BufferToInt64(buf[count:])
	count += 8
	header.dataLen = common.BufferToUint16(buf[count:])
	count += 2
	return count
}

func checkDecodeFrame(buf []byte, header *frameHeader) int32 {
	count := decodeFramePreffixHeader(buf, header)
	if count < 0 {
		return count
	}
	if !bytes.Equal(buf[count+int32(header.dataLen):], frameSuffixBytes) {
		log.Infof("buf: %v   compare: %v ---- %v", buf, buf[count+int32(header.dataLen)], frameSuffixBytes)
		return -3
	}
	return 1
}

func toFrameBuf(buf []byte) []byte {
	sh := (*reflect.SliceHeader)(unsafe.Pointer(&buf))
	sh.Data -= uintptr(framePrefixAndHeaderLength)
	sh.Cap += framePrefixAndHeaderLength
	sh.Len += framePrefixHeaderSuffixLength
	return buf
}

func encodeDataFrame(buf []byte, header *frameHeader) []byte {
	dataLen := len(buf)
	buf = toFrameBuf(buf)
	// 前綴和幀頭
	header.dataLen = uint16(dataLen)
	encodeFramePreffixHeader(header, buf)
	// 後綴
	copy(buf[framePrefixAndHeaderLength+dataLen:], frameSuffixBytes)
	return buf
}

var (
	stepSize  = []int32{64, 128, 256, 384, 512, 768, 1024, 1280, 1536}
	stepPools []*sync.Pool
	stored    sync.Map
)

func init() {
	stepPools = make([]*sync.Pool, len(stepSize))
	for i := 0; i < len(stepSize); i++ {
		s := stepSize[i]
		stepPools[i] = &sync.Pool{
			New: func() any {
				buf := make([]byte, s)
				return buf
			},
		}
	}
}

func _getBuffer(s int32, store bool) []byte {
	for i := 0; i < len(stepSize); i++ {
		if stepSize[i] >= s {
			b := stepPools[i].Get().([]byte)
			if store {
				pb := (*reflect.SliceHeader)(unsafe.Pointer(&b)).Data
				stored.Store(pb, true)
			}
			return b
		}
	}
	return nil
}

func _putBuffer(b []byte, store bool) {
	c := cap(b)
	found := false
	for i := 0; i < len(stepSize); i++ {
		if c == int(stepSize[i]) {
			if store {
				pb := (*reflect.SliceHeader)(unsafe.Pointer(&b)).Data
				if _, o := stored.LoadAndDelete(pb); !o {
					panic(fmt.Sprintf("gsnet(kcp): not store buffer %p: %v, cant put buffer", b, b))
				}
			}
			stepPools[i].Put(b)
			found = true
			break
		}
	}
	if !found {
		panic(fmt.Sprintf("gsnet(kcp): not found buffer %v", b))
	}
}

func getKcpMtuBuffer(s int32) []byte {
	b := _getBuffer(s+int32(framePrefixHeaderSuffixLength), true)
	return b[framePrefixAndHeaderLength:]
}

func putKcpMtuBuffer(b []byte) {
	sh := (*reflect.SliceHeader)(unsafe.Pointer(&b))
	sh.Data -= uintptr(framePrefixAndHeaderLength)
	sh.Cap += framePrefixAndHeaderLength
	sh.Len = sh.Cap
	_putBuffer(b, true)
}
