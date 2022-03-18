package gsnet

import "time"

// 消息客户端
type MsgClient struct {
	*Client
	dispatcher *MsgDispatcher
}

func NewMsgClient(msgDecoder IMsgDecoder, options ...Option) *MsgClient {
	c := &MsgClient{}
	c.Client = NewClient(c.dispatcher, options...)
	c.dispatcher = NewMsgDispatcher(msgDecoder)
	c.Client.handler = c.dispatcher
	return c
}

func (c *MsgClient) SetConnectHandle(handle func(ISession)) {
	c.dispatcher.SetConnectHandle(handle)
}

func (c *MsgClient) SetDisconnectHandle(handle func(ISession, error)) {
	c.dispatcher.SetDisconnectHandle(handle)
}

func (c *MsgClient) SetTickHandle(handle func(ISession, time.Duration)) {
	c.dispatcher.SetTickHandle(handle)
}

func (c *MsgClient) SetErrorHandle(handle func(error)) {
	c.dispatcher.SetErrorHandle(handle)
}

func (c *MsgClient) RegisterHandle(msgid uint32, handle func(ISession, []byte) error) {
	c.dispatcher.RegisterHandle(msgid, handle)
}

func (c *MsgClient) Send(msgid uint32, data []byte) error {
	sess := c.Client.sess
	return c.dispatcher.SendMsg(sess, msgid, data)
}

func NewDefaultMsgClient(options ...Option) *MsgClient {
	return NewMsgClient(&DefaultMsgDecoder{}, options...)
}
