package common

import (
	"reflect"

	"github.com/huoshan017/gsnet/msg"
	"github.com/huoshan017/gsnet/test/tproto"
)

const (
	MsgAgentClientName    = "MsgAgentClient"
	TestAddress           = "127.0.0.1:9000"
	MsgAgentServerAddress = "127.0.0.1:9001"
)

const (
	MsgIdPing    = msg.MsgIdType(1)
	MsgIdPong    = msg.MsgIdType(2)
	SendListMode = 1
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
