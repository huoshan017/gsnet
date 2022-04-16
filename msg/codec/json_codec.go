package codec

import "encoding/json"

type JsonCodec struct{}

func NewJsonCodec() *JsonCodec {
	return &JsonCodec{}
}

func (c JsonCodec) Encode(i any) ([]byte, error) {
	return json.Marshal(i)
}

func (c JsonCodec) Decode(d []byte, i any) error {
	return json.Unmarshal(d, i)
}
