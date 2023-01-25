package kcp

import (
	"context"
	"math/rand"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/huoshan017/gsnet/common"
	"github.com/huoshan017/gsnet/control"
	"github.com/huoshan017/gsnet/log"
	"github.com/huoshan017/gsnet/options"
	kcp "github.com/huoshan017/kcpgo"
	plist "github.com/huoshan017/ponu/list"
	pt "github.com/huoshan017/ponu/time"
)

const (
	defaultConnChanLen     = 4096
	defaultReqListLength   = 2048
	maxWriteChanCount      = 20
	writeChanCount         = 5
	defaultWriteListLength = 2048
)

type reqInfo struct {
	slice mBufferSlice
	addr  net.Addr
}

type dataInfo struct {
	slice mBufferSlice
	conn  *uConn
}

type Acceptor struct {
	listenConn    net.PacketConn //*net.UDPConn
	options       *options.ServerOptions
	network       string
	localAddress  *net.UDPAddr
	state         int32
	connMap       sync.Map
	stateMap      sync.Map
	deletedMap    map[string]uint8
	convIdCounter uint32
	tokenCounter  int64
	reqCh         chan reqInfo
	connCh        chan net.Conn
	dataCh        chan dataInfo
	discCh        chan string
	writeChArray  [maxWriteChanCount]chan struct {
		raddr *net.UDPAddr
		data  []byte
	}
	tw      *pt.Wheel
	ran     *rand.Rand
	closeCh chan struct{}
}

func NewAcceptor(ops *options.ServerOptions) *Acceptor {
	currToken := time.Now().UnixMilli()
	a := &Acceptor{
		closeCh:      make(chan struct{}),
		tokenCounter: currToken,
		ran:          rand.New(rand.NewSource(time.Now().UnixMilli())),
	}
	a.options = ops
	if a.options.GetConnChanLen() <= 0 {
		a.options.SetConnChanLen(defaultConnChanLen)
	}
	a.connCh = make(chan net.Conn, a.options.GetConnChanLen())
	if a.options.GetKcpMtu() <= 0 {
		a.options.SetKcpMtu(defaultMtu)
	}
	if a.options.GetBacklogLength() <= 0 {
		a.options.SetBacklogLength(defaultReqListLength)
	}
	a.reqCh = make(chan reqInfo, a.options.GetBacklogLength())
	if a.options.GetRecvListLen() <= 0 {
		a.options.SetRecvListLen(common.DefaultConnRecvListLen)
	}
	a.dataCh = make(chan dataInfo, a.options.GetRecvListLen()*10)
	if a.options.GetSendListLen() <= 0 {
		a.options.SetSendListLen(defaultWriteListLength)
	}
	a.discCh = make(chan string, 256)
	for i := 0; i < writeChanCount; i++ {
		a.writeChArray[i] = make(chan struct {
			raddr *net.UDPAddr
			data  []byte
		}, a.options.GetSendListLen())
	}
	a.deletedMap = make(map[string]uint8)
	return a
}

func (a *Acceptor) Listen(addr string) error {
	var (
		laddr *net.UDPAddr
		err   error
	)
	a.network = common.NetProto2Network(a.options.GetNetProto())
	if !strings.Contains(a.network, "udp") {
		a.network = "udp"
	}
	laddr, err = net.ResolveUDPAddr(a.network, addr)
	if err != nil {
		return err
	}
	a.localAddress = laddr
	var ctrlOptions control.CtrlOptions
	if a.options.GetReuseAddr() {
		ctrlOptions.ReuseAddr = 1
	}
	if a.options.GetReusePort() {
		ctrlOptions.ReusePort = 1
	}
	var lc = net.ListenConfig{
		Control: control.GetControl(ctrlOptions),
	}
	listener, err := lc.ListenPacket(context.Background(), a.network, addr)
	/*var c *net.UDPConn
	c, err = net.ListenUDP(network, laddr)*/
	if err != nil {
		return err
	}
	a.listenConn = listener
	a.state = STATE_LISTENING
	return nil
}

