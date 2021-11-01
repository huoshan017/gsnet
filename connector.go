package gsnet

import (
	"net"
	"time"
)

type Connector struct {
	*Conn
	options         ConnOptions
	asyncResultCh   chan error
	connectCallback func(error)
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
	}()
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
	c.Conn = NewConn(conn, &c.options)
	c.Run()
	return nil
}
