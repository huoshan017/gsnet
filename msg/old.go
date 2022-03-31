package msg

type IMsgDecoder interface {
	Encode(msgid uint32, msg []byte) []byte
	Decode(data []byte) (msgid uint32, msg []byte)
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
