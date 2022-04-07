package common

import (
	"github.com/huoshan017/gsnet/common/packet"
	"github.com/huoshan017/gsnet/common/pool"
)

func GetSendData(data interface{}) ([]byte, *[]byte, [][]byte, []*[]byte) {
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

func FreeSendData(mmt packet.MemoryManagementType, data interface{}) bool {
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
