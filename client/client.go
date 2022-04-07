package client

import (
	"context"
	"time"

	"github.com/huoshan017/gsnet/common"
	"github.com/huoshan017/gsnet/common/packet"
)

// 数据客户端
type Client struct {
	conn     *Connector
	sess     common.ISession
	handler  common.ISessionEventHandler
	options  ClientOptions
	lastTime time.Time
	ctx      context.Context
	cancel   context.CancelFunc
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
	if c.options.GetPacketBuilder() == nil {
		c.options.SetPacketBuilder(packet.GetDefaultPacketBuilder())
	}
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
	if err == nil && c.handler != nil {
		c.handler.OnConnect(c.sess)
	}
	c.sess = common.NewSessionNoId(c.conn.GetConn())
}

func (c *Client) Send(data []byte, copyData bool) error {
	return c.conn.Send(data, copyData)
}

func (c *Client) Update() error {
	// 连接状态
	if c.sess == nil {
		c.conn.WaitResult(0)
		return nil
	}

	pak, err := c.conn.RecvNonblock()
	// 没有数据
	if err == common.ErrRecvChanEmpty {
		c.options.GetPacketPool().Put(pak)
		return nil
	}

	// process resend
	resend := c.conn.GetResendData()
	if err == nil {
		var res int32
		if resend != nil {
			res = resend.OnAck(pak)
			if res < 0 {
				common.GetLogger().Fatalf("gsnet: length of rend list less than ack num")
				err = common.ErrResendDataInvalid
			}
		}
		if res == 0 {
			err = c.handler.OnPacket(c.sess, pak)
		}
		if resend != nil && err == nil {
			resend.OnProcessed(1)
		}
		//err = c.handler.OnPacket(c.sess, pak)
	}
	if resend != nil {
		resend.OnUpdate(c.conn.GetConn())
	}

	c.options.GetPacketPool().Put(pak)
	if err != nil {
		if !common.IsNoDisconnectError(err) {
			c.handler.OnDisconnect(c.sess, err)
			c.conn.Close()
		} else {
			c.handler.OnError(err)
		}
	}
	return err
}

func (c *Client) Run() {
	var err error
	for {
		err = c.handle()
		if err != nil {
			break
		}
	}

	if err != nil && !common.IsNoDisconnectError(err) {
		if c.handler != nil {
			c.handler.OnDisconnect(c.sess, err)
		}
		c.conn.Close()
	}
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

func (c *Client) handle() error {
	var (
		pak packet.IPacket
		err error
	)
	resend := c.conn.GetResendData()
	pak, err = c.conn.Wait(c.ctx)
	if err == nil {
		if pak != nil { // net packet handle
			//err = c.handler.OnPacket(c.sess, pak)
			var res int32
			if resend != nil {
				res = resend.OnAck(pak)
				if res < 0 {
					common.GetLogger().Fatalf("gsnet: length of rend list less than ack num")
					err = common.ErrResendDataInvalid
				}
			}
			if res == 0 {
				err = c.handler.OnPacket(c.sess, pak)
			}
			if resend != nil && err == nil {
				resend.OnProcessed(1)
			}
		} else { // tick handle
			c.handleTick()
			if resend != nil {
				err = resend.OnUpdate(c.conn.GetConn())
			}
		}
	}
	c.options.GetPacketPool().Put(pak)
	if err != nil {
		if !common.IsNoDisconnectError(err) {
		} else {
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
