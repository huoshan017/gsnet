package msg

import (
	"time"

	"github.com/huoshan017/gsnet/common"
	"github.com/huoshan017/gsnet/server"
)

// MsgServer struct
type MsgServer struct {
	*server.Server
	handler *msgHandler
}

// NewMsgServer create new message server
func NewMsgServer(msgCodec IMsgCodec, idMsgMapper *IdMsgMapper, options ...common.Option) *MsgServer {
	s := &MsgServer{}
	handler := newMsgHandler(msgCodec, idMsgMapper)
	s.Server = server.NewServerWithHandler(handler, options...)
	s.handler = handler
	return s
}

// MsgServer.SetConnectHandle set connected handle with session
func (c *MsgServer) SetConnectHandle(handle func(common.ISession)) {
	c.handler.SetConnectHandle(handle)
}

// MsgServer.SetDisconnectHandle set disconnect handle with session and error
func (c *MsgServer) SetDisconnectHandle(handle func(common.ISession, error)) {
	c.handler.SetDisconnectHandle(handle)
}

// MsgServer.SetTickHandle set tick timer handle with session
func (c *MsgServer) SetTickHandle(handle func(common.ISession, time.Duration)) {
	c.handler.SetTickHandle(handle)
}

// MsgServer.SetErrorHandle set error handle
func (c *MsgServer) SetErrorHandle(handle func(error)) {
	c.handler.SetErrorHandle(handle)
}

// MsgServer.RegisterMsgHandle register handle for msg id
func (c *MsgServer) RegisterMsgHandle(msgid MsgIdType, handle func(common.ISession, interface{}) error) {
	c.handler.RegisterHandle(msgid, handle)
}

// MsgServer.Send send to session a message
func (c *MsgServer) Send(sess common.ISession, msgid MsgIdType, msg interface{}) error {
	return c.handler.SendMsg(sess, msgid, msg)
}

// MsgServer.SendNoCopy send to session a message with no copy data inside
func (c *MsgServer) SendNoCopy(sess common.ISession, msgid MsgIdType, msg interface{}) error {
	return c.handler.SendMsgNoCopy(sess, msgid, msg)
}

// NewPBMsgServer create a protobuf message server
func NewPBMsgServer(idMsgMapper *IdMsgMapper, options ...common.Option) *MsgServer {
	return NewMsgServer(&ProtobufCodec{}, idMsgMapper, options...)
}

// NewJsonMsgServer create a json message server
func NewJsonMsgServer(idMsgMapper *IdMsgMapper, options ...common.Option) *MsgServer {
	return NewMsgServer(&JsonCodec{}, idMsgMapper, options...)
}
