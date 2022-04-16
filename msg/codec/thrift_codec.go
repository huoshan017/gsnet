package codec

import (
	thrifter "github.com/thrift-iterator/go"
)

type ThriftCodec struct {
}

func NewThriftCodec() *ThriftCodec {
	return &ThriftCodec{}
}

func (c *ThriftCodec) Encode(i any) ([]byte, error) {
	return thrifter.Marshal(i)
}

func (c *ThriftCodec) Decode(d []byte, i any) error {
	return thrifter.Unmarshal(d, i)
}
