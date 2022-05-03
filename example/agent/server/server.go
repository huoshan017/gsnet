package main

import (
	"time"

	"github.com/huoshan017/gsnet/client"
	"github.com/huoshan017/gsnet/common"
	"github.com/huoshan017/gsnet/log"
	"github.com/huoshan017/gsnet/packet"
	"github.com/huoshan017/gsnet/server"

	ecommon "github.com/huoshan017/gsnet/example/agent/common"
)

func createAgentClient() (*client.AgentClient, error) {
	c := client.NewAgentClient()
	if err := c.Dial(ecommon.AgentServerAddress); err != nil {
		return nil, err
	}
	log.Infof("agent client connected")
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

type serverHandlerUseAgentClient struct {
	agentClient *client.AgentClient
	agentSess   common.ISession
}

func newServerHandlerUseAgentClient(args ...any) common.ISessionEventHandler {
	agentClient := args[0].(*client.AgentClient)
	return &serverHandlerUseAgentClient{agentClient: agentClient}
}

func (h *serverHandlerUseAgentClient) OnConnect(sess common.ISession) {
	log.Infof("session %v connected to server", sess.GetId())
}

func (h *serverHandlerUseAgentClient) OnDisconnect(sess common.ISession, err error) {
	log.Infof("session %v disconnected from server", sess.GetId())
}

func (h *serverHandlerUseAgentClient) OnPacket(sess common.ISession, pak packet.IPacket) error {
	ws := h.getWorkerSess(sess)
	return ws.Send(pak.Data(), true)
}

func (h *serverHandlerUseAgentClient) OnTick(sess common.ISession, tick time.Duration) {
}

func (h *serverHandlerUseAgentClient) OnError(err error) {
	log.Infof("occur err %v on server", err)
}

func (h *serverHandlerUseAgentClient) getWorkerSess(sess common.ISession) common.ISession {
	if h.agentSess == nil {
		h.agentSess = h.agentClient.BoundHandleAndGetAgentSession(sess, h.OnPacketFromWorkerServer)
	}
	return h.agentSess
}

func (h *serverHandlerUseAgentClient) OnPacketFromWorkerServer(sess common.ISession, pak packet.IPacket) error {
	return sess.Send(pak.Data(), true)
}

func createServerUseAgentClient(address string) *server.Server {
	agentClient, err := createAgentClient()
	if err != nil {
		log.Fatalf("create agent client err %v", err)
		return nil
	}
	s := server.NewServer(newServerHandlerUseAgentClient, server.WithNewSessionHandlerFuncArgs(agentClient), common.WithTickSpan(100*time.Millisecond))
	if err = s.Listen(address); err != nil {
		log.Infof("test server listen err %v", err)
		return nil
	}
	return s
}

func main() {
	s := createServerUseAgentClient(ecommon.TestAddress)
	if s == nil {
		return
	}
	defer s.End()
	s.Start()
}
