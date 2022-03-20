package gsnet

import (
	"time"

	"github.com/huoshan017/gsnet/client"
	"github.com/huoshan017/gsnet/common"
)

// 消息客户端
type MsgClient struct {
	*client.Client
	dispatcher *common.MsgDispatcher
}

func NewMsgClient(msgDecoder common.IMsgDecoder, options ...common.Option) *MsgClient {
	c := &MsgClient{}
	dispatcher := common.NewMsgDispatcher(msgDecoder)
	c.Client = client.NewClient(dispatcher, options...)
	c.dispatcher = dispatcher
	return c
}

func (c *MsgClient) SetConnectHandle(handle func(common.ISession)) {
	c.dispatcher.SetConnectHandle(handle)
}

func (c *MsgClient) SetDisconnectHandle(handle func(common.ISession, error)) {
	c.dispatcher.SetDisconnectHandle(handle)
}

func (c *MsgClient) SetTickHandle(handle func(common.ISession, time.Duration)) {
	c.dispatcher.SetTickHandle(handle)
}

func (c *MsgClient) SetErrorHandle(handle func(error)) {
	c.dispatcher.SetErrorHandle(handle)
}

func (c *MsgClient) RegisterHandle(msgid uint32, handle func(common.ISession, []byte) error) {
	c.dispatcher.RegisterHandle(msgid, handle)
}

func (c *MsgClient) Send(msgid uint32, data []byte) error {
	return c.dispatcher.SendMsg(c.GetSession(), msgid, data)
}

func NewDefaultMsgClient(options ...common.Option) *MsgClient {
	return NewMsgClient(&common.DefaultMsgDecoder{}, options...)
}
