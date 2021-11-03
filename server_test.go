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

func (h *testServerHandler) Init(args ...interface{}) {
	h.t = args[0].(*testing.T)
	h.state = args[1].(int32)
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
	return NewServer(&testServerHandler{}, []interface{}{t, state})
}

func TestServer(t *testing.T) {
	// 创建并启动服务器
	ts := createTestServer(t, 1)
	err := ts.Listen(testAddress)
	if err != nil {
		t.Errorf("test server listen address %v err: %v", testAddress, err)
		return
	}
	go func() {
		ts.Start()
	}()

	t.Logf("test server is running")

	clientNum := 100
	var wg sync.WaitGroup
	wg.Add(clientNum)

	// 创建一堆客户端
	for i := 0; i < clientNum; i++ {
		tc := createTestClient(t, 2)
		go func(c *Client) {
			err := c.Connect(testAddress)
			if err != nil {
				t.Errorf("client for test server connect address %v err: %v", testAddress, err)
				return
			}
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
			wg.Done()
		}(tc)
	}

	wg.Wait()

	t.Logf("test done")
}