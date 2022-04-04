package codec

import "encoding/json"

type JsonCodec struct{}

func NewJsonCodec() *JsonCodec {
	return &JsonCodec{}
}

func (c JsonCodec) Encode(i interface{}) ([]byte, error) {
	return json.Marshal(i)
}

func (c JsonCodec) Decode(d []byte, i interface{}) error {
	return json.Unmarshal(d, i)
}
