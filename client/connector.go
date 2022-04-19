package client

import (
	"net"
	"sync/atomic"
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
		}),
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
	return c.connect(address, 0)
}

// 带超时的同步连接
func (c *Connector) ConnectWithTimeout(address string, timeout time.Duration) (net.Conn, error) {
	return c.connect(address, timeout)
}

// 异步连接
func (c *Connector) ConnectAsync(address string, timeout time.Duration, connectCB func(error)) {
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
		c.asyncResultCh = nil
	}()
	atomic.StoreInt32(&c.state, ConnStateConnecting)
}

func (c *Connector) GetConn() net.Conn {
	return c.conn
}

// 等待结果，wait参数为等待时间，如何这个时间内有了结果就提前返回
func (c *Connector) WaitResult(wait time.Duration) {
	var timer *time.Timer
	if wait > 0 {
		timer = time.NewTimer(wait)
	}
	if timer == nil {
		select {
		case d, o := <-c.asyncResultCh:
			if !o {
				return
			}
			if d.err == nil {
				c.conn = d.conn
			}
			c.connectCallback(d.err)
		default:
		}
	} else {
		select {
		case <-timer.C:
		case d, o := <-c.asyncResultCh:
			if !o {
				return
			}
			if d.err == nil {
				c.conn = d.conn
			}
			c.connectCallback(d.err)

		}
		timer.Stop()
	}
}

// 是否已连接
func (c *Connector) IsConnected() bool {
	return atomic.LoadInt32(&c.state) == ConnStateConnected
}

// 是否正在连接
func (c *Connector) IsConnecting() bool {
	return atomic.LoadInt32(&c.state) == ConnStateConnecting
}

// 是否断连或未连接
func (c *Connector) IsDisconnected() bool {
	return atomic.LoadInt32(&c.state) == ConnStateNotConnect
}

// 是否正在断连
func (c *Connector) IsDisconnecting() bool {
	return atomic.LoadInt32(&c.state) == ConnStateDisconnecting
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
	atomic.StoreInt32(&c.state, ConnStateConnected)
	return conn, nil
}
