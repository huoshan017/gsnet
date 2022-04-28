package server

import (
	"context"
	"net"
	"reflect"
	"sync"
	"sync/atomic"
	"time"

	"github.com/huoshan017/gsnet/common"
	"github.com/huoshan017/gsnet/log"
	"github.com/huoshan017/gsnet/packet"
)

const (
	DefaultServerMaxConnCount  = 20000
	DefaultSessionCloseChanLen = 100
	DefaultSessionHandleTick   = 20 * time.Millisecond // 默认会话定时器间隔
)

type sessionCloseInfo struct {
	err       error
	sessionId uint64
}

// 服务器
type Server struct {
	acceptor          *Acceptor
	sessHandlerType   reflect.Type
	newHandlerFunc    NewSessionHandlerFunc
	mainTickHandle    func(time.Duration)
	options           ServerOptions
	sessCloseInfoChan chan *sessionCloseInfo
	sessMap           map[uint64]*common.Session
	ctx               context.Context
	cancel            context.CancelFunc
	waitWg            sync.WaitGroup
	endLoopCh         chan struct{}
}

func NewServer(newFunc NewSessionHandlerFunc, options ...common.Option) *Server {
	s := &Server{
		newHandlerFunc: newFunc,
		sessMap:        make(map[uint64]*common.Session),
		endLoopCh:      make(chan struct{}),
	}
	s.initWith(options...)
	return s
}

func NewServerWithHandler(handler common.ISessionEventHandler, options ...common.Option) *Server {
	rf := reflect.TypeOf(handler)
	s := &Server{
		sessHandlerType: rf,
		sessMap:         make(map[uint64]*common.Session),
		endLoopCh:       make(chan struct{}),
	}
	s.initWith(options...)
	return s
}

func NewServerWithOptions(newFunc NewSessionHandlerFunc, options *ServerOptions) *Server {
	s := &Server{
		newHandlerFunc: newFunc,
		options:        *options,
		sessMap:        make(map[uint64]*common.Session),
		endLoopCh:      make(chan struct{}),
	}
	s.init()
	return s
}

func (s *Server) initWith(options ...common.Option) {
	for _, option := range options {
		option(&s.options.Options)
	}
	s.init()
}

func (s *Server) init() {
	if s.options.GetConnMaxCount() <= 0 {
		s.options.SetConnMaxCount(DefaultServerMaxConnCount)
	}
	if s.options.GetErrChanLen() <= 0 {
		s.options.SetErrChanLen(DefaultSessionCloseChanLen)
	}
	if s.options.GetSessionHandleTick() <= 0 {
		s.options.SetSessionHandleTick(DefaultSessionHandleTick)
	}
	if s.options.GetPacketPool() == nil {
		s.options.SetPacketPool(packet.GetDefaultPacketPool())
	}

	s.acceptor = NewAcceptor(s.options)
	s.sessCloseInfoChan = make(chan *sessionCloseInfo, s.options.GetErrChanLen())
	s.ctx, s.cancel = context.WithCancel(context.Background())
}

func (s *Server) Listen(addr string) error {
	err := s.acceptor.Listen(addr)
	if err != nil {
		return err
	}
	return nil
}

func (s *Server) ListenAndServe(addr string) error {
	err := s.Listen(addr)
	if err == nil {
		s.Start()
	}
	return err
}

func (s *Server) SetMainTickHandle(handle func(time.Duration)) {
	s.mainTickHandle = handle
}

func (s *Server) Start() {
	go func() {
		defer func() {
			if err := recover(); err != nil {
				log.WithStack(err)
			}
		}()
		s.acceptor.Serve()
	}()

	var (
		ticker   *time.Ticker
		lastTime time.Time
		tickerCh <-chan time.Time
		conn     net.Conn
		o        bool = true
	)

	if s.mainTickHandle != nil && s.options.GetTickSpan() > 0 {
		ticker = time.NewTicker(s.options.GetTickSpan())
		lastTime = time.Now()
	}

	if ticker != nil {
		tickerCh = ticker.C
	}

	for o {
		select {
		case <-s.endLoopCh:
			o = false
		case conn, o = <-s.acceptor.GetNewConnChan():
			if !o { // 已关闭
				continue
			}
			s.handleConn(conn)
		case <-tickerCh:
			s.handleTick(&lastTime)
		case c := <-s.sessCloseInfoChan:
			s.handleClose(c)
		}
	}
	if ticker != nil {
		ticker.Stop()
	}
}

// 結束
func (s *Server) End() {
	s.acceptor.Close()
	s.cancel()
	s.waitWg.Wait()
	close(s.endLoopCh)
}

func (s *Server) getSessCloseInfoChan() chan *sessionCloseInfo {
	return s.sessCloseInfoChan
}

func (s *Server) handleTick(lastTime *time.Time) {
	now := time.Now()
	tick := now.Sub(*lastTime)
	s.mainTickHandle(tick)
	*lastTime = now
}

func (s *Server) handleHandshake(conn common.IConn, basePacketHandler common.IBasePacketHandler) (bool, error) {
	var (
		pak packet.IPacket
		res int32
		err error
	)
	pak, _, err = conn.Wait(s.ctx, nil)
	if err == nil && pak != nil {
		res, err = basePacketHandler.OnHandleHandshake(pak)
	}
	if pak != nil {
		s.options.GetPacketPool().Put(pak)
	}
	return res == 1, err
}

