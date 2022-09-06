package main

import (
	"time"

	"github.com/huoshan017/gsnet/common"
	"github.com/huoshan017/gsnet/log"
	"github.com/huoshan017/gsnet/options"
	"github.com/huoshan017/gsnet/packet"
	"github.com/huoshan017/gsnet/server"

	"net/http"
	_ "net/http/pprof"

	ecommon "github.com/huoshan017/gsnet/example/agent/common"
)

type serverHandler struct {
}

func newServerHandlerUseAgentClient(args ...any) common.ISessionEventHandler {
	return &serverHandler{}
}

func (h *serverHandler) OnConnect(sess common.ISession) {
	log.Infof("session %v connected to server", sess.GetId())
}

func (h *serverHandler) OnReady(sess common.ISession) {
	log.Infof("session %v ready", sess.GetId())
}

func (h *serverHandler) OnDisconnect(sess common.ISession, err error) {
	log.Infof("session %v disconnected from server", sess.GetId())
}

func (h *serverHandler) OnPacket(sess common.ISession, pak packet.IPacket) error {
	return sess.Send(pak.Data(), true)
}

func (h *serverHandler) OnTick(sess common.ISession, tick time.Duration) {
}

func (h *serverHandler) OnError(err error) {
	log.Infof("occur err %v on server", err)
}

func createServer(address string) *server.Server {
	s := server.NewServer(newServerHandlerUseAgentClient, options.WithTickSpan(100*time.Millisecond))
	if err := s.Listen(address); err != nil {
		log.Infof("test server listen err %v", err)
		return nil
	}
	log.Infof("server listening %v", address)
	return s
}

func main() {
	s := createServer(ecommon.TestAddress)
	if s == nil {
		return
	}
	defer s.End()
	go func() {
		http.ListenAndServe("0.0.0.0:6060", nil)
	}()
	s.Serve()
}
