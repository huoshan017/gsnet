package test

import (
	"bytes"
	"fmt"
	"math/rand"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"net/http"
	_ "net/http/pprof"

	"github.com/huoshan017/gsnet/msg"
	"github.com/huoshan017/gsnet/options"
	"github.com/huoshan017/gsnet/packet"

	"github.com/huoshan017/gsnet/test/tproto"
)

const (
	MsgIdPing = msg.MsgIdType(1)
	MsgIdPong = msg.MsgIdType(2)
	sendCount = 1000
	clientNum = 3000
)

var (
	ch          = make(chan struct{})
	idMsgMapper *msg.IdMsgMapper
)

type testMsgConfig struct {
	useUDPKcp         bool
	useHeartbeat      bool
	useResend         bool
	useSnappyCompress bool
	useAesCrypto      bool
}

func init() {
	idMsgMapper = msg.CreateIdMsgMapperWith(map[msg.MsgIdType]reflect.Type{
		MsgIdPing: reflect.TypeOf(&tproto.MsgPing{}),
		MsgIdPong: reflect.TypeOf(&tproto.MsgPong{}),
	})
}

func newPBMsgClient(config *testMsgConfig, t *testing.T) (*msg.MsgClient, error) {
	var c *msg.MsgClient
	var ops = []options.Option{
		options.WithTickSpan(10 * time.Millisecond),
	}
	if config.useUDPKcp {
		ops = append(ops, options.WithNetProto(options.NetProtoUDP))
	}
	if config.useResend {
		ops = append(ops, options.WithResendConfig(&options.ResendConfig{}))
	}
	if config.useHeartbeat {
		ops = append(ops, options.WithUseHeartbeat(true))
	}

	c = msg.NewProtobufMsgClient(idMsgMapper, ops...)

	c.SetConnectHandle(func(sess *msg.MsgSession) {
		t.Logf("connected")
	})

	c.SetReadyHandle(func(sess *msg.MsgSession) {
		t.Logf("ready")
	})

	c.SetDisconnectHandle(func(sess *msg.MsgSession, err error) {
		t.Logf("disconnected, err %v", err)
	})

	var n int
	var sendList [][]byte
	ran := rand.New(rand.NewSource(time.Now().UnixNano()))
	c.SetTickHandle(func(sess *msg.MsgSession, tick time.Duration) {
		if n < sendCount {
			var ping tproto.MsgPing
			bs := randBytes(100, ran)
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
	c.RegisterMsgHandle(MsgIdPong, func(sess *msg.MsgSession, msg any) error {
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

func (h *testPBMsgHandler) OnReady(sess *msg.MsgSession) {
	h.t.Logf("session %v ready", sess.GetId())
}

func (h *testPBMsgHandler) OnDisconnected(sess *msg.MsgSession, err error) {
	h.t.Logf("session %v disconnected", sess.GetId())
}

func (h *testPBMsgHandler) OnTick(sess *msg.MsgSession, tick time.Duration) {

}

func (h *testPBMsgHandler) OnError(err error) {
	h.t.Logf("session err: %v", err)
}

func (h *testPBMsgHandler) OnMsgHandle(sess *msg.MsgSession, msgid msg.MsgIdType, msgobj any) error {
	if msgid == MsgIdPing {
		m, o := msgobj.(*tproto.MsgPing)
		if !o {
			h.t.Errorf("server receive message must Ping")
		}
		var rm tproto.MsgPong
		rm.Content = m.Content
		return sess.SendMsg(MsgIdPong, &rm)
	}
	return nil
}

func newTestPBMsgHandler(args ...any) msg.IMsgSessionHandler {
	handler := &testPBMsgHandler{
		t: args[0].(*testing.T),
	}
	return handler
}

func newPBMsgServer(config *testMsgConfig, t *testing.T) (*msg.MsgServer, error) {
	var s *msg.MsgServer
	var ops = []options.Option{}
	if config.useUDPKcp {
		ops = append(ops, options.WithNetProto(options.NetProtoUDP))
	}
	if config.useResend {
		ops = append(ops, options.WithResendConfig(&options.ResendConfig{}))
	}
	if config.useHeartbeat {
		ops = append(ops, options.WithUseHeartbeat(true))
	}
	s = msg.NewProtobufMsgServer(newTestPBMsgHandler, []any{t}, idMsgMapper, ops...)
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
	ran      *rand.Rand
}

func newTestPBMsgClientHandler(t *testing.T, c *msg.MsgClient) *testPBMsgClientHandler {
	return &testPBMsgClientHandler{t: t, ran: rand.New(rand.NewSource(time.Now().UnixNano()))}
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
		d := randBytes(50, h.ran)
		ping.Content = string(d)
		err := sess.SendMsg(MsgIdPing, &ping)
		if err != nil {
			h.t.Logf("client send message err: %v", err)
		}
		h.sendList = append(h.sendList, d)
		h.sn += 1
		//h.t.Logf("client send message %v", d)
	}
}

func (h *testPBMsgClientHandler) onMsgPong(sess *msg.MsgSession, msgobj any) error {
	if h.rn >= sendCount {
		return nil
	}
	m := msgobj.(*tproto.MsgPong)
	if !bytes.Equal([]byte(m.Content), h.sendList[0]) {
		err := fmt.Errorf("compare failed: %v to %v", []byte(m.Content), h.sendList[0])
		panic(err)
	}
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
	var ops = []options.Option{
		options.WithTickSpan(10 * time.Millisecond),
	}
	if config.useUDPKcp {
		ops = append(ops, options.WithNetProto(options.NetProtoUDP))
	}
	if config.useResend {
		ops = append(ops, options.WithResendConfig(&options.ResendConfig{UseLockFree: true}))
	}
	if config.useHeartbeat {
		ops = append(ops, options.WithUseHeartbeat(true))
	}
	c = msg.NewProtobufMsgClient(idMsgMapper, ops...)

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

func (h *testPBMsgHandler2) OnReady(sess *msg.MsgSession) {
	h.t.Logf("session %v ready", sess.GetId())
}

func (h *testPBMsgHandler2) OnDisconnected(sess *msg.MsgSession, err error) {
	h.t.Logf("session %v disconnected", sess.GetId())
}

func (h *testPBMsgHandler2) OnTick(sess *msg.MsgSession, tick time.Duration) {

}

func (h *testPBMsgHandler2) OnError(err error) {
	h.t.Logf("session err: %v", err)
}

func (h *testPBMsgHandler2) OnMsgHandle(sess *msg.MsgSession, msgid msg.MsgIdType, msgobj any) error {
	if msgid == MsgIdPing {
		m, o := msgobj.(*tproto.MsgPing)
		if !o {
			h.t.Errorf("server receive message must Ping")
		}
		//h.t.Logf("session %v OnMsgHandle %v", sess.GetId(), []byte(m.Content))
		var rm tproto.MsgPong
		rm.Content = m.Content
		return sess.SendMsg(MsgIdPong, &rm)
	}
	return nil
}

func newTestPBMsgHandler2(args ...any) msg.IMsgSessionHandler {
	t := args[0].(*testing.T)
	handler := &testPBMsgHandler2{
		t: t,
	}
	return handler
}

func newPBMsgServer2(config *testMsgConfig, t *testing.T) (*msg.MsgServer, error) {
	var s *msg.MsgServer
	var ops = []options.Option{}
	if config.useUDPKcp {
		ops = append(ops, options.WithNetProto(options.NetProtoUDP))
	}
	if config.useResend {
		ops = append(ops, options.WithResendConfig(&options.ResendConfig{UseLockFree: true}))
	}
	if config.useSnappyCompress {
		ops = append(ops, options.WithPacketCompressType(packet.CompressSnappy))
	}
	if config.useAesCrypto {
		ops = append(ops, options.WithPacketEncryptionType(packet.EncryptionAes))
	}
	if config.useHeartbeat {
		ops = append(ops, options.WithUseHeartbeat(true))
	}
	ops = append(ops, msg.WithHeaderFormatFunc(msg.DefaultMsgHeaderFormat), msg.WithHeaderUnformatFunc(msg.DefaultMsgHeaderUnformat))
	s = msg.NewProtobufMsgServer(newTestPBMsgHandler2, []any{t}, idMsgMapper, ops...)
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
	go s.Serve()

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

func testPBMsgServer(useUDPKcp bool, useHeartbeat bool, useResend bool, useSnappyCompress bool, useAesCrypto bool, t *testing.T) {
	config := &testMsgConfig{useUDPKcp: useUDPKcp, useHeartbeat: useHeartbeat, useResend: useResend, useSnappyCompress: useSnappyCompress, useAesCrypto: useAesCrypto}
	s, err := newPBMsgServer2(config, t)
	if err != nil {
		t.Errorf("%v", err)
		return
	}
	defer func() {
		s.End()
		t.Logf("server end")
	}()
	go s.Serve()

	go func() {
		http.ListenAndServe("0.0.0.0:6060", nil)
	}()

	t.Logf("server started")

	var wg sync.WaitGroup
	wg.Add(clientNum)
	remainCount := int32(clientNum)
	for i := 0; i < clientNum; i++ {
		go func() {
			c, err := newPBMsgClient2(config, t)
			if err != nil {
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

	wg.Wait()
}

func TestPBMsgClient(t *testing.T) {
	testPBMsgClient(false, t)
}

func TestPBMsgClientUseResend(t *testing.T) {
	testPBMsgClient(true, t)
}

func TestPBMsgServer(t *testing.T) {
	testPBMsgServer(false, false, false, false, false, t)
}

func TestPBMsgServerUseCompressAndEncryption(t *testing.T) {
	testPBMsgServer(false, false, false, true, true, t)
}

func TestPBMsgServerUseResend(t *testing.T) {
	testPBMsgServer(false, false, true, false, false, t)
}

func TestPBMsgServerUseCompressEncryptionAndResend(t *testing.T) {
	testPBMsgServer(false, false, true, true, true, t)
}

func TestPBMsgServerUseUDPKcp(t *testing.T) {
	testPBMsgServer(true, true, false, false, false, t)
}
