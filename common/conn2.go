package common

import (
	"bufio"
	"context"
	"net"
	"sync/atomic"
	"time"

	"github.com/huoshan017/gsnet/common/packet"
	"github.com/huoshan017/gsnet/common/pool"
)

type wrapperSendData struct {
	data     []byte
	poolData *[]byte
}

type Conn2 struct {
	conn       net.Conn
	options    Options
	writer     *bufio.Writer
	reader     *bufio.Reader
	recvCh     chan packet.IPacket  // 缓存从网络接收的数据，对应一个接收者一个发送者
	sendCh     chan wrapperSendData // 缓存发往网络的数据，对应一个接收者一个发送者
	closeCh    chan struct{}        // 关闭通道
	closed     int32                // 是否关闭
	errCh      chan error           // 错误通道
	errWriteCh chan error           // 写错误通道
	ticker     *time.Ticker         // 定时器
}

// 创建新连接
func NewConn2(conn net.Conn, options Options) *Conn2 {
	c := &Conn2{
		conn:       conn,
		options:    options,
		closeCh:    make(chan struct{}),
		errCh:      make(chan error, 1),
		errWriteCh: make(chan error, 1),
	}

	if c.options.writeBuffSize <= 0 {
		c.writer = bufio.NewWriter(conn)
	} else {
		c.writer = bufio.NewWriterSize(conn, c.options.writeBuffSize)
	}

	if c.options.readBuffSize <= 0 {
		c.reader = bufio.NewReader(conn)
	} else {
		c.reader = bufio.NewReaderSize(conn, c.options.readBuffSize)
	}

	if c.options.recvChanLen <= 0 {
		c.options.recvChanLen = DefaultConnRecvChanLen
	}
	c.recvCh = make(chan packet.IPacket, c.options.recvChanLen)

	if c.options.sendChanLen <= 0 {
		c.options.sendChanLen = DefaultConnSendChanLen
	}
	c.sendCh = make(chan wrapperSendData, c.options.sendChanLen)

	if c.options.dataProto == nil {
		c.options.dataProto = &DefaultDataProto{}
	}

	if tcpConn, ok := c.conn.(*net.TCPConn); ok {
		if c.options.noDelay {
			tcpConn.SetNoDelay(c.options.noDelay)
		}
		if c.options.keepAlived {
			tcpConn.SetKeepAlive(c.options.keepAlived)
		}
		if c.options.keepAlivedPeriod > 0 {
			tcpConn.SetKeepAlivePeriod(c.options.keepAlivedPeriod)
		}
	}

	if c.options.tickSpan > 0 && c.options.tickSpan < MinConnTick {
		c.options.tickSpan = MinConnTick
	}

	return c
}

func (c *Conn2) Run() {
	go c.readLoop()
	go c.writeLoop()
}

// 读循环
func (c *Conn2) readLoop() {
	defer func() {
		if err := recover(); err != nil {
			GetLogger().WithStack(err)
		}
	}()

	var (
		pak packet.IPacket
		err error
	)
	for err == nil {
		if c.options.readTimeout != 0 {
			err = c.conn.SetReadDeadline(time.Now().Add(c.options.readTimeout))
			if err != nil {
				break
			}
		}
		pak, err = c.options.GetPacketBuilder().DecodeReadFrom(c.reader)
		if err == nil {
			select {
			case err = <-c.errWriteCh:
			case <-c.closeCh:
				err = c.genErrConnClosed()
			case c.recvCh <- pak:
			}
		}
	}
	// 错误写入通道
	if err != nil {
		c.errCh <- err
	}
	// 关闭错误通道
	close(c.errCh)
	// 关闭接收通道
	close(c.recvCh)
}

// 写循环
func (c *Conn2) writeLoop() {
	defer func() {
		if err := recover(); err != nil {
			GetLogger().WithStack(err)
		}
	}()
	var err error
	for d := range c.sendCh {
		if c.options.writeTimeout != 0 {
			err = c.conn.SetWriteDeadline(time.Now().Add(c.options.writeTimeout))
		}
		if err == nil {
			var data []byte
			if d.data != nil {
				data = d.data
			} else if d.poolData != nil {
				data = *d.poolData
			} else {
				panic("gsnet: wrapper send data type must []byte or *[]byte")
			}
			err = c.options.GetPacketBuilder().EncodeWriteTo(packet.PacketNormal, data, c.writer)
			if err == nil {
				// 数据还在缓冲
				if c.writer.Buffered() > 0 {
					if c.options.writeTimeout != 0 {
						err = c.conn.SetWriteDeadline(time.Now().Add(c.options.writeTimeout))
					}
					if err == nil {
						err = c.writer.Flush()
					}
				}
			}
		}
		if d.poolData != nil {
			pool.GetBuffPool().Free(d.poolData)
		}
		if err != nil {
			break
		}
	}
	// 错误写入通道由读协程接收
	if err != nil {
		c.errWriteCh <- err
	}
	// 关闭写错误通道
	close(c.errWriteCh)
}

