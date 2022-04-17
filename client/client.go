package client

import (
	"context"
	"errors"
	"time"

	"github.com/huoshan017/gsnet/common"
	"github.com/huoshan017/gsnet/packet"
)

var (
	ErrClientRunUpdateMode   = errors.New("gsnet: client run update mode")
	ErrClientRunMainLoopMode = errors.New("gsnet: client run main loop mode")
)

// 数据客户端
type Client struct {
	conn              *Connector
	sess              common.ISession
	handler           common.ISessionEventHandler
	basePacketHandler common.IBasePacketHandler
	options           ClientOptions
	lastTime          time.Time
	ctx               context.Context
	cancel            context.CancelFunc
}

func NewClient(handler common.ISessionEventHandler, options ...common.Option) *Client {
	c := &Client{
		handler: handler,
	}
	c.ctx, c.cancel = context.WithCancel(context.Background())
	for _, option := range options {
		option(&c.options.Options)
	}
	if c.options.GetPacketPool() == nil {
		c.options.SetPacketPool(packet.GetDefaultPacketPool())
	}
	//if c.options.GetPacketBuilder() == nil {
	//	c.options.SetPacketBuilder(packet.GetDefaultPacketBuilder())
	//}
	return c
}

func (c *Client) Connect(addr string) error {
	conn := c.newConnector()
	err := conn.Connect(addr)
	c.doConnectResult(err)
	return err
}

func (c *Client) ConnectWithTimeout(addr string, timeout time.Duration) error {
	conn := c.newConnector()
	err := conn.ConnectWithTimeout(addr, timeout)
	c.doConnectResult(err)
	return err
}

func (c *Client) ConnectAsync(addr string, timeout time.Duration, callback func(error)) {
	conn := c.newConnector()
	conn.ConnectAsync(addr, timeout, func(err error) {
		callback(err)
		c.doConnectResult(err)
	})
}

func (c *Client) newConnector() *Connector {
	c.conn = NewConnector(&c.options.Options)
	return c.conn
}

func (c *Client) doConnectResult(err error) {
	c.sess = common.NewSessionNoId(c.conn.GetConn())
	var resend common.IResendEventHandler
	if c.conn.GetResendData() != nil {
		resend = c.conn.GetResendData()
	}
	c.basePacketHandler = common.NewDefaultBasePacketHandler(true, c.conn.GetConn(), resend, &c.options.Options)
	// update模式下先把握手处理掉
	if c.options.GetRunMode() == RunModeOnlyUpdate {
		for {
			res, err := c.handleHandshake(0)
			if err != nil || res {
				break
			}
		}
		if err == nil {
			c.handler.OnConnect(c.sess)
		}
	}
}

func (c *Client) Send(data []byte, copyData bool) error {
	return c.conn.Send(data, copyData)
}

func (c *Client) Update() error {
	if c.options.GetRunMode() != RunModeOnlyUpdate {
		return ErrClientRunUpdateMode
	}

	// 连接状态
	if c.sess == nil {
		c.conn.WaitResult(0)
		return nil
	}

	err := c.handle(1)
	return c.handleErr(err)
}

func (c *Client) Run() {
	if c.options.GetRunMode() != RunModeAsMainLoop {
		c.handler.OnError(ErrClientRunMainLoopMode)
		return
	}

	var (
		res bool
		err error
	)

	for {
		res, err = c.handleHandshake(0)
		if err != nil || res {
			break
		}
	}

	if err == nil {
		c.handler.OnConnect(c.sess)
	}

	for err == nil {
		err = c.handle(0)
	}

	c.handleErr(err)
}

func (c *Client) Quit() {
	c.cancel()
}

func (c *Client) Close() {
	if c.options.GetConnCloseWaitSecs() > 0 {
		c.conn.CloseWait(c.options.GetConnCloseWaitSecs())
	} else {
		c.conn.Close()
	}
}

func (c *Client) CloseWait(secs int) {
	c.conn.CloseWait(secs)
}

func (c *Client) GetSession() common.ISession {
	return c.sess
}

func (c *Client) IsConnecting() bool {
	if c.conn == nil {
		return false
	}
	return c.conn.IsConnecting()
}

func (c *Client) IsConnected() bool {
	if c.conn == nil {
		return false
	}
	return c.conn.IsConnected()
}

func (c *Client) IsDisconnected() bool {
	return c.conn == nil || c.conn.IsDisconnected()
}

func (c *Client) IsDisconnecting() bool {
	if c.conn == nil {
		return false
	}
	return c.conn.IsDisconnecting()
}

func (c *Client) handleHandshake(mode int32) (bool, error) {
	var (
		pak packet.IPacket
		res int32
		err error
	)
	err = c.basePacketHandler.OnUpdateHandle()
	if err != nil {
		return false, err
	}
	if mode == 0 {
		pak, err = c.conn.Wait(c.ctx)
	} else {
		pak, err = c.conn.RecvNonblock()
	}
	if err == nil && pak != nil {
		res, err = c.basePacketHandler.OnHandleHandshake(pak)
	}
	c.options.GetPacketPool().Put(pak)
	if err == common.ErrRecvChanEmpty {
		err = nil
	}
	return res == 2, err
}

func (c *Client) handle(mode int32) error {
	var (
		pak packet.IPacket
		res int32
		err error
	)

	if mode == 0 {
		pak, err = c.conn.Wait(c.ctx)
	} else {
		pak, err = c.conn.RecvNonblock()
	}
	if err == nil {
		if pak != nil { // net packet handle
			res, err = c.basePacketHandler.OnPreHandle(pak)
			if err == nil {
				if res == 0 {
					err = c.handler.OnPacket(c.sess, pak)
				}
			}
			if err == nil {
				err = c.basePacketHandler.OnPostHandle(pak)
			}
			c.options.GetPacketPool().Put(pak)
		} else { // tick handle
			c.handleTick()
			err = c.basePacketHandler.OnUpdateHandle()
		}
	}

	if mode != 0 && err == common.ErrRecvChanEmpty {
		err = nil
	}

	if err != nil {
		if common.IsNoDisconnectError(err) {
			c.handler.OnError(err)
		}
	}

	return err
}

func (c *Client) handleTick() {
	now := time.Now()
	tick := now.Sub(c.lastTime)
	c.handler.OnTick(c.sess, tick)
	c.lastTime = now
}

func (c *Client) handleErr(err error) error {
	if err != nil {
		// if no disconnect error, reset err to nil
		if common.IsNoDisconnectError(err) {
			err = nil
		} else {
			if c.handler != nil {
				c.handler.OnDisconnect(c.sess, err)
			}
			c.conn.Close()
		}
	}
	return err
}
