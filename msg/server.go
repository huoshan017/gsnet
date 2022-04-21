package msg

import (
	"github.com/huoshan017/gsnet/common"
	"github.com/huoshan017/gsnet/msg/codec"
	"github.com/huoshan017/gsnet/server"
)

// NewMsgSessionHandlerFunc function for creating interface IMsgSessionHandler instance
type NewMsgSessionHandlerFunc func(args ...any) IMsgSessionEventHandler

// MsgServer struct
type MsgServer struct {
	*server.Server
	newFunc NewMsgSessionHandlerFunc
	codec   IMsgCodec
	mapper  *IdMsgMapper
}

// NewMsgServer create new message server
func NewMsgServer(newFunc NewMsgSessionHandlerFunc, funcArgs []any, codec IMsgCodec, mapper *IdMsgMapper, options ...common.Option) *MsgServer {
	s := &MsgServer{
		newFunc: newFunc,
		codec:   codec,
		mapper:  mapper,
	}
	var newSessionHandler server.NewSessionHandlerFunc = func(args ...any) common.ISessionEventHandler {
		msgSessionHandler := s.newFunc(args...)
		ms := newMsgHandlerServer(msgSessionHandler, s.codec, s.mapper)
		return common.ISessionEventHandler(ms)
	}
	if len(funcArgs) > 0 {
		options = append(options, server.WithNewSessionHandlerFuncArgs(funcArgs...))
	}
	s.Server = server.NewServer(newSessionHandler, options...)
	return s
}

// NewProtoBufMsgServer create a protobuf message server
func NewProtoBufMsgServer(newFunc NewMsgSessionHandlerFunc, funcArgs []any, idMsgMapper *IdMsgMapper, options ...common.Option) *MsgServer {
	return NewMsgServer(newFunc, funcArgs, &codec.ProtobufCodec{}, idMsgMapper, options...)
}

// NewJsonMsgServer create a json message server
func NewJsonMsgServerr(newFunc NewMsgSessionHandlerFunc, funcArgs []any, idMsgMapper *IdMsgMapper, options ...common.Option) *MsgServer {
	return NewMsgServer(newFunc, funcArgs, &codec.JsonCodec{}, idMsgMapper, options...)
}

// NewGobMsgServer create a gob message server
func NewGobMsgServer(newFunc NewMsgSessionHandlerFunc, funcArgs []any, idMsgMapper *IdMsgMapper, options ...common.Option) *MsgServer {
	return NewMsgServer(newFunc, funcArgs, &codec.GobCodec{}, idMsgMapper, options...)
}

// NewThriftMsgServer create a thrift message server
func NewThriftMsgServer(newFunc NewMsgSessionHandlerFunc, funcArgs []any, idMsgMapper *IdMsgMapper, options ...common.Option) *MsgServer {
	return NewMsgServer(newFunc, funcArgs, &codec.ThriftCodec{}, idMsgMapper, options...)
}

// NewMsgpackMsgServer create a msgpack message server
func NewMsgpackMsgServer(newFunc NewMsgSessionHandlerFunc, funcArgs []any, idMsgMapper *IdMsgMapper, options ...common.Option) *MsgServer {
	return NewMsgServer(newFunc, funcArgs, &codec.MsgpackCodec{}, idMsgMapper, options...)
}
