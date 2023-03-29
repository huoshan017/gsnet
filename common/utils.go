package common

import (
	"fmt"
	"net"

	"github.com/huoshan017/gsnet/options"
	"github.com/huoshan017/gsnet/packet"
	"github.com/huoshan017/gsnet/pool"
)

func GetSendData(data any) ([]byte, *[]byte, [][]byte, []*[]byte) {
	var (
		b   []byte
		pb  *[]byte
		ba  [][]byte
		pba []*[]byte
		o   bool
	)
	if b, o = data.([]byte); !o {
		if pb, o = data.(*[]byte); !o {
			if ba, o = data.([][]byte); !o {
				if pba, o = data.([]*[]byte); !o {
					panic("gsnet: wrapper send data type must in []byte, *[]byte, [][]byte and []*[]byte")
				}
			}
		}
	}
	return b, pb, ba, pba
}

func FreeSendData(mmt packet.MemoryManagementType, data any) bool {
	b, pb, ba, pba := GetSendData(data)
	return FreeSendData2(mmt, b, pb, ba, pba)
}

func FreeSendData2(mmt packet.MemoryManagementType, b []byte, pb *[]byte, ba [][]byte, pba []*[]byte) bool {
	if mmt == packet.MemoryManagementPoolUserManualFree {
		if b != nil {
			panic("gsnet: type []byte cant free with pool")
		}
		if ba != nil {
			panic("gsnet: type [][]byte cant free with pool")
		}
		if pb != nil {
			pool.GetBuffPool().Free(pb)
		} else if pba != nil {
			for i := 0; i < len(pba); i++ {
				pool.GetBuffPool().Free(pba[i])
			}
		}
		return true
	}
	return false
}

func Int64ToBuffer(num int64, buffer []byte) {
	for i := 0; i < 8; i++ {
		buffer[i] = byte(num >> (8 * i) & 0xff)
	}
}

func BufferToInt64(buffer []byte) int64 {
	var num int64
	for i := 0; i < 8; i++ {
		num += (int64(buffer[i]) << (8 * i))
	}
	return num
}

func Uint64ToBuffer(num uint64, buffer []byte) {
	for i := 0; i < 8; i++ {
		buffer[i] = byte(num >> (8 * i) & 0xff)
	}
}

func BufferToUint64(buffer []byte) uint64 {
	var num uint64
	for i := 0; i < 8; i++ {
		num += (uint64(buffer[i]) << (8 * i))
	}
	return num
}

func Int32ToBuffer(num int32, buffer []byte) {
	for i := 0; i < 4; i++ {
		buffer[i] = byte(num >> (8 * i) & 0xff)
	}
}

func BufferToInt32(buffer []byte) int32 {
	var num int32
	for i := 0; i < 4; i++ {
		num += (int32(buffer[i]) << (8 * i))
	}
	return num
}

func Uint32ToBuffer(num uint32, buffer []byte) {
	for i := 0; i < 4; i++ {
		buffer[i] = byte(num >> (8 * i) & 0xff)
	}
}

func BufferToUint32(buffer []byte) uint32 {
	var num uint32
	for i := 0; i < 4; i++ {
		num += (uint32(buffer[i]) << (8 * i))
	}
	return num
}

func Int16ToBuffer(num int16, buffer []byte) {
	for i := 0; i < 2; i++ {
		buffer[i] = byte(num >> (8 * i) & 0xff)
	}
}

func BufferToInt16(buffer []byte) int16 {
	var num int16
	for i := 0; i < 2; i++ {
		num += (int16(buffer[i]) << (8 * i))
	}
	return num
}

func Uint16ToBuffer(num uint16, buffer []byte) {
	for i := 0; i < 2; i++ {
		buffer[i] = byte(num >> (8 * i) & 0xff)
	}
}

func BufferToUint16(buffer []byte) uint16 {
	var num uint16
	for i := 0; i < 2; i++ {
		num += (uint16(buffer[i]) << (8 * i))
	}
	return num
}

func IsTimeoutError(err error) bool {
	if net_err, ok := err.(net.Error); ok && net_err.Timeout() {
		return true
	}
	return false
}

func NetProto2Network(proto options.NetProto) string {
	switch proto {
	case options.NetProtoTCP:
		return "tcp"
	case options.NetProtoTCP4:
		return "tcp4"
	case options.NetProtoTCP6:
		return "tcp6"
	case options.NetProtoUDP:
		return "udp"
	case options.NetProtoUDP4:
		return "udp4"
	case options.NetProtoUDP6:
		return "udp6"
	default:
		panic(fmt.Sprintf("gsnet: unsupported network protocol %v", proto))
	}
}
