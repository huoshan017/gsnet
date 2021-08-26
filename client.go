package gsnet

import (
	"time"
)

// 数据客户端
type Client struct {
	conn     *Connector
	callback IClientCallback
	handler  IClientHandler
	options  ClientOptions
	lastTime time.Time
}

func NewClient(callback IClientCallback, handler IClientHandler, options ...Option) *Client {
	c := &Client{
		callback: callback,
		handler:  handler,
	}
	for _, option := range options {
		option(&c.options.Options)
	}
	return c
}

func (c *Client) Connect(addr string) error {
	c.conn = NewConnector(&ConnOptions{
		ReadBuffSize:  c.options.ReadBuffSize,
		WriteBuffSize: c.options.WriteBuffSize,
		RecvChanLen:   c.options.RecvChanLen,
		SendChanLen:   c.options.SendChanLen,
		DataProto:     c.options.DataProto,
	})
	err := c.conn.Connect(addr)
	if err == nil && c.callback != nil {
		c.callback.OnConnect()
	}
	return err
}

func (c *Client) Send(data []byte) error {
	return c.conn.Send(data)
}

func (c *Client) Run() {
	var ticker *time.Ticker
	if c.callback != nil && c.options.tickSpan > 0 {
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

	if err != nil {
		if c.callback != nil {
			c.callback.OnDisconnect(err)
		}
		c.conn.Close()
	}
}

func (c *Client) Close() {
	c.conn.Close()
}

func (c *Client) handle(ticker *time.Ticker) error {
	var d []byte
	var err error
	if ticker != nil {
		select {
		case d = <-c.conn.getRecvCh():
		case <-ticker.C:
			now := time.Now()
			tick := now.Sub(c.lastTime)
			c.callback.OnTick(tick)
			c.lastTime = now
		case err = <-c.conn.getErrCh():
		}
	} else {
		d, err = c.conn.Recv()
	}
	if err == nil && (d != nil || len(d) > 0) {
		err = c.handler.OnData(d)
	}
	if err != nil {
		if IsNoDisconnectError(err) {
			err = nil
		} else {
			if c.callback != nil {
				c.callback.OnError(err)
			}
		}
	}
	return err
}

// 消息客户端
type MsgClient struct {
	*Client
	dispatcher *ClientMsgDispatcher
}

func NewMsgClient(callback IClientCallback, options ...Option) *MsgClient {
	c := &MsgClient{}
	c.Client = NewClient(callback, c.dispatcher, options...)
	if c.options.MsgProto == nil {
		c.dispatcher = NewClientMsgDispatcher(&DefaultMsgProto{})
	} else {
		c.dispatcher = NewClientMsgDispatcher(c.options.MsgProto)
	}
	c.Client.handler = c.dispatcher
	return c
}

func (c *MsgClient) RegisterHandle(msgid uint32, handle func([]byte) error) {
	c.dispatcher.RegisterHandle(msgid, handle)
}

func (c *MsgClient) Send(msgid uint32, data []byte) error {
	ed := c.dispatcher.Encode(msgid, data)
	return c.Client.Send(ed)
}
