package common

import (
	"bufio"
	"context"
	"io"
	"net"
	"sync/atomic"
	"time"

	"github.com/huoshan017/gsnet/log"
	"github.com/huoshan017/gsnet/packet"
)

const ()

type IDataProto interface {
	GetHeaderLen() uint8
	GetBodyLen(header []byte) uint32
	EncodeBodyLen([]byte) []byte
	Compress([]byte) []byte
	Decompress([]byte) ([]byte, bool)
	Encrypt([]byte) []byte
	Decrypt([]byte) ([]byte, bool)
}

// 默认数据协议
type DefaultDataProto struct {
}

func (p DefaultDataProto) GetHeaderLen() uint8 {
	return 3
}

func (p DefaultDataProto) GetBodyLen(header []byte) uint32 {
	l := uint32(header[0]) << 16 & 0xff0000
	l += uint32(header[1]) << 8 & 0xff00
	l += uint32(header[2]) & 0xff
	return l
}

func (p DefaultDataProto) EncodeBodyLen(data []byte) []byte {
	dl := len(data)
	// todo 用内存池优化
	bh := make([]byte, 3)
	bh[0] = byte(dl >> 16 & 0xff)
	bh[1] = byte(dl >> 8 & 0xff)
	bh[2] = byte(dl & 0xff)
	return bh
}

func (p DefaultDataProto) Compress(data []byte) []byte {
	return data
}

func (p DefaultDataProto) Decompress(data []byte) ([]byte, bool) {
	return data, true
}

func (p DefaultDataProto) Encrypt(data []byte) []byte {
	return data
}

func (p DefaultDataProto) Decrypt(data []byte) ([]byte, bool) {
	return data, true
}

type SimpleConn struct {
	conn       net.Conn
	options    Options
	writer     *bufio.Writer
	reader     *bufio.Reader
	recvCh     chan packet.IPacket // 缓存从网络接收的数据，对应一个接收者一个发送者
	sendCh     chan []byte         // 缓存发往网络的数据，对应一个接收者一个发送者
	closeCh    chan struct{}       // 关闭通道
	closed     int32               // 是否关闭
	errCh      chan error          // 错误通道
	errWriteCh chan error          // 写错误通道
	ticker     *time.Ticker        // 定时器
	dataProto  IDataProto
}

