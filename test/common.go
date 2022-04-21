package test

import (
	"bytes"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/huoshan017/gsnet/client"
	"github.com/huoshan017/gsnet/common"
	"github.com/huoshan017/gsnet/packet"
	"github.com/huoshan017/gsnet/server"
)

const (
	connDataType = 0 // 连接数据类型
	testAddress  = "127.0.0.1:9998"
)

type sendDataInfo struct {
	list        [][]byte
	num         int32
	cnum        int32
	numCh       chan int32
	numChClosed bool
}

func createSendDataInfo(cnum int32) *sendDataInfo {
	return &sendDataInfo{
		list:  make([][]byte, 0),
		cnum:  cnum,
		numCh: make(chan int32, 1),
	}
}

// 发送goroutine中调用
func (info *sendDataInfo) appendSendData(data []byte) {
	info.list = append(info.list, data)
}

// 在逻辑goroutine中调用
func (info *sendDataInfo) compareData(data []byte, isForward bool) (bool, error) {
	if bytes.Equal(info.list[0], data) {
		if isForward {
			info.compareForward(false)
		}
		return true, nil
	}
	return false, fmt.Errorf("data %v compare info.list[0] %v failed", data, info.list[0])
}

func (info *sendDataInfo) compareForward(toLock bool) {
	info.list = info.list[1:]
	info.num += 1
	if !info.numChClosed && info.num >= info.cnum {
		info.numCh <- info.num
		close(info.numCh)
		info.numChClosed = true
	}
}

func (info *sendDataInfo) waitEnd() int32 {
	num := <-info.numCh
	return num
}

func (info *sendDataInfo) getComparedNum() int32 {
	return info.num
}

type testClientHandler struct {
	t            *testing.T
	b            *testing.B
	state        int32 // 1 客户端模式   2 服务器模式
	sendDataList *sendDataInfo
	ran          *rand.Rand
}

func newTestClientHandler(args ...any) common.ISessionEventHandler {
	if len(args) < 2 {
		panic("At least need 2 arguments")
	}
	h := &testClientHandler{ran: rand.New(rand.NewSource(time.Now().UnixNano()))}
	var o bool
	h.t, o = args[0].(*testing.T)
	if !o {
		h.b, _ = args[0].(*testing.B)
	}
	h.state = args[1].(int32)
	if len(args) > 2 {
		h.sendDataList, _ = args[2].(*sendDataInfo)
	}
	return h
}

func (h *testClientHandler) OnConnect(sess common.ISession) {
	if h.state == 2 {
		if h.t != nil {
			h.t.Logf("connected")
		} else if h.b != nil {
			h.b.Logf("connected")
		}
	}
}

func (h *testClientHandler) OnDisconnect(sess common.ISession, err error) {
	if h.state == 2 {
		if h.t != nil {
			h.t.Logf("disconnected, err: %v", err)
		} else if h.b != nil {
			h.b.Logf("disconnected, err: %v", err)
		}
	}
}

func (h *testClientHandler) OnPacket(sess common.ISession, packet packet.IPacket) error {
	var (
		o bool
		e error
	)
	data := packet.Data()
	if o, e = h.sendDataList.compareData(data, true); !o {
		err := fmt.Errorf("compare err: %v", e)
		if h.t != nil {
			panic(err)
		} else if h.b != nil {
			panic(err)
		}
	}
	h.t.Logf("session %v  OnPacket compared %v", sess.GetId(), data)
	return nil
}

func (h *testClientHandler) OnTick(sess common.ISession, tick time.Duration) {
	d := randBytes(100, h.ran)
	err := sess.Send(d, false)
	if err != nil {
		if h.t != nil {
			h.t.Logf("sess send data err: %v", err)
		} else if h.b != nil {
			h.t.Logf("sess send data err: %v", err)
		}
		return
	}
	h.sendDataList.appendSendData(d)
}

func (h *testClientHandler) OnError(err error) {
	if h.state == 2 {
		if h.t != nil {
			h.t.Logf("occur err: %v", err)
		} else if h.b != nil {
			h.t.Logf("occur err: %v", err)
		}
	}
}

