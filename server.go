package gsnet

import (
	"reflect"
	"time"
)

const (
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
	handlerTemplate   ISessionHandler
	sessHandlerType   reflect.Type
	mainTickHandle    func(time.Duration)
	options           ServiceOptions
	sessCloseInfoChan chan *sessionCloseInfo
	sessionIdCounter  uint64
	sessMap           map[uint64]*Session
	// todo 增加最大连接数限制
}

func NewServer(handler ISessionHandler, options ...Option) *Server {
	rf := reflect.TypeOf(handler)
	s := &Server{
		handlerTemplate: handler,
		sessHandlerType: rf,
		sessMap:         make(map[uint64]*Session),
	}
	for _, option := range options {
		option(&s.options.Options)
	}

	if s.options.errChanLen <= 0 {
		s.options.errChanLen = DefaultSessionCloseChanLen
	}
	if s.options.sessionHandleTick <= 0 {
		s.options.sessionHandleTick = DefaultSessionHandleTick
	}
	s.sessCloseInfoChan = make(chan *sessionCloseInfo, s.options.errChanLen)
	return s
}

func (s *Server) Init(handler ISessionHandler) {
	s.handlerTemplate = handler
}

func (s *Server) Listen(addr string) error {
	aop := &AcceptorOptions{}
	aop.WriteBuffSize = s.options.writeBuffSize
	aop.ReadBuffSize = s.options.readBuffSize
	aop.SendChanLen = s.options.sendChanLen
	aop.RecvChanLen = s.options.recvChanLen
	aop.DataProto = s.options.dataProto

	s.acceptor = NewAcceptor(aop)
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
				getLogger().WithStack(err)
			}
		}()
		s.acceptor.Serve()
	}()

	var ticker *time.Ticker
	var lastTime time.Time
	if s.mainTickHandle != nil && s.options.tickSpan > 0 {
		ticker = time.NewTicker(s.options.tickSpan)
		lastTime = time.Now()
	}

	var conn IConn
	var o bool = true
	if ticker != nil {
		for o {
			select {
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

func (s *Server) End() {
	s.acceptor.Close()
	close(s.sessCloseInfoChan)
}

func (s *Server) getSessCloseInfoChan() chan *sessionCloseInfo {
	return s.sessCloseInfoChan
}

func (s *Server) handleConn(conn IConn) {
	s.sessionIdCounter += 1
	sess := NewSession(conn, s.sessionIdCounter)
	s.sessMap[sess.id] = sess

	conn.Run()

	go func(conn IConn) {
		defer func() {
			if err := recover(); err != nil {
				getLogger().WithStack(err)
			}
		}()

		var handler ISessionHandler

		// 创建handler
		if s.options.createHandlerFunc == nil {
			v := reflect.New(s.sessHandlerType.Elem())
			it := v.Interface()
			handler = it.(ISessionHandler)
		} else {
			if s.options.createHandlerFuncArgs == nil {
				handler = s.options.createHandlerFunc()
			} else {
				handler = s.options.createHandlerFunc(s.options.createHandlerFuncArgs...)
			}
		}

		handler.OnConnect(sess)

		// 会话处理时间间隔设置到连接
		conn.SetTick(s.options.sessionHandleTick)

		var lastTime time.Time = time.Now()
		var data []byte
		var err error
		for {
			data, err = conn.WaitSelect()
			if err == nil {
				if data != nil {
					err = handler.OnData(sess, data)
				} else {
					now := time.Now()
					handler.OnTick(sess, now.Sub(lastTime))
					lastTime = now
				}
			}
			if err != nil {
				if !IsNoDisconnectError(err) {
					break
				}
				handler.OnError(err)
			}
		}

		handler.OnDisconnect(sess, err)
		sess.Close()

		s.getSessCloseInfoChan() <- &sessionCloseInfo{sessionId: sess.id, err: err}
	}(conn)
}

func (s *Server) handleClose(err *sessionCloseInfo) {
	_, o := s.sessMap[err.sessionId]
	if o {
		delete(s.sessMap, err.sessionId)
	}
}
