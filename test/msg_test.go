package test

import (
	"bytes"
	"fmt"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/huoshan017/gsnet/common"
	"github.com/huoshan017/gsnet/msg"
	"github.com/huoshan017/gsnet/server"

	"github.com/huoshan017/gsnet/test/tproto"
)

const (
	MsgIdPing = msg.MsgIdType(1)
	MsgIdPong = msg.MsgIdType(2)
	sendCount = 5000
	clientNum = 1000
)

var (
	ch          = make(chan struct{})
	idMsgMapper *msg.IdMsgMapper
	clientsCh   = make(chan *msg.MsgClient, 128)
)

func init() {
	idMsgMapper = msg.CreateIdMsgMapper()
	idMsgMapper.AddMap(MsgIdPing, reflect.TypeOf(&tproto.MsgPing{}))
	idMsgMapper.AddMap(MsgIdPong, reflect.TypeOf(&tproto.MsgPong{}))
}

func newPBMsgClient(t *testing.T) (*msg.MsgClient, error) {
	c := msg.NewPBMsgClient(idMsgMapper, common.WithTickSpan(10*time.Millisecond))

	c.SetConnectHandle(func(sess *msg.MsgSession) {
		t.Logf("connected")
	})

	c.SetDisconnectHandle(func(sess *msg.MsgSession, err error) {
		t.Logf("disconnected, err %v", err)
	})

	var n int
	c.SetTickHandle(func(sess *msg.MsgSession, tick time.Duration) {
		if n < sendCount {
			var ping tproto.MsgPing
			ping.Content = "pingpingping"
			err := c.Send(MsgIdPing, &ping)
			if err != nil {
				t.Logf("client send message err: %v", err)
			}
			n += 1
		} else {
			close(ch)
		}
	})

	c.SetErrorHandle(func(err error) {
		t.Logf("get error: %v", err)
	})

	var rn int
	c.RegisterMsgHandle(MsgIdPong, func(sess *msg.MsgSession, msg interface{}) error {
		t.Logf("received Pong message %v", rn)
		rn += 1
		return nil
	})

	err := c.Connect(testAddress)
	if err != nil {
		return nil, fmt.Errorf("TestPBMsgClient connect address %v err %v", testAddress, err)
	}

	return c, nil
}

type testPBMsgHandler struct {
	t *testing.T
}

func (h *testPBMsgHandler) OnConnected(sess *msg.MsgSession) {
	h.t.Logf("session %v connected", sess.GetId())
}

func (h *testPBMsgHandler) OnDisconnected(sess *msg.MsgSession, err error) {
	h.t.Logf("session %v disconnected", sess.GetId())
}

func (h *testPBMsgHandler) OnTick(sess *msg.MsgSession, tick time.Duration) {

}

func (h *testPBMsgHandler) OnError(err error) {
	h.t.Logf("session err: %v", err)
}

func (h *testPBMsgHandler) OnMsgHandle(sess *msg.MsgSession, msgid msg.MsgIdType, msgobj interface{}) error {
	if msgid == MsgIdPing {
		m, o := msgobj.(*tproto.MsgPing)
		if !o {
			h.t.Errorf("server receive message must Ping")
		}
		h.t.Logf("received session %v message %v", sess.GetId(), m.Content)
		var rm tproto.MsgPong
		rm.Content = "pongpongpong"
		return sess.SendMsg(MsgIdPong, &rm)
	}
	return nil
}

func newTestPBMsgHandler(args ...interface{}) msg.IMsgSessionHandler {
	handler := &testPBMsgHandler{
		t: args[0].(*testing.T),
	}
	return handler
}