func createTestClient(t *testing.T, state int32, userData any) *client.Client {
	// 启用tick处理
	return client.NewClient(newTestClientHandler(t, state, userData), common.WithTickSpan(time.Millisecond*10), common.WithConnDataType(connDataType))
}

func createTestClient2(t *testing.T, state int32, userData any) *client.Client {
	h := newTestClientHandler(t, state, userData)
	return client.NewClient(h, common.WithConnDataType(connDataType))
}

func createBenchmarkClient(b *testing.B, state int32, userData any) *client.Client {
	return client.NewClient(newTestClientHandler(b, state, userData), common.WithConnDataType(connDataType))
}

type testServerHandler struct {
	t     *testing.T
	b     *testing.B
	state int32 // 1 表示服务器模式  2 表示客户端模式
}

func newTestServerHandler(args ...any) common.ISessionEventHandler {
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
		if h.t != nil {
			h.t.Logf("new client(session_id: %v) connected", sess.GetId())
		} else if h.b != nil {
			h.b.Logf("new client(session_id: %v) connected", sess.GetId())
		}
	}
}

func (h *testServerHandler) OnDisconnect(sess common.ISession, err error) {
	if h.state == 1 {
		if h.t != nil {
			h.t.Logf("client(session_id: %v) disconnected, err: %v", sess.GetId(), err)
		} else if h.b != nil {
			h.b.Logf("client(session_id: %v) disconnected, err: %v", sess.GetId(), err)
		}
	}
}

func (h *testServerHandler) OnPacket(sess common.ISession, packet packet.IPacket) error {
	if h.t != nil {
		h.t.Logf("testServerHandler.OnPacket packet %v", packet.Data())
	}
	err := sess.Send(packet.Data(), true)
	if err != nil {
		str := fmt.Sprintf("OnData with session %v send err: %v", sess.GetId(), err)
		if h.state == 1 {
			if h.t != nil {
				h.t.Logf(str)
			} else if h.b != nil {
				h.b.Logf(str)
			}
		}
	}
	return err
}

func (h *testServerHandler) OnTick(sess common.ISession, tick time.Duration) {

}

func (h *testServerHandler) OnError(err error) {
	if h.state == 1 {
		if h.t != nil {
			h.t.Logf("server occur err: %v @@@ @@@", err)
		} else if h.b != nil {
			h.t.Logf("server occur err: %v @@@ @@@", err)
		}
	}
}

func createTestServer(t *testing.T, state int32) *server.Server {
	return server.NewServer(newTestServerHandler,
		server.WithNewSessionHandlerFuncArgs(t, state),
		common.WithReadBuffSize(10*4096),
		common.WithWriteBuffSize(5*4096),
		common.WithConnDataType(connDataType),
		common.WithPacketCompressType(packet.CompressSnappy),
		common.WithPacketEncryptionType(packet.EncryptionDes),
	)
}

func createTestServerWithHandler(t *testing.T, state int32) *server.Server {
	return server.NewServerWithHandler(&testServerHandler{
		t:     t,
		state: state,
	}, common.WithConnDataType(connDataType))
}

func createTestServerWithReusePort(t *testing.T, state int32) []*server.Server {
	return []*server.Server{
		server.NewServerWithHandler(&testServerHandler{
			t:     t,
			state: state,
		}, server.WithReuseAddr(true), common.WithConnDataType(connDataType)),
		server.NewServerWithHandler(&testServerHandler{
			t:     t,
			state: state,
		}, server.WithReuseAddr(true), common.WithConnDataType(connDataType)),
	}
}

func createBenchmarkServerWithHandler(b *testing.B, state int32) *server.Server {
	return server.NewServerWithHandler(&testServerHandler{
		b:     b,
		state: state,
	}, common.WithConnDataType(connDataType))
}

var letters = []byte("abcdefghijklmnopqrstuvwxyz01234567890~!@#$%^&*()_+-={}[]|:;'<>?/.,")
var lettersLen = len(letters)

func randBytes(n int, ran *rand.Rand) []byte {
	b := make([]byte, n)
	for i := 0; i < len(b); i++ {
		r := ran.Int31n(int32(lettersLen))
		b[i] = letters[r]
	}
	return b
}
