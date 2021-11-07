package gsnet

import (
	"time"
)

// 数据客户端
type Client struct {
	conn     *Connector
	sess     ISession
	handler  ISessionHandler
	options  ClientOptions
	lastTime time.Time
}

func NewClient(handler ISessionHandler, options ...Option) *Client {
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
	c.conn = NewConnector(&ConnOptions{
		ReadBuffSize:  c.options.ReadBuffSize,
		WriteBuffSize: c.options.WriteBuffSize,
		RecvChanLen:   c.options.RecvChanLen,
		SendChanLen:   c.options.SendChanLen,
		DataProto:     c.options.DataProto,
	})
	return c.conn
}

func (c *Client) doConnectResult(err error) {
	if err == nil && c.handler != nil {
		c.handler.OnConnect(c.sess)
	}
	c.sess = NewSessionNoId(c.conn)
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
	if err == ErrRecvChanEmpty {
		return nil
	}
	if err == nil && (d != nil || len(d) > 0) {
		err = c.handler.OnData(c.sess, d)
	}
	if err != nil {
		if !IsNoDisconnectError(err) {
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
	if c.handler != nil && c.options.tickSpan > 0 {
		ticker = time.NewTicker(c.options.tickSpan)
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

	if err != nil && !IsNoDisconnectError(err) {
		if c.handler != nil {
			c.handler.OnDisconnect(c.sess, err)
		}
		c.conn.Close()
	}
}

func (c *Client) Close() {
	c.conn.Close()
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
		case d = <-c.conn.getRecvCh():
		case <-ticker.C:
			c.handleTick()
		case err = <-c.conn.getErrCh():
		}
	} else {
		d, err = c.conn.Recv()
	}
	if err == nil && (d != nil || len(d) > 0) {
		err = c.handler.OnData(c.sess, d)
	}
	if err != nil {
		if IsNoDisconnectError(err) {
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

// 消息客户端
type MsgClient struct {
	*Client
	dispatcher *MsgDispatcher
}

func NewMsgClient(options ...Option) *MsgClient {
	c := &MsgClient{}
	c.Client = NewClient(c.dispatcher, options...)
	if c.options.MsgProto == nil {
		c.dispatcher = NewMsgDispatcher(&DefaultMsgProto{})
	} else {
		c.dispatcher = NewMsgDispatcher(c.options.MsgProto)
	}
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
