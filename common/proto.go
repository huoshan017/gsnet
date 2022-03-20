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

type IMsgDecoder interface {
	Encode(msgid uint32, msg []byte) []byte
	Decode(data []byte) (msgid uint32, msg []byte)
}

// 默认数据协议
type DefaultDataProto struct {
}

func (p DefaultDataProto) GetHeaderLen() uint8 {
	return 4
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
	bh := make([]byte, 4)
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

// 默认消息协议
type DefaultMsgDecoder struct {
}

func (p DefaultMsgDecoder) Encode(msgid uint32, msg []byte) (data []byte) {
	data = make([]byte, 4+len(msg))
	for i := 0; i < 4; i++ {
		data[i] = byte(msgid >> (8 * (4 - i - 1)))
	}
	copy(data[4:], msg[:])
	return
}

func (p DefaultMsgDecoder) Decode(data []byte) (msgid uint32, msg []byte) {
	for i := 0; i < 4; i++ {
		msgid += uint32(data[i] << (8 * (4 - i - 1)))
	}
	msg = data[4:]
	return
}
