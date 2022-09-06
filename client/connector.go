package client

import (
	"net"
	"sync/atomic"
	"time"

	"github.com/huoshan017/gsnet/common"
	"github.com/huoshan017/gsnet/kcp"
	"github.com/huoshan017/gsnet/log"
	"github.com/huoshan017/gsnet/options"
)

const (
	ConnStateNotConnect    = iota // 非连接状态
	ConnStateConnecting    = 1    // 连接中状态
	ConnStateConnected     = 2    // 已连接状态
	ConnStateDisconnecting = 3    // 正在断开状态
)

type Connector struct {
	conn          net.Conn
	options       *options.Options
	asyncResultCh chan struct {
		conn net.Conn
		err  error
	}
	connectCallback func(error)
	state           int32
}

// 创建连接器
func NewConnector(options *options.Options) *Connector {
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
	atomic.StoreInt32(&c.state, ConnStateNotConnect)
}

// 同步连接
func (c *Connector) Connect(address string) (net.Conn, error) {
	conn, err := c.connect(address, 0)
	atomic.StoreInt32(&c.state, ConnStateConnected)
	return conn, err
}

// 带超时的同步连接
func (c *Connector) ConnectWithTimeout(address string, timeout time.Duration) (net.Conn, error) {
	conn, err := c.connect(address, timeout)
	atomic.StoreInt32(&c.state, ConnStateConnected)
	return conn, err
}

// 异步连接
func (c *Connector) ConnectAsync(address string, timeout time.Duration, connectCB func(error)) {
	atomic.StoreInt32(&c.state, ConnStateConnecting)
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

func (c *Connector) WaitResult(isBlock bool) (int32, error) {
	var err error
	if isBlock {
		d, o := <-c.asyncResultCh
		if !o {
			return ConnStateConnected, nil
		}
		c.doAsyncResult(d.err, d.conn)
		err = d.err
	} else {
		select {
		case d, o := <-c.asyncResultCh:
			if !o {
				return ConnStateConnected, nil
			}
			c.doAsyncResult(d.err, d.conn)
			err = d.err
		default:
		}
	}
	return c.state, err
}

func (c *Connector) IsNotConnect() bool {
	return atomic.LoadInt32(&c.state) == ConnStateNotConnect
}

// 是否已连接
func (c *Connector) IsConnected() bool {
	return atomic.LoadInt32(&c.state) == ConnStateConnected //atomic.LoadInt32(&c.state) == ConnStateConnected
}

// 是否正在连接
func (c *Connector) IsConnecting() bool {
	return atomic.LoadInt32(&c.state) == ConnStateConnecting //atomic.LoadInt32(&c.state) == ConnStateConnecting
}

// 是否断连或未连接
func (c *Connector) IsDisconnected() bool {
	return atomic.LoadInt32(&c.state) == ConnStateNotConnect //atomic.LoadInt32(&c.state) == ConnStateNotConnect
}

// 是否正在断连
func (c *Connector) IsDisconnecting() bool {
	return atomic.LoadInt32(&c.state) == ConnStateDisconnecting //atomic.LoadInt32(&c.state) == ConnStateDisconnecting
}

func (c *Connector) GetState() int32 {
	return atomic.LoadInt32(&c.state)
}

// 内部连接函数
func (c *Connector) connect(address string, timeout time.Duration) (net.Conn, error) {
	var (
		conn net.Conn
		err  error
	)

	var netProto = c.options.GetNetProto()
	switch netProto {
	case options.NetProtoTCP:
		if timeout > 0 {
			conn, err = net.DialTimeout("tcp", address, timeout)
		} else {
			conn, err = net.Dial("tcp", address)
		}
	case options.NetProtoTCP4:
		if timeout > 0 {
			conn, err = net.DialTimeout("tcp4", address, timeout)
		} else {
			conn, err = net.Dial("tcp4", address)
		}
	case options.NetProtoTCP6:
		if timeout > 0 {
			conn, err = net.DialTimeout("tcp6", address, timeout)
		} else {
			conn, err = net.Dial("tcp6", address)
		}
	case options.NetProtoUDP:
		conn, err = kcp.DialUDP("udp", address)
	case options.NetProtoUDP4:
		conn, err = kcp.DialUDP("udp4", address)
	case options.NetProtoUDP6:
		conn, err = kcp.DialUDP("udp6", address)
	default:
		err = common.ErrUnknownNetwork
	}
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func (c *Connector) doAsyncResult(err error, conn net.Conn) {
	if err == nil {
		c.conn = conn
		c.state = ConnStateConnected
	} else {
		c.state = ConnStateNotConnect
	}
	c.connectCallback(err)
}
