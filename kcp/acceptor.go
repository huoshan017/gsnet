package kcp

import (
	"net"
	"sync"
	"time"

	"github.com/huoshan017/gsnet/log"
	"github.com/huoshan017/gsnet/options"
	pt "github.com/huoshan017/ponu/time"
)

const (
	FRAME_SYN     = 0
	FRAME_SYN_ACK = 1
	FRAME_ACK     = 2
	FRAME_FIN     = 3
	FRAME_PAYLOAD = 4
)

const (
	STATE_CLOSED      int32 = 0
	STATE_LISTENING   int32 = 1
	STATE_SYN_RECV    int32 = 2
	STATE_SYN_SEND    int32 = 3
	STATE_ESTABLISHED int32 = 4
)

const (
	defaultConnChanLen = 4096
)

type reqInfo struct {
	slice mBufferSlice
	addr  *net.UDPAddr
}

type Acceptor struct {
	listenConn    *net.UDPConn
	options       options.ServerOptions
	connMap       sync.Map
	stateMap      sync.Map
	convIdCounter uint32
	tokenCounter  int64
	mtu           int32
	reqCh         chan reqInfo
	tw            *pt.Wheel
	connCh        chan net.Conn
	closeCh       chan struct{}
}

func NewAcceptor(ops options.ServerOptions) *Acceptor {
	currToken := time.Now().UnixMilli()
	a := &Acceptor{
		options:      ops,
		reqCh:        make(chan reqInfo, 4096),
		closeCh:      make(chan struct{}),
		tokenCounter: currToken,
	}
	if a.options.GetConnChanLen() <= 0 {
		a.options.SetConnChanLen(defaultConnChanLen)
		a.connCh = make(chan net.Conn, a.options.GetConnChanLen())
	}
	if a.options.GetKcpMtu() <= 0 {
		a.mtu = defaultMtu
	}
	return a
}

func (a *Acceptor) Listen(addr string) error {
	var (
		laddr *net.UDPAddr
		err   error
	)
	laddr, err = net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return err
	}
	var c *net.UDPConn
	c, err = net.ListenUDP("udp", laddr)
	if err != nil {
		return err
	}
	a.listenConn = c
	return nil
}

func (a *Acceptor) Serve() error {
	// receive new connection
	var (
		mbuf   *mBuffer = getMBuffer()
		slice  mBufferSlice
		raddr  *net.UDPAddr
		err    error
		header frameHeader
	)

	// timer run
	a.tw = pt.NewWheel(100*time.Millisecond, time.Hour)
	defer a.tw.Stop()

	go a.handleConnectRequest()

	for err == nil {
		if mbuf == nil {
			mbuf = getMBuffer()
		}
		slice, raddr, err = ReadFromUDP2MBuffer(a.listenConn, mbuf)
		// 小于mtu标记为可回收，等后续引用计数为0后就能回收到对象池
		if mbuf.left() < a.mtu {
			mbuf.markRecycle()
			mbuf = nil
		}
		if err == nil {
			c, o := a.connMap.Load(raddr.String())
			if !o {
				a.reqCh <- reqInfo{slice: slice, addr: raddr}
			} else {
				cnt := decodeFrameHeader(slice.getData(), &header)
				if slice.skip(cnt) {
					if header.frm == FRAME_PAYLOAD {
						conn := c.(*uConn)
						if conn.recvList == nil {
							conn.recvList = make(chan mBufferSlice, 4096)
						}
						if conn.convId == header.convId && conn.token == header.token {
							conn.recvList <- slice
							continue
						}
					}
				}
				slice.finish(putMBuffer)
			}
		}
	}
	return err
}

func (a *Acceptor) GetNewConnChan() chan net.Conn {
	return a.connCh
}

func (a *Acceptor) Close() {
	close(a.closeCh)
}

