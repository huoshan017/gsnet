package gsnet

import (
	"sync"
	"testing"
	"time"
)

const (
	testAddress = "127.0.0.1:9999"
)

type testServerHandler struct {
	t     *testing.T
	state int32 // 1 表示服务器模式  2 表示客户端模式
}

func newTestServerHandler(args ...interface{}) ISessionHandler {
	h := &testServerHandler{}
	h.t = args[0].(*testing.T)
	h.state = args[1].(int32)
	return h
}

func (h *testServerHandler) OnConnect(sess ISession) {
	if h.state == 1 {
		h.t.Logf("new client(session_id: %v) connected", sess.GetId())
	}
}

func (h *testServerHandler) OnDisconnect(sess ISession, err error) {
	if h.state == 1 {
		h.t.Logf("client(session_id: %v) disconnected, err: %v", sess.GetId(), err)
	}
}

func (h *testServerHandler) OnData(sess ISession, data []byte) error {
	return sess.Send(data)
}

func (h *testServerHandler) OnTick(sess ISession, tick time.Duration) {

}

func (h *testServerHandler) OnError(err error) {
	if h.state == 1 {
		h.t.Logf("occur err: %v", err)
	}
}

func createTestServer(t *testing.T, state int32) *Server {
	return NewServer(newTestServerHandler, SetNewSessionHandlerFuncArgs(t, state))
}

func createTestServerWithHandler(t *testing.T, state int32) *Server {
	return NewServerWithHandler(&testClientHandler{
		t:     t,
		state: state,
	})
}

func testServer(t *testing.T, state int32, typ int32) {
	var ts *Server
	if typ == 0 {
		ts = createTestServer(t, state)
	} else {
		ts = createTestServerWithHandler(t, state)
	}

	defer ts.End()

	err := ts.Listen(testAddress)
	if err != nil {
		t.Errorf("test server listen address %v err: %v", testAddress, err)
		return
	}
	go func() {
		ts.Start()
	}()

	t.Logf("test server is running")

	clientNum := 1000
	var wg sync.WaitGroup
	wg.Add(clientNum)

	// 创建一堆客户端
	for i := 0; i < clientNum; i++ {
		tc := createTestClient(t, 2)
		go func(c *Client, idx int) {
			err := c.Connect(testAddress)
			if err != nil {
				t.Errorf("client for test server connect address %v err: %v", testAddress, err)
				return
			}
			t.Logf("client %v connected server", idx)
			// 另起goroutine执行Client的Run函数
			go func(c *Client) {
				c.Run()
			}(c)
			for i := 0; i < 1000; i++ {
				err := c.Send([]byte("abcdefg"))
				if err != nil {
					t.Errorf("client for test server send data err: %v", err)
					return
				}
			}
			c.Close()
			t.Logf("client %v test done", idx)
			wg.Done()
		}(tc, i)
	}

	wg.Wait()
}

func TestServer(t *testing.T) {
	// 创建并启动服务器
	testServer(t, 1, 0)
	t.Logf("test server done")
	testServer(t, 1, 1)
	t.Logf("test server with handler done")
}
