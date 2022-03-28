package server

import (
	"context"
	"reflect"
	"sync"
	"time"

	"github.com/huoshan017/gsnet/common"
	"github.com/huoshan017/gsnet/common/packet"
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
	sessionIdCounter  uint64
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
	s.init(options...)
	return s
}

func NewServerWithHandler(handler common.ISessionHandler, options ...common.Option) *Server {
	rf := reflect.TypeOf(handler)
	s := &Server{
		sessHandlerType: rf,
		sessMap:         make(map[uint64]*common.Session),
		endLoopCh:       make(chan struct{}),
	}
	s.init(options...)
	return s
}

func (s *Server) init(options ...common.Option) {
	for _, option := range options {
		option(&s.options.Options)
	}
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
	if s.options.GetPacketBuilder() == nil {
		s.options.SetPacketBuilder(packet.GetDefaultPacketBuilder())
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

func (s *Server) SetMainTickHandle(handle func(time.Duration)) {
	s.mainTickHandle = handle
}

func (s *Server) Start() {
	go func() {
		defer func() {
			if err := recover(); err != nil {
				common.GetLogger().WithStack(err)
			}
		}()
		s.acceptor.Serve()
	}()

	var ticker *time.Ticker
	var lastTime time.Time
	if s.mainTickHandle != nil && s.options.GetTickSpan() > 0 {
		ticker = time.NewTicker(s.options.GetTickSpan())
		lastTime = time.Now()
	}

	var conn common.IConn
	var o bool = true
	if ticker != nil {
		for o {
			select {
			case <-s.endLoopCh:
				o = false
			case conn, o = <-s.acceptor.GetNewConnChan():
				if !o { // 已关闭
					continue
				}
				s.handleConn(conn)
			case <-ticker.C:
				now := time.Now()
				tick := now.Sub(lastTime)
				s.mainTickHandle(tick)
				lastTime = now
			case c := <-s.sessCloseInfoChan:
				s.handleClose(c)
			}
		}
		ticker.Stop()
	} else {
		for o {
			select {
			case <-s.endLoopCh:
				o = false
			case conn, o = <-s.acceptor.GetNewConnChan():
				if !o {
					continue
				}
				s.handleConn(conn)
			case c := <-s.sessCloseInfoChan:
				s.handleClose(c)
			}
		}
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

func (s *Server) handleConn(conn common.IConn) {
	if len(s.sessMap) >= s.options.GetConnMaxCount() {
		common.GetLogger().Info("gsnet: connection to server is maximum")
		return
	}

	// 先讓連接跑起來
	conn.Run()

	// 创建會話處理器
	var handler common.ISessionHandler
	if s.newHandlerFunc == nil {
		v := reflect.New(s.sessHandlerType.Elem())
		it := v.Interface()
		handler = it.(common.ISessionHandler)
	} else {
		if s.options.GetNewSessionHandlerFuncArgs() == nil {
			handler = s.newHandlerFunc()
		} else {
			handler = s.newHandlerFunc(s.options.GetNewSessionHandlerFuncArgs()...)
		}
	}

	// 創建會話
	s.sessionIdCounter += 1
	sess := common.NewSession(conn, s.sessionIdCounter)
	s.sessMap[sess.GetId()] = sess
	s.waitWg.Add(1)

	// 會話處理綫程
	go func(conn common.IConn) {
		defer func() {
			if err := recover(); err != nil {
				common.GetLogger().WithStack(err)
			}
		}()

		handler.OnConnect(sess)

		var (
			lastTime time.Time = time.Now()
			pak      packet.IPacket
			err      error
			run      bool = true
		)
		for run {
			pak, err = conn.Wait(s.ctx)
			if err == nil {
				if pak != nil {
					err = handler.OnPacket(sess, pak)
				} else {
					now := time.Now()
					handler.OnTick(sess, now.Sub(lastTime))
					lastTime = now
				}
			}
			s.options.GetPacketPool().Put(pak)
			if err != nil {
				if !common.IsNoDisconnectError(err) {
					run = false
				} else {
					handler.OnError(err)
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
		delete(s.sessMap, err.sessionId)
		s.waitWg.Done()
		common.GetLogger().Info("handleClose sess count ", len(s.sessMap), ", sessionId: ", err.sessionId)
	}
}