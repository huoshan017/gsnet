package common

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/huoshan017/gsnet/log"
	"github.com/huoshan017/gsnet/options"
	"github.com/huoshan017/gsnet/packet"
	"github.com/huoshan017/gsnet/pool"
)

const (
	chunkBufferDefaultLength = 4096
)

type chunk struct {
	rawbuf  *[]byte
	datalen int32
	isend   bool
}

var (
	chunkPool *sync.Pool
)

func init() {
	chunkPool = &sync.Pool{
		New: func() any {
			return &chunk{}
		},
	}
}

func getChunk() *chunk {
	c := chunkPool.Get().(*chunk)
	return c
}

func putChunk(c *chunk) {
	//if c.isend {
	//	pool.GetBuffPool().Free(c.rawbuf)
	//}
	c.rawbuf = nil
	c.datalen = 0
	c.isend = false
	chunkPool.Put(c)
}

// BConn struct
type BConn struct {
	conn        net.Conn
	options     *options.Options
	writer      *bufio.Writer
	recvCh      chan *chunk // 缓存从网络接收的数据，对应一个接收者一个发送者
	blist       packet.BytesList
	sendCh      chan wrapperSendData // 缓存发往网络的数据，对应一个接收者一个发送者
	csendList   ISendList            // 缓存发送队列
	closeCh     chan struct{}        // 关闭通道
	closed      int32                // 是否关闭
	errCh       chan error           // 错误通道
	errWriteCh  chan error           // 写错误通道
	ticker      *time.Ticker         // 定时器
	packetCodec IPacketCodec         // 包解码器
}

// NewConn create Conn instance use resend
func NewBConn(conn net.Conn, packetCodec IPacketCodec, ops *options.Options) *BConn {
	c := &BConn{
		conn:        conn,
		options:     ops,
		closeCh:     make(chan struct{}),
		errCh:       make(chan error, 1),
		errWriteCh:  make(chan error, 1),
		packetCodec: packetCodec,
	}

	if c.options.GetWriteBuffSize() <= 0 {
		c.writer = bufio.NewWriter(conn)
	} else {
		c.writer = bufio.NewWriterSize(conn, c.options.GetWriteBuffSize())
	}

	if c.options.GetRecvListLen() <= 0 {
		c.options.SetRecvListLen(DefaultConnRecvListLen)
	}
	c.recvCh = make(chan *chunk, c.options.GetRecvListLen())
	c.blist = packet.NewBytesList((packet.MaxPacketLength + chunkBufferDefaultLength - 1) / chunkBufferDefaultLength)

	if c.options.GetSendListMode() >= 0 {
		c.csendList = newSendListFuncMap[c.options.GetSendListMode()](int32(c.options.GetSendListLen()))
	} else {
		if c.options.GetSendListLen() <= 0 {
			c.options.SetSendListLen(DefaultConnSendListLen)
		}
		c.sendCh = make(chan wrapperSendData, c.options.GetSendListLen())
	}

	if tcpConn, ok := c.conn.(*net.TCPConn); ok {
		if c.options.GetNodelay() {
			tcpConn.SetNoDelay(c.options.GetNodelay())
		}
		if c.options.GetKeepAlived() {
			tcpConn.SetKeepAlive(c.options.GetKeepAlived())
		}
		if c.options.GetKeepAlivedPeriod() > 0 {
			tcpConn.SetKeepAlivePeriod(c.options.GetKeepAlivedPeriod())
		}
	}

	var tickSpan = c.options.GetTickSpan()
	if tickSpan > 0 && tickSpan < MinConnTick {
		c.options.SetTickSpan(MinConnTick)
	}

	// resend config
	resendConfig := c.options.GetResendConfig()
	if resendConfig != nil {
		if tickSpan <= 0 || tickSpan > resendConfig.AckSentSpan {
			tickSpan = resendConfig.AckSentSpan
			c.options.SetTickSpan(tickSpan)
		}
	}

	return c
}

// Conn.LocalAddr get local address for connection
func (c *BConn) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
}

// Conn.RemoteAddr get remote address for connection
func (c *BConn) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

// Conn.Run read loop and write loop runing in goroutine
func (c *BConn) Run() {
	go c.readLoop()
	if c.options.GetSendListMode() >= 0 {
		go c.newWriteLoop()
	} else {
		go c.writeLoop()
	}
}

