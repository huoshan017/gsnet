package test

import (
	"fmt"
	"testing"
	"time"

	"github.com/huoshan017/gsnet/client"
	"github.com/huoshan017/gsnet/common"
	"github.com/huoshan017/gsnet/common/packet"
)

func TestClient(t *testing.T) {
	ts := createTestServer(t, 2)
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
	tc := createTestClient2(t, 1, sd)
	err = tc.Connect(testAddress)
	if err != nil {
		t.Errorf("test client connect err: %+v", err)
		return
	}
	defer tc.Close()

	go tc.Run()

	t.Logf("test client running")

	for i := 0; i < sendNum; i++ {
		d := []byte("abcdefghijklmnopqrstuvwxyz0123456789")
		err := tc.Send(d, false)
		if err != nil {
			t.Errorf("test client send err: %+v", err)
			return
		}
		sd.appendSendData(d)
		time.Sleep(time.Millisecond)
	}

	time.Sleep(time.Second)

	t.Logf("test done")
}

type testClientUseUpdateHandler struct {
	t            *testing.T
	state        int32 // 1 客户端模式   2 服务器模式
	sendDataList *sendDataInfo
	compareNum   int32
}

func newTestClientUseUpdateHandler(args ...interface{}) common.ISessionEventHandler {
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
	if h.state == 1 {
		if h.t != nil {
			h.t.Logf("connected")
		}
	}
}

func (h *testClientUseUpdateHandler) OnDisconnect(sess common.ISession, err error) {
	if h.state == 1 {
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
	if h.compareNum >= 100 {
		sess.Close()
	}
	h.t.Logf("compared %v", h.compareNum)
	return nil
}

func (h *testClientUseUpdateHandler) OnTick(sess common.ISession, tick time.Duration) {
}

func (h *testClientUseUpdateHandler) OnError(err error) {
	if h.state == 1 {
		if h.t != nil {
			h.t.Logf("occur err: %v", err)
		}
	}
}

func createTestClientUseUpdate(t *testing.T, state int32, userData interface{}) *client.Client {
	// 启用tick处理
	return client.NewClient(newTestClientUseUpdateHandler(t, state, userData), common.WithTickSpan(time.Millisecond*100))
}

func TestClientUseUpdate(t *testing.T) {
	ts := createTestServer(t, 2)
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
	tc := createTestClientUseUpdate(t, 1, sd)
	err = tc.Connect(testAddress)
	if err != nil {
		t.Errorf("test client connect err: %+v", err)
		return
	}
	defer tc.Close()

	for {
		d := []byte("abcdefghijklmnopqrstuvwxyz0123456789")
		err = tc.Send(d, false)
		if err != nil {
			t.Logf("test client send err: %+v", err)
			break
		}
		sd.appendSendData(d)
		err = tc.Update()
		if err != nil {
			t.Logf("test client update err %v", err)
			break
		}
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
