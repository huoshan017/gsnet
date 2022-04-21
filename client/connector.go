package client

import (
	"net"
	"time"

	"github.com/huoshan017/gsnet/common"
	"github.com/huoshan017/gsnet/log"
)

const (
	ConnStateNotConnect    = iota // 非连接状态
	ConnStateConnecting    = 1    // 连接中状态
	ConnStateConnected     = 2    // 已连接状态
	ConnStateDisconnecting = 3    // 正在断开状态
)

type Connector struct {
	conn          net.Conn
	options       *common.Options
	asyncResultCh chan struct {
		conn net.Conn
		err  error
	}
	connectCallback func(error)
	state           int32
}

// 创建连接器
func NewConnector(options *common.Options) *Connector {
	c := &Connector{
		options: options,
		asyncResultCh: make(chan struct {
			conn net.Conn
			err  error
		}, 1),
	}
	return c
}

// 重置
func (c *Connector) Reset() {
	if c.asyncResultCh == nil {
		c.asyncResultCh = make(chan struct {
			conn net.Conn
			err  error
		})
	}
}

// 同步连接
func (c *Connector) Connect(address string) (net.Conn, error) {
	conn, err := c.connect(address, 0)
	c.state = ConnStateConnected
	return conn, err
}

// 带超时的同步连接
func (c *Connector) ConnectWithTimeout(address string, timeout time.Duration) (net.Conn, error) {
	conn, err := c.connect(address, timeout)
	c.state = ConnStateConnected
	return conn, err
}

// 异步连接
func (c *Connector) ConnectAsync(address string, timeout time.Duration, connectCB func(error)) {
	c.state = ConnStateConnecting
	go func() {
		defer func() {
			if err := recover(); err != nil {
				log.WithStack(err)
			}
		}()
		c.connectCallback = connectCB
		conn, err := c.connect(address, timeout)
		c.asyncResultCh <- struct {
			conn net.Conn
			err  error
		}{conn, err}
		close(c.asyncResultCh)
	}()
}

func (c *Connector) GetConn() net.Conn {
	return c.conn
}

// 等待结果，wait参数为等待时间，如何这个时间内有了结果就提前返回
func (c *Connector) WaitResult() (int32, error) {
	var err error
	select {
	case d, o := <-c.asyncResultCh:
		if !o {
			return ConnStateConnected, nil
		}
		if d.err == nil {
			c.conn = d.conn
			c.state = ConnStateConnected
		} else {
			c.state = ConnStateNotConnect
			err = d.err
		}
		c.connectCallback(d.err)
	default:
	}
	return c.state, err
}

func (c *Connector) IsNotConnect() bool {
	return c.state == ConnStateNotConnect
}

// 是否已连接
func (c *Connector) IsConnected() bool {
	return c.state == ConnStateConnected //atomic.LoadInt32(&c.state) == ConnStateConnected
}

// 是否正在连接
func (c *Connector) IsConnecting() bool {
	return c.state == ConnStateConnecting //atomic.LoadInt32(&c.state) == ConnStateConnecting
}

// 是否断连或未连接
func (c *Connector) IsDisconnected() bool {
	return c.state == ConnStateNotConnect //atomic.LoadInt32(&c.state) == ConnStateNotConnect
}

// 是否正在断连
func (c *Connector) IsDisconnecting() bool {
	return c.state == ConnStateDisconnecting //atomic.LoadInt32(&c.state) == ConnStateDisconnecting
}

func (c *Connector) GetState() int32 {
	return c.state
}

// 内部连接函数
func (c *Connector) connect(address string, timeout time.Duration) (net.Conn, error) {
	var conn net.Conn
	var err error
	if timeout > 0 {
		conn, err = net.DialTimeout("tcp", address, timeout)
	} else {
		conn, err = net.Dial("tcp", address)
	}
	if err != nil {
		return nil, err
	}
	return conn, nil
}
