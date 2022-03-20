package test

import (
	"sync"
	"testing"

	"github.com/huoshan017/gsnet/client"
	"github.com/huoshan017/gsnet/server"
)

func testServer(t *testing.T, state int32, typ int32) {
	var ts []*server.Server
	if typ == 0 {
		ts = []*server.Server{createTestServer(t, state)}
	} else if typ == 1 {
		ts = []*server.Server{createTestServerWithHandler(t, state)}
	} else if typ == 2 {
		ts = createTestServerWithReusePort(t, state)
	} else {
		panic("server invalid type !!!!!!")
	}

	defer func() {
		for _, s := range ts {
			s.End()
		}
	}()

	for _, s := range ts {
		err := s.Listen(testAddress)
		if err != nil {
			t.Errorf("test server listen address %v err: %v", testAddress, err)
			return
		}
		go func(s *server.Server) {
			s.Start()
		}(s)
	}

	t.Logf("test server is running")

	clientNum := 4000
	var wg sync.WaitGroup
	wg.Add(clientNum)

	// 创建一堆客户端
	for i := 0; i < clientNum; i++ {
		tc := createTestClient(t, 2)
		go func(c *client.Client, idx int) {
			defer wg.Done()
			err := c.Connect(testAddress)
			if err != nil {
				t.Errorf("client for test server connect address %v err: %v", testAddress, err)
				return
			}
			t.Logf("client %v connected server", idx)
			// 另起goroutine执行Client的Run函数
			go func(c *client.Client) {
				c.Run()
			}(c)
			for i := 0; i < 1000; i++ {
				err := c.Send([]byte("abcdefghijklmnopqrstuvwxyz01234567890~!@#$%^&*()_+-={}[]|:;'<>?/.,"))
				if err != nil {
					t.Errorf("client for test server send data err: %v", err)
					break
				}
			}
			c.Close()
			t.Logf("client %v test done", idx)
		}(tc, i)
	}

	wg.Wait()
}

func TestServer(t *testing.T) {
	// 创建并启动服务器
	testServer(t, 1, 0)
	t.Logf("test server done")
	//testServer(t, 1, 1)
	//t.Logf("test server with handler done")
	//testServer(t, 1, 2)
	//t.Logf("test server with reuse port")
}

func BenchmarkServer(b *testing.B) {
	bs := createBenchmarkServerWithHandler(b, 2)
	defer bs.End()

	err := bs.Listen(testAddress)
	if err != nil {
		b.Errorf("benchmark server listen address %v err: %v", testAddress, err)
		return
	}
	go func(s *server.Server) {
		s.Start()
	}(bs)

	b.Logf("benchmark server is running")

	clientNum := 4000
	var wg sync.WaitGroup
	wg.Add(clientNum)

	// 创建一堆客户端
	for i := 0; i < clientNum; i++ {
		tc := createBenchmarkClient(b, 2)
		go func(c *client.Client, idx int) {
			defer wg.Done()
			err := c.Connect(testAddress)
			if err != nil {
				b.Errorf("client for benchmark server connect address %v err: %v", testAddress, err)
				return
			}
			b.Logf("client %v connected server", idx)
			// 另起goroutine执行Client的Run函数
			go func(c *client.Client) {
				c.Run()
			}(c)
			for i := 0; i < 1000; i++ {
				err := c.Send([]byte("abcdefghijklmnopqrstuvwxyz01234567890~!@#$%^&*()_+-={}[]|:;'<>?/.,"))
				if err != nil {
					b.Errorf("client for benchmark server send data err: %v", err)
					break
				}
			}
			c.Close()
			b.Logf("client %v test done", idx)
		}(tc, i)
	}

	wg.Wait()
}
