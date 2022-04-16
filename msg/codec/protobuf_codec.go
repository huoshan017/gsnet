package codec

import (
	"github.com/gogo/protobuf/proto"
)

type ProtobufCodec struct{}

func NewProtobufCodec() *ProtobufCodec {
	return &ProtobufCodec{}
}

func (c *ProtobufCodec) Encode(i any) ([]byte, error) {
	return proto.Marshal(i.(proto.Message))
}

func (c *ProtobufCodec) Decode(d []byte, i any) error {
	return proto.Unmarshal(d, i.(proto.Message))
}
