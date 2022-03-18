package gsnet

import (
	"bufio"
	"context"
	"net"
	"sync/atomic"
	"time"
)

const (
	DefaultConnRecvChanLen = 100
	DefaultConnSendChanLen = 100
	MaxDataBodyLength      = 128 * 1024
	MinConnTick            = 10 * time.Millisecond
	DefaultConnTick        = 30 * time.Millisecond
)

type Conn struct {
	conn       net.Conn
	options    Options
	writer     *bufio.Writer
	reader     *bufio.Reader
	recvCh     chan []byte   // 缓存从网络接收的数据，对应一个接收者一个发送者
	sendCh     chan []byte   // 缓存发往网络的数据，对应一个接收者一个发送者
	closeCh    chan struct{} // 关闭通道
	closed     int32         // 是否关闭
	errCh      chan error    // 错误通道
	errWriteCh chan error    // 写错误通道
	ticker     *time.Ticker  // 定时器
}

// 创建新连接
func NewConn(conn net.Conn, options Options) *Conn {
	c := &Conn{
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
	c.recvCh = make(chan []byte, c.options.recvChanLen)

	if c.options.sendChanLen <= 0 {
		c.options.sendChanLen = DefaultConnSendChanLen
	}
	c.sendCh = make(chan []byte, c.options.sendChanLen)

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

	return c
}

func (c *Conn) Run() {
	go c.readLoop()
	go c.writeLoop()
}

// 读循环
func (c *Conn) readLoop() {
	defer func() {
		if err := recover(); err != nil {
			getLogger().WithStack(err)
		}
	}()

	var err error
	var closed bool
	header := make([]byte, c.options.dataProto.GetHeaderLen())
	for err == nil {
		closed, err = c._read(header)
		if err != nil || closed {
			break
		}
		c.options.dataProto.SetHeader(header)
		// todo  1. 判断长度是否超过限制 2. 用内存池来优化
		bodyLen := c.options.dataProto.GetBodyLen()
		if bodyLen > MaxDataBodyLength {
			err = ErrBodyLenInvalid
			break
		}
		body := make([]byte, bodyLen)
		closed, err = c._read(body)
		if closed || err != nil {
			break
		}

		select {
		case err = <-c.errWriteCh:
		case <-c.closeCh:
			err = ErrConnClosed
		case c.recvCh <- body:
		}
	}
	// 停止定时器
	if c.ticker != nil {
		c.ticker.Stop()
	}
	// 关闭接收通道
	close(c.recvCh)
	// 错误写入通道
	if err != nil {
		c.errCh <- err
	}
	// 关闭错误通道
	close(c.errCh)
}

// 读数据包
func (c *Conn) _read(data []byte) (closed bool, err error) {
	var n int
	for n < len(data) {
		select {
		case <-c.closeCh:
			closed = true
			return
		case err = <-c.errWriteCh: // 接收写协程的错误
			return
		default:
		}

		if c.options.readTimeout != 0 {
			c.conn.SetReadDeadline(time.Now().Add(c.options.readTimeout))
		}

		var nn int
		nn, err = c.reader.Read(data[n:])
		if err != nil {
			return
		}
		n += nn
	}
	return
}

// 写循环
func (c *Conn) writeLoop() {
	defer func() {
		if err := recover(); err != nil {
			getLogger().WithStack(err)
		}
	}()
	var err error
	for d := range c.sendCh {
		closed := false
		// 写入数据头
		closed, err = c._write(c.options.dataProto.EncodeBodyLen(d))
		if err != nil || closed {
			break
		}
		// 写入数据
		closed, err = c._write(d)
		if err != nil || closed {
			break
		}
		if c.writer.Buffered() > 0 {
			if c.options.writeTimeout != 0 {
				c.conn.SetWriteDeadline(time.Now().Add(c.options.writeTimeout))
			}
			err = c.writer.Flush()
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

// 写数据包
func (c *Conn) _write(data []byte) (closed bool, err error) {
	var n int
	for n < len(data) {
		select {
		case <-c.closeCh:
			closed = true
			return
		default:
		}

		if c.options.writeTimeout != 0 {
			c.conn.SetWriteDeadline(time.Now().Add(c.options.writeTimeout))
		}

		var nn int
		nn, err = c.writer.Write(data[n:])
		if err != nil {
			return
		}
		n += nn
	}
	return
}

// 正常关闭
func (c *Conn) Close() {
	c.closeWait(0)
}

// 等待關閉
func (c *Conn) CloseWait(secs int) {
	c.closeWait(secs)
}

// 關閉
func (c *Conn) closeWait(secs int) {
	if !atomic.CompareAndSwapInt32(&c.closed, 0, 1) {
		return
	}
	if secs != 0 {
		c.conn.(*net.TCPConn).SetLinger(secs)
	}
	c.conn.Close()
	close(c.sendCh)
	close(c.closeCh)
}

// 是否关闭
func (c *Conn) IsClosed() bool {
	return atomic.LoadInt32(&c.closed) > 0
}

// 发送数据，必須與Close函數在同一goroutine調用
func (c *Conn) Send(data []byte) error {
	if atomic.LoadInt32(&c.closed) > 0 {
		return ErrConnClosed
	}
	select {
	case err, o := <-c.errCh:
		if !o {
			return ErrConnClosed
		}
		return err
	case c.sendCh <- data:
	}
	return nil
}

// 非阻塞发送
func (c *Conn) SendNonblock(data []byte) error {
	if atomic.LoadInt32(&c.closed) > 0 {
		return ErrConnClosed
	}
	err := c._recvErr()
	if err != nil {
		return err
	}
	select {
	case c.sendCh <- data:
	default:
		return ErrSendChanFull
	}
	return nil
}

// 接收数据
func (c *Conn) Recv() ([]byte, error) {
	if atomic.LoadInt32(&c.closed) > 0 {
		return nil, ErrConnClosed
	}
	err := c._recvErr()
	if err != nil {
		return nil, err
	}

	data, o := <-c.recvCh
	if !o {
		return nil, ErrConnClosed
	}
	return data, nil
}

// 非阻塞接收数据
func (c *Conn) RecvNonblock() ([]byte, error) {
	if atomic.LoadInt32(&c.closed) > 0 {
		return nil, ErrConnClosed
	}
	err := c._recvErr()
	if err != nil {
		return nil, err
	}

	var data []byte
	var o bool
	select {
	case data, o = <-c.recvCh:
		if !o {
			return nil, ErrConnClosed
		}
	default:
		return nil, ErrRecvChanEmpty
	}
	return data, nil
}

// 接收错误
func (c *Conn) _recvErr() error {
	select {
	case err, o := <-c.errCh:
		if !o {
			return ErrConnClosed
		}
		return err
	default:
	}
	return nil
}

func (c *Conn) getRecvCh() chan []byte {
	return c.recvCh
}

func (c *Conn) getErrCh() chan error {
	return c.errCh
}

func (c *Conn) SetTick(tick time.Duration) {
	if tick < MinConnTick {
		tick = MinConnTick
	}
	c.ticker = time.NewTicker(tick)
}

type ServConn struct {
	Conn
}

func NewServConn(conn net.Conn, options Options) *ServConn {
	return &ServConn{
		Conn: *NewConn(conn, options),
	}
}

// 等待选择结果
func (c *ServConn) Wait(ctx context.Context) ([]byte, error) {
	if atomic.LoadInt32(&c.closed) > 0 {
		return nil, ErrConnClosed
	}

	if c.ticker == nil && c.options.tickHandle != nil {
		c.ticker = time.NewTicker(DefaultConnTick)
	}

	var (
		d   []byte
		err error
		o   bool
	)
	select {
	case <-ctx.Done():
		err = ErrCancelWait
	case d, o = <-c.recvCh:
		if !o {
			err = ErrConnClosed
		}
	case <-c.ticker.C:
	case err, o = <-c.errCh:
		if !o {
			err = ErrConnClosed
		}
	}
	return d, err
}
