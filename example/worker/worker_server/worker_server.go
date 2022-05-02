package main

import (
	"time"

	"github.com/huoshan017/gsnet/common"
	"github.com/huoshan017/gsnet/log"
	"github.com/huoshan017/gsnet/packet"
	"github.com/huoshan017/gsnet/worker"

	ecommon "github.com/huoshan017/gsnet/example/worker/common"
)

type workerServerHandler struct {
}

func newWorkerServerHandler(args ...any) common.ISessionEventHandler {
	return &workerServerHandler{}
}

func (h *workerServerHandler) OnConnect(sess common.ISession) {
	log.Infof("session %v connected to worker server", sess.GetId())
}

func (h *workerServerHandler) OnDisconnect(sess common.ISession, err error) {
	log.Infof("session %v disconnected from worker server", sess.GetId())
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
	log.Infof("occur err %v on worker server", err)
}

func createWorkerServer(address string) *worker.Server {
	s := worker.NewServer(newWorkerServerHandler)
	if err := s.Listen(address); err != nil {
		log.Infof("worker server listen and serve err %v", err)
		return nil
	}

	return s
}

func main() {
	s := createWorkerServer(ecommon.WorkerServerAddress)
	if s == nil {
		return
	}
	defer s.End()
	s.Start()
}
