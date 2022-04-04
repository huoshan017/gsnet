package codec

import (
	"github.com/gogo/protobuf/proto"
)

type ProtobufCodec struct{}

func NewProtobufCodec() *ProtobufCodec {
	return &ProtobufCodec{}
}

func (c *ProtobufCodec) Encode(i interface{}) ([]byte, error) {
	return proto.Marshal(i.(proto.Message))
}

func (c *ProtobufCodec) Decode(d []byte, i interface{}) error {
	return proto.Unmarshal(d, i.(proto.Message))
}
