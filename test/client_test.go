package test

import (
	"bytes"
	"fmt"
	"testing"
	"time"

	"github.com/huoshan017/gsnet/client"
	"github.com/huoshan017/gsnet/common"
	"github.com/huoshan017/gsnet/packet"
)

type testClientUseRunHandler struct {
	t          *testing.T
	b          *testing.B
	state      int32 // 1 客户端模式   2 服务器模式
	sentList   [][]byte
	compareNum int32
}

func newTestClientUseRunHandler(args ...any) common.ISessionEventHandler {
	if len(args) < 2 {
		panic("At least need 2 arguments")
	}
	h := &testClientUseRunHandler{}
	var o bool
	h.t, o = args[0].(*testing.T)
	if !o {
		h.b, _ = args[0].(*testing.B)
	}
	h.state = args[1].(int32)
	return h
}

func (h *testClientUseRunHandler) OnConnect(sess common.ISession) {
	if h.state == 2 {
		if h.t != nil {
			h.t.Logf("TestClientUseRun connected")
		} else if h.b != nil {
			h.b.Logf("TestClientUseRun connected")
		}
	}
}

func (h *testClientUseRunHandler) OnDisconnect(sess common.ISession, err error) {
	if h.state == 2 {
		if h.t != nil {
			h.t.Logf("TestClientUseRun disconnected, err: %v", err)
		} else if h.b != nil {
			h.b.Logf("TestClientUseRun disconnected, err: %v", err)
		}
	}
}

func (h *testClientUseRunHandler) OnPacket(sess common.ISession, packet packet.IPacket) error {
	var (
		e error
	)
	data := *packet.Data()
	if !bytes.Equal(data, h.sentList[0]) {
		err := fmt.Errorf("compare err: %v", e)
		if h.t != nil {
			panic(err)
		} else if h.b != nil {
			panic(err)
		}
	}
	h.sentList = h.sentList[1:]
	h.compareNum += 1
	if h.compareNum >= 100 {
		sess.Close()
	}
	h.t.Logf("testClientUseRunHandler.OnPacket compared %v", data)
	return nil
}

func (h *testClientUseRunHandler) OnTick(sess common.ISession, tick time.Duration) {
	d := randBytes(100)
	err := sess.Send(d, false)
	if err != nil {
		if h.t != nil {
			h.t.Logf("TestClientUseRun sess send data err: %v", err)
		} else if h.b != nil {
			h.t.Logf("TestClientUseRun sess send data err: %v", err)
		}
		return
	}
	h.sentList = append(h.sentList, d)
}

func (h *testClientUseRunHandler) OnError(err error) {
	if h.state == 2 {
		if h.t != nil {
			h.t.Logf("TestClientUseRun occur err: %v", err)
		} else if h.b != nil {
			h.t.Logf("TestClientUseRun occur err: %v", err)
		}
	}
}

func createTestClientUseRun(t *testing.T, state int32) *client.Client {
	// 启用tick处理
	return client.NewClient(newTestClientUseRunHandler(t, state), common.WithTickSpan(time.Millisecond*100), common.WithConnDataType(connDataType))
}

func TestClientUseRun(t *testing.T) {
	ts := createTestServer(t, 1)
	err := ts.Listen(testAddress)
	if err != nil {
		t.Errorf("server for TestClientUseRun listen err: %+v", err)
		return
	}
	defer ts.End()

	go ts.Start()

	t.Logf("server for TestClientUseRun running")

	tc := createTestClientUseRun(t, 2)
	err = tc.Connect(testAddress)
	if err != nil {
		t.Errorf("TestClientUseRun connect err: %+v", err)
		return
	}
	defer tc.Close()

	t.Logf("TestClientUseRun running")

	tc.Run()

	t.Logf("TestClientUseRun done")
}

type testClientUseUpdateHandler struct {
	t            *testing.T
	state        int32 // 1 客户端模式   2 服务器模式
	sendDataList *sendDataInfo
	compareNum   int32
}

