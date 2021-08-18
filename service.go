package gsnet

import (
	"fmt"
	"time"
)

const (
	DefaultServiceErrChanLen = 100
)

type serviceErrInfo struct {
	err       error
	sessionId uint64
}

// 服务
type Service struct {
	acceptor         *Acceptor
	callback         IServiceCallback
	handler          IHandler
	options          ServiceOptions
	errInfoChan      chan *serviceErrInfo
	sessionIdCounter uint64
	sessMap          map[uint64]*Session
}

func NewService(callback IServiceCallback, handler IHandler, options ...Option) *Service {
	s := &Service{
		callback: callback,
		handler:  handler,
		sessMap:  make(map[uint64]*Session),
	}
	for _, option := range options {
		option(&s.options.Options)
	}

	if s.options.ErrChanLen <= 0 {
		s.options.ErrChanLen = DefaultServiceErrChanLen
	}
	s.errInfoChan = make(chan *serviceErrInfo, s.options.ErrChanLen)
	return s
}

func (s *Service) Listen(addr string) error {
	aop := &AcceptorOptions{}
	aop.WriteBuffSize = s.options.WriteBuffSize
	aop.ReadBuffSize = s.options.ReadBuffSize
	aop.SendChanLen = s.options.SendChanLen
	aop.RecvChanLen = s.options.RecvChanLen
	aop.DataProto = s.options.DataProto

	s.acceptor = NewAcceptor(aop)
	err := s.acceptor.Listen(addr)
	if err != nil {
		return err
	}
	return nil
}

func (s *Service) Start() {
	go s.acceptor.Serve()

	var ticker *time.Ticker
	var lastTime time.Time
	if s.callback != nil && s.options.tickSpan > 0 {
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
				s.callback.OnTick(tick)
				lastTime = now
			case err := <-s.errInfoChan:
				s.handleErr(err)
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
			case err := <-s.errInfoChan:
				s.handleErr(err)
			}
		}
	}
}

func (s *Service) End() {
	s.acceptor.Close()
	close(s.errInfoChan)
}

func (s *Service) getErrInfoChan() chan *serviceErrInfo {
	return s.errInfoChan
}

func (s *Service) handleConn(conn IConn) {
	s.sessionIdCounter += 1
	sess := NewSession(conn, s.sessionIdCounter)
	s.sessMap[sess.id] = sess
	if s.callback != nil {
		s.callback.OnConnect(sess.id)
	}

	conn.Run()

	go func(conn IConn) {
		var data []byte
		var err error
		for {
			data, err = conn.Recv()
			if err != nil {
				break
			}
			err = s.handler.HandleData(sess, data)
			if err != nil {
				break
			}
		}
		s.getErrInfoChan() <- &serviceErrInfo{sessionId: sess.id, err: err}
	}(conn)
}

func (s *Service) handleErr(err *serviceErrInfo) {
	if IsNoDisconnectError(err.err) {
		if s.callback != nil {
			s.callback.OnError(err.err)
		}
	} else {
		if s.callback != nil {
			s.callback.OnDisconnect(err.sessionId, err.err)
		}
		sess, o := s.sessMap[err.sessionId]
		if o {
			sess.Close()
			delete(s.sessMap, err.sessionId)
		} else {
			err := fmt.Errorf("netlib: session %v not found, close failed", err.sessionId)
			if s.callback != nil {
				s.callback.OnError(err)
			}
		}
	}
}
