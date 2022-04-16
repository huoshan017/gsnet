package client

import (
	"context"
	"net"
	"sync/atomic"
	"time"

	"github.com/huoshan017/gsnet/common"
	"github.com/huoshan017/gsnet/log"
	"github.com/huoshan017/gsnet/packet"
)

const (
	ConnStateNotConnect    = iota // 非连接状态
	ConnStateConnecting    = 1    // 连接中状态
	ConnStateConnected     = 2    // 已连接状态
	ConnStateDisconnecting = 3    // 正在断开状态
)

type Connector struct {
	conn            common.IConn
	options         *common.Options
	asyncResultCh   chan error
	connectCallback func(error)
	state           int32
	resend          *common.ResendData
}

// 创建连接器
func NewConnector(options *common.Options) *Connector {
	c := &Connector{
		options:       options,
		asyncResultCh: make(chan error),
	}
	resendConfig := c.options.GetResendConfig()
	if resendConfig != nil {
		c.resend = common.NewResendData(resendConfig)
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
				log.WithStack(err)
			}
		}()
		c.connectCallback = connectCB
		err := c.connect(address, timeout)
		c.asyncResultCh <- err
		close(c.asyncResultCh)
		c.asyncResultCh = nil
	}()
	atomic.StoreInt32(&c.state, ConnStateConnecting)
}

func (c *Connector) GetConn() common.IConn {
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

func (c *Connector) Recv() (packet.IPacket, error) {
	return c.conn.Recv()
}

func (c *Connector) RecvNonblock() (packet.IPacket, error) {
	return c.conn.RecvNonblock()
}

func (c *Connector) Send(data []byte, copyData bool) error {
	return c.conn.Send(packet.PacketNormalData, data, copyData)
}

func (c *Connector) Wait(ctx context.Context) (packet.IPacket, error) {
	return c.conn.Wait(ctx)
}

func (c *Connector) CloseWait(secs int) {
	c.conn.CloseWait(secs)
	atomic.StoreInt32(&c.state, ConnStateNotConnect)
}

// 关闭
func (c *Connector) Close() {
	c.conn.Close()
	atomic.StoreInt32(&c.state, ConnStateNotConnect)
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

func (c *Connector) GetResendData() *common.ResendData {
	return c.resend
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
	switch c.options.GetConnDataType() {
	case 1:
		c.conn = common.NewConn(conn, *c.options)
	default:
		if c.resend != nil {
			c.conn = common.NewConn2UseResend(conn, c.resend, *c.options)
		} else {
			c.conn = common.NewConn2(conn, *c.options)
		}
	}

	c.conn.Run()
	return nil
}