func newPBMsgServer(t *testing.T) (*msg.MsgServer, error) {
	s := msg.NewPBMsgServer(newTestPBMsgHandler, idMsgMapper, server.WithNewSessionHandlerFuncArgs(t))
	err := s.Listen(testAddress)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func newPBMsgClient2(t *testing.T) (*msg.MsgClient, error) {
	c := msg.NewPBMsgClient(idMsgMapper, common.WithTickSpan(10*time.Millisecond))

	c.SetConnectHandle(func(sess *msg.MsgSession) {
		t.Logf("connected")
	})

	c.SetDisconnectHandle(func(sess *msg.MsgSession, err error) {
		t.Logf("disconnected, err %v", err)
	})

	var sn, rn int
	var sendList [][]byte
	c.SetTickHandle(func(sess *msg.MsgSession, tick time.Duration) {
		if sn < sendCount {
			var ping tproto.MsgPing
			d := randBytes(50)
			ping.Content = string(d)
			err := c.Send(MsgIdPing, &ping)
			if err != nil {
				t.Logf("client send message err: %v", err)
			}
			sendList = append(sendList, d)
			sn += 1
		}
	})

	c.SetErrorHandle(func(err error) {
		t.Logf("get error: %v", err)
	})

	c.RegisterMsgHandle(MsgIdPong, func(sess *msg.MsgSession, msg interface{}) error {
		if rn >= sendCount {
			return nil
		}
		m := msg.(*tproto.MsgPong)
		if !bytes.Equal([]byte(m.Content), sendList[rn]) {
			err := fmt.Errorf("compare failed: %v to %v", m.Content, sendList[rn])
			panic(err)
		}
		rn += 1
		if rn >= sendCount {
			clientsCh <- c
		}
		return nil
	})

	err := c.Connect(testAddress)
	if err != nil {
		return nil, fmt.Errorf("TestPBMsgClient connect address %v err %v", testAddress, err)
	}

	return c, nil
}

type testPBMsgHandler2 struct {
	t *testing.T
}

func (h *testPBMsgHandler2) OnConnected(sess *msg.MsgSession) {
	h.t.Logf("session %v connected", sess.GetId())
}

func (h *testPBMsgHandler2) OnDisconnected(sess *msg.MsgSession, err error) {
	h.t.Logf("session %v disconnected", sess.GetId())
}

func (h *testPBMsgHandler2) OnTick(sess *msg.MsgSession, tick time.Duration) {

}

func (h *testPBMsgHandler2) OnError(err error) {
	h.t.Logf("session err: %v", err)
}

func (h *testPBMsgHandler2) OnMsgHandle(sess *msg.MsgSession, msgid msg.MsgIdType, msgobj interface{}) error {
	if msgid == MsgIdPing {
		m, o := msgobj.(*tproto.MsgPing)
		if !o {
			h.t.Errorf("server receive message must Ping")
		}
		var rm tproto.MsgPong
		rm.Content = m.Content
		return sess.SendMsgNoCopy(MsgIdPong, &rm)
	}
	return nil
}

func newTestPBMsgHandler2(args ...interface{}) msg.IMsgSessionHandler {
	t := args[0].(*testing.T)
	handler := &testPBMsgHandler2{
		t: t,
	}
	return handler
}

func newPBMsgServer2(t *testing.T) (*msg.MsgServer, error) {
	s := msg.NewPBMsgServer(newTestPBMsgHandler2, idMsgMapper, server.WithNewSessionHandlerFuncArgs(t))
	err := s.Listen(testAddress)
	if err != nil {
		return nil, err
	}

	return s, nil
}

func TestPBMsgClient(t *testing.T) {
	s, err := newPBMsgServer(t)
	if err != nil {
		t.Errorf("%v", err)
		return
	}
	defer func() {
		s.End()
		t.Logf("server end")
	}()
	go s.Start()

	t.Logf("server started")

	c, err := newPBMsgClient(t)
	if err != nil {
		t.Errorf("%v", err)
		return
	}
	t.Logf("client start")
	go func() {
		<-ch
		c.Close()
	}()
	c.Run()
	t.Logf("client end")
}

func TestPBMsgServer(t *testing.T) {
	s, err := newPBMsgServer2(t)
	if err != nil {
		t.Errorf("%v", err)
		return
	}
	defer func() {
		s.End()
		t.Logf("server end")
	}()
	go s.Start()

	t.Logf("server started")

	var wg sync.WaitGroup
	wg.Add(clientNum)
	remainCount := int32(clientNum)
	for i := 0; i < clientNum; i++ {
		go func() {
			c, err := newPBMsgClient2(t)
			if err != nil {
				//t.Errorf("%v", err)
				wg.Done()
				count := atomic.AddInt32(&remainCount, -1)
				t.Logf("new protobuf message client err: %v, remain client %v", err, count)
				return
			}
			t.Logf("client start")
			c.Run()
			t.Logf("client end")
			count := atomic.AddInt32(&remainCount, -1)
			t.Logf("clients count remain %v", count)
			wg.Done()
		}()
	}

	go func() {
		for {
			client, o := <-clientsCh
			if !o {
				break
			}
			client.Close()
		}
	}()

	wg.Wait()
}
