package test

import (
	"math/rand"
	"sync"
	"sync/atomic"
	"testing"
	"time"

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
	sendNum := 1000
	n := int32(0)
	var wg sync.WaitGroup
	wg.Add(clientNum)

	// 创建一堆客户端
	for i := 0; i < clientNum; i++ {
		go func(idx int) {
			defer wg.Done()

			sd := createSendDataInfo(int32(sendNum))
			c := createTestClient(t, 2, sd)
			err := c.Connect(testAddress)
			if err != nil {
				nn := atomic.AddInt32(&n, 1)
				t.Logf("client idx(%v) count(%v) for test server connect address %v err: %v", idx, nn, testAddress, err)
				return
			}
			t.Logf("client %v connected server", idx)

			defer c.Close()

			go func() {
				sd.waitEnd()
				c.Quit()
			}()

			c.Run()

			nn := atomic.AddInt32(&n, 1)
			t.Logf("client idx(%v) count(%v) test done", idx, nn)
		}(i)
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

func TestServer2(t *testing.T) {
	srv := createTestServer(t, 1)
	err := srv.Listen(testAddress)
	if err != nil {
		t.Errorf("test server listen address %v err: %v", testAddress, err)
		return
	}
	defer srv.End()

	go func(s *server.Server) {
		s.Start()
	}(srv)

	t.Logf("test server is running")

	clientNum := 5000
	sendNum := 1000
	n := int32(0)
	var wg sync.WaitGroup
	wg.Add(clientNum)

	// 创建一堆客户端
	for i := 0; i < clientNum; i++ {
		go func(idx int) {
			defer wg.Done()

			sd := createSendDataInfo(int32(sendNum))
			c := createTestClient2(t, 2, sd)
			err := c.Connect(testAddress)
			if err != nil {
				nn := atomic.AddInt32(&n, 1)
				t.Logf("client idx(%v) count(%v) for test server connect address %v err: %v", idx, nn, testAddress, err)
				return
			}
			t.Logf("client %v connected server", idx)
			defer c.Close()

			ran := rand.New(rand.NewSource(time.Now().UnixNano()))
			ss := 0
			for {
				if ss < sendNum {
					d := randBytes(100, ran)
					err := c.Send(d, false)
					if err != nil {
						nn := atomic.AddInt32(&n, 1)
						t.Logf("client idx(%v) count(%v) for test server send data err: %v", idx, nn, err)
						return
					}
					sd.appendSendData(d)
					ss += 1
				}
				err = c.Update()
				if err != nil {
					return
				}
				if sd.getComparedNum() >= int32(sendNum) {
					break
				}
				time.Sleep(time.Millisecond * 1)
			}
			nn := atomic.AddInt32(&n, 1)
			t.Logf("client idx(%v) count(%v) test done", idx, nn)
		}(i)
	}

	wg.Wait()
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

	sendNum := 1000
	n := int32(0)
	var wg sync.WaitGroup
	wg.Add(clientNum)

	// 创建一堆客户端
	for i := 0; i < b.N; i++ {

		go func(idx int) {
			defer wg.Done()

			sd := createSendDataInfo(int32(sendNum))
			c := createBenchmarkClient(b, 2, sd)
			err := c.Connect(testAddress)
			if err != nil {
				b.Errorf("client for benchmark server connect address %v err: %v", testAddress, err)
				return
			}
			b.Logf("client %v connected server", idx)

			defer c.Close()

			go func() {
				sd.waitEnd()
				c.Quit()
			}()

			c.Run()

			nn := atomic.AddInt32(&n, 1)
			b.Logf("client idx(%v) count(%v) test done", idx, nn)
		}(i)
	}

	wg.Wait()
}
