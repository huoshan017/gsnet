package gsnet

import (
	"time"
)

// 数据客户端
type Client struct {
	conn     *Connector
	callback IClientCallback
	handler  IHandler
	options  ClientOptions
	sess     *Session
}

func NewClient(callback IClientCallback, handler IHandler, options ...Option) *Client {
	c := &Client{
		callback: callback,
		handler:  handler,
		sess:     &Session{},
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
	var lastTime time.Time
	if c.callback != nil && c.options.tickSpan > 0 {
		ticker = time.NewTicker(c.options.tickSpan)
		lastTime = time.Now()
	}

	var err error
	for {
		if ticker != nil {
			select {
			case <-ticker.C:
				now := time.Now()
				tick := now.Sub(lastTime)
				c.callback.OnTick(tick)
				lastTime = now
			default:
				err = c.handle(true)
				if err != nil {
					break
				}
			}
		} else {
			err = c.handle(false)
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

func (c *Client) handle(nonblock bool) error {
	var d []byte
	var err error
	if nonblock {
		d, err = c.conn.RecvNonblock()
	} else {
		d, err = c.conn.Recv()
	}
	if err == nil {
		c.sess.conn = c.conn.Conn
		err = c.handler.HandleData(c.sess, d)
	}
	if err != nil && c.callback != nil {
		c.callback.OnError(err)
	}
	return err
}

// 消息客户端
type MsgClient struct {
	*Client
	dispatcher *MsgDispatcher
}

func NewMsgClient(callback IClientCallback, options ...Option) *MsgClient {
	c := &MsgClient{}
	if c.options.MsgProto == nil {
		c.dispatcher = NewMsgDispatcher(&DefaultMsgProto{})
	} else {
		c.dispatcher = NewMsgDispatcher(c.options.MsgProto)
	}
	c.Client = NewClient(callback, c.dispatcher, options...)
	return c
}

func (c *MsgClient) RegisterHandle(msgid uint32, handle func(*Session, []byte) error) {
	c.dispatcher.RegisterHandle(msgid, handle)
}

func (c *MsgClient) Send(sess *Session, msgid uint32, data []byte) error {
	return c.dispatcher.SendMsg(sess, msgid, data)
}