func newTestClientUseUpdateHandler(args ...any) common.ISessionEventHandler {
	if len(args) < 2 {
		panic("At least need 2 arguments")
	}
	h := &testClientUseUpdateHandler{}
	var o bool
	h.t, o = args[0].(*testing.T)
	if !o {
		panic("cant transfer to *testing.T")
	}
	h.state = args[1].(int32)
	if len(args) > 2 {
		h.sendDataList, _ = args[2].(*sendDataInfo)
	}
	return h
}

func (h *testClientUseUpdateHandler) OnConnect(sess common.ISession) {
	if h.state == 2 {
		if h.t != nil {
			h.t.Logf("connected")
		}
	}
}

func (h *testClientUseUpdateHandler) OnDisconnect(sess common.ISession, err error) {
	if h.state == 2 {
		if h.t != nil {
			h.t.Logf("disconnected, err: %v", err)
		}
	}
}

func (h *testClientUseUpdateHandler) OnPacket(sess common.ISession, packet packet.IPacket) error {
	var (
		o bool
		e error
	)
	data := *packet.Data()
	if o, e = h.sendDataList.compareData(data, true); !o {
		err := fmt.Errorf("compare err: %v", e)
		if h.t != nil {
			panic(err)
		}
	}
	h.compareNum += 1
	if h.compareNum >= 1000 {
		sess.Close()
	}
	h.t.Logf("compared %v", h.compareNum)
	return nil
}

func (h *testClientUseUpdateHandler) OnTick(sess common.ISession, tick time.Duration) {
}

func (h *testClientUseUpdateHandler) OnError(err error) {
	if h.state == 2 {
		if h.t != nil {
			h.t.Logf("occur err: %v", err)
		}
	}
}

func createTestClientUseUpdate(t *testing.T, state int32, userData any) *client.Client {
	// 启用tick处理
	return client.NewClient(newTestClientUseUpdateHandler(t, state, userData), client.WithRunMode(client.RunModeOnlyUpdate))
}

func TestClientUseUpdate(t *testing.T) {
	ts := createTestServer(t, 1)
	err := ts.Listen(testAddress)
	if err != nil {
		t.Errorf("server for test client listen err: %+v", err)
		return
	}
	defer ts.End()

	go ts.Start()

	t.Logf("server for test client running")

	sendNum := 10
	sd := createSendDataInfo(int32(sendNum))
	tc := createTestClientUseUpdate(t, 2, sd)
	err = tc.Connect(testAddress)
	if err != nil {
		t.Errorf("test client connect err: %+v", err)
		return
	}
	defer tc.Close()

	for {
		err = tc.Update()
		if err != nil {
			t.Logf("test client update err %v", err)
			break
		}
		d := []byte("abcdefghijklmnopqrstuvwxyz0123456789")
		err = tc.Send(d, false)
		if err != nil {
			t.Logf("test client send err: %+v", err)
			break
		}
		sd.appendSendData(d)
		time.Sleep(time.Millisecond)
	}

	time.Sleep(time.Second)

	t.Logf("test done")
}

func BenchmarkClient(b *testing.B) {
	bs := createBenchmarkServerWithHandler(b, 1)
	err := bs.Listen(testAddress)
	if err != nil {
		b.Errorf("server for benchmark client listen err: %+v", err)
		return
	}
	defer bs.End()

	go bs.Start()

	b.Logf("server for benchmark client running")

	sendNum := 10
	sd := createSendDataInfo(int32(sendNum))
	bc := createBenchmarkClient(b, 1, sd)
	err = bc.Connect(testAddress)
	if err != nil {
		b.Errorf("benchmark client connect err: %+v", err)
		return
	}
	go bc.Run()

	b.Logf("benchmark client running")

	for i := 0; i < b.N; i++ {
		err := bc.Send([]byte("abcdefghijklmnopqrstuvwxyz0123456789"), false)
		if err != nil {
			b.Errorf("benchmark client send err: %+v", err)
			return
		}
		time.Sleep(time.Millisecond)
	}

	b.Logf("benchmark done")
}