func (a *Acceptor) handleConnectRequest() {
	defer func() {
		if err := recover(); err != nil {
			log.WithStack(err)
		}
	}()
	var (
		loop   = true
		err    error
		header frameHeader
	)
	for loop {
		select {
		case info, o := <-a.reqCh:
			if !o {
				break
			}
			decodeFrameHeader(info.slice.getData(), &header)
			addrStr := info.addr.String()
			switch header.frm {
			case FRAME_SYN: // 第一次握手
				conn := getStreamConn()
				conn.state = STATE_SYN_RECV
				if _, o := a.stateMap.LoadOrStore(addrStr, conn); o {
					break
				}
				a.convIdCounter += 1
				a.tokenCounter += 1
				if err = sendSynAck(a.listenConn, info.addr, a.convIdCounter, a.tokenCounter); err != nil {
					log.Infof("gsnet: kcp connection send synack err %v", err)
					break
				}
				tid := a.tw.Add(3*time.Second, func(args []any) {
					if conn.cn >= 2 { // 超出重试次数
						a.stateMap.LoadAndDelete(info.addr.String())
						return
					}
					conn.cn += 1 // timeout once
					if err = sendSynAck(a.listenConn, info.addr, a.convIdCounter, a.tokenCounter); err != nil {
						a.stateMap.Delete(addrStr)
						log.Infof("gsnet: kcp connetion send synack err %v with resend %v", err, conn.cn)
					}
				}, nil)
				conn.state = STATE_SYN_RECV
				conn.tid = tid
				conn.convId = a.convIdCounter
				conn.token = a.tokenCounter
				log.Infof("gsnet: acceptor receive syn on handshake, conversation(%v) token(%v)", a.convIdCounter, a.tokenCounter)
			case FRAME_ACK: // 第三次握手
				c, o := a.stateMap.Load(addrStr)
				if !o {
					log.Infof("gsnet: kcp connection not found state conn with frame ack received")
					break
				}
				conn := c.(*uConn)
				if err := checkAck(conn, header.convId, header.token); err != nil {
					log.Infof("%v", err)
					break
				}
				a.stateMap.Delete(addrStr)
				conn.state = STATE_ESTABLISHED
				conn.conn = a.listenConn
				conn.raddr = info.addr
				a.connMap.Store(addrStr, conn)
				if conn.tid > 0 {
					a.tw.Remove(conn.tid)
				}
				a.connCh <- conn
				log.Infof("gsnet: acceptor receive ack on handshake, new kcp conversation(%v) established", conn.convId)
			default:
				log.Infof("gsnet: kcp connection received unexpected frame with type %v", header.frm)
			}
			info.slice.finish(putMBuffer)
		case t, o := <-a.tw.C:
			if o {
				t.ExecuteFunc()
			}
		case <-a.closeCh:
			loop = false
		}
	}
}

func sendSyn(conn *net.UDPConn, header *frameHeader) error {
	var (
		buf [maxFrameHeaderLength]byte
		err error
	)

	header.frm = FRAME_SYN
	cnt := encodeFrameHeader(header, buf[:])
	_, err = conn.Write(buf[:cnt])
	if err != nil {
		return err
	}
	return nil
}

func recvSynAck(conn *net.UDPConn, timeout time.Duration, header *frameHeader) error {
	var (
		buf [maxFrameHeaderLength]byte
	)
	// recv syn+ack
	conn.SetReadDeadline(time.Now().Add(timeout))
	n, err := conn.Read(buf[:])
	if err != nil {
		return err
	}

	decodeFrameHeader(buf[:n], header)
	if header.frm != FRAME_SYN_ACK {
		return ErrNeedSynAck
	}
	return nil
}

func sendSynAck(conn *net.UDPConn, remoteAddr *net.UDPAddr, conversation uint32, token int64) error {
	var (
		buf    [maxFrameHeaderLength]byte
		header frameHeader
		err    error
	)
	header.frm = FRAME_SYN_ACK
	header.convId = conversation
	header.token = token
	cnt := encodeFrameHeader(&header, buf[:])
	_, err = conn.WriteToUDP(buf[:cnt], remoteAddr)
	if err != nil {
		return err
	}
	return nil
}

func checkAck(conn *uConn, conversation uint32, token int64) error {
	if conversation != conn.convId || token != conn.token {
		return ErrUDPConvToken
	}
	return nil
}

func sendAck(conn *net.UDPConn, conversation uint32, token int64) error {
	var (
		buf    [maxFrameHeaderLength]byte
		header frameHeader
		err    error
	)
	header.frm = FRAME_ACK
	header.convId = conversation
	header.token = token
	cnt := encodeFrameHeader(&header, buf[:])
	_, err = conn.Write(buf[:cnt])
	if err != nil {
		return err
	}
	return nil
}

var (
	streamConnPool sync.Pool
)

func init() {
	streamConnPool = sync.Pool{
		New: func() any {
			return newUConn()
		},
	}
}

func getStreamConn() *uConn {
	return streamConnPool.Get().(*uConn)
}

func putStreamConn(conn *uConn) {
	conn.conn = nil
	conn.convId = 0
	conn.state = 0
	streamConnPool.Put(conn)
}
