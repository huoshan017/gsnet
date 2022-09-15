package kcp

import (
	"net"
	"testing"
	"time"

	"github.com/huoshan017/gsnet/options"
)

func createAcceptor(t *testing.T, address string, options *options.ServerOptions) (*Acceptor, error) {
	acceptor := NewAcceptor(options)
	err := acceptor.Listen(address)
	if err != nil {
		return nil, err
	}
	go acceptor.Serve()
	return acceptor, nil
}

func TestConnect(t *testing.T) {
	var (
		sops     options.ServerOptions
		cops     options.ClientOptions
		acceptor *Acceptor
		conn     net.Conn
		err      error
	)
	acceptor, err = createAcceptor(t, "127.0.0.1:9000", &sops)
	if err != nil {
		t.Errorf("create acceptor err: %v", err)
		return
	}

	conn, err = DialUDP("127.0.0.1:9000", &cops.Options)
	if err != nil {
		t.Errorf("dial udp err: %v", err)
		return
	}

	time.Sleep(time.Second)

	_, err = conn.Write([]byte("hello"))
	if err != nil {
		t.Errorf("uConn write err: %v", err)
		return
	}

	time.Sleep(3 * time.Second)

	acceptor.Close()
}

func TestMultiConnect(t *testing.T) {
	const (
		count = 2000
	)
	var (
		sops     options.ServerOptions
		cops     options.ClientOptions
		acceptor *Acceptor
		err      error
	)
	acceptor, err = createAcceptor(t, "127.0.0.1:9000", &sops)
	if err != nil {
		t.Errorf("create acceptor err: %v", err)
		return
	}

	for i := 0; i < count; i++ {
		go func() {
			conn, e := DialUDP("127.0.0.1:9000", &cops.Options)
			if e != nil {
				t.Errorf("dial udp err: %v", err)
				return
			}
			_, e = conn.Write([]byte("hello"))
			if e != nil {
				t.Errorf("uConn write err: %v", err)
				return
			}
		}()
	}

	time.Sleep(3 * time.Second)

	acceptor.Close()
}

func TestConnectTimeout(t *testing.T) {

}
