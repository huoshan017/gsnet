package test

import (
	"net"
	"testing"
	"time"

	"github.com/huoshan017/gsnet/client"
	"github.com/huoshan017/gsnet/common"
)

func TestConnector(t *testing.T) {
	ts, err := net.Listen("tcp4", testAddress)
	if err != nil {
		t.Errorf("create server listener err: %v", err)
		return
	}

	defer ts.Close()

	go func() {
		var con net.Conn
		for {
			con, err = ts.Accept()
			if err != nil {
				if err != net.ErrClosed {
					t.Errorf("server accept err %v", err)
					break
				}
			}
			t.Logf("accept remote client %v", con.RemoteAddr())
		}
	}()

	var conn net.Conn
	connector := client.NewConnector(&common.Options{})
	conn, err = connector.Connect(testAddress)
	if err != nil {
		t.Errorf("test connector connect address %v err: %v", testAddress, err)
		return
	}
	conn.Close()

	t.Logf("test connector connect done")

	conn, err = connector.ConnectWithTimeout(testAddress, time.Second*10)
	if err != nil {
		t.Errorf("test connector connect address %v with timeout err: %v", testAddress, err)
		return
	}
	conn.Close()
	t.Logf("test connector connect timeout done")

	connected := false
	connector.ConnectAsync(testAddress, time.Second*5, func(e error) {
		if err != nil {
			t.Errorf("test connector async connect err: %v", err)
			return
		}
		connected = true
	})

	waitTime := time.Millisecond * 100
	ticker := time.NewTicker(waitTime)
	for connected == false {
		connector.WaitResult(true)
		<-ticker.C
		t.Logf("wait %v", waitTime)
	}

	t.Logf("test connector async connect done")
}
