package main

import (
	"time"

	"github.com/huoshan017/gsnet/common"
	"github.com/huoshan017/gsnet/log"
	"github.com/huoshan017/gsnet/packet"
	"github.com/huoshan017/gsnet/server"
	"github.com/huoshan017/gsnet/worker"

	ecommon "github.com/huoshan017/gsnet/example/worker/common"
)

func createWorkerClient() (*worker.Client, error) {
	c := worker.NewClient(ecommon.WorkerClientName)
	if err := c.Dial(ecommon.WorkerServerAddress); err != nil {
		return nil, err
	}
	c.SetConnectHandle(func(sess common.ISession) {
		log.Infof("worker client (sess %v) connected to server", sess.GetId())
	})
	c.SetDisconnectHandle(func(sess common.ISession, err error) {
		log.Infof("worker client (sess %v) disconnected from server", sess.GetId())
	})
	c.SetTickHandle(func(sess common.ISession, tick time.Duration) {
	})
	c.SetErrorHandle(func(err error) {
		log.Infof("worker client ocurr err %v", err)
	})
	return c, nil
}

type serverHandlerUseWorkerClient struct {
	workerSess common.ISession
}

func newServerHandlerUseWorkerClient(args ...any) common.ISessionEventHandler {
	return &serverHandlerUseWorkerClient{}
}

func (h *serverHandlerUseWorkerClient) OnConnect(sess common.ISession) {
	log.Infof("session %v connected to server", sess.GetId())
}

func (h *serverHandlerUseWorkerClient) OnReady(sess common.ISession) {
	log.Infof("session %v ready", sess.GetId())
}

func (h *serverHandlerUseWorkerClient) OnDisconnect(sess common.ISession, err error) {
	log.Infof("session %v disconnected from server", sess.GetId())
}

func (h *serverHandlerUseWorkerClient) OnPacket(sess common.ISession, pak packet.IPacket) error {
	ws := h.getWorkerSess(sess)
	return ws.Send(pak.Data(), true)
}

func (h *serverHandlerUseWorkerClient) OnTick(sess common.ISession, tick time.Duration) {
}

func (h *serverHandlerUseWorkerClient) OnError(err error) {
	log.Infof("occur err %v on server", err)
}

func (h *serverHandlerUseWorkerClient) getWorkerSess(sess common.ISession) common.ISession {
	if h.workerSess == nil {
		if c := worker.GetClient(ecommon.WorkerClientName); c != nil {
			c.BoundPacketHandle(sess, h.OnPacketFromWorkerServer)
			h.workerSess = c.NewSessionChannel(sess)
		}
	}
	return h.workerSess
}

func (h *serverHandlerUseWorkerClient) OnPacketFromWorkerServer(sess common.ISession, agentId int32, pak packet.IPacket) error {
	return sess.Send(pak.Data(), true)
}

func createServerUseWorkerClient(address string) *server.Server {
	var err error
	if _, err = createWorkerClient(); err != nil {
		log.Infof("create worker client err %v", err)
		return nil
	}
	s := server.NewServer(newServerHandlerUseWorkerClient, common.WithTickSpan(100*time.Millisecond))
	if err = s.Listen(address); err != nil {
		log.Infof("test server listen err %v", err)
		return nil
	}
	return s
}

func main() {
	s := createServerUseWorkerClient(ecommon.TestAddress)
	if s == nil {
		return
	}
	defer s.End()
	s.Serve()
}
