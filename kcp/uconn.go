package kcp

import (
	"errors"
	"net"
	"strings"
	"sync/atomic"
	"time"

	"github.com/huoshan017/gsnet/common"
	"github.com/huoshan017/gsnet/log"
	"github.com/huoshan017/gsnet/options"
	kcp "github.com/huoshan017/kcpgo"
)

// implementing net.Conn
type uConn struct {
	conn     *net.UDPConn
	raddr    *net.UDPAddr
	options  *options.Options
	convId   uint32
	token    int64
	state    int32
	cn       uint8
	tid      uint32
	recvList chan struct {
		buffer     mBufferSlice
		dataOffset int16
		dataLen    uint16
	}
	writeCh chan writeInfo
}

func newUConn(ops *options.Options) *uConn {
	c := &uConn{options: ops}
	if c.options.GetKcpMtu() <= 0 {
		c.options.SetKcpMtu(int32(defaultMtu))
	}
	return c
}

func DialUDP(address string, ops *options.Options) (net.Conn, error) {
	network := common.NetProto2Network(ops.GetNetProto())
	if !strings.Contains(network, "udp") {
		network = "udp"
	}
	addr, err := net.ResolveUDPAddr(network, address)
	if err != nil {
		return nil, err
	}

	conn, err := net.DialUDP(network, nil, addr)
	if err != nil {
		return nil, err
	}

	var (
		state  int32
		header frameHeader
	)

	// 三次握手
	for i := 0; i < len(timeoutSynAckTimeList); i++ {
		// send syn
		err = sendSyn(conn, &header)
		if err != nil {
			return nil, err
		}

		state = STATE_SYN_SEND

		// recv synack
		err = recvSynAck(conn, time.Duration(timeoutSynAckTimeList[i])*time.Second, &header)
		if err != nil {
			if common.IsTimeoutError(err) {
				continue
			}
			return nil, err
		}
		break
	}

	// maybe it's timeout error
	if err != nil {
		return nil, err
	}

	// send ack
	err = sendAck(conn, header.convId, header.token)
	if err != nil {
		return nil, err
	}

	state = STATE_ESTABLISHED

	log.Infof("gsnet: kcp client connection established, conversation(%v) token(%v)", header.convId, header.token)

	c := newUConn(ops)
	c.state = state
	c.conn = conn
	c.convId = header.convId
	c.token = header.token

	go c.readLoop()
	return c, nil
}

func (c *uConn) Read(buf []byte) (int, error) {
	return -1, errors.New("gsnet: kcp uConn.Read not allowed call")
}

// 發送data寫入udp，data默認為getKcpMtuBuffer獲得的緩存
func (c *uConn) Write(data []byte) (int, error) {
	if atomic.LoadInt32(&c.state) == STATE_CLOSED {
		return 0, common.ErrConnClosed
	}

	if c.writeCh == nil {
		// 打包數據幀
		var header = frameHeader{frm: FRAME_DATA, convId: c.convId, token: c.token}
		var sdata = encodeDataFrame(data, &header)
		n, e := c.writeDirectly(sdata)
		if e != nil {
			return 0, e
		}
		kcp.RecycleOutputBuffer(data)
		return n, nil
	}

	select {
	case c.writeCh <- struct {
		raddr  *net.UDPAddr
		data   []byte
		convId uint32
		token  int64
	}{raddr: c.raddr, data: data}:
	default:
		kcp.RecycleOutputBuffer(data)
	}
	return 0, nil
}

func (c *uConn) Close() error {
	atomic.StoreInt32(&c.state, STATE_CLOSED)
	var err error
	if c.raddr == nil { //  客户端连接
		err = c.conn.Close()
	} else { // 服务器连接
		if c.recvList != nil {
			close(c.recvList)
		}
	}
	return err
}

func (c *uConn) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
}

func (c *uConn) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

func (c *uConn) SetReadDeadline(t time.Time) error {
	return c.conn.SetReadDeadline(t)
}

func (c *uConn) SetWriteDeadline(t time.Time) error {
	return c.conn.SetWriteDeadline(t)
}

func (c *uConn) SetDeadline(t time.Time) error {
	return c.conn.SetDeadline(t)
}

func (c *uConn) recv(slice mBufferSlice, dataLen uint16) {
	if atomic.LoadInt32(&c.state) == STATE_CLOSED {
		return
	}
	if c.recvList == nil {
		if c.options.GetRecvListLen() == 0 {
			c.options.SetRecvListLen(common.DefaultConnRecvListLen)
		}
		c.recvList = make(chan struct {
			buffer     mBufferSlice
			dataOffset int16
			dataLen    uint16
		}, c.options.GetRecvListLen())
	}
	c.recvList <- struct {
		buffer     mBufferSlice
		dataOffset int16
		dataLen    uint16
	}{buffer: slice, dataOffset: int16(framePrefixAndHeaderLength), dataLen: dataLen}
}

func (c *uConn) readLoop() { // 客户端连接的接收线程
	defer func() {
		if err := recover(); err != nil {
			log.WithStack(err)
		}
	}()
	var (
		mbuf   *mBuffer
		slice  mBufferSlice
		header frameHeader
		o      bool
		err    error
	)
	for err == nil {
		if atomic.LoadInt32(&c.state) == STATE_CLOSED { //
			if mbuf != nil {
				slice, o = toLastSlice(mbuf)
			}
			break
		}
		if mbuf == nil {
			mbuf = getMBuffer()
		}
		slice, err = Read2MBuffer(c.conn, mbuf)
		if err != nil {
			slice, o = toLastSlice(mbuf)
			mbuf = nil
			log.Infof("gsnet: uConn.readLoop Read2MBuffer err: %v", err)
			continue
		}
		// 小于mtu标记为可回收，等后续引用计数为0后就能回收到对象池
		if mbuf.left() < c.options.GetKcpMtu() {
			mbuf.markRecycle()
			mbuf = nil
		}
		if c := checkDecodeFrame(slice.getData(), &header); c < 0 {
			slice, o = toLastSlice(mbuf)
			mbuf = nil
			log.Infof("gsnet: uConn.readLoop checkDecodeFrame failed: %v", c)
			continue
		}
		c.recv(slice, header.dataLen)
	}
	if o {
		slice.finish(putMBuffer)
	}
	if c.recvList != nil {
		close(c.recvList)
	}
}

func (c *uConn) writeDirectly(data []byte) (int, error) {
	var (
		n   int
		err error
	)
	if c.raddr != nil {
		n, err = c.conn.WriteToUDP(data, c.raddr)
	} else {
		n, err = c.conn.Write(data)
	}
	return n, err
}

func toLastSlice(mbuf *mBuffer) (mBufferSlice, bool) {
	var (
		slice mBufferSlice
		o     bool
	)
	if mbuf != nil {
		slice, o = mbuf.lastSlice()
		if o {
			mbuf.markRecycle()
		}
	}
	return slice, o
}

func (c *uConn) reset() {
	c.conn = nil
	c.raddr = nil
	c.options = nil
	c.convId = 0
	c.token = 0
	c.state = 0
	c.cn = 0
	c.tid = 0
	close(c.recvList)
}
