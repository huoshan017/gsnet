package common

import (
	"reflect"

	"github.com/huoshan017/gsnet/msg"
	"github.com/huoshan017/gsnet/test/tproto"
)

const (
	GateAddress = "127.0.0.1:9900"
)

var (
	BackendAddress = []string{"127.0.0.1:9901", "127.0.0.1:9902", "127.0.0.1:9903"}
)

const (
	MsgIdPing    = msg.MsgIdType(1)
	MsgIdPong    = msg.MsgIdType(2)
	SendListMode = 0
	SendCount    = 5000
	ClientNum    = 5000
)

var (
	Ch          = make(chan struct{})
	IdMsgMapper *msg.IdMsgMapper
)

func init() {
	IdMsgMapper = msg.CreateIdMsgMapperWith(map[msg.MsgIdType]reflect.Type{
		MsgIdPing: reflect.TypeOf(&tproto.MsgPing{}),
		MsgIdPong: reflect.TypeOf(&tproto.MsgPong{}),
	})
}
