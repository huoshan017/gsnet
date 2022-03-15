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
	acceptor              *Acceptor
	sessHandlerType       reflect.Type
	newHandlerFunc        NewSessionHandlerFunc
	mainTickHandle        func(time.Duration)
	options               ServiceOptions
	sessCloseInfoChan     chan *sessionCloseInfo
	sessionIdCounter      uint64
	sessMap               map[uint64]*Session
	stopSessCloseInfoChan chan struct{} // 停止會話關閉信息通道
	// todo 增加最大连接数限制
}

func NewServer(newFunc NewSessionHandlerFunc, options ...Option) *Server {
	s := &Server{
		newHandlerFunc: newFunc,
		sessMap:        make(map[uint64]*Session),
	}
	s.init(options...)
	return s
}

func NewServerWithHandler(handler ISessionHandler, options ...Option) *Server {
	rf := reflect.TypeOf(handler)
	s := &Server{
		sessHandlerType: rf,
		sessMap:         make(map[uint64]*Session),
	}
	s.init(options...)
	return s
}

func (s *Server) init(options ...Option) {
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
	s.stopSessCloseInfoChan = make(chan struct{})
	s.acceptor = NewAcceptor(options...)
}

func (s *Server) Listen(addr string) error {
	/*aop := &AcceptorOptions{}
	aop.WriteBuffSize = s.options.writeBuffSize
	aop.ReadBuffSize = s.options.readBuffSize
	aop.SendChanLen = s.options.sendChanLen
	aop.RecvChanLen = s.options.recvChanLen
	aop.DataProto = s.options.dataProto
	if s.options.reuseAddr {
		aop.ReuseAddr = 1
	}
	if s.options.reusePort {
		aop.ReusePort = 1
	}
	*/
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

// 結束
func (s *Server) End() {
	s.acceptor.Close()
	// todo 不能這麽關閉channel，因爲這是多個發送一個接收的通道
	close(s.stopSessCloseInfoChan)
	// todo 缺少通知所有連接關閉的處理
}

func (s *Server) getSessCloseInfoChan() chan *sessionCloseInfo {
	return s.sessCloseInfoChan
}

func (s *Server) handleConn(conn IConn) {
	conn.Run()

	// 创建會話處理器
	var handler ISessionHandler
	if s.newHandlerFunc == nil {
		v := reflect.New(s.sessHandlerType.Elem())
		it := v.Interface()
		handler = it.(ISessionHandler)
	} else {
		if s.options.createHandlerFuncArgs == nil {
			handler = s.newHandlerFunc()
		} else {
			handler = s.newHandlerFunc(s.options.createHandlerFuncArgs...)
		}
	}

	// 創建會話
	s.sessionIdCounter += 1
	sess := NewSession(conn, s.sessionIdCounter)
	s.sessMap[sess.id] = sess

	// 會話處理綫程
	go func(conn IConn) {
		defer func() {
			if err := recover(); err != nil {
				getLogger().WithStack(err)
			}
		}()

		handler.OnConnect(sess)

		// 会话处理时间间隔设置到连接
		conn.SetTick(s.options.sessionHandleTick)

		var (
			lastTime time.Time = time.Now()
			data     []byte
			err      error
			run      bool = true
		)
		for run {
			select {
			case <-s.stopSessCloseInfoChan:
				run = false
				continue
			default:
			}
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
