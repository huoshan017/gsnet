package codec

import (
	"bytes"
	"encoding/gob"
)

type GobCodec struct{}

func NewGobCodec() *GobCodec {
	return &GobCodec{}
}

func (c GobCodec) Encode(i any) ([]byte, error) {
	var buffer bytes.Buffer
	encoder := gob.NewEncoder(&buffer)
	err := encoder.Encode(i)
	if err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func (c GobCodec) Decode(d []byte, i any) error {
	decoder := gob.NewDecoder(bytes.NewReader(d))
	return decoder.Decode(i)
}
