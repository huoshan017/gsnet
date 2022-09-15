package kcp

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"sync/atomic"
	"time"

	"github.com/huoshan017/gsnet/common"
	"github.com/huoshan017/gsnet/log"
	"github.com/huoshan017/gsnet/options"
	"github.com/huoshan017/gsnet/packet"
	"github.com/huoshan017/gsnet/pool"
	kcp "github.com/huoshan017/kcpgo"
)

const (
	defaultTickSpan = 50 * time.Millisecond
)

// KConn struct
type KConn struct {
	conn        *uConn
	options     *options.Options
	writer      *bufio.Writer
	kcpCB       *kcp.KcpCB
	blist       packet.BytesList
	sendCh      chan []byte
	closeCh     chan struct{}       // 关闭通道
	closed      int32               // 是否关闭
	errCh       chan error          // 错误通道
	errWriteCh  chan error          // 写错误通道
	ticker      *time.Ticker        // 定时器
	packetCodec common.IPacketCodec // 包解码器
}

// NewConn create Conn instance use resend
func NewKConn(conn net.Conn, packetCodec common.IPacketCodec, options *options.Options) *KConn {
	kconn := conn.(*uConn)
	c := &KConn{
		conn:        kconn,
		options:     options,
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
		c.options.SetRecvListLen(common.DefaultConnRecvListLen)
	}

	c.blist = packet.NewBytesList(128)

	if c.options.GetSendListLen() <= 0 {
		c.options.SetSendListLen(common.DefaultConnSendListLen)
	}
	c.sendCh = make(chan []byte, c.options.GetSendListLen())

	var tickSpan = c.options.GetTickSpan()
	if tickSpan > 0 && tickSpan < common.MinConnTick {
		c.options.SetTickSpan(common.MinConnTick)
	} else if tickSpan == 0 {
		c.options.SetTickSpan(defaultTickSpan)
	}

	// resend config
	resendConfig := c.options.GetResendConfig()
	if resendConfig != nil {
		if tickSpan <= 0 || tickSpan > resendConfig.AckSentSpan {
			tickSpan = resendConfig.AckSentSpan
			c.options.SetTickSpan(tickSpan)
		}
	}

	var mtu int32 = options.GetKcpMtu()
	if mtu == 0 {
		mtu = defaultMtu
	}
	c.kcpCB = kcp.New(kconn.convId, nil, c.outputData, kcp.WithMtu(mtu), kcp.WithStream(true), kcp.WithInterval(int32(c.options.GetTickSpan().Milliseconds())), kcp.WithUserFreeOutputBuf(true))

	return c
}

// Conn.LocalAddr get local address for connection
func (c *KConn) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
}

// Conn.RemoteAddr get remote address for connection
func (c *KConn) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

// Conn.Run read loop and write loop runing in goroutine
func (c *KConn) Run() {
}

// Conn.Close close connection
func (c *KConn) Close() error {
	return c.closeWait(0)
}

// Conn.CloseWait close connection wait seconds
func (c *KConn) CloseWait(secs int) error {
	return c.closeWait(secs)
}

// Conn.closeWait implementation for close connection
func (c *KConn) closeWait(secs int) error {
	defer func() {
		// 清理接收通道内存池分配的内存
		for d := range c.conn.recvList {
			d.finish(putMBuffer)
		}
	}()

	if !atomic.CompareAndSwapInt32(&c.closed, 0, 1) {
		return nil
	}
	// 连接断开
	err := c.conn.Close()
	// 停止定时器
	if c.ticker != nil {
		c.ticker.Stop()
	}
	close(c.closeCh)
	if c.sendCh != nil {
		close(c.sendCh)
	}
	return err
}

// Conn.IsClosed the connection is closed
func (c *KConn) IsClosed() bool {
	return atomic.LoadInt32(&c.closed) > 0
}

// Conn.Send send bytes data (发送数据，必須與Close函數在同一goroutine調用)
func (c *KConn) Send(pt packet.PacketType, data []byte, copyData bool) error {
	header, data, err := c.packetCodec.Encode(pt, data)
	if err != nil {
		return err
	}
	c.kcpCB.Send(header)
	c.kcpCB.Send(data)
	return nil
}

// Conn.SendPoolBuffer send buffer data with pool allocated (发送内存池缓存)
func (c *KConn) SendPoolBuffer(pt packet.PacketType, pData *[]byte, mmType packet.MemoryManagementType) error {
	header, data, err := c.packetCodec.Encode(pt, *pData)
	if err == nil {
		c.kcpCB.Send(header)
		c.kcpCB.Send(data)
	}
	common.FreeSendData2(mmType, nil, pData, nil, nil)
	return err
}

