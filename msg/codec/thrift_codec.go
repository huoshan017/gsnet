package codec

import (
	"github.com/thrift-iterator/go"
)

type ThriftCodec struct {
}

func NewThriftCodec() *ThriftCodec {
	return &ThriftCodec{}
}

func (c *ThriftCodec) Encode(i interface{}) ([]byte, error) {
	return thrifter.Marshal(i)
}

func (c *ThriftCodec) Decode(d []byte, i interface{}) error {
	return thrifter.Unmarshal(d, i)
}
