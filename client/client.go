package client

import (
	"context"
	"time"

	"github.com/huoshan017/gsnet/common"
)

// 数据客户端
type Client struct {
	conn     *Connector
	sess     common.ISession
	handler  common.ISessionHandler
	options  ClientOptions
	lastTime time.Time
	ctx      context.Context
	cancel   context.CancelFunc
}

func NewClient(handler common.ISessionHandler, options ...common.Option) *Client {
	c := &Client{
		handler: handler,
	}
	c.ctx, c.cancel = context.WithCancel(context.Background())
	for _, option := range options {
		option(&c.options.Options)
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
	c.sess = common.NewSessionNoId(c.conn)
}

func (c *Client) Send(data []byte) error {
	return c.conn.Send(data)
}

func (c *Client) Update() error {
	// 连接状态
	if c.sess == nil {
		c.conn.WaitResult(0)
		return nil
	}

	d, err := c.conn.RecvNonblock()
	// 没有数据
	if err == common.ErrRecvChanEmpty {
		return nil
	}
	if err == nil {
		err = c.handler.OnData(c.sess, d)
	}
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
		data interface{}
		err  error
	)
	data, err = c.conn.Wait(c.ctx)
	if err == nil {
		if data != nil {
			err = c.handler.OnData(c.sess, data)
		} else {
			c.handleTick()
		}
	}
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
