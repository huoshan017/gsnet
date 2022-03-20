package test

import (
	"testing"
	"time"

	"github.com/huoshan017/gsnet/client"
	"github.com/huoshan017/gsnet/common"
	"github.com/huoshan017/gsnet/server"
)

const (
	testAddress = "127.0.0.1:9999"
)

type testClientHandler struct {
	t     *testing.T
	b     *testing.B
	state int32 // 1 客户端模式   2 服务器模式
}

func newTestClientHandler(args ...interface{}) common.ISessionHandler {
	h := &testClientHandler{}
	var o bool
	h.t, o = args[0].(*testing.T)
	if !o {
		h.b, _ = args[0].(*testing.B)
	}
	h.state = args[1].(int32)
	return h
}

func (h *testClientHandler) OnConnect(sess common.ISession) {
	if h.state == 1 {
		if h.t != nil {
			h.t.Logf("connected")
		} else if h.b != nil {
			h.b.Logf("connected")
		}
	}
}

func (h *testClientHandler) OnDisconnect(sess common.ISession, err error) {
	if h.state == 1 {
		if h.t != nil {
			h.t.Logf("disconnected")
		} else if h.b != nil {
			h.b.Logf("disconnected")
		}
	}
}

func (h *testClientHandler) OnData(sess common.ISession, data []byte) error {
	return nil
}

func (h *testClientHandler) OnTick(sess common.ISession, tick time.Duration) {
}

func (h *testClientHandler) OnError(err error) {
	if h.state == 1 {
		if h.t != nil {
			h.t.Errorf("occur err: %v", err)
		} else if h.b != nil {
			h.t.Errorf("occur err: %v", err)
		}
	}
}

func createTestClient(t *testing.T, state int32) *client.Client {
	h := newTestClientHandler(t, state)
	return client.NewClient(h)
}

func createBenchmarkClient(b *testing.B, state int32) *client.Client {
	return client.NewClient(newTestClientHandler(b, state))
}

type testServerHandler struct {
	t     *testing.T
	b     *testing.B
	state int32 // 1 表示服务器模式  2 表示客户端模式
}

func newTestServerHandler(args ...interface{}) common.ISessionHandler {
	h := &testServerHandler{}
	var o bool
	h.t, o = args[0].(*testing.T)
	if !o {
		h.b, _ = args[0].(*testing.B)
	}
	h.state = args[1].(int32)
	return h
}

func (h *testServerHandler) OnConnect(sess common.ISession) {
	if h.state == 1 {
		h.t.Logf("new client(session_id: %v) connected", sess.GetId())
	}
}

func (h *testServerHandler) OnDisconnect(sess common.ISession, err error) {
	if h.state == 1 {
		h.t.Logf("client(session_id: %v) disconnected, err: %v", sess.GetId(), err)
	}
}

func (h *testServerHandler) OnData(sess common.ISession, data []byte) error {
	return sess.Send(data)
}

func (h *testServerHandler) OnTick(sess common.ISession, tick time.Duration) {

}

func (h *testServerHandler) OnError(err error) {
	if h.state == 1 {
		h.t.Logf("server occur err: %v @@@ @@@", err)
	}
}

func createTestServer(t *testing.T, state int32) *server.Server {
	return server.NewServer(newTestServerHandler, common.WithNewSessionHandlerFuncArgs(t, state))
}

func createTestServerWithHandler(t *testing.T, state int32) *server.Server {
	return server.NewServerWithHandler(&testServerHandler{
		t:     t,
		state: state,
	})
}

func createTestServerWithReusePort(t *testing.T, state int32) []*server.Server {
	return []*server.Server{
		server.NewServerWithHandler(&testServerHandler{
			t:     t,
			state: state,
		}, common.WithReuseAddr(true)),
		server.NewServerWithHandler(&testServerHandler{
			t:     t,
			state: state,
		}, common.WithReuseAddr(true)),
	}
}

func createBenchmarkServerWithHandler(b *testing.B, state int32) *server.Server {
	return server.NewServerWithHandler(&testServerHandler{
		b:     b,
		state: state,
	})
}
