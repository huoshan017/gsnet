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

// wrapperSendData send data wrapper
type wrapperSendData struct {
	data   interface{}
	pt_mmt int32
}

// wrapperSendData.getData transfer interface{} data to the right type
func (sd *wrapperSendData) getData() ([]byte, *[]byte, [][]byte, []*[]byte) {
	return GetSendData(sd.data)
}

// only free to returns from wrapperSendData.getData with same instance
func (sd *wrapperSendData) toFree(b []byte, pb *[]byte, ba [][]byte, pba []*[]byte) {
	mmt := sd.pt_mmt & 0xffff
	FreeSendData2(packet.MemoryManagementType(mmt), b, pb, ba, pba)
}

// wrapperSendData.recycle recycle free send data to pool
func (sd *wrapperSendData) recycle() {
	if sd.data == nil {
		panic("gsnet: wrapper send data nil")
	}
	b, pb, ba, pba := sd.getData()
	sd.toFree(b, pb, ba, pba)
}

// Conn2 struct
type Conn2 struct {
	conn               net.Conn
	options            Options
	writer             *bufio.Writer
	reader             *bufio.Reader
	recvCh             chan packet.IPacket  // 缓存从网络接收的数据，对应一个接收者一个发送者
	sendCh             chan wrapperSendData // 缓存发往网络的数据，对应一个接收者一个发送者
	closeCh            chan struct{}        // 关闭通道
	closed             int32                // 是否关闭
	errCh              chan error           // 错误通道
	errWriteCh         chan error           // 写错误通道
	ticker             *time.Ticker         // 定时器
	resendEventHandler IResendEventHandler  // 重发事件处理器
}

