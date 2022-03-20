package test

import (
	"testing"
	"time"

	"github.com/huoshan017/gsnet/client"
	"github.com/huoshan017/gsnet/common"
)

func TestConnector(t *testing.T) {
	ts := createTestServer(t, 2)
	err := ts.Listen(testAddress)
	if err != nil {
		t.Errorf("server for test connector listen address %v err: %v", testAddress, err)
		return
	}
	defer ts.End()

	go ts.Start()

	connector := client.NewConnector(&common.Options{})
	err = connector.Connect(testAddress)
	if err != nil {
		t.Errorf("test connector connect address %v err: %v", testAddress, err)
		return
	}

	t.Logf("test connector connect done")

	err = connector.ConnectWithTimeout(testAddress, time.Second*10)
	if err != nil {
		t.Errorf("test connector connect address %v with timeout err: %v", testAddress, err)
		return
	}

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
		connector.WaitResult(0)
		<-ticker.C
		t.Logf("wait %v", waitTime)
	}

	t.Logf("test connector async connect done")
}
