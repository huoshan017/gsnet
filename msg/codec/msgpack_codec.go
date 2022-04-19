package codec

import "github.com/vmihailenco/msgpack/v5"

type MsgpackCodec struct {
}

func NewMsgpackCodec() *MsgpackCodec {
	return &MsgpackCodec{}
}

func (c *MsgpackCodec) Encode(i any) ([]byte, error) {
	return msgpack.Marshal(i)
}

func (c *MsgpackCodec) Decode(d []byte, i any) error {
	return msgpack.Unmarshal(d, i)
}
