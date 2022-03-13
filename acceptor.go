package gsnet

import (
	"context"
	"net"
	"time"

	"github.com/huoshan017/gsnet/control"
)

type Acceptor struct {
	listener net.Listener
	connCh   chan IConn
	options  *AcceptorOptions
	closeCh  chan struct{}
	closed   bool
}

const (
	DefaultConnChanLen = 100
)

type AcceptorOptions struct {
	ConnOptions
	control.CtrlOptions
	ConnChanLen int
}

func NewAcceptor(options *AcceptorOptions) *Acceptor {
	if options.ConnChanLen <= 0 {
		options.ConnChanLen = DefaultConnChanLen
	}
	s := &Acceptor{
		connCh:  make(chan IConn, options.ConnChanLen),
		options: options,
		closeCh: make(chan struct{}),
	}
	return s
}

func (s *Acceptor) Listen(addr string) error {
	var lc = net.ListenConfig{
		Control: control.GetControl(s.options.CtrlOptions),
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
		c := NewConn(conn, &s.options.ConnOptions)
		s.connCh <- c
	}
	return err
}

func (s *Acceptor) GetNewConnChan() chan IConn {
	return s.connCh
}

func (s *Acceptor) Close() {
	if s.closed {
		return
	}
	close(s.closeCh)
	s.closed = true
}