// Conn.SendBytesArray send bytes array data (发送缓冲数组)
func (c *KConn) SendBytesArray(pt packet.PacketType, datas [][]byte, copyData bool) error {
	header, datas, err := c.packetCodec.EncodeBytesArray(pt, datas)
	if err == nil {
		c.kcpCB.Send(header)
		for i := 0; i < len(datas); i++ {
			c.kcpCB.Send(datas[i])
		}
	}
	return err
}

// Conn.SendPoolBufferArray send buffer array data with pool allocated (发送内存池缓存数组)
func (c *KConn) SendPoolBufferArray(pt packet.PacketType, pDatas []*[]byte, mmType packet.MemoryManagementType) error {
	header, datas, err := c.packetCodec.EncodeBytesPointerArray(pt, pDatas)
	if err == nil {
		c.kcpCB.Send(header)
		for i := 0; i < len(datas); i++ {
			c.kcpCB.Send(datas[i])
		}
	}
	common.FreeSendData2(mmType, nil, nil, nil, pDatas)
	return err
}

// Conn.RecvNonblock recv packet no blocked (非阻塞接收数据)
func (c *KConn) RecvNonblock() (packet.IPacket, error) {
	if atomic.LoadInt32(&c.closed) > 0 {
		return nil, common.ErrConnClosed
	}

	var (
		slice mBufferSlice
		p     packet.IPacket
		ok    bool
		err   error
	)

	// decode first, if get packet or error then return
	p, err = c.packetCodec.Decode(&c.blist)
	if err != nil {
		return nil, err
	}
	if p != nil {
		return p, nil
	}

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
	case slice, ok = <-c.conn.recvList:
		if !ok {
			err = c.genErrConnClosed()
			break
		}
		p, err = c.recvMBufferSlice(slice)
	default:
		err = common.ErrRecvListEmpty
	}
	return p, err
}

// Conn.Wait wait packet or timer (等待选择结果)
func (c *KConn) Wait(ctx context.Context, chPak chan common.IdWithPacket) (packet.IPacket, int32, error) {
	if atomic.LoadInt32(&c.closed) > 0 {
		return nil, 0, common.ErrConnClosed
	}

	var (
		slice    mBufferSlice
		p        packet.IPacket
		id       int32
		err      error
		tickerCh <-chan time.Time
		ok       bool
	)

	if c.ticker == nil && c.options.GetTickSpan() > 0 {
		c.ticker = time.NewTicker(c.options.GetTickSpan())
	}

	if c.ticker != nil {
		tickerCh = c.ticker.C
	}

	// decode first, if get packet or error then return
	p, err = c.packetCodec.Decode(&c.blist)
	if err != nil {
		return nil, 0, err
	}

	if p == nil {
		select {
		case <-ctx.Done():
			err = common.ErrCancelWait
		case slice, ok = <-c.conn.recvList:
			if !ok {
				err = common.ErrConnClosed
				break
			}
			p, err = c.recvMBufferSlice(slice)
		case <-tickerCh:
			c.kcpCB.Update(currMs())
			id = -1
		case pak, o := <-chPak:
			if o {
				p = pak.GetPak()
				id = pak.GetId()
			} else {
				log.Infof("gsnet: inbound channel is closed")
			}
		case err = <-c.errCh:
			if err == nil {
				err = common.ErrConnClosed
			}
		case err = <-c.errWriteCh:
			if err == nil {
				err = common.ErrConnClosed
			}
		}
	}

	return p, id, err
}

func (c *KConn) outputData(data []byte, user any) int32 {
	n, err := c.conn.Write(data)
	if err != nil {
		log.Infof("gsnet: KConn.outputData err: %v", err)
		return -1
	}
	return int32(n)
}

func (c *KConn) recvMBufferSlice(slice mBufferSlice) (packet.IPacket, error) {
	c.kcpCB.Input(slice.getData())
	slice.finish(putMBuffer)

	var s = c.kcpCB.PeekSize()
	for s > 0 {
		buf := pool.GetBuffPool().Alloc(s)
		rn := c.kcpCB.Recv(*buf)
		if rn != s {
			panic(fmt.Sprintf("gsnet: kcp peek size %v not equal to recv size %v", s, rn))
		}
		if !c.blist.PushBytes(buf, rn) {
			panic("BytesList blist is full, PushBytes failed")
		}
		s = c.kcpCB.PeekSize()
	}
	return c.packetCodec.Decode(&c.blist)
}

// Conn.genErrConnClosed generate connection closed error
func (c *KConn) genErrConnClosed() error {
	return common.ErrConnClosed
}

var (
	initTime time.Time
)

func init() {
	initTime = time.Now()
}

func currMs() int32 {
	return int32(time.Since(initTime).Milliseconds())
}
