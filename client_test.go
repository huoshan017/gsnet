package gsnet

import (
	"testing"
	"time"
)

type testClientHandler struct {
	t     *testing.T
	state int32 // 1 客户端模式   2 服务器模式
}

func newTestClientHandler(args ...interface{}) ISessionHandler {
	h := &testClientHandler{}
	h.t = args[0].(*testing.T)
	h.state = args[1].(int32)
	return h
}

func (h *testClientHandler) OnConnect(sess ISession) {
	if h.state == 1 {
		h.t.Logf("connected")
	}
}

func (h *testClientHandler) OnDisconnect(sess ISession, err error) {
	if h.state == 1 {
		h.t.Logf("disconnected")
	}
}

func (h *testClientHandler) OnData(sess ISession, data []byte) error {
	return nil
}

func (h *testClientHandler) OnTick(sess ISession, tick time.Duration) {
}

func (h *testClientHandler) OnError(err error) {
	if h.state == 1 {
		h.t.Errorf("occur err: %v", err)
	}
}

func createTestClient(t *testing.T, state int32) *Client {
	h := newTestClientHandler(t, state)
	return NewClient(h)
}

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

	tc := createTestClient(t, 1)
	err = tc.Connect(testAddress)
	if err != nil {
		t.Errorf("test client connect err: %+v", err)
		return
	}
	go tc.Run()

	t.Logf("test client running")

	for i := 0; i < 10000; i++ {
		err := tc.Send([]byte("abcdefghijklmnopqrstuvwxyz0123456789"))
		if err != nil {
			t.Errorf("test client send err: %+v", err)
			return
		}
		time.Sleep(time.Millisecond)
	}

	t.Logf("test done")
}