// Conn.readLoop read loop goroutine
func (c *BConn) readLoop() {
	defer func() {
		if err := recover(); err != nil {
			log.WithStack(err)
		}
	}()

	var (
		buf    *[]byte
		offset int32
		r      int
		err    error
	)
	for err == nil {
		if c.options.GetReadTimeout() != 0 {
			err = c.conn.SetReadDeadline(time.Now().Add(c.options.GetReadTimeout()))
			if err != nil {
				break
			}
		}
		if buf == nil {
			buf = pool.GetBuffPool().Alloc(chunkBufferDefaultLength)
			offset = 0
		}
		r, err = c.conn.Read((*buf)[offset:])
		if err == nil {
			offset += int32(r)
			isend := int(offset) == len(*buf)
			select {
			case <-c.closeCh:
				err = c.genErrConnClosed()
			case c.recvCh <- func() *chunk {
				chunk := getChunk()
				chunk.rawbuf = buf
				chunk.datalen = int32(r)
				chunk.isend = isend
				//log.Infof("receive chunk %+v", *chunk)
				return chunk
			}():
				if isend {
					buf = nil
				}
			}
		} else if IsTimeoutError(err) {
			err = nil
		}
	}
	if buf != nil && offset == 0 {
		pool.GetBuffPool().Free(buf)
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

func (c *BConn) realSend(d *wrapperSendData) error {
	if d.data == nil {
		panic("gsnet: wrapper send data nil")
	}

	pt := d.getPacketType()
	b, pb, ba, pba := d.getData()

	var (
		header []byte
		data   []byte
		datas  [][]byte
		err    error
	)
	if b != nil {
		header, data, err = c.packetCodec.Encode(pt, b)
	} else if pb != nil {
		header, data, err = c.packetCodec.Encode(pt, *pb)
	} else if ba != nil {
		header, datas, err = c.packetCodec.EncodeBytesArray(pt, ba)
	} else if pba != nil {
		header, datas, err = c.packetCodec.EncodeBytesPointerArray(pt, pba)
	}

	if err == nil {
		if c.options.GetWriteTimeout() != 0 {
			if err = c.conn.SetWriteDeadline(time.Now().Add(c.options.GetWriteTimeout())); err != nil {
				return err
			}
		}
		_, err = c.writer.Write(header)
		if err != nil {
			return err
		}

		if data != nil {
			var err error
			if c.options.GetWriteTimeout() != 0 {
				if err = c.conn.SetWriteDeadline(time.Now().Add(c.options.GetWriteTimeout())); err != nil {
					return err
				}
			}
			_, err = c.writer.Write(data)
		} else if datas != nil {
			var err error
			if c.options.GetWriteTimeout() != 0 {
				if err = c.conn.SetWriteDeadline(time.Now().Add(c.options.GetWriteTimeout())); err != nil {
					return err
				}
			}
			for i := 0; i < len(datas); i++ {
				_, err = c.writer.Write(datas[i])
				if err != nil {
					break
				}
			}
		}
		// have data in buffer
		if err == nil && c.writer.Buffered() > 0 {
			if c.options.GetWriteTimeout() != 0 {
				err = c.conn.SetWriteDeadline(time.Now().Add(c.options.GetWriteTimeout()))
			}
			if err == nil {
				err = c.writer.Flush()
			}
		}
	}

	// no use resend
	if !d.getResend() {
		d.toFree(b, pb, ba, pba)
	}

	return err
}

// Conn.newWriteLoop new write loop goroutine
func (c *BConn) newWriteLoop() {
	defer func() {
		c.csendList.Finalize()
		if err := recover(); err != nil {
			log.WithStack(err)
		}
	}()
	var err error
	for {
		sd, o := c.csendList.PopFront()
		if !o {
			break
		}
		if err = c.realSend(&sd); err != nil {
			break
		}
	}
	if err != nil {
		c.errWriteCh <- err
	}
	// 关闭写错误通道
	close(c.errWriteCh)
}

// Conn.writeLoop write loop goroutine
func (c *BConn) writeLoop() {
	defer func() {
		// 退出时回收内存池分配的内存
		for d := range c.sendCh {
			d.recycle()
		}
		if err := recover(); err != nil {
			log.WithStack(err)
		}
	}()

	var err error
	for d := range c.sendCh {
		if err = c.realSend(&d); err != nil {
			break
		}
	}
	if err != nil {
		c.errWriteCh <- err
	}
	// 关闭写错误通道
	close(c.errWriteCh)
}

// Conn.Close close connection
func (c *BConn) Close() error {
	return c.closeWait(0)
}

// Conn.CloseWait close connection wait seconds
func (c *BConn) CloseWait(secs int) error {
	return c.closeWait(secs)
}

// Conn.closeWait implementation for close connection
func (c *BConn) closeWait(secs int) error {
	defer func() {
		// 清理接收通道内存池分配的内存
		for d := range c.recvCh {
			putChunk(d)
		}
	}()

	if !atomic.CompareAndSwapInt32(&c.closed, 0, 1) {
		return nil
	}
	if secs != 0 {
		if conn, o := c.conn.(*net.TCPConn); o {
			conn.SetLinger(secs)
		}
	}
	// 连接断开
	err := c.conn.Close()
	// 停止定时器
	if c.ticker != nil {
		c.ticker.Stop()
	}
	close(c.closeCh)
	if c.options.GetSendListMode() >= 0 {
		c.csendList.Close()
	} else {
		if c.sendCh != nil {
			close(c.sendCh)
		}
	}
	return err
}

// Conn.IsClosed the connection is closed
func (c *BConn) IsClosed() bool {
	return atomic.LoadInt32(&c.closed) > 0
}

// Conn.Send send bytes data (发送数据，必須與Close函數在同一goroutine調用)
func (c *BConn) Send(pt packet.PacketType, data []byte, copyData bool) error {
	return c.sendData(pt, data, nil, copyData, nil, nil, packet.MemoryManagementNone, nil)
}

// Conn.SendPoolBuffer send buffer data with pool allocated (发送内存池缓存)
func (c *BConn) SendPoolBuffer(pt packet.PacketType, pData *[]byte, mmType packet.MemoryManagementType) error {
	return c.sendData(pt, nil, nil, false, pData, nil, mmType, nil)
}

// Conn.SendBytesArray send bytes array data (发送缓冲数组)
func (c *BConn) SendBytesArray(pt packet.PacketType, datas [][]byte, copyData bool) error {
	return c.sendData(pt, nil, datas, copyData, nil, nil, packet.MemoryManagementNone, nil)
}

// Conn.SendPoolBufferArray send buffer array data with pool allocated (发送内存池缓存数组)
func (c *BConn) SendPoolBufferArray(pt packet.PacketType, pDatas []*[]byte, mmType packet.MemoryManagementType) error {
	return c.sendData(pt, nil, nil, false, nil, pDatas, mmType, nil)
}

// Conn.send
func (c *BConn) send(data []byte, copyData bool, resendEventHandler IResendEventHandler) error {
	return c.sendData(packet.PacketNormalData, data, nil, copyData, nil, nil, packet.MemoryManagementNone, resendEventHandler)
}

// Conn.sendBytesArray
func (c *BConn) sendBytesArray(datas [][]byte, copyData bool, resendEventHandler IResendEventHandler) error {
	return c.sendData(packet.PacketNormalData, nil, datas, copyData, nil, nil, packet.MemoryManagementNone, resendEventHandler)
}

// Conn.sendPoolBuffer
func (c *BConn) sendPoolBuffer(pData *[]byte, mmType packet.MemoryManagementType, resendEventHandler IResendEventHandler) error {
	return c.sendData(packet.PacketNormalData, nil, nil, false, pData, nil, mmType, resendEventHandler)
}

// Conn.sendPoolBufferArray
func (c *BConn) sendPoolBufferArray(pDatas []*[]byte, mmType packet.MemoryManagementType, resendEventHandler IResendEventHandler) error {
	return c.sendData(packet.PacketNormalData, nil, nil, false, nil, pDatas, mmType, resendEventHandler)
}

// Conn.sendData
func (c *BConn) sendData(pt packet.PacketType, data []byte, datas [][]byte, copyData bool, pData *[]byte, pDataArray []*[]byte, mmType packet.MemoryManagementType, resendEventHandler IResendEventHandler) error {
	if atomic.LoadInt32(&c.closed) > 0 {
		return c.genErrConnClosed()
	}
	var err error
	if c.options.GetSendListMode() >= 0 {
		select {
		case err = <-c.errCh:
			if err == nil {
				return c.genErrConnClosed()
			}
			return err
		case err = <-c.errWriteCh:
			if err == nil {
				return c.genErrConnClosed()
			}
		case <-c.closeCh:
			return c.genErrConnClosed()
		default:
			var useResend bool
			if resendEventHandler != nil {
				useResend = true
			}
			sd := getWrapperSendData(pt, data, datas, copyData, pData, pDataArray, mmType, useResend)
			// 多线程情况下，在发送数据之前调用重发的接口缓存数据，防止从发送完数据到对方收到再回确认包过来时还没来得及缓存造成确认失败
			if resendEventHandler != nil && pt == packet.PacketNormalData {
				resendEventHandler.OnSent(sd.data, sd.pt_mmt)
			}
			err = c.csendList.PushBack(sd)
		}
	} else {
		select {
		case err = <-c.errCh:
			if err == nil {
				return c.genErrConnClosed()
			}
			return err
		case err = <-c.errWriteCh:
			if err == nil {
				return c.genErrConnClosed()
			}
		case <-c.closeCh:
			return c.genErrConnClosed()
		case c.sendCh <- getWrapperSendData(pt, data, datas, copyData, pData, pDataArray, mmType, false):
		default:
			err = ErrSendListFull
		}
	}
	return err
}

// Conn.SendNonblock send data no bloacked (非阻塞发送)
func (c *BConn) SendNonblock(pt packet.PacketType, data []byte, copyData bool) error {
	if atomic.LoadInt32(&c.closed) > 0 {
		return c.genErrConnClosed()
	}
	var err error
	select {
	case err = <-c.errCh:
		if err == nil {
			err = c.genErrConnClosed()
		}
	case err = <-c.errWriteCh:
		if err == nil {
			err = c.genErrConnClosed()
		}
	case <-c.closeCh:
		err = c.genErrConnClosed()
	case c.sendCh <- getWrapperSendBytes(pt, data, copyData, false):
	default:
		err = ErrSendListFull
	}
	return err
}

// Conn.Recv recv packet (接收数据)
func (c *BConn) Recv() (packet.IPacket, error) {
	if atomic.LoadInt32(&c.closed) > 0 {
		return nil, c.genErrConnClosed()
	}
	var (
		pak packet.IPacket
		err error
	)
	select {
	case err = <-c.errCh:
		if err == nil {
			err = c.genErrConnClosed()
		}
	case err = <-c.errWriteCh:
		if err == nil {
			err = c.genErrConnClosed()
		}
	case <-c.closeCh:
		err = c.genErrConnClosed()
	case chk := <-c.recvCh:
		if chk == nil {
			return nil, c.genErrConnClosed()
		}
		pak, err = c.recvChunk(chk)
	}
	return pak, err
}

// Conn.RecvNonblock recv packet no blocked (非阻塞接收数据)
func (c *BConn) RecvNonblock() (packet.IPacket, error) {
	if atomic.LoadInt32(&c.closed) > 0 {
		return nil, ErrConnClosed
	}

	var (
		pak packet.IPacket
		err error
	)

	select {
	case err = <-c.errCh:
		if err == nil {
			err = c.genErrConnClosed()
		}
	case err = <-c.errWriteCh:
		if err == nil {
			err = c.genErrConnClosed()
		}
	case <-c.closeCh:
		err = c.genErrConnClosed()
	case chk := <-c.recvCh:
		if chk == nil {
			err = c.genErrConnClosed()
			break
		}
		pak, err = c.recvChunk(chk)
	default:
		err = ErrRecvListEmpty
	}
	return pak, err
}

// Conn.genErrConnClosed generate connection closed error
func (c *BConn) genErrConnClosed() error {
	return ErrConnClosed
}

// Conn.Wait wait packet or timer (等待选择结果)
func (c *BConn) Wait(ctx context.Context, chPak chan IdWithPacket) (packet.IPacket, int32, error) {
	if atomic.LoadInt32(&c.closed) > 0 {
		return nil, 0, ErrConnClosed
	}

	var (
		p        packet.IPacket
		id       int32
		err      error
		tickerCh <-chan time.Time
		loop     bool = true
	)

	if c.ticker == nil && c.options.GetTickSpan() > 0 {
		c.ticker = time.NewTicker(c.options.GetTickSpan())
	}

	if c.ticker != nil {
		tickerCh = c.ticker.C
	}

	for loop {
		select {
		case <-ctx.Done():
			err = ErrCancelWait
		case chk := <-c.recvCh:
			if chk == nil {
				err = ErrConnClosed
				break
			}
			p, err = c.recvChunk(chk)
		case <-tickerCh:
			id = -1
			loop = false
		case pak, o := <-chPak:
			if o {
				id = pak.id
			} else {
				log.Infof("gsnet: inbound channel is closed")
			}
		case err = <-c.errCh:
			if err == nil {
				err = ErrConnClosed
			}
		case err = <-c.errWriteCh:
			if err == nil {
				err = ErrConnClosed
			}
		}
		if err != nil || p != nil || id > 0 {
			loop = false
		}
	}
	return p, id, err
}

func (c *BConn) recvChunk(chk *chunk) (packet.IPacket, error) {
	if !c.blist.PushBytes(chk.rawbuf, chk.datalen) {
		panic(fmt.Sprintf("!!!!! bl.PushBytes chk.rawbuf(%p) chk.datalen(%v) failed, BytesList %+v", chk.rawbuf, chk.datalen, c.blist))
	}
	putChunk(chk)
	return c.packetCodec.Decode(&c.blist)
}
