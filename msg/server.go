package msg

import (
	"github.com/huoshan017/gsnet/common"
	"github.com/huoshan017/gsnet/msg/codec"
	"github.com/huoshan017/gsnet/options"
	"github.com/huoshan017/gsnet/server"
)

// NewMsgSessionHandlerFunc function for creating interface IMsgSessionHandler instance
type NewMsgSessionHandlerFunc func(args ...any) IMsgSessionEventHandler

// MsgServer struct
type MsgServer struct {
	*server.Server
	options MsgServerOptions
}

func prepareMsgServer(newFunc NewMsgSessionHandlerFunc, funcArgs []any, codec IMsgCodec, mapper *IdMsgMapper, msgServerOptions *MsgServerOptions, options ...options.Option) func(args ...any) common.ISessionHandler {
	for i := 0; i < len(options); i++ {
		options[i](&msgServerOptions.Options)
	}
	if len(funcArgs) > 0 {
		msgServerOptions.SetNewSessionHandlerFuncArgs(funcArgs...)
	}
	var newSessionHandlerFunc server.NewSessionHandlerFunc = func(args ...any) common.ISessionHandler {
		msgSessionHandler := newFunc(args...)
		ms := newMsgHandlerServer(msgSessionHandler, codec, mapper, &msgServerOptions.MsgOptions)
		return common.ISessionHandler(ms)
	}
	return newSessionHandlerFunc
}

// NewMsgServer create new message server
func NewMsgServer(newFunc NewMsgSessionHandlerFunc, funcArgs []any, codec IMsgCodec, mapper *IdMsgMapper, options ...options.Option) *MsgServer {
	s := &MsgServer{}
	newSessionHandlerFunc := prepareMsgServer(newFunc, funcArgs, codec, mapper, &s.options, options...)
	s.Server = server.NewServerWithOptions(newSessionHandlerFunc, &s.options.ServerOptions)
	return s
}

// NewProtobufMsgServer create a protobuf message server
func NewProtobufMsgServer(newFunc NewMsgSessionHandlerFunc, funcArgs []any, idMsgMapper *IdMsgMapper, options ...options.Option) *MsgServer {
	return NewMsgServer(newFunc, funcArgs, &codec.ProtobufCodec{}, idMsgMapper, options...)
}

// NewJsonMsgServer create a json message server
func NewJsonMsgServerr(newFunc NewMsgSessionHandlerFunc, funcArgs []any, idMsgMapper *IdMsgMapper, options ...options.Option) *MsgServer {
	return NewMsgServer(newFunc, funcArgs, &codec.JsonCodec{}, idMsgMapper, options...)
}

// NewGobMsgServer create a gob message server
func NewGobMsgServer(newFunc NewMsgSessionHandlerFunc, funcArgs []any, idMsgMapper *IdMsgMapper, options ...options.Option) *MsgServer {
	return NewMsgServer(newFunc, funcArgs, &codec.GobCodec{}, idMsgMapper, options...)
}

// NewThriftMsgServer create a thrift message server
func NewThriftMsgServer(newFunc NewMsgSessionHandlerFunc, funcArgs []any, idMsgMapper *IdMsgMapper, options ...options.Option) *MsgServer {
	return NewMsgServer(newFunc, funcArgs, &codec.ThriftCodec{}, idMsgMapper, options...)
}

// NewMsgpackMsgServer create a msgpack message server
func NewMsgpackMsgServer(newFunc NewMsgSessionHandlerFunc, funcArgs []any, idMsgMapper *IdMsgMapper, options ...options.Option) *MsgServer {
	return NewMsgServer(newFunc, funcArgs, &codec.MsgpackCodec{}, idMsgMapper, options...)
}
