package common

import (
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
