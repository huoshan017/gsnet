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
	clientNum = 2000
)

var (
	ch          = make(chan struct{})
	idMsgMapper *msg.IdMsgMapper
	//clientsCh   = make(chan *msg.MsgClient, 128)
)

type testMsgConfig struct {
	useHeartbeat bool
	useResend    bool
}

func init() {
	idMsgMapper = msg.CreateIdMsgMapper()
	idMsgMapper.AddMap(MsgIdPing, reflect.TypeOf(&tproto.MsgPing{}))
	idMsgMapper.AddMap(MsgIdPong, reflect.TypeOf(&tproto.MsgPong{}))
}

func newPBMsgClient(config *testMsgConfig, t *testing.T) (*msg.MsgClient, error) {
	var c *msg.MsgClient
	var options = []common.Option{
		common.WithTickSpan(10 * time.Millisecond),
	}
	if config.useResend {
		options = append(options, common.WithResendConfig(&common.ResendConfig{}))
	}
	if config.useHeartbeat {
		options = append(options, common.WithUseHeartbeat(true))
	}

	c = msg.NewPBMsgClient(idMsgMapper, options...)

	c.SetConnectHandle(func(sess *msg.MsgSession) {
		t.Logf("connected")
	})

	c.SetDisconnectHandle(func(sess *msg.MsgSession, err error) {
		t.Logf("disconnected, err %v", err)
	})

	var n int
	var sendList [][]byte
	c.SetTickHandle(func(sess *msg.MsgSession, tick time.Duration) {
		if n < sendCount {
			var ping tproto.MsgPing
			bs := randBytes(100)
			sendList = append(sendList, bs)
			ping.Content = string(bs)
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
		m := msg.(*tproto.MsgPong)
		if !bytes.Equal([]byte(m.Content), sendList[0]) {
			err := fmt.Errorf("compare failed: %v to %v", []byte(m.Content), sendList[0])
			panic(err)
		}
		rn += 1
		sendList = sendList[1:]
		t.Logf("received Pong %v message %v", rn, m.Content)
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
		//h.t.Logf("received session %v message %v", sess.GetId(), m.Content)
		var rm tproto.MsgPong
		rm.Content = m.Content
		return sess.SendMsg(MsgIdPong, &rm)
	}
	return nil
}

func newTestPBMsgHandler(args ...interface{}) msg.IMsgSessionEventHandler {
	handler := &testPBMsgHandler{
		t: args[0].(*testing.T),
	}
	return handler
}

func newPBMsgServer(config *testMsgConfig, t *testing.T) (*msg.MsgServer, error) {
	var s *msg.MsgServer
	var options = []common.Option{
		server.WithNewSessionHandlerFuncArgs(t),
	}
	if config.useResend {
		options = append(options, common.WithResendConfig(&common.ResendConfig{}))
	}
	if config.useHeartbeat {
		options = append(options, common.WithUseHeartbeat(true))
	}
	s = msg.NewPBMsgServer(newTestPBMsgHandler, idMsgMapper, options...)
	err := s.Listen(testAddress)
	if err != nil {
		return nil, err
	}
	return s, nil
}

type testPBMsgClientHandler struct {
	t        *testing.T
	sn, rn   int
	sendList [][]byte
}

func newTestPBMsgClientHandler(t *testing.T, c *msg.MsgClient) *testPBMsgClientHandler {
	return &testPBMsgClientHandler{t: t}
}

func (h *testPBMsgClientHandler) OnConnect(sess *msg.MsgSession) {
	h.t.Logf("connected")
}

func (h *testPBMsgClientHandler) OnDisconnect(sess *msg.MsgSession, err error) {
	h.t.Logf("disconnected, err %v", err)
}

func (h *testPBMsgClientHandler) OnTick(sess *msg.MsgSession, tick time.Duration) {
	if h.sn < sendCount {
		var ping tproto.MsgPing
		d := randBytes(100)
		ping.Content = string(d)
		err := sess.SendMsg(MsgIdPing, &ping)
		if err != nil {
			h.t.Logf("client send message err: %v", err)
		}
		h.sendList = append(h.sendList, d)
		h.sn += 1
	}
}

func (h *testPBMsgClientHandler) onMsgPong(sess *msg.MsgSession, msgobj interface{}) error {
	if h.rn >= sendCount {
		return nil
	}
	m := msgobj.(*tproto.MsgPong)
	if !bytes.Equal([]byte(m.Content), h.sendList[0]) {
		err := fmt.Errorf("compare failed: %v to %v", []byte(m.Content), h.sendList[0])
		panic(err)
	}
	//h.t.Logf("session %v received %v", sess.GetId(), m.Content)
	h.sendList = h.sendList[1:]
	h.rn += 1
	if h.rn >= sendCount {
		sess.Close()
	}
	return nil
}

func (h *testPBMsgClientHandler) OnError(err error) {
	h.t.Logf("get error: %v", err)
}

func newPBMsgClient2(config *testMsgConfig, t *testing.T) (*msg.MsgClient, error) {
	var c *msg.MsgClient
	var options = []common.Option{
		common.WithTickSpan(10 * time.Millisecond),
	}
	if config.useResend {
		options = append(options, common.WithResendConfig(&common.ResendConfig{UseLockFree: true}))
	}
	if config.useHeartbeat {
		options = append(options, common.WithUseHeartbeat(true))
	}
	c = msg.NewPBMsgClient(idMsgMapper, options...)

	handler := newTestPBMsgClientHandler(t, c)
	c.SetConnectHandle(handler.OnConnect)
	c.SetDisconnectHandle(handler.OnDisconnect)
	c.SetTickHandle(handler.OnTick)
	c.SetErrorHandle(handler.OnError)
	c.RegisterMsgHandle(MsgIdPong, handler.onMsgPong)

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

func newTestPBMsgHandler2(args ...interface{}) msg.IMsgSessionEventHandler {
	t := args[0].(*testing.T)
	handler := &testPBMsgHandler2{
		t: t,
	}
	return handler
}

func newPBMsgServer2(config *testMsgConfig, t *testing.T) (*msg.MsgServer, error) {
	var s *msg.MsgServer
	var options = []common.Option{
		server.WithNewSessionHandlerFuncArgs(t),
	}
	if config.useResend {
		options = append(options, common.WithResendConfig(&common.ResendConfig{UseLockFree: true}))
	}
	if config.useHeartbeat {
		options = append(options, common.WithUseHeartbeat(true))
	}
	s = msg.NewPBMsgServer(newTestPBMsgHandler2, idMsgMapper, options...)
	err := s.Listen(testAddress)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func testPBMsgClient(useResend bool, t *testing.T) {
	config := &testMsgConfig{useResend: useResend, useHeartbeat: true}
	s, err := newPBMsgServer(config, t)
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

	c, err := newPBMsgClient(config, t)
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

func testPBMsgServer(useResend bool, t *testing.T) {
	config := &testMsgConfig{useResend: useResend, useHeartbeat: true}
	s, err := newPBMsgServer2(config, t)
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
			c, err := newPBMsgClient2(config, t)
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

	/*go func() {
		for {
			client, o := <-clientsCh
			if !o {
				break
			}
			client.Close()
		}
	}()*/

	wg.Wait()
}

func TestPBMsgClient(t *testing.T) {
	testPBMsgClient(false, t)
}

func TestPBMsgClientUseResend(t *testing.T) {
	testPBMsgClient(true, t)
}

func TestPBMsgServer(t *testing.T) {
	testPBMsgServer(false, t)
}

func TestPBMsgServerUseResend(t *testing.T) {
	testPBMsgServer(true, t)
}