func (a *Acceptor) Serve() error {
	var (
		mbuf  *mBuffer = getMBuffer()
		slice mBufferSlice
		o     bool
		raddr net.Addr
		err   error
	)

	// timer run
	a.tw = pt.NewWheel(time.Hour, pt.WithInterval(100*time.Millisecond))
	defer a.tw.Stop()
	go a.tw.Run()

	go a.handleConnectRequest()
	go a.handle()
	go a.writeLoop()

	for err == nil {
		if mbuf == nil {
			mbuf = getMBuffer()
		}
		slice, raddr, err = ReadFrom2MBuffer(a.listenConn, mbuf)
		// 小于mtu标记为可回收，等后续引用计数为0后就能回收到对象池
		if mbuf.left() < a.options.GetKcpMtu() {
			mbuf.markRecycle()
			mbuf = nil
		}

		if err != nil {
			if mbuf != nil {
				if slice, o = mbuf.lastSlice(); o {
					mbuf.markRecycle()
				}
			}
			continue
		}

		var c any
		c, o = a.connMap.Load(raddr.String())
		if !o { // receive new connection
			a.reqCh <- reqInfo{slice: slice, addr: raddr}
		} else {
			a.dataCh <- dataInfo{slice: slice, conn: c.(*uConn)}
		}
	}
	if o {
		slice.finish(putMBuffer)
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
		writeIndex = a.ran.Int31n(writeChanCount)
		loop       = true
		tid        uint32
		err        error
		header     frameHeader
	)
	for loop {
		select {
		case info, o := <-a.reqCh:
			if !o {
				break
			}
			addrStr := info.addr.String()
			decodeFrameHeader(info.slice.getData(), &header)
			info.slice.finish(putMBuffer)
			if header.frm == FRAME_SYN { // 第一次握手
				_, o = a.stateMap.Load(addrStr)
				if o {
					log.Infof("gsnet: kcp connection received duplicate syn frame for address %v", addrStr)
					break
				}
				conn := getUConn(&a.options.Options)
				conn.state = STATE_SYN_RECV
				conn.writeCh = a.writeChArray[writeIndex]
				conn.raddr = info.addr.(*net.UDPAddr)
				a.stateMap.Store(addrStr, conn)
				a.convIdCounter += 1
				a.tokenCounter += 1
				tid, err = a.sendSynAckWithTimeout(conn, addrStr)
				if tid == 0 || err != nil {
					break
				}
				conn.state = STATE_SYN_RECV
				conn.tid = tid
				conn.convId = a.convIdCounter
				conn.token = a.tokenCounter
			} else if header.frm == FRAME_ACK { // 第三次握手
				var c any
				c, o = a.stateMap.Load(addrStr) // 找不到对应的连接
				if !o {
					if _, o = a.connMap.Load(addrStr); !o { // 没有此连接，应该是synack三次超时没有回复后删掉了连接状态
						var n uint8
						if n, o = a.deletedMap[addrStr]; o {
							log.Infof("gsnet: kcp connection from %v not found state on frame ack received, but found in deleted map value %v", addrStr, n)
						} else {
							log.Infof("gsnet: kcp connection from %v not found state on frame ack received, not found in deleted map", addrStr)
						}
						break
					}
					// 已经建立连接，正常情况不会跑到这里
					break
				}
				conn := c.(*uConn)
				if conn.state == STATE_ESTABLISHED { // 已经建立连接
					break
				}
				a.stateMap.Delete(addrStr)
				a.deletedMap[addrStr] = conn.cn
				if err := checkAck(conn, header.convId, header.token); err != nil {
					sendRst(conn)
					a.discCh <- addrStr
					break
				}
				if conn.tid > 0 {
					a.tw.Cancel(conn.tid)
				}
				conn.state = STATE_ESTABLISHED
				if a.options.GetReusePort() {
					// todo 新建连接绑定到监听端口
					conn.conn, err = net.DialUDP(a.network, a.localAddress, conn.raddr)
					if err != nil {
						sendRst(conn)
						a.discCh <- addrStr
						log.Infof("gsnet: kcp acceptor new connection err: %v", err)
						break
					}
				} else {
					conn.conn = a.listenConn.(*net.UDPConn)
				}
				a.connCh <- conn
				a.connMap.Store(addrStr, conn)
			} else {
				log.Infof("gsnet: kcp connection received unexpected frame with type %v", header.frm)
			}
		case t, o := <-a.tw.C:
			if o {
				t.ExecuteFunc()
			}
		case <-a.closeCh:
			loop = false
		}
	}
}

