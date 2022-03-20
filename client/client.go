package client

import (
	"time"

	"github.com/huoshan017/gsnet/common"
)

// 数据客户端
type Client struct {
	conn     *Connector
	sess     common.ISession
	handler  common.ISessionHandler
	options  common.ClientOptions
	lastTime time.Time
}

func NewClient(handler common.ISessionHandler, options ...common.Option) *Client {
	c := &Client{
		handler: handler,
	}
	for _, option := range options {
		option(&c.options.Options)
	}
	return c
}

func (c *Client) Connect(addr string) error {
	c.newConnector()
	err := c.conn.Connect(addr)
	c.doConnectResult(err)
	return err
}

func (c *Client) ConnectAsync(addr string, timeout time.Duration, callback func(error)) {
	c.newConnector()
	c.conn.ConnectAsync(addr, timeout, func(err error) {
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
	if err == nil && (d != nil || len(d) > 0) {
		err = c.handler.OnData(c.sess, d)
	}
	if err != nil {
		if !common.IsNoDisconnectError(err) {
			c.handler.OnDisconnect(c.sess, err)
		} else {
			c.handler.OnError(err)
		}
		c.conn.Close()
	}
	return err
}

func (c *Client) Run() {
	var ticker *time.Ticker
	if c.handler != nil && c.options.GetTickSpan() > 0 {
		ticker = time.NewTicker(c.options.GetTickSpan())
		c.lastTime = time.Now()
	}

	var err error
	for {
		if ticker != nil {
			err = c.handle(ticker)
			if err != nil {
				break
			}
		} else {
			err = c.handle(nil)
			if err != nil {
				break
			}
		}
	}

	if ticker != nil {
		ticker.Stop()
	}

	if err != nil && !common.IsNoDisconnectError(err) {
		if c.handler != nil {
			c.handler.OnDisconnect(c.sess, err)
		}
		c.conn.Close()
	}
}

func (c *Client) Close() {
	if c.options.GetConnCloseWaitSecs() > 0 {
		c.conn.CloseWait(c.options.GetConnCloseWaitSecs())
	} else {
		c.conn.Close()
	}
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

func (c *Client) handle(ticker *time.Ticker) error {
	var d []byte
	var err error
	if ticker != nil {
		select {
		case d = <-c.conn.GetRecvCh():
		case <-ticker.C:
			c.handleTick()
		case err = <-c.conn.GetErrCh():
		}
	} else {
		d, err = c.conn.Recv()
	}
	if err == nil && (d != nil || len(d) > 0) {
		err = c.handler.OnData(c.sess, d)
	}
	if err != nil {
		if common.IsNoDisconnectError(err) {
			err = nil
		} else {
			if c.handler != nil {
				c.handler.OnError(err)
			}
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
