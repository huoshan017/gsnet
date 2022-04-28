package test

import (
	"math/rand"
	"testing"
	"time"

	"github.com/huoshan017/gsnet/common"
	"github.com/huoshan017/gsnet/packet"
	"github.com/huoshan017/gsnet/server"
	"github.com/huoshan017/gsnet/worker"
)

const (
	workerServerAddress = "127.0.0.1:9001"
	workerClientName    = "TestWorkerClient"
)

func createWorkerClient(t *testing.T) (*worker.Client, error) {
	c := worker.NewClient(workerClientName)
	if err := c.Dial(workerServerAddress); err != nil {
		return nil, err
	}
	c.SetConnectHandle(func(sess common.ISession) {
		t.Logf("worker client (sess %v) connected to server", sess.GetId())
	})
	c.SetDisconnectHandle(func(sess common.ISession, err error) {
		t.Logf("worker client (sess %v) disconnected from server", sess.GetId())
	})
	c.SetTickHandle(func(sess common.ISession, tick time.Duration) {
	})
	c.SetErrorHandle(func(err error) {
		t.Logf("worker client ocurr err %v", err)
	})
	return c, nil
}

type serverHandlerUseWorkerClient struct {
	t          *testing.T
	workerSess common.ISession
}

func newServerHandlerUseWorkerClient(args ...any) common.ISessionEventHandler {
	t := args[0].(*testing.T)
	return &serverHandlerUseWorkerClient{
		t: t,
	}
}

func (h *serverHandlerUseWorkerClient) OnConnect(sess common.ISession) {
	h.t.Logf("session %v connected to server", sess.GetId())
}

func (h *serverHandlerUseWorkerClient) OnDisconnect(sess common.ISession, err error) {
	h.t.Logf("session %v disconnected from server", sess.GetId())
}

func (h *serverHandlerUseWorkerClient) OnPacket(sess common.ISession, pak packet.IPacket) error {
	ws := h.getWorkerSess(sess)
	return ws.Send(pak.Data(), true)
}

func (h *serverHandlerUseWorkerClient) OnTick(sess common.ISession, tick time.Duration) {
}

func (h *serverHandlerUseWorkerClient) OnError(err error) {
	h.t.Logf("occur err %v on server", err)
}

func (h *serverHandlerUseWorkerClient) getWorkerSess(sess common.ISession) common.ISession {
	if h.workerSess == nil {
		if c := worker.GetClient(workerClientName); c != nil {
			c.AttachPacketHandle(sess, h.OnPacketFromWorkerServer)
			h.workerSess = c.NewSessionChannel(sess)
		}
	}
	return h.workerSess
}

func (h *serverHandlerUseWorkerClient) OnPacketFromWorkerServer(sess common.ISession, pak packet.IPacket) error {
	return sess.Send(pak.Data(), true)
}

func createServerUseWorkerClient(address string, t *testing.T) *server.Server {
	var err error
	if _, err = createWorkerClient(t); err != nil {
		t.Logf("create worker client err %v", err)
		return nil
	}
	s := server.NewServer(newServerHandlerUseWorkerClient, server.WithNewSessionHandlerFuncArgs(t), common.WithTickSpan(100*time.Millisecond))
	if err = s.Listen(address); err != nil {
		t.Logf("test server listen err %v", err)
		return nil
	}
	go s.Start()
	return s
}

type workerServerHandler struct {
	t *testing.T
}

func newWorkerServerHandler(args ...any) common.ISessionEventHandler {
	t := args[0].(*testing.T)
	return &workerServerHandler{
		t: t,
	}
}

func (h *workerServerHandler) OnConnect(sess common.ISession) {
	h.t.Logf("session %v connected to worker server", sess.GetId())
}

func (h *workerServerHandler) OnDisconnect(sess common.ISession, err error) {
	h.t.Logf("session %v disconnected from worker server", sess.GetId())
}

func (h *workerServerHandler) OnPacket(sess common.ISession, pak packet.IPacket) error {
	if _, o := sess.(*common.SessionChannel); !o {
		panic("!!!!! Must *SessionChannel type")
	}
	return sess.Send(pak.Data(), true)
}

func (h *workerServerHandler) OnTick(sess common.ISession, tick time.Duration) {
}

func (h *workerServerHandler) OnError(err error) {
	h.t.Logf("occur err %v on worker server", err)
}

func createWorkerServer(address string, t *testing.T) *worker.Server {
	s := worker.NewServer(newWorkerServerHandler, server.WithNewSessionHandlerFuncArgs(t))
	if err := s.Listen(address); err != nil {
		t.Logf("worker server listen and serve err %v", err)
		return nil
	}
	go s.Start()
	return s
}

func TestWorkerClient(t *testing.T) {
	ws := createWorkerServer(workerServerAddress, t)
	if ws == nil {
		return
	}
	defer ws.End()

	swc := createServerUseWorkerClient(testAddress, t)
	if swc == nil {
		return
	}
	defer swc.End()

	sendNum := 1000
	sd := createSendDataInfo(int32(sendNum))
	client := createTestClientUseUpdate(t, 1, sd)
	err := client.Connect(testAddress)
	if err != nil {
		t.Logf("test client connect err %v", err)
		return
	}
	defer client.Close()

	t.Logf("test client connected server")

	n := 100
	ran := rand.New(rand.NewSource(time.Now().UnixNano()))
	for {
		err = client.Update()
		if err != nil {
			t.Logf("test client update err %v", err)
			break
		}
		if client.IsConnected() && n > 0 {
			rn := ran.Intn(100000) + 1
			d := randBytes(rn, ran)
			err = client.Send(d, false)
			if err != nil {
				t.Logf("test client send err: %+v", err)
				break
			}
			sd.appendSendData(d)
			n -= 1
		}
		time.Sleep(time.Millisecond)
	}

	time.Sleep(time.Second)

	t.Logf("test done")
}
