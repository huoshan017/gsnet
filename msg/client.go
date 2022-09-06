package msg

import (
	"time"

	"github.com/huoshan017/gsnet/client"
	"github.com/huoshan017/gsnet/msg/codec"
	"github.com/huoshan017/gsnet/options"
)

// MsgClient struct
type MsgClient struct {
	*client.Client
	options MsgClientOptions
	handler *msgHandlerClient
}

// NewMsgClient create a message client
func NewMsgClient(msgCodec IMsgCodec, idMsgMapper *IdMsgMapper, ops ...options.Option) *MsgClient {
	c := &MsgClient{}
	c.handler = newMsgHandlerClient(msgCodec, idMsgMapper, &c.options.MsgOptions)
	for i := 0; i < len(ops); i++ {
		ops[i](&c.options.Options)
	}
	c.Client = client.NewClientWithOptions(c.handler, &c.options.ClientOptions)
	return c
}

// MsgClient.SetConnectHandle set connected handle with message session
func (c *MsgClient) SetConnectHandle(handle func(*MsgSession)) {
	c.handler.SetConnectHandle(handle)
}

// MsgClient.SetReadyHandle set ready handle with message session
func (c *MsgClient) SetReadyHandle(handle func(*MsgSession)) {
	c.handler.SetReadyHandle(handle)
}

// MsgClient.SetDisconnectHandle set disconnected handle with message session
func (c *MsgClient) SetDisconnectHandle(handle func(*MsgSession, error)) {
	c.handler.SetDisconnectHandle(handle)
}

// MsgClient.SetTickHandle set tick timer handle with message session
func (c *MsgClient) SetTickHandle(handle func(*MsgSession, time.Duration)) {
	c.handler.SetTickHandle(handle)
}

// MsgClient.SetErrorHandle set error handle
func (c *MsgClient) SetErrorHandle(handle func(error)) {
	c.handler.SetErrorHandle(handle)
}

// MsgClient.RegisterMsgHandle register a handle for message id
func (c *MsgClient) RegisterMsgHandle(msgid MsgIdType, handle func(*MsgSession, any) error) {
	c.handler.RegisterHandle(msgid, handle)
}

// MsgClient.Send send message to server
func (c *MsgClient) Send(msgid MsgIdType, msg any) error {
	return c.handler.SendMsg(c.GetSession(), msgid, msg)
}

// NewProtobufMsgClient create protobuf message client
func NewProtobufMsgClient(idMsgMapper *IdMsgMapper, options ...options.Option) *MsgClient {
	return NewMsgClient(&codec.ProtobufCodec{}, idMsgMapper, options...)
}

// NewJsonMsgClient create json message client
func NewJsonMsgClient(idMsgMapper *IdMsgMapper, options ...options.Option) *MsgClient {
	return NewMsgClient(&codec.JsonCodec{}, idMsgMapper, options...)
}

// NewGobMsgClient create gob message client
func NewGobMsgClient(idMsgMapper *IdMsgMapper, options ...options.Option) *MsgClient {
	return NewMsgClient(&codec.GobCodec{}, idMsgMapper, options...)
}

// NewThriftMsgClient create thrift message client
func NewThriftMsgClient(idMsgMapper *IdMsgMapper, options ...options.Option) *MsgClient {
	return NewMsgClient(&codec.ThriftCodec{}, idMsgMapper, options...)
}

// NewMsgpackMsgClient create msgpack message client
func NewMsgpackMsgClient(idMsgMapper *IdMsgMapper, options ...options.Option) *MsgClient {
	return NewMsgClient(&codec.MsgpackCodec{}, idMsgMapper, options...)
}
