package main

import (
	"time"

	"github.com/huoshan017/gsnet/common"
	"github.com/huoshan017/gsnet/log"
	"github.com/huoshan017/gsnet/packet"
	"github.com/huoshan017/gsnet/server"

	ecommon "github.com/huoshan017/gsnet/example/agent/common"
)

type agentServerHandler struct {
}

func newAgentServerHandler(args ...any) common.ISessionEventHandler {
	return &agentServerHandler{}
}

func (h *agentServerHandler) OnConnect(sess common.ISession) {
	log.Infof("session %v connected to agent server", sess.GetId())
}

func (h *agentServerHandler) OnReady(sess common.ISession) {
	log.Infof("session %v ready", sess.GetId())
}

func (h *agentServerHandler) OnDisconnect(sess common.ISession, err error) {
	log.Infof("session %v disconnected from agent server", sess.GetId())
}

func (h *agentServerHandler) OnPacket(sess common.ISession, pak packet.IPacket) error {
	if _, o := sess.(*common.AgentSession); !o {
		panic("!!!!! Must *AgentSession type")
	}
	return sess.Send(pak.Data(), true)
}

func (h *agentServerHandler) OnTick(sess common.ISession, tick time.Duration) {
}

func (h *agentServerHandler) OnError(err error) {
	log.Infof("occur err %v on agent server", err)
}

func createAgentServer(address string) *server.AgentServer {
	s := server.NewAgentServer(newAgentServerHandler)
	if err := s.Listen(address); err != nil {
		log.Infof("agent server listen and serve err %v", err)
		return nil
	}
	log.Infof("listening %v", address)
	return s
}

func main() {
	s := createAgentServer(ecommon.AgentServerAddress)
	if s == nil {
		return
	}
	defer s.End()
	log.Infof("agent server started")
	s.Serve()
}
