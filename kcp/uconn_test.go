package kcp

import (
	"net"
	"sync"
	"testing"
	"time"

	"github.com/huoshan017/gsnet/options"
	kcp "github.com/huoshan017/kcpgo"
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
	kcp.UserMtuBufferFunc(getKcpMtuBuffer, putKcpMtuBuffer)
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

	data := []byte("hello")
	wdata := getKcpMtuBuffer(int32(len(data)))
	uconn := conn.(*uConn)
	copy(wdata, data)
	_, err = uconn.writeDirectly(encodeDataFrame(wdata, &frameHeader{frm: FRAME_DATA, convId: uconn.convId, token: uconn.token}))
	if err != nil {
		t.Errorf("uConn write err: %v", err)
		return
	}
	putKcpMtuBuffer(wdata)

	time.Sleep(3 * time.Second)

	acceptor.Close()
}

func TestMultiConnect(t *testing.T) {
	const (
		count = 10000
		data  = "hello"
	)
	var (
		sops     options.ServerOptions
		cops     options.ClientOptions
		acceptor *Acceptor
		wg       sync.WaitGroup
		err      error
	)
	acceptor, err = createAcceptor(t, "127.0.0.1:9000", &sops)
	if err != nil {
		t.Errorf("create acceptor err: %v", err)
		return
	}

	wg.Add(count)
	for i := 0; i < count; i++ {
		go func() {
			defer wg.Done()
			conn, e := DialUDP("127.0.0.1:9000", &cops.Options)
			if e != nil {
				t.Errorf("dial udp err: %v", e)
				return
			}
			wdata := getKcpMtuBuffer(int32(len(data)))
			uconn := conn.(*uConn)
			copy(wdata, data)
			_, e = uconn.writeDirectly(encodeDataFrame(wdata, &frameHeader{frm: FRAME_DATA, convId: uconn.convId, token: uconn.token}))
			if e != nil {
				t.Errorf("uConn write err: %v", e)
				return
			}
		}()
	}

	wg.Wait()
	acceptor.Close()
	time.Sleep(1 * time.Second)
}

func TestConnectTimeout(t *testing.T) {

}