func (a *Acceptor) sendSynAckWithTimeout(conn *uConn, addrStr string) (uint32, error) {
	if int(conn.cn) >= len(timeoutAckTimeList) { // 超出重试次数
		a.stateMap.Delete(addrStr)
		a.discCh <- addrStr
		//a.deletedMap[addrStr] = conn.cn
		return 0, nil
	}
	var err error
	if err = sendSynAck(conn, a.convIdCounter, a.tokenCounter); err != nil {
		a.stateMap.Delete(addrStr)
		a.discCh <- addrStr
		//a.deletedMap[addrStr] = conn.cn
		return 0, err
	}
	var tid uint32
	tid = a.tw.Add(time.Duration(timeoutAckTimeList[conn.cn])*time.Second, func(args []any) {
		conn.cn += 1 // timeout once
		tid, err = a.sendSynAckWithTimeout(conn, addrStr)
		if tid > 0 {
			conn.tid = tid
		} else {
			a.stateMap.Delete(addrStr)
			a.discCh <- addrStr
			//a.deletedMap[addrStr] = conn.cn
		}
	}, nil)
	return tid, nil
}

func (a *Acceptor) handle() {
	defer func() {
		if err := recover(); err != nil {
			log.WithStack(err)
		}
	}()
	var (
		loop = true
	)
	for loop {
		select {
		case d, o := <-a.dataCh:
			if o {
				d.conn.recv(d.slice)
			}
		case addr, o := <-a.discCh:
			if o {
				var c any
				c, o = a.stateMap.Load(addr)
				if o {
					conn := c.(*uConn)
					conn.Close()
					putUConn(conn)
				}
			}
		case <-a.closeCh:
			loop = false
		}
	}
}

func (a *Acceptor) writeLoop() {
	defer func() {
		if err := recover(); err != nil {
			log.WithStack(err)
		}
	}()
	var (
		v struct {
			raddr *net.UDPAddr
			data  []byte
		}
		o   bool
		run = true
	)
	for run {
		select {
		case v, o = <-a.writeChArray[0]:
		case v, o = <-a.writeChArray[1]:
		case v, o = <-a.writeChArray[2]:
		case v, o = <-a.writeChArray[3]:
		case v, o = <-a.writeChArray[4]:
		case v, o = <-a.writeChArray[5]:
		case v, o = <-a.writeChArray[6]:
		case v, o = <-a.writeChArray[7]:
		case v, o = <-a.writeChArray[8]:
		case v, o = <-a.writeChArray[9]:
		case v, o = <-a.writeChArray[10]:
		case v, o = <-a.writeChArray[11]:
		case v, o = <-a.writeChArray[12]:
		case v, o = <-a.writeChArray[13]:
		case v, o = <-a.writeChArray[14]:
		case v, o = <-a.writeChArray[15]:
		case v, o = <-a.writeChArray[16]:
		case v, o = <-a.writeChArray[17]:
		case v, o = <-a.writeChArray[18]:
		case v, o = <-a.writeChArray[19]:
		case <-a.closeCh:
			run = false
		}
		if run && o {
			_, err := a.listenConn.WriteTo(v.data, v.raddr)
			if err != nil {
				log.Infof("gsnet: kcp Acceptor WriteToUDP to listen socket err: %v", err)
			}
			kcp.RecycleOutputBuffer(v.data)
		}
		o = false
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
	conn.SetReadDeadline(time.Time{})

	decodeFrameHeader(buf[:n], header)
	if header.frm != FRAME_SYN_ACK {
		return ErrNeedSynAck
	}
	return nil
}

func sendSynAck(conn *uConn, conversation uint32, token int64) error {
	var (
		buf    [maxFrameHeaderLength]byte
		header frameHeader
	)
	header.frm = FRAME_SYN_ACK
	header.convId = conversation
	header.token = token
	cnt := encodeFrameHeader(&header, buf[:])
	_, err := conn.Write(buf[:cnt])
	return err
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

func sendRst(conn *uConn) error {
	var (
		buf    [maxFrameHeaderLength]byte
		header frameHeader
		err    error
	)
	header.frm = FRAME_RST
	cnt := encodeFrameHeader(&header, buf[:])
	_, err = conn.Write(buf[:cnt])
	if err != nil {
		return err
	}
	return nil
}

var (
	uconnList *plist.List
	uconnMap  map[*uConn]bool
)

func init() {
	uconnList = plist.New()
	uconnMap = make(map[*uConn]bool, 100)
}

func getUConn(ops *options.Options) *uConn {
	var uconn *uConn
	v, o := uconnList.PopFront()
	if !o {
		uconn = newUConn(ops)
	} else {
		uconn = v.(*uConn)
	}
	delete(uconnMap, uconn)
	return uconn
}

func putUConn(uconn *uConn) {
	if _, o := uconnMap[uconn]; !o {
		return
	}
	uconn.reset()
	uconnList.PushBack(uconn)
}