// NewConn2WithResend create Conn2 instance use resend
func NewConn2UseResend(conn net.Conn, resend IResendEventHandler, options Options) *Conn2 {
	c := &Conn2{
		conn:               conn,
		options:            options,
		closeCh:            make(chan struct{}),
		errCh:              make(chan error, 1),
		errWriteCh:         make(chan error, 1),
		resendEventHandler: resend,
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

	var tickSpan = c.options.GetTickSpan()
	if tickSpan > 0 && tickSpan < MinConnTick {
		c.options.tickSpan = MinConnTick
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

// NewConn2 create a Conn2 instance
func NewConn2(conn net.Conn, options Options) *Conn2 {
	return NewConn2UseResend(conn, nil, options)
}

// Conn2.LocalAddr get local address for connection
func (c *Conn2) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
}

// Conn2.RemoteAddr get remote address for connection
func (c *Conn2) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

// Conn2.Run read loop and write loop runing in goroutine
func (c *Conn2) Run() {
	go c.readLoop()
	go c.writeLoop()
}

// Conn2.readLoop read loop goroutine
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
			// todo 是否一定需要接收goroutine也等待发送goroutine的错误
			//case err = <-c.errWriteCh:
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

// Conn2.writeLoop write loop goroutine
func (c *Conn2) writeLoop() {
	defer func() {
		// 退出时回收内存池分配的内存
		for d := range c.sendCh {
			d.recycle()
		}
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
			if d.data == nil {
				panic("gsnet: wrapper send data nil")
			}

			// 多线程情况下，在发送数据之前调用重发的接口缓存数据，防止发送完对方收到再发确认包过来时还没来得及缓存造成确认失败
			if c.resendEventHandler != nil {
				mmt := packet.MemoryManagementType(d.pt_mmt & 0xffff)
				c.resendEventHandler.OnSent(d.data, mmt)
			}

			pt := packet.PacketType((d.pt_mmt >> 16) & 0xffff)
			b, pb, ba, pba := d.getData()

			if b != nil {
				err = c.options.GetPacketBuilder().EncodeWriteTo(pt, b, c.writer)
			} else if pb != nil {
				err = c.options.GetPacketBuilder().EncodeWriteTo(pt, *pb, c.writer)
			} else if ba != nil {
				err = c.options.GetPacketBuilder().EncodeBytesArrayWriteTo(pt, ba, c.writer)
			} else if pba != nil {
				err = c.options.GetPacketBuilder().EncodeBytesPointerArrayWriteTo(pt, pba, c.writer)
			}
			if err == nil {
				// have data in buffer
				if c.writer.Buffered() > 0 {
					if c.options.writeTimeout != 0 {
						err = c.conn.SetWriteDeadline(time.Now().Add(c.options.writeTimeout))
					}
					if err == nil {
						err = c.writer.Flush()
					}
				}
			}
			// no use resend
			if c.resendEventHandler == nil {
				d.toFree(b, pb, ba, pba)
			}
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

// Conn2.Close close connection
func (c *Conn2) Close() {
	c.closeWait(0)
}

// Conn2.CloseWait close connection wait seconds
func (c *Conn2) CloseWait(secs int) {
	c.closeWait(secs)
}

// Conn2.closeWait implementation for close connection
func (c *Conn2) closeWait(secs int) {
	defer func() {
		// 清理接收通道内存池分配的内存
		for d := range c.recvCh {
			if d.MMType() == packet.MemoryManagementPoolUserManualFree {
				pool.GetBuffPool().Free(d.Data())
			}
		}
	}()

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

// Conn2.IsClosed the connection is closed
func (c *Conn2) IsClosed() bool {
	return atomic.LoadInt32(&c.closed) > 0
}

// Conn2.Send send bytes data (发送数据，必須與Close函數在同一goroutine調用)
func (c *Conn2) Send(pt packet.PacketType, data []byte, copyData bool) error {
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
	case err = <-c.errWriteCh:
		if err == nil {
			return c.genErrConnClosed()
		}
	case <-c.closeCh:
		return c.genErrConnClosed()
	case c.sendCh <- c.getWrapperSendBytes(pt, data, copyData):
	}
	return nil
}

// Conn2.SendPoolBuffer send buffer data with pool allocated (发送内存池缓存)
func (c *Conn2) SendPoolBuffer(pt packet.PacketType, pData *[]byte, mmType packet.MemoryManagementType) error {
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
	case err = <-c.errWriteCh:
		if err == nil {
			return c.genErrConnClosed()
		}
	case <-c.closeCh:
		return c.genErrConnClosed()
	case c.sendCh <- c.getWrapperSendPoolBuffer(pt, pData, mmType):
	}
	return nil
}

// Conn2.SendBytesArray send bytes array data (发送缓冲数组)
func (c *Conn2) SendBytesArray(pt packet.PacketType, datas [][]byte, copyData bool) error {
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
	case err = <-c.errWriteCh:
		if err == nil {
			return c.genErrConnClosed()
		}
	case <-c.closeCh:
		return c.genErrConnClosed()
	case c.sendCh <- c.getWrapperSendBytesArray(pt, datas, copyData):
	}
	return nil
}

// Conn2.SendPoolBufferArray send buffer array data with pool allocated (发送内存池缓存数组)
func (c *Conn2) SendPoolBufferArray(pt packet.PacketType, pDatas []*[]byte, mmType packet.MemoryManagementType) error {
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
	case err = <-c.errWriteCh:
		if err == nil {
			return c.genErrConnClosed()
		}
	case <-c.closeCh:
		return c.genErrConnClosed()
	case c.sendCh <- c.getWrapperSendPoolBufferArray(pt, pDatas, mmType):
	}
	return nil
}

// Conn2.SendNonblock send data no bloacked (非阻塞发送)
func (c *Conn2) SendNonblock(pt packet.PacketType, data []byte, copyData bool) error {
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
	case c.sendCh <- c.getWrapperSendBytes(pt, data, copyData):
	default:
		err = ErrSendChanFull
	}
	return err
}

func mergePacketTypeAndMMT(pt packet.PacketType, mmt packet.MemoryManagementType) int32 {
	return (int32(pt) << 16 & 0x7fff0000) | int32(mmt)
}

// Conn2.getWrapperSendBytes wrap bytes data for send
func (c *Conn2) getWrapperSendBytes(pt packet.PacketType, data []byte, copyData bool) wrapperSendData {
	if !copyData {
		return wrapperSendData{data: data, pt_mmt: mergePacketTypeAndMMT(pt, packet.MemoryManagementSystemGC)}
	}
	b := pool.GetBuffPool().Alloc(int32(len(data)))
	copy(*b, data)
	return wrapperSendData{data: b, pt_mmt: mergePacketTypeAndMMT(pt, packet.MemoryManagementPoolUserManualFree)}
}

