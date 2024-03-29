package test

import (
	"bytes"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/huoshan017/gsnet/client"
	"github.com/huoshan017/gsnet/common"
	"github.com/huoshan017/gsnet/options"
	"github.com/huoshan017/gsnet/packet"
)

type testClientUseRunHandler struct {
	t          *testing.T
	b          *testing.B
	sentList   [][]byte
	compareNum int32
	totalNum   int32
	ran        *rand.Rand
	isReady    bool
}

func newTestClientUseRunHandler(args ...any) common.ISessionHandler {
	if len(args) < 2 {
		panic("At least need 2 arguments")
	}
	h := &testClientUseRunHandler{ran: rand.New(rand.NewSource(time.Now().UnixNano()))}
	var o bool
	h.t, o = args[0].(*testing.T)
	if !o {
		h.b, _ = args[0].(*testing.B)
	}
	h.totalNum = args[1].(int32)
	return h
}

func (h *testClientUseRunHandler) OnConnect(sess common.ISession) {
	if h.t != nil {
		h.t.Logf("TestClientUseRun connected")
	} else if h.b != nil {
		h.b.Logf("TestClientUseRun connected")
	}
}

func (h *testClientUseRunHandler) OnReady(sess common.ISession) {
	h.isReady = true
	h.compareNum = 0
	if h.t != nil {
		h.t.Logf("TestClientUseRun ready (session %v)", sess.GetId())
	} else if h.b != nil {
		h.b.Logf("TestClientUseRun ready (session %v)", sess.GetId())
	}
}

func (h *testClientUseRunHandler) OnDisconnect(sess common.ISession, err error) {
	if h.t != nil {
		h.t.Logf("TestClientUseRun disconnected, err: %v", err)
	} else if h.b != nil {
		h.b.Logf("TestClientUseRun disconnected, err: %v", err)
	}
}

func (h *testClientUseRunHandler) OnPacket(sess common.ISession, packet packet.IPacket) error {
	data := packet.Data()
	if !bytes.Equal(data, h.sentList[0]) {
		err := fmt.Errorf("compare err with	  %v\r\n			%v", data, h.sentList[0])
		if h.t != nil {
			panic(err)
		} else if h.b != nil {
			panic(err)
		}
	}
	h.sentList = h.sentList[1:]
	h.compareNum += 1
	if h.totalNum > 0 && h.compareNum >= h.totalNum {
		sess.Close()
	}
	h.t.Logf("testClientUseRunHandler.OnPacket compared count %v", h.compareNum)
	return nil
}

func (h *testClientUseRunHandler) OnTick(sess common.ISession, tick time.Duration) {
	if !h.isReady || sess.IsClosed() {
		return
	}
	d := randBytes(100, h.ran)
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
	if h.t != nil {
		h.t.Logf("TestClientUseRun occur err: %v", err)
	} else if h.b != nil {
		h.t.Logf("TestClientUseRun occur err: %v", err)
	}
}

func createTestClientUseRun(t *testing.T, totalNum int32) *client.Client {
	// 启用tick处理
	return client.NewClient(newTestClientUseRunHandler(t, totalNum), options.WithTickSpan(time.Second), options.WithConnDataType(connDataType))
}

func createTestClientUseRunAndUDPKcp(t *testing.T, totalNum int32) *client.Client {
	return client.NewClient(newTestClientUseRunHandler(t, totalNum), options.WithTickSpan(time.Second), options.WithNetProto(options.NetProtoUDP))
}