// 正常关闭
func (c *Conn2) Close() {
	c.closeWait(0)
}

// 等待關閉
func (c *Conn2) CloseWait(secs int) {
	c.closeWait(secs)
}

// 關閉
func (c *Conn2) closeWait(secs int) {
	if !atomic.CompareAndSwapInt32(&c.closed, 0, 1) {
		return
	}
	if secs != 0 {
		c.conn.(*net.TCPConn).SetLinger(secs)
	}
	// 连接断开
	c.conn.Close()
	// 停止定时器
	if c.ticker != nil {
		c.ticker.Stop()
	}
	close(c.closeCh)
	close(c.sendCh)
}

// 是否关闭
func (c *Conn2) IsClosed() bool {
	return atomic.LoadInt32(&c.closed) > 0
}

// 发送数据，必須與Close函數在同一goroutine調用
func (c *Conn2) Send(data []byte, deepCopy bool) error {
	if atomic.LoadInt32(&c.closed) > 0 {
		return c.genErrConnClosed()
	}
	var err error
	select {
	case err = <-c.errCh:
		if err == nil {
			return c.genErrConnClosed()
		}
		return err
	case <-c.closeCh:
		return c.genErrConnClosed()
	case c.sendCh <- c.getWrapperSendData(data, deepCopy):
	}
	return nil
}

// 非阻塞发送
func (c *Conn2) SendNonblock(data []byte, deepCopy bool) error {
	if atomic.LoadInt32(&c.closed) > 0 {
		return c.genErrConnClosed()
	}
	err := c.recvErr()
	if err != nil {
		return err
	}
	select {
	case <-c.closeCh:
		return c.genErrConnClosed()
	case c.sendCh <- c.getWrapperSendData(data, deepCopy):
	default:
		return ErrSendChanFull
	}
	return nil
}

func (c *Conn2) getWrapperSendData(data []byte, deepCopy bool) wrapperSendData {
	if !deepCopy {
		return wrapperSendData{data: data}
	}
	b := pool.GetBuffPool().Alloc(int32(len(data)))
	copy(*b, data)
	return wrapperSendData{poolData: b}
}

// 接收数据
func (c *Conn2) Recv() (packet.IPacket, error) {
	if atomic.LoadInt32(&c.closed) > 0 {
		return nil, c.genErrConnClosed()
	}
	err := c.recvErr()
	if err != nil {
		return nil, err
	}

	pak := <-c.recvCh
	if pak == nil {
		return nil, c.genErrConnClosed()
	}
	return pak, nil
}

// 非阻塞接收数据
func (c *Conn2) RecvNonblock() (packet.IPacket, error) {
	if atomic.LoadInt32(&c.closed) > 0 {
		return nil, ErrConnClosed
	}
	err := c.recvErr()
	if err != nil {
		return nil, err
	}

	var pak packet.IPacket
	select {
	case pak = <-c.recvCh:
		if pak == nil {
			return nil, c.genErrConnClosed()
		}
	default:
		return nil, ErrRecvChanEmpty
	}
	return pak, nil
}

// 接收错误
func (c *Conn2) recvErr() error {
	select {
	case err := <-c.errCh:
		if err == nil {
			return c.genErrConnClosed()
		}
		return err
	case <-c.closeCh:
		return c.genErrConnClosed()
	default:
	}
	return nil
}

func (c *Conn2) genErrConnClosed() error {
	//debug.PrintStack()
	return ErrConnClosed
}

// 等待选择结果
func (c *Conn2) Wait(ctx context.Context) (packet.IPacket, error) {
	if atomic.LoadInt32(&c.closed) > 0 {
		return nil, ErrConnClosed
	}

	if c.ticker == nil && c.options.tickSpan > 0 {
		c.ticker = time.NewTicker(c.options.tickSpan)
	}

	var (
		p   packet.IPacket
		err error
	)

	if c.ticker != nil {
		select {
		case <-ctx.Done():
			err = ErrCancelWait
		case p = <-c.recvCh:
			if p == nil {
				err = ErrConnClosed
			}
		case <-c.ticker.C:
		case err = <-c.errCh:
			if err == nil {
				err = ErrConnClosed
			}
		}
	} else {
		select {
		case <-ctx.Done():
			err = ErrCancelWait
		case p = <-c.recvCh:
			if p == nil {
				err = ErrConnClosed
			}
		case err = <-c.errCh:
			if err == nil {
				err = ErrConnClosed
			}
		}
	}
	return p, err
}