// Conn2.getWrapperSendBytesArray wrap bytes array data for send
func (c *Conn2) getWrapperSendBytesArray(pt packet.PacketType, datas [][]byte, copyData bool) wrapperSendData {
	if !copyData {
		return wrapperSendData{data: datas, pt_mmt: mergePacketTypeAndMMT(pt, packet.MemoryManagementSystemGC)}
	}

	var ds [][]byte
	for i := 0; i < len(datas); i++ {
		b := pool.GetBuffPool().Alloc(int32(len(datas[i])))
		copy(*b, datas[i])
		ds = append(ds, *b)
	}

	return wrapperSendData{data: ds, pt_mmt: mergePacketTypeAndMMT(pt, packet.MemoryManagementPoolUserManualFree)}
}

// Conn2.getWrapperSendPoolBuffer wrap pool buffer for send
func (c *Conn2) getWrapperSendPoolBuffer(pt packet.PacketType, pData *[]byte, mt packet.MemoryManagementType) wrapperSendData {
	var sd wrapperSendData
	switch mt {
	case packet.MemoryManagementSystemGC:
		sd.data = pData
		sd.pt_mmt = mergePacketTypeAndMMT(pt, mt) //(int32(pt)<<16)&0x7fff0000 | int32(mt)
	case packet.MemoryManagementPoolFrameworkFree:
		b := pool.GetBuffPool().Alloc(int32(len(*pData)))
		copy(*b, *pData)
		sd.data = b
		sd.pt_mmt = mergePacketTypeAndMMT(pt, mt) //(int32(pt)<<16)&0x7fff0000 | packet.MemoryManagementPoolUserManualFree
	case packet.MemoryManagementPoolUserManualFree:
		sd.data = pData
		sd.pt_mmt = mergePacketTypeAndMMT(pt, mt) //(int32(pt)<<16)&0x7fff0000 | int32(mt)
	}
	return sd
}

// Conn2.getWrapperSendPoolBufferArray wrap pool buffer array data for send
func (c *Conn2) getWrapperSendPoolBufferArray(pt packet.PacketType, pDataArray []*[]byte, mt packet.MemoryManagementType) wrapperSendData {
	var sd wrapperSendData
	switch mt {
	case packet.MemoryManagementSystemGC:
		sd.data = pDataArray
		sd.pt_mmt = mergePacketTypeAndMMT(pt, mt) //(int32(pt)<<16)&0x7fff0000 | int32(mt)
	case packet.MemoryManagementPoolFrameworkFree:
		var pda []*[]byte
		for i := 0; i < len(pDataArray); i++ {
			b := pool.GetBuffPool().Alloc(int32(len(*pDataArray[i])))
			copy(*b, *pDataArray[i])
			pda = append(pda, b)
		}
		sd.data = pda
		sd.pt_mmt = mergePacketTypeAndMMT(pt, mt) //(int32(pt)<<16)&0x7fff0000 | packet.MemoryManagementPoolUserManualFree
	case packet.MemoryManagementPoolUserManualFree:
		sd.data = pDataArray
		sd.pt_mmt = mergePacketTypeAndMMT(pt, mt) //(int32(pt)<<16)&0x7fff0000 | int32(mt)
	}
	return sd
}

// Conn2.Recv recv packet (接收数据)
func (c *Conn2) Recv() (packet.IPacket, error) {
	if atomic.LoadInt32(&c.closed) > 0 {
		return nil, c.genErrConnClosed()
	}
	var pak packet.IPacket
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
	case pak = <-c.recvCh:
		if pak == nil {
			return nil, c.genErrConnClosed()
		}
	}
	return pak, err
}

// Conn2.RecvNonblock recv packet no blocked (非阻塞接收数据)
func (c *Conn2) RecvNonblock() (packet.IPacket, error) {
	if atomic.LoadInt32(&c.closed) > 0 {
		return nil, ErrConnClosed
	}
	var pak packet.IPacket
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
	case pak = <-c.recvCh:
		if pak == nil {
			err = c.genErrConnClosed()
		}
	default:
		err = ErrRecvChanEmpty
	}
	return pak, err
}

// Conn2.genErrConnClosed generate connection closed error
func (c *Conn2) genErrConnClosed() error {
	//debug.PrintStack()
	return ErrConnClosed
}

// Conn2.Wait wait packet or timer (等待选择结果)
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
		case err = <-c.errWriteCh:
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
		case err = <-c.errWriteCh:
			if err == nil {
				err = ErrConnClosed
			}
		}
	}
	return p, err
}