func TestClientUseRun(t *testing.T) {
	ts := createTestServer(t, 1, 0)
	err := ts.Listen(testAddress)
	if err != nil {
		t.Errorf("server for TestClientUseRun listen err: %+v", err)
		return
	}
	defer ts.End()

	go ts.Serve()

	t.Logf("server for TestClientUseRun running")

	tc := createTestClientUseRun(t, 200)
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

func TestClientUseRunAndUDPKcp(t *testing.T) {
	ts := createTestServerWithUDPKcp(t)
	err := ts.Listen(testAddress)
	if err != nil {
		t.Errorf("server for TestClientUseRunAndUDPKcp listen err: %+v", err)
		return
	}
	defer ts.End()

	go ts.Serve()

	t.Logf("server for TestClientUseRunAndUDPKcp running")

	tc := createTestClientUseRunAndUDPKcp(t, 200)
	err = tc.Connect(testAddress)
	if err != nil {
		t.Errorf("TestClientUseRunAnUDPKcp connect err: %+v", err)
		return
	}
	defer tc.Close()

	t.Logf("TestClientUseRunAndUDPKcp running")

	tc.Run()

	t.Logf("TestClientUseRunAndUDPKcp done")
}

type testClientUseUpdateHandler struct {
	t            *testing.T
	state        int32 // 1 客户端模式   2 服务器模式
	sendDataList *sendDataInfo
	totalNum     int32
	compareNum   int32
}

func newTestClientUseUpdateHandler(args ...any) common.ISessionHandler {
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
	if len(args) > 3 {
		h.totalNum = args[3].(int32)
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

func (h *testClientUseUpdateHandler) OnReady(sess common.ISession) {
	if h.state == 2 {
		if h.t != nil {
			h.t.Logf("ready")
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
	data := packet.Data()
	if o, e = h.sendDataList.compareData(data, true); !o {
		err := fmt.Errorf("compare err: %v", e)
		if h.t != nil {
			panic(err)
		}
	}
	h.compareNum += 1
	if h.compareNum >= h.totalNum {
		sess.Close()
		h.t.Logf("compare %v num, to end", h.compareNum)
	}
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

func createTestClientUseUpdate(t *testing.T, state int32, userData any, count int32, connType int) *client.Client {
	// 启用tick处理
	return client.NewClient(newTestClientUseUpdateHandler(t, state, userData, count), options.WithRunMode(options.RunModeOnlyUpdate), options.WithConnDataType(connType))
}

func testClientUseUpdate(t *testing.T, connType int) {
	ts := createTestServer(t, 1, connType)
	err := ts.Listen(testAddress)
	if err != nil {
		t.Errorf("server for test client listen err: %+v", err)
		return
	}
	defer ts.End()

	go ts.Serve()

	t.Logf("server for test client running")

	sendNum := 10
	sd := createSendDataInfo(int32(sendNum))
	tc := createTestClientUseUpdate(t, 2, sd, 100, connType)
	err = tc.Connect(testAddress)
	if err != nil {
		t.Errorf("test client connect err: %+v", err)
		return
	}
	defer tc.Close()

	ran := rand.New(rand.NewSource(time.Now().UnixNano()))
	for {
		err = tc.Update()
		if err != nil {
			t.Logf("test client update err %v", err)
			break
		}
		d := randBytes(30, ran)
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

func TestClientUseUpdate(t *testing.T) {
	testClientUseUpdate(t, 0)
}

func TestClientWithKConnUseUpdate(t *testing.T) {
	testClientUseUpdate(t, 2)
}

func TestClientAsyncConnect(t *testing.T) {
	ts := createTestServer(t, 1, 0)
	err := ts.Listen(testAddress)
	if err != nil {
		t.Errorf("server for test client listen err: %+v", err)
		return
	}
	defer ts.End()

	go ts.Serve()

	t.Logf("server for test client running")

	sendNum := 10
	sd := createSendDataInfo(int32(sendNum))
	tc := createTestClientUseUpdate(t, 2, sd, 100, 0)
	tc.ConnectAsync(testAddress, 0, func(err error) {
		if err != nil {
			t.Logf("test client connect async err %v", err)
		}
	})
	defer tc.Close()

	ran := rand.New(rand.NewSource(time.Now().UnixNano()))
	for {
		err = tc.Update()
		if err != nil {
			t.Logf("test client update err %v", err)
			break
		}
		if tc.IsConnected() {
			d := randBytes(30, ran)
			err = tc.Send(d, false)
			if err != nil {
				t.Logf("test client send err: %+v", err)
				break
			}
			sd.appendSendData(d)
		}
		time.Sleep(time.Millisecond)
	}

	time.Sleep(time.Second)

	t.Logf("test done")
}

func TestConnectAsyncConnect2(t *testing.T) {
	ts := createTestServer(t, 1, 0)
	err := ts.Listen(testAddress)
	if err != nil {
		t.Errorf("server for test client listen err: %+v", err)
		return
	}
	defer func() {
		ts.End()
		t.Logf("server end")
	}()

	go ts.Serve()

	t.Logf("server for test client running")

	tc := createTestClientUseRun(t, 200)
	tc.ConnectAsync(testAddress, 3*time.Second, func(err error) {
		if err != nil {
			t.Logf("test client connect async err %v", err)
		} else {
			t.Logf("test client connected")
		}
	})
	defer func() {
		tc.Close()
		t.Logf("client end")
	}()

	t.Logf("connecting")

	tc.Run()

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

	go bs.Serve()

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

func createTestClientUseReconnect(t *testing.T, totalNum int32) *client.Client {
	// 启用tick处理
	return client.NewClient(newTestClientUseRunHandler(t, totalNum), options.WithAutoReconnect(true), options.WithTickSpan(time.Second))
}

func TestClientReconnect(t *testing.T) {
	ts := createTestServer(t, 1, 0)
	err := ts.Listen(testAddress)
	if err != nil {
		t.Errorf("server for test client listen err: %+v", err)
		return
	}
	defer func() {
		ts.End()
		t.Logf("server end")
	}()

	go ts.Serve()

	t.Logf("server for test client running")

	tc := createTestClientUseReconnect(t, 10)
	tc.ConnectAsync(testAddress, 3*time.Second, func(err error) {
		if err != nil {
			t.Logf("test client connect async err %v", err)
		} else {
			t.Logf("test client connected")
		}
	})

	t.Logf("connecting")

	go func() {
		time.Sleep(120 * time.Second)
		tc.Close()
	}()

	tc.Run()

	t.Logf("test done")
}