// 创建新连接
func NewSimpleConn(conn net.Conn, options Options) *SimpleConn {
	c := &SimpleConn{
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
	c.sendCh = make(chan []byte, c.options.sendChanLen)

	if c.dataProto == nil {
		c.dataProto = &DefaultDataProto{}
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

func (c *SimpleConn) Run() {
	go c.readLoop()
	go c.writeLoop()
}

// 读循环
func (c *SimpleConn) readLoop() {
	defer func() {
		if err := recover(); err != nil {
			log.WithStack(err)
		}
	}()

	var err error
	header := make([]byte, c.dataProto.GetHeaderLen())
	for err == nil {
		err = c.readBytes(header)
		if err != nil {
			break
		}

		// todo  1. 判断长度是否超过限制 2. 用内存池来优化
		bodyLen := c.dataProto.GetBodyLen(header)
		if bodyLen > packet.MaxPacketLength {
			err = packet.ErrBodyLengthTooLong
			break
		}

		body := make([]byte, bodyLen)
		err = c.readBytes(body)
		if err != nil {
			break
		}

		pp := packet.BytesPacket(body)

		select {
		case err = <-c.errWriteCh:
		case <-c.closeCh:
			err = c.genErrConnClosed()
		case c.recvCh <- &pp:
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

// 读数据包
func (c *SimpleConn) readBytes(data []byte) (err error) {
	select {
	case <-c.closeCh:
		err = c.genErrConnClosed()
	case err = <-c.errWriteCh: // 接收写协程的错误
	default:
	}

	if err != nil {
		return
	}

	if c.options.readTimeout != 0 {
		c.conn.SetReadDeadline(time.Now().Add(c.options.readTimeout))
	}

	_, err = io.ReadFull(c.reader, data)
	if err != nil {
		log.Infof("gsnet: io.ReadFull err: %v", err)
	}
	return
}

// 写循环
func (c *SimpleConn) writeLoop() {
	defer func() {
		if err := recover(); err != nil {
			log.WithStack(err)
		}
	}()
	var err error
	for d := range c.sendCh {
		// 写入数据头
		err = c.writeBytes(c.dataProto.EncodeBodyLen(d))
		if err != nil {
			break
		}
		// 写入数据
		err = c.writeBytes(d)
		if err != nil {
			break
		}
		// 数据还在缓冲
		if c.writer.Buffered() > 0 {
			if c.options.writeTimeout != 0 {
				c.conn.SetWriteDeadline(time.Now().Add(c.options.writeTimeout))
			}
			err = c.writer.Flush()
			if err != nil {
				break
			}
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
func (c *SimpleConn) writeBytes(data []byte) (err error) {
	select {
	case <-c.closeCh:
		err = c.genErrConnClosed()
		return
	default:
	}

	if c.options.writeTimeout != 0 {
		c.conn.SetWriteDeadline(time.Now().Add(c.options.writeTimeout))
	}

	_, err = c.writer.Write(data)
	if err != nil {
		log.Infof("gsnet: w.writer.Write err: %v", err)
	}
	return
}

// 正常关闭
func (c *SimpleConn) Close() {
	c.closeWait(0)
}

// 等待關閉
func (c *SimpleConn) CloseWait(secs int) {
	c.closeWait(secs)
}

// 關閉
func (c *SimpleConn) closeWait(secs int) {
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
func (c *SimpleConn) IsClosed() bool {
	return atomic.LoadInt32(&c.closed) > 0
}

// 发送数据，必須與Close函數在同一goroutine調用
func (c *SimpleConn) Send(typ packet.PacketType, data []byte, toCopy bool) error {
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
	case c.sendCh <- data:
	}
	return nil
}

func (c *SimpleConn) SendPoolBuffer(packet.PacketType, *[]byte, packet.MemoryManagementType) error {
	return ErrNotImplement("Conn.SendPoolBuffer")
}

func (c *SimpleConn) SendBytesArray(packet.PacketType, [][]byte, bool) error {
	return ErrNotImplement("Conn.SendBytesArray")
}

func (c *SimpleConn) SendPoolBufferArray(packet.PacketType, []*[]byte, packet.MemoryManagementType) error {
	return ErrNotImplement("Conn.SendPoolBufferArray")
}

// 非阻塞发送
func (c *SimpleConn) SendNonblock(pt packet.Packet, data []byte, toCopy bool) error {
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
	case c.sendCh <- data:
	default:
		return ErrSendChanFull
	}
	return nil
}

// 接收数据
func (c *SimpleConn) Recv() (packet.IPacket, error) {
	if atomic.LoadInt32(&c.closed) > 0 {
		return nil, c.genErrConnClosed()
	}
	err := c.recvErr()
	if err != nil {
		return nil, err
	}

	data := <-c.recvCh
	if data == nil {
		return nil, c.genErrConnClosed()
	}
	return data, nil
}

// 非阻塞接收数据
func (c *SimpleConn) RecvNonblock() (packet.IPacket, error) {
	if atomic.LoadInt32(&c.closed) > 0 {
		return nil, ErrConnClosed
	}
	err := c.recvErr()
	if err != nil {
		return nil, err
	}

	var data packet.IPacket
	select {
	case data = <-c.recvCh:
		if data == nil {
			return nil, c.genErrConnClosed()
		}
	default:
		return nil, ErrRecvChanEmpty
	}
	return data, nil
}

// 接收错误
func (c *SimpleConn) recvErr() error {
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

func (c *SimpleConn) genErrConnClosed() error {
	//debug.PrintStack()
	return ErrConnClosed
}

// 等待选择结果
func (c *SimpleConn) Wait(ctx context.Context) (packet.IPacket, error) {
	if atomic.LoadInt32(&c.closed) > 0 {
		return nil, ErrConnClosed
	}

	if c.ticker == nil && c.options.tickSpan > 0 {
		c.ticker = time.NewTicker(c.options.tickSpan)
	}

	var (
		d   packet.IPacket
		err error
	)

	if c.ticker != nil {
		select {
		case <-ctx.Done():
			err = ErrCancelWait
		case d = <-c.recvCh:
			if d == nil {
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
		case d = <-c.recvCh:
			if d == nil {
				err = ErrConnClosed
			}
		case err = <-c.errCh:
			if err == nil {
				err = ErrConnClosed
			}
		}
	}
	return d, err
}

func (c *SimpleConn) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
}

func (c *SimpleConn) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}
