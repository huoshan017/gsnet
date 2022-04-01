package msg

import (
	"time"

	"github.com/huoshan017/gsnet/client"
	"github.com/huoshan017/gsnet/common"
)

// MsgClient struct
type MsgClient struct {
	*client.Client
	handler *msgHandler
}

// NewMsgClient create a message client
func NewMsgClient(msgCodec IMsgCodec, idMsgMapper *IdMsgMapper, options ...common.Option) *MsgClient {
	handler := newMsgHandler(msgCodec, idMsgMapper)
	c := &MsgClient{
		Client:  client.NewClient(handler, options...),
		handler: handler,
	}
	return c
}

// MsgClient.SetConnectHandle set connected handle with session
func (c *MsgClient) SetConnectHandle(handle func(common.ISession)) {
	c.handler.SetConnectHandle(handle)
}

// MsgClient.SetDisconnectHandle set disconnected handle with session
func (c *MsgClient) SetDisconnectHandle(handle func(common.ISession, error)) {
	c.handler.SetDisconnectHandle(handle)
}

// MsgClient.SetTickHandle set tick timer handle with session
func (c *MsgClient) SetTickHandle(handle func(common.ISession, time.Duration)) {
	c.handler.SetTickHandle(handle)
}

// MsgClient.SetErrorHandle set error handle
func (c *MsgClient) SetErrorHandle(handle func(error)) {
	c.handler.SetErrorHandle(handle)
}

// MsgClient.RegisterMsgHandle register a handle for message id
func (c *MsgClient) RegisterMsgHandle(msgid MsgIdType, handle func(*MsgSession, interface{}) error) {
	c.handler.RegisterHandle(msgid, handle)
}

// MsgClient.Send send message to server
func (c *MsgClient) Send(msgid MsgIdType, msg interface{}) error {
	return c.handler.SendMsg(c.GetSession(), msgid, msg)
}

// NewPBMsgClient create protobuf message client
func NewPBMsgClient(idMsgMapper *IdMsgMapper, options ...common.Option) *MsgClient {
	return NewMsgClient(&ProtobufCodec{}, idMsgMapper, options...)
}

// NewJsonMsgClient create json message client
func NewJsonMsgClient(idMsgMapper *IdMsgMapper, options ...common.Option) *MsgClient {
	return NewMsgClient(&JsonCodec{}, idMsgMapper, options...)
}

// NewGobMsgClient create gob message client
func NewGobMsgClient(idMsgMapper *IdMsgMapper, options ...common.Option) *MsgClient {
	return NewMsgClient(&GobCodec{}, idMsgMapper, options...)
}
