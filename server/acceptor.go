package server

import (
	"context"
	"net"
	"time"

	"github.com/huoshan017/gsnet/common"
	"github.com/huoshan017/gsnet/control"
)

type Acceptor struct {
	listener net.Listener
	connCh   chan common.IConn
	options  ServerOptions
	closeCh  chan struct{}
	closed   bool
}

const (
	DefaultConnChanLen = 100
)

func NewAcceptor(options ServerOptions) *Acceptor {
	a := &Acceptor{
		options: options,
		closeCh: make(chan struct{}),
	}
	if a.options.GetConnChanLen() <= 0 {
		a.options.SetConnChanLen(DefaultConnChanLen)
		a.connCh = make(chan common.IConn, a.options.GetConnChanLen())
	}
	return a
}

func (s *Acceptor) Listen(addr string) error {
	var ctrlOptions control.CtrlOptions
	if s.options.GetReuseAddr() {
		ctrlOptions.ReuseAddr = 1
	}
	if s.options.GetReusePort() {
		ctrlOptions.ReusePort = 1
	}
	var lc = net.ListenConfig{
		Control: control.GetControl(ctrlOptions),
	}
	listener, err := lc.Listen(context.Background(), "tcp", addr)
	if err != nil {
		return err
	}
	s.listener = listener
	return nil
}

func (s *Acceptor) ListenAndServe(addr string) error {
	err := s.Listen(addr)
	if err != nil {
		return err
	}
	return s.serve(s.listener)
}

func (s *Acceptor) Serve() error {
	return s.serve(s.listener)
}

func (s *Acceptor) serve(listener net.Listener) error {
	var delay time.Duration
	var conn net.Conn
	var err error
	for {
		select {
		case <-s.closeCh:
			s.listener.Close()
		default:
		}
		conn, err = listener.Accept()
		if err != nil {
			if net_err, ok := err.(net.Error); ok && net_err.Temporary() {
				if delay == 0 {
					delay = 5 * time.Millisecond
				} else {
					delay *= 2
				}
				if max := 1 * time.Second; delay > max {
					delay = max
				}
				time.Sleep(delay)
				continue
			}
			close(s.connCh)
			break
		}
		var c common.IConn
		switch s.options.GetConnDataType() {
		case 1:
			c = common.NewConn(conn, s.options.Options)
		default:
			c = common.NewConn2(conn, s.options.Options)
		}
		s.connCh <- c
	}
	return err
}

func (s *Acceptor) GetNewConnChan() chan common.IConn {
	return s.connCh
}

func (s *Acceptor) Close() {
	if s.closed {
		return
	}
	close(s.closeCh)
	s.closed = true
}
