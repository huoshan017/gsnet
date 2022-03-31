package common

type IDataProto interface {
	GetHeaderLen() uint8
	GetBodyLen(header []byte) uint32
	EncodeBodyLen([]byte) []byte
	Compress([]byte) []byte
	Decompress([]byte) ([]byte, bool)
	Encrypt([]byte) []byte
	Decrypt([]byte) ([]byte, bool)
}

// 默认数据协议
type DefaultDataProto struct {
}

func (p DefaultDataProto) GetHeaderLen() uint8 {
	return 3
}

func (p DefaultDataProto) GetBodyLen(header []byte) uint32 {
	l := uint32(header[0]) << 16 & 0xff0000
	l += uint32(header[1]) << 8 & 0xff00
	l += uint32(header[2]) & 0xff
	return l
}

func (p DefaultDataProto) EncodeBodyLen(data []byte) []byte {
	dl := len(data)
	// todo 用内存池优化
	bh := make([]byte, 3)
	bh[0] = byte(dl >> 16 & 0xff)
	bh[1] = byte(dl >> 8 & 0xff)
	bh[2] = byte(dl & 0xff)
	return bh
}

func (p DefaultDataProto) Compress(data []byte) []byte {
	return data
}

func (p DefaultDataProto) Decompress(data []byte) ([]byte, bool) {
	return data, true
}

func (p DefaultDataProto) Encrypt(data []byte) []byte {
	return data
}

func (p DefaultDataProto) Decrypt(data []byte) ([]byte, bool) {
	return data, true
}
