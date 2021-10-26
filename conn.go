package gsnet

import (
	"bufio"
	"net"
	"sync/atomic"
	"time"
)

const (
	DefaultConnRecvChanLen = 100
	DefaultConnSendChanLen = 100
	MaxDataBodyLength      = 128 * 1024
	MinConnTick            = 10 * time.Millisecond
)

type Conn struct {
	conn       net.Conn
	options    ConnOptions
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

type ConnOptions struct {
	ReadBuffSize  int        // 读取缓冲区大小
	WriteBuffSize int        // 发送缓冲区大小
	RecvChanLen   int        // 接收缓冲通道长度
	SendChanLen   int        // 发送缓冲通道长度
	DataProto     IDataProto // 数据包协议

	// todo 以下是需要实现的配置逻辑
	FlushWriteInterval       time.Duration // 写缓冲数据刷新到网络IO的最小时间间隔
	GracefulCloseWaitingTime time.Duration // 优雅关闭等待时间
	HeartbeatInterval        time.Duration // 心跳间隔
}

// 创建新连接
func NewConn(conn net.Conn, options *ConnOptions) *Conn {
	c := &Conn{
		conn:       conn,
		options:    *options,
		closeCh:    make(chan struct{}),
		errCh:      make(chan error, 1),
		errWriteCh: make(chan error, 1),
	}

	if c.options.WriteBuffSize <= 0 {
		c.writer = bufio.NewWriter(conn)
	} else {
		c.writer = bufio.NewWriterSize(conn, c.options.WriteBuffSize)
	}

	if c.options.ReadBuffSize <= 0 {
		c.reader = bufio.NewReader(conn)
	} else {
		c.reader = bufio.NewReaderSize(conn, c.options.ReadBuffSize)
	}

	if c.options.RecvChanLen <= 0 {
		c.options.RecvChanLen = DefaultConnRecvChanLen
	}
	c.recvCh = make(chan []byte, c.options.RecvChanLen)

	if c.options.SendChanLen <= 0 {
		c.options.SendChanLen = DefaultConnSendChanLen
	}
	c.sendCh = make(chan []byte, c.options.SendChanLen)

	if c.options.DataProto == nil {
		c.options.DataProto = &DefaultDataProto{}
	}

	return c
}

func (c *Conn) Run() {
	go c.readLoop()
	go c.writeLoop()
}

// 读循环
func (c *Conn) readLoop() {
	var err error
	var closed bool
	header := make([]byte, c.options.DataProto.GetHeaderLen())
	for err == nil {
		closed, err = c._read(header)
		if err != nil || closed {
			break
		}
		c.options.DataProto.SetHeader(header)
		// todo  1. 判断长度是否超过限制 2. 用内存池来优化
		bodyLen := c.options.DataProto.GetBodyLen()
		if bodyLen > MaxDataBodyLength {
			err = ErrBodyLenInvalid
			break
		}
		body := make([]byte, bodyLen)
		closed, err = c._read(body)
		if closed || err != nil {
			break
		}
		c.recvCh <- body
	}
	// 关闭接收通道
	close(c.recvCh)
	// 错误写入通道
	if err != nil {
		c.errCh <- err
	}
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
	var err error
	for d := range c.sendCh {
		closed := false
		// 写入数据头
		closed, err = c._write(c.options.DataProto.EncodeBodyLen(d))
		if err != nil || closed {
			break
		}
		// 写入数据
		closed, err = c._write(d)
		if err != nil || closed {
			break
		}
		err = c.writer.Flush()
		if err != nil {
			break
		}
	}
	// 错误写入通道由读协程接收
	if err != nil {
		c.errWriteCh <- err
	}
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
	if !atomic.CompareAndSwapInt32(&c.closed, 0, 1) {
		return
	}
	c.conn.Close()
	// 关闭发送通道
	close(c.sendCh)
	close(c.closeCh)
}

// 是否关闭
func (c *Conn) IsClosed() bool {
	return atomic.LoadInt32(&c.closed) > 0
}

// 发送数据
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
	default:
		c.sendCh <- data
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

// 等待选择结果
func (c *Conn) WaitSelect() ([]byte, error) {
	if atomic.LoadInt32(&c.closed) > 0 {
		return nil, ErrConnClosed
	}
	var d []byte
	var err error
	var o bool
	if c.ticker != nil {
		select {
		case d, o = <-c.recvCh:
			if !o {
				err = ErrRecvChanEmpty
			}
		case <-c.ticker.C:
		case err, o = <-c.errCh:
			if !o {
				err = ErrConnClosed
			}
		}
	} else {
		select {
		case d, o = <-c.recvCh:
			if !o {
				err = ErrRecvChanEmpty
			}
		case err, o = <-c.errCh:
			if !o {
				err = ErrConnClosed
			}
		}
	}
	return d, err
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

// 接收超时设置
func (c *Conn) SetRecvDeadline(deadline time.Time) {
	c.conn.SetReadDeadline(deadline)
}

// 发送超时设置
func (c *Conn) SetSendDeadline(deadline time.Time) {
	c.conn.SetWriteDeadline(deadline)
}