func (s *Server) handleConn(c net.Conn) {
	if len(s.sessMap) >= s.options.GetConnMaxCount() {
		c.Close()
		log.Info("gsnet: connection to server is maximum")
		return
	}

	var (
		conn          common.IConn
		resendData    *common.ResendData
		packetBuilder *common.PacketBuilder
	)

	// 创建连接
	switch s.options.GetConnDataType() {
	case 1:
		conn = common.NewSimpleConn(c, s.options.Options)
	default:
		packetBuilder = common.NewPacketBuilder(&s.options.Options)
		resendConfig := s.options.GetResendConfig()
		if resendConfig != nil {
			resendData = common.NewResendData(resendConfig)
			conn = common.NewConnUseResend(c, packetBuilder, resendData, &s.options.Options)
		} else {
			conn = common.NewConn(c, packetBuilder, &s.options.Options)
		}
	}

	// 创建包创建器参数获取者
	var argsGetter common.IPacketBuilderArgsGetter
	if packetBuilder != nil {
		argsGetter = &packetBuilderArgsGetter{packetBuilder}
	}

	// 类型的指针值为空，其包含的接口类型的值不一定为空
	resendEventHandler := func() common.IResendEventHandler {
		if resendData == nil {
			return nil
		}
		return resendData
	}()

	// 创建基础包处理器
	basePacketHandler := common.NewDefaultBasePacketHandler4Server(conn, argsGetter, resendEventHandler, &s.options.Options)

	// 先讓連接跑起來
	conn.Run()

	// 创建會話處理器
	var handler common.ISessionEventHandler
	if s.newHandlerFunc == nil {
		v := reflect.New(s.sessHandlerType.Elem())
		handler = v.Interface().(common.ISessionEventHandler)
	} else {
		if s.options.GetNewSessionHandlerFuncArgs() == nil {
			handler = s.newHandlerFunc()
		} else {
			handler = s.newHandlerFunc(s.options.GetNewSessionHandlerFuncArgs()...)
		}
	}

	// 創建會話
	sess := common.NewSession(conn, getNextSessionId())
	sess.SetResendData(resendData)
	s.sessMap[sess.GetId()] = sess
	s.waitWg.Add(1)

	// 會話處理綫程
	go func(conn common.IConn) {
		defer func() {
			if err := recover(); err != nil {
				log.WithStack(err)
			}
		}()

		var (
			err error
			run = true
		)

		// handle handshake
		var complete bool
		for !complete {
			complete, err = s.handleHandshake(conn, basePacketHandler)
			if err != nil {
				break
			}
		}

		// handle packet
		if err == nil {
			handler.OnConnect(sess)

			var (
				lastTime time.Time = time.Now()
				pak      packet.IPacket
				id       int32
			)
			for run {
				pak, id, err = conn.Wait(s.ctx, sess.GetPacketChannel())
				if err == nil {
					if pak != nil {
						if id != 0 {
							inboundHandle := sess.GetInboundHandles()[id]
							if inboundHandle != nil {
								err = inboundHandle(sess, pak)
							} else {
								log.Infof("gsnet: inbound handle with id %v not found", id)
							}
						} else {
							var res, err = basePacketHandler.OnPreHandle(pak)
							if err == nil && res == 0 {
								err = handler.OnPacket(sess, pak)
							}
							if err == nil {
								err = basePacketHandler.OnPostHandle(pak)
							}
						}
						// free packet to pool
						s.options.GetPacketPool().Put(pak)
					} else {
						now := time.Now()
						handler.OnTick(sess, now.Sub(lastTime))
						lastTime = now
						err = basePacketHandler.OnUpdateHandle()
					}
				}

				// process error
				if err != nil {
					if !common.IsNoDisconnectError(err) {
						run = false
					} else {
						handler.OnError(err)
					}
				}
			}
		}

		handler.OnDisconnect(sess, err)
		if s.options.GetConnCloseWaitSecs() > 0 {
			conn.CloseWait(s.options.GetConnCloseWaitSecs())
		} else {
			conn.Close()
		}

		s.getSessCloseInfoChan() <- &sessionCloseInfo{sessionId: sess.GetId(), err: err}
	}(conn)
}

func (s *Server) handleClose(err *sessionCloseInfo) {
	_, o := s.sessMap[err.sessionId]
	if o {
		sess := s.sessMap[err.sessionId]
		if sess != nil {
			sess.GetResendData()
		}
		delete(s.sessMap, err.sessionId)
		s.waitWg.Done()
		log.Info("handleClose sess count ", len(s.sessMap), ", sessionId: ", err.sessionId)
	}
}

type packetBuilderArgsGetter struct {
	getter *common.PacketBuilder
}

func (h *packetBuilderArgsGetter) Get() []any {
	return []any{h.getter.GetCryptoKey()}
}

var (
	globalSessionIdCounter uint64
)

func getNextSessionId() uint64 {
	return atomic.AddUint64(&globalSessionIdCounter, 1)
}
