package common

import (
	"bufio"
	"context"
	"net"
	"sync/atomic"
	"time"

	"github.com/huoshan017/gsnet/log"
	"github.com/huoshan017/gsnet/packet"
	"github.com/huoshan017/gsnet/pool"
)

// Conn struct
type Conn struct {
	conn               net.Conn
	options            *Options
	writer             *bufio.Writer
	reader             *bufio.Reader
	recvCh             chan packet.IPacket  // 缓存从网络接收的数据，对应一个接收者一个发送者
	sendCh             chan wrapperSendData // 缓存发往网络的数据，对应一个接收者一个发送者
	csendList          *condSendList        // 缓存发送队列
	closeCh            chan struct{}        // 关闭通道
	closed             int32                // 是否关闭
	errCh              chan error           // 错误通道
	errWriteCh         chan error           // 写错误通道
	ticker             *time.Ticker         // 定时器
	packetBuilder      IPacketBuilder       // 包创建器
	resendEventHandler IResendEventHandler  // 重发事件处理器
}

// NewConnWithResend create Conn2 instance use resend
func NewConnUseResend(conn net.Conn, packetBuilder IPacketBuilder, resend IResendEventHandler, options *Options) *Conn {
	c := &Conn{
		conn:               conn,
		options:            options,
		closeCh:            make(chan struct{}),
		errCh:              make(chan error, 1),
		errWriteCh:         make(chan error, 1),
		packetBuilder:      packetBuilder,
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

	if c.options.GetRecvChanLen() <= 0 {
		c.options.SetRecvChanLen(DefaultConnRecvChanLen)
	}
	c.recvCh = make(chan packet.IPacket, c.options.recvChanLen)

	if c.options.GetSendListMode() == 0 {
		c.csendList = newCondSendList()
	} else {
		if c.options.GetSendChanLen() <= 0 {
			c.options.SetSendChanLen(DefaultConnSendChanLen)
		}
		c.sendCh = make(chan wrapperSendData, c.options.sendChanLen)
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

// NewConn create a Conn2 instance
func NewConn(conn net.Conn, packetBuilder IPacketBuilder, options *Options) *Conn {
	return NewConnUseResend(conn, packetBuilder, nil, options)
}

// Conn.LocalAddr get local address for connection
func (c *Conn) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
}

// Conn.RemoteAddr get remote address for connection
func (c *Conn) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

// Conn.Run read loop and write loop runing in goroutine
func (c *Conn) Run() {
	go c.readLoop()
	if c.options.GetSendListMode() == 0 {
		go c.newWriteLoop()
	} else {
		go c.writeLoop()
	}
}

// Conn.readLoop read loop goroutine
func (c *Conn) readLoop() {
	defer func() {
		if err := recover(); err != nil {
			log.WithStack(err)
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
		pak, err = c.packetBuilder.DecodeReadFrom(c.reader)
		if err == nil {
			select {
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

func (c *Conn) realSend(d *wrapperSendData) error {
	var err error
	if c.options.writeTimeout != 0 {
		err = c.conn.SetWriteDeadline(time.Now().Add(c.options.writeTimeout))
	}
	if err == nil {
		if d.data == nil {
			panic("gsnet: wrapper send data nil")
		}

		// 多线程情况下，在发送数据之前调用重发的接口缓存数据，防止发送完对方收到再发确认包过来时还没来得及缓存造成确认失败
		if c.resendEventHandler != nil {
			if !c.resendEventHandler.OnSent(d.data, d.pt_mmt) {
				err = ErrSentPacketCacheFull
			}
		}

		if err == nil {
			pt := d.getPacketType()
			b, pb, ba, pba := d.getData()

			if b != nil {
				err = c.packetBuilder.EncodeWriteTo(pt, b, c.writer)
			} else if pb != nil {
				err = c.packetBuilder.EncodeWriteTo(pt, *pb, c.writer)
			} else if ba != nil {
				err = c.packetBuilder.EncodeBytesArrayWriteTo(pt, ba, c.writer)
			} else if pba != nil {
				err = c.packetBuilder.EncodeBytesPointerArrayWriteTo(pt, pba, c.writer)
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
	}
	return err
}

// Conn.newWriteLoop new write loop goroutine
func (c *Conn) newWriteLoop() {
	defer func() {
		c.csendList.recycle()
		if err := recover(); err != nil {
			log.WithStack(err)
		}
	}()
	var err error
	for {
		sd, o := c.csendList.popFront()
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
func (c *Conn) writeLoop() {
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
		err = c.realSend(&d)
		if err != nil {
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
func (c *Conn) Close() {
	c.closeWait(0)
}

// Conn.CloseWait close connection wait seconds
func (c *Conn) CloseWait(secs int) {
	c.closeWait(secs)
}

// Conn.closeWait implementation for close connection
func (c *Conn) closeWait(secs int) {
	defer func() {
		// 清理接收通道内存池分配的内存
		for d := range c.recvCh {
			c.options.GetPacketPool().Put(d)
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
	c.packetBuilder.Close()
	close(c.closeCh)
	if c.options.GetSendListMode() == 0 {
		c.csendList.close()
	} else {
		if c.sendCh != nil {
			close(c.sendCh)
		}
	}
}

// Conn.IsClosed the connection is closed
func (c *Conn) IsClosed() bool {
	return atomic.LoadInt32(&c.closed) > 0
}

// Conn.Send send bytes data (发送数据，必須與Close函數在同一goroutine調用)
func (c *Conn) Send(pt packet.PacketType, data []byte, copyData bool) error {
	return c.sendData(pt, data, nil, copyData, nil, nil, packet.MemoryManagementNone)
}

// Conn.SendPoolBuffer send buffer data with pool allocated (发送内存池缓存)
func (c *Conn) SendPoolBuffer(pt packet.PacketType, pData *[]byte, mmType packet.MemoryManagementType) error {
	return c.sendData(pt, nil, nil, false, pData, nil, mmType)
}

// Conn.SendBytesArray send bytes array data (发送缓冲数组)
func (c *Conn) SendBytesArray(pt packet.PacketType, datas [][]byte, copyData bool) error {
	return c.sendData(pt, nil, datas, copyData, nil, nil, packet.MemoryManagementNone)
}

// Conn.SendPoolBufferArray send buffer array data with pool allocated (发送内存池缓存数组)
func (c *Conn) SendPoolBufferArray(pt packet.PacketType, pDatas []*[]byte, mmType packet.MemoryManagementType) error {
	return c.sendData(pt, nil, nil, false, nil, pDatas, mmType)
}

// Conn.sendData
func (c *Conn) sendData(pt packet.PacketType, data []byte, datas [][]byte, copyData bool, pData *[]byte, pDataArray []*[]byte, mmType packet.MemoryManagementType) error {
	if atomic.LoadInt32(&c.closed) > 0 {
		return c.genErrConnClosed()
	}
	var err error
	if c.options.GetSendListMode() == 0 {
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
			sd := c.getWrapperSendData(pt, data, datas, copyData, pData, pDataArray, mmType)
			c.csendList.pushBack(sd)
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
		case c.sendCh <- c.getWrapperSendData(pt, data, datas, copyData, pData, pDataArray, mmType):
		}
	}
	return nil
}

// Conn.SendNonblock send data no bloacked (非阻塞发送)
func (c *Conn) SendNonblock(pt packet.PacketType, data []byte, copyData bool) error {
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

// Conn.getWrapperSendData  wrapper send data
func (c *Conn) getWrapperSendData(pt packet.PacketType, data []byte, datas [][]byte, copyData bool, pData *[]byte, pDataArray []*[]byte, mt packet.MemoryManagementType) wrapperSendData {
	if data != nil {
		return c.getWrapperSendBytes(pt, data, copyData)
	}
	if datas != nil {
		return c.getWrapperSendBytesArray(pt, datas, copyData)
	}
	if pData != nil {
		return c.getWrapperSendPoolBuffer(pt, pData, mt)
	}
	return c.getWrapperSendPoolBufferArray(pt, pDataArray, mt)
}

// Conn.getWrapperSendBytes  wrapper bytes data for send
func (c *Conn) getWrapperSendBytes(pt packet.PacketType, data []byte, copyData bool) wrapperSendData {
	if !copyData {
		return wrapperSendData{data: data, pt_mmt: mergePacketTypeAndMMT(pt, packet.MemoryManagementSystemGC)}
	}
	b := pool.GetBuffPool().Alloc(int32(len(data)))
	copy(*b, data)
	return wrapperSendData{data: b, pt_mmt: mergePacketTypeAndMMT(pt, packet.MemoryManagementPoolUserManualFree)}
}

// Conn.getWrapperSendBytesArray wrap bytes array data for send
func (c *Conn) getWrapperSendBytesArray(pt packet.PacketType, datas [][]byte, copyData bool) wrapperSendData {
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

// Conn.getWrapperSendPoolBuffer wrap pool buffer for send
func (c *Conn) getWrapperSendPoolBuffer(pt packet.PacketType, pData *[]byte, mt packet.MemoryManagementType) wrapperSendData {
	var sd wrapperSendData
	switch mt {
	case packet.MemoryManagementSystemGC, packet.MemoryManagementPoolUserManualFree:
		sd.data = pData
		sd.pt_mmt = mergePacketTypeAndMMT(pt, mt)
	case packet.MemoryManagementPoolFrameworkFree:
		b := pool.GetBuffPool().Alloc(int32(len(*pData)))
		copy(*b, *pData)
		sd.data = b
		sd.pt_mmt = mergePacketTypeAndMMT(pt, mt)
	}
	return sd
}

// Conn.getWrapperSendPoolBufferArray wrap pool buffer array data for send
func (c *Conn) getWrapperSendPoolBufferArray(pt packet.PacketType, pDataArray []*[]byte, mt packet.MemoryManagementType) wrapperSendData {
	var sd wrapperSendData
	switch mt {
	case packet.MemoryManagementSystemGC, packet.MemoryManagementPoolUserManualFree:
		sd.data = pDataArray
		sd.pt_mmt = mergePacketTypeAndMMT(pt, mt)
	case packet.MemoryManagementPoolFrameworkFree:
		var pda []*[]byte
		for i := 0; i < len(pDataArray); i++ {
			b := pool.GetBuffPool().Alloc(int32(len(*pDataArray[i])))
			copy(*b, *pDataArray[i])
			pda = append(pda, b)
		}
		sd.data = pda
		sd.pt_mmt = mergePacketTypeAndMMT(pt, mt)
	}
	return sd
}

// Conn.Recv recv packet (接收数据)
func (c *Conn) Recv() (packet.IPacket, error) {
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
	case pak = <-c.recvCh:
		if pak == nil {
			return nil, c.genErrConnClosed()
		}
	}
	return pak, err
}

// Conn.RecvNonblock recv packet no blocked (非阻塞接收数据)
func (c *Conn) RecvNonblock() (packet.IPacket, error) {
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
	case pak = <-c.recvCh:
		if pak == nil {
			err = c.genErrConnClosed()
		}
	default:
		err = ErrRecvChanEmpty
	}
	return pak, err
}

// Conn.genErrConnClosed generate connection closed error
func (c *Conn) genErrConnClosed() error {
	//debug.PrintStack()
	return ErrConnClosed
}

// Conn.Wait wait packet or timer (等待选择结果)
func (c *Conn) Wait(ctx context.Context, chPak chan IdWithPacket) (packet.IPacket, int32, error) {
	if atomic.LoadInt32(&c.closed) > 0 {
		return nil, 0, ErrConnClosed
	}

	var (
		p        packet.IPacket
		id       int32
		err      error
		tickerCh <-chan time.Time
	)

	if c.ticker == nil && c.options.tickSpan > 0 {
		c.ticker = time.NewTicker(c.options.tickSpan)
	}

	if c.ticker != nil {
		tickerCh = c.ticker.C
	}

	select {
	case <-ctx.Done():
		err = ErrCancelWait
	case p = <-c.recvCh:
		if p == nil {
			err = ErrConnClosed
		}
	case <-tickerCh:
	case pak, o := <-chPak:
		if o {
			p = pak.pak
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
	return p, id, err
}
