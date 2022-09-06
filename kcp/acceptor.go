package kcp

import (
	"net"
	"sync"
	"time"

	"github.com/gogo/protobuf/proto"

	"github.com/huoshan017/gsnet/common"
	"github.com/huoshan017/gsnet/log"
	"github.com/huoshan017/gsnet/options"
	"github.com/huoshan017/gsnet/protocol"
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

type uConn struct {
	conn     *net.UDPConn
	convId   uint32
	token    int64
	state    int32
	cn       uint8
	tid      uint32
	recvList chan mBufferSlice
}

func newUConn() *uConn {
	return &uConn{}
}

func DialUDP(network, address string) (net.Conn, error) {
	addr, err := net.ResolveUDPAddr(network, address)
	if err != nil {
		return nil, err
	}
	conn, err := net.DialUDP(network, nil, addr)
	if err != nil {
		return nil, err
	}

	var (
		c      uConn
		convId uint32
		token  int64
	)

	// 三次握手
	for i := 0; i < 3; i++ {
		// send syn
		err = sendSyn(conn)
		if err != nil {
			break
		}

		c.state = STATE_SYN_SEND

		// recv synack
		err = recvSynAck(conn, 3*time.Second, &convId, &token)
		if err != nil {
			if common.IsTimeoutError(err) {
				continue
			}
		}

		// send ack
		err = sendAck(conn, convId, token)
		if err != nil {
			break
		}

		c.state = STATE_ESTABLISHED
		c.conn = conn
		c.convId = convId
		c.token = token
	}

	if err != nil {
		return nil, err
	}

	uc := newUConn()
	*uc = c
	return uc, nil
}

func (c *uConn) Read(buf []byte) (int, error) {
	return c.conn.Read(buf)
}

func (c *uConn) Write(buf []byte) (int, error) {
	return c.conn.Write(buf)
}

func (c *uConn) Close() error {
	c.state = STATE_CLOSED
	close(c.recvList)
	return c.conn.Close()
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
	return &Acceptor{
		options:      ops,
		reqCh:        make(chan reqInfo, 1024),
		closeCh:      make(chan struct{}),
		tokenCounter: currToken,
	}
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
		//buf   = make([]byte, a.mtu)
		mbuf  *mBuffer = getMBuffer()
		slice mBufferSlice
		raddr *net.UDPAddr
		err   error
	)

	// timer run
	a.tw = pt.NewWheel(100*time.Millisecond, time.Hour)
	defer a.tw.Stop()

	go a.handleConnectRequest()

	for err == nil {
		//r, raddr, err = a.listenConn.ReadFromUDP(buf[:])
		if mbuf.left() < a.mtu {
			mbuf = getMBuffer()
		}
		slice, raddr, err = ReadFromUDP2MBuffer(a.listenConn, mbuf)
		if err == nil {
			c, o := a.connMap.Load(raddr.String())
			if !o {
				a.reqCh <- reqInfo{slice: slice, addr: raddr}
			} else {
				conn := c.(*uConn)
				if conn.recvList == nil {
					conn.recvList = make(chan mBufferSlice, 1024)
				}
				var buf [1]byte
				if slice.read(buf[:]) > 0 {
					if buf[0] == FRAME_PAYLOAD {
						conn.recvList <- slice
						continue
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
	var (
		loop = true
		err  error
	)
	for loop {
		select {
		case info, o := <-a.reqCh:
			if !o {
				break
			}
			d := info.slice.getData()
			frm := d[0]
			buf := d[1:]
			addrStr := info.addr.String()
			switch frm {
			case FRAME_SYN: // 第一次握手
				conn := getStreamConn()
				conn.state = STATE_SYN_RECV
				if _, o := a.stateMap.LoadOrStore(addrStr, conn); o {
					break
				}
				a.convIdCounter += 1
				a.tokenCounter += 1
				if err = sendSynAck(conn, a.convIdCounter, a.tokenCounter); err != nil {
					log.Infof("gsnet: kcp connection send synack err %v", err)
					break
				}
				tid := a.tw.Add(3*time.Second, func(args []any) {
					if conn.cn >= 2 { // 超出重试次数
						a.stateMap.LoadAndDelete(info.addr.String())
						return
					}
					conn.cn += 1 // timeout once
					if err = sendSynAck(conn, a.convIdCounter, a.tokenCounter); err != nil {
						a.stateMap.Delete(addrStr)
						log.Infof("gsnet: kcp connetion send synack err %v with resend %v", err, conn.cn)
					}
				}, nil)
				conn.state = STATE_SYN_RECV
				conn.tid = tid
			case FRAME_ACK: // 第三次握手
				c, o := a.stateMap.Load(addrStr)
				if !o {
					log.Infof("gsnet: kcp connection not found state conn with frame ack received")
					break
				}
				conn := c.(*uConn)
				if err := checkAck(conn, buf); err != nil {
					log.Infof("gsnet: kcp connection check ack err %v", err)
					break
				}
				a.stateMap.Delete(addrStr)
				localAddr := a.listenConn.LocalAddr().(*net.UDPAddr)
				conn.conn, err = net.DialUDP("udp", localAddr, info.addr)
				if err != nil {
					log.Infof("gsnet: kcp connection dial to client address %v err %v", info.addr, err)
					break
				}
				conn.state = STATE_ESTABLISHED
				a.connMap.Store(addrStr, conn)
				if conn.tid > 0 {
					a.tw.Remove(conn.tid)
				}
				a.connCh <- conn
			default:
				log.Infof("gsnet: kcp connection received unexpected frame with type %v", frm)
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

func sendSyn(conn *net.UDPConn) error {
	var (
		buf [16]byte
		syn protocol.KcpSendSyn
	)

	buf[0] = FRAME_SYN
	n, err := syn.MarshalTo(buf[1:])
	if err != nil {
		return err
	}
	_, err = conn.Write(buf[:n+1])
	if err != nil {
		return err
	}
	return nil
}

func recvSynAck(conn *net.UDPConn, timeout time.Duration, conversation *uint32, token *int64) error {
	var (
		buf    [64]byte
		synack protocol.KcpSendSynAck
	)
	// recv syn+ack
	conn.SetReadDeadline(time.Now().Add(timeout))
	n, err := conn.Read(buf[:])
	if err != nil {
		return err
	}
	if buf[0] != FRAME_SYN_ACK {
		return ErrNeedSynAck
	}
	err = proto.Unmarshal(buf[1:n], &synack)
	if err != nil {
		return err
	}
	*conversation = synack.Conversation
	*token = synack.Token
	return nil
}

func sendSynAck(conn *uConn, conversation uint32, token int64) error {
	var synack protocol.KcpSendSynAck
	synack.Conversation = conversation
	synack.Token = token
	var buf [64]byte
	buf[0] = FRAME_SYN_ACK
	n, err := synack.MarshalTo(buf[1:])
	if err != nil {
		return err
	}
	_, err = conn.Write(buf[:n+1])
	if err != nil {
		return err
	}
	return nil
}

func checkAck(conn *uConn, buf []byte) error {
	var ack protocol.KcpSendAck
	// recv syn+ack
	err := proto.Unmarshal(buf, &ack)
	if err != nil {
		return err
	}
	if ack.Conversation != conn.convId || ack.Token != conn.token {
		return ErrConvToken
	}
	return nil
}

func sendAck(conn *net.UDPConn, conversation uint32, token int64) error {
	var ack protocol.KcpSendAck
	ack.Conversation = conversation
	ack.Token = token
	var buf [16]byte
	buf[0] = FRAME_ACK
	n, err := ack.MarshalTo(buf[1:])
	if err != nil {
		return err
	}
	_, err = conn.Write(buf[:n+1])
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
