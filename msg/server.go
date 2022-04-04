package msg

import (
	"github.com/huoshan017/gsnet/common"
	"github.com/huoshan017/gsnet/msg/codec"
	"github.com/huoshan017/gsnet/server"
)

// NewMsgSessionHandlerFunc function for creating interface IMsgSessionHandler instance
type NewMsgSessionHandlerFunc func(args ...interface{}) IMsgSessionEventHandler

// MsgServer struct
type MsgServer struct {
	*server.Server
	newFunc NewMsgSessionHandlerFunc
	codec   IMsgCodec
	mapper  *IdMsgMapper
}

// NewMsgServer create new message server
func NewMsgServer(newFunc NewMsgSessionHandlerFunc, codec IMsgCodec, mapper *IdMsgMapper, options ...common.Option) *MsgServer {
	s := &MsgServer{
		newFunc: newFunc,
		codec:   codec,
		mapper:  mapper,
	}
	var newSessionHandler server.NewSessionHandlerFunc = func(args ...interface{}) common.ISessionEventHandler {
		msgSessionHandler := s.newFunc(args...)
		proxy := newMsgHandlerServerProxy(msgSessionHandler, s.codec, s.mapper)
		return common.ISessionEventHandler(proxy)
	}
	s.Server = server.NewServer(newSessionHandler, options...)
	return s
}

// NewPBMsgServer create a protobuf message server
func NewPBMsgServer(newFunc NewMsgSessionHandlerFunc, idMsgMapper *IdMsgMapper, options ...common.Option) *MsgServer {
	return NewMsgServer(newFunc, &codec.ProtobufCodec{}, idMsgMapper, options...)
}

// NewJsonMsgServer create a json message server
func NewJsonMsgServerr(newFunc NewMsgSessionHandlerFunc, idMsgMapper *IdMsgMapper, options ...common.Option) *MsgServer {
	return NewMsgServer(newFunc, &codec.JsonCodec{}, idMsgMapper, options...)
}

// NewGobMsgServer create a gob message server
func NewGobMsgServer(newFunc NewMsgSessionHandlerFunc, idMsgMapper *IdMsgMapper, options ...common.Option) *MsgServer {
	return NewMsgServer(newFunc, &codec.GobCodec{}, idMsgMapper, options...)
}

// NewThriftMsgServer create a thrift message server
func NewThriftMsgServer(newFunc NewMsgSessionHandlerFunc, idMsgMapper *IdMsgMapper, options ...common.Option) *MsgServer {
	return NewMsgServer(newFunc, &codec.ThriftCodec{}, idMsgMapper, options...)
}
