package gsnet

import (
	"net"
	"time"
)

type Acceptor struct {
	listener net.Listener
	connCh   chan IConn
	options  *AcceptorOptions
}

const (
	DefaultConnChanLen = 100
)

type AcceptorOptions struct {
	ConnOptions
	ConnChanLen int
}

func NewAcceptor(options *AcceptorOptions) *Acceptor {
	if options.ConnChanLen <= 0 {
		options.ConnChanLen = DefaultConnChanLen
	}
	s := &Acceptor{
		connCh:  make(chan IConn, options.ConnChanLen),
		options: options,
	}
	return s
}

func (s *Acceptor) Listen(addr string) error {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	s.listener = listener

	return nil
}

func (s *Acceptor) Serve() error {
	var delay time.Duration
	var conn net.Conn
	var err error
	for {
		conn, err = s.listener.Accept()
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
	s.listener.Close()
	close(s.connCh)
}
