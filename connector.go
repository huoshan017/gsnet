package gsnet

import (
	"net"
	"sync/atomic"
	"time"
)

const (
	ConnStateNotConnect    = iota // 非连接状态
	ConnStateConnecting    = 1    // 连接中状态
	ConnStateConnected     = 2    // 已连接状态
	ConnStateDisconnecting = 3    // 正在断开状态
)

type Connector struct {
	*Conn
	options         ConnOptions
	asyncResultCh   chan error
	connectCallback func(error)
	state           int32
}

// 创建连接器
func NewConnector(options *ConnOptions) *Connector {
	c := &Connector{
		options:       *options,
		asyncResultCh: make(chan error),
	}
	return c
}

// 重置
func (c *Connector) Reset() {
	if c.asyncResultCh == nil {
		c.asyncResultCh = make(chan error)
	}
}

// 同步连接
func (c *Connector) Connect(address string) error {
	return c.connect(address, 0)
}

// 带超时的同步连接
func (c *Connector) ConnectWithTimeout(address string, timeout time.Duration) error {
	return c.connect(address, timeout)
}

// 异步连接
func (c *Connector) ConnectAsync(address string, timeout time.Duration, connectCB func(error)) {
	go func() {
		defer func() {
			if err := recover(); err != nil {
				getLogger().WithStack(err)
			}
		}()
		c.connectCallback = connectCB
		err := c.connect(address, timeout)
		c.asyncResultCh <- err
		close(c.asyncResultCh)
		c.asyncResultCh = nil
		atomic.StoreInt32(&c.state, ConnStateConnected)
	}()
	atomic.StoreInt32(&c.state, ConnStateConnecting)
}

// 等待结果，wait参数为等待时间，如何这个时间内有了结果就提前返回
func (c *Connector) WaitResult(wait time.Duration) {
	var timer *time.Timer
	if wait > 0 {
		timer = time.NewTimer(wait)
	}
	if timer == nil {
		select {
		case err, o := <-c.asyncResultCh:
			if !o {
				return
			}
			c.connectCallback(err)
		default:
		}
	} else {
		select {
		case <-timer.C:
		case err, o := <-c.asyncResultCh:
			if !o {
				return
			}
			c.connectCallback(err)

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
func (c *Connector) connect(address string, timeout time.Duration) error {
	var conn net.Conn
	var err error
	if timeout > 0 {
		conn, err = net.DialTimeout("tcp", address, timeout)
	} else {
		conn, err = net.Dial("tcp", address)
	}
	if err != nil {
		return err
	}
	atomic.StoreInt32(&c.state, ConnStateConnected)
	c.Conn = NewConn(conn, &c.options)
	c.Run()
	return nil
}
