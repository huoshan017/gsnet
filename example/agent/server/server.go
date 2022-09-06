package main

import (
	"net/http"
	"time"

	"github.com/huoshan017/gsnet/client"
	"github.com/huoshan017/gsnet/common"
	"github.com/huoshan017/gsnet/log"
	"github.com/huoshan017/gsnet/options"
	"github.com/huoshan017/gsnet/packet"
	"github.com/huoshan017/gsnet/server"

	_ "net/http/pprof"

	ecommon "github.com/huoshan017/gsnet/example/agent/common"
)

func createAgentClient() (*client.AgentClient, error) {
	c := client.NewAgentClient(options.WithSendListMode(ecommon.SendListMode))
	if err := c.Dial(ecommon.AgentServerAddress); err != nil {
		return nil, err
	}
	log.Infof("agent client connected")
	c.SetConnectHandle(func(sess common.ISession) {
		log.Infof("agent client (sess %v) connected to server", sess.GetId())
	})
	c.SetReadyHandle(func(sess common.ISession) {
		log.Infof("agent cleint (sess %v) ready", sess.GetId())
	})
	c.SetDisconnectHandle(func(sess common.ISession, err error) {
		log.Infof("agent client (sess %v) disconnected from server", sess.GetId())
	})
	c.SetTickHandle(func(sess common.ISession, tick time.Duration) {
	})
	c.SetErrorHandle(func(err error) {
		log.Infof("agent client ocurr err %v", err)
	})
	return c, nil
}

type serverSessionHandlerUseAgentClient struct {
	agentClient *client.AgentClient
	agentSess   *common.AgentSession
}

func newServerSessionHandlerUseAgentClient(args ...any) common.ISessionEventHandler {
	agentClient := args[0].(*client.AgentClient)
	return &serverSessionHandlerUseAgentClient{agentClient: agentClient}
}

func (h *serverSessionHandlerUseAgentClient) OnConnect(sess common.ISession) {
	log.Infof("session %v connected to server", sess.GetId())
}

func (h *serverSessionHandlerUseAgentClient) OnReady(sess common.ISession) {
	log.Infof("session %v ready", sess.GetId())
}

func (h *serverSessionHandlerUseAgentClient) OnDisconnect(sess common.ISession, err error) {
	if h.agentSess != nil {
		h.agentClient.UnboundServerSession(sess, h.agentSess)
	}
	log.Infof("session %v disconnected from server", sess.GetId())
}

func (h *serverSessionHandlerUseAgentClient) OnPacket(sess common.ISession, pak packet.IPacket) error {
	ws := h.getWorkerSess(sess)
	return ws.Send(pak.Data(), true)
}

func (h *serverSessionHandlerUseAgentClient) OnTick(sess common.ISession, tick time.Duration) {
}

func (h *serverSessionHandlerUseAgentClient) OnError(err error) {
	log.Infof("occur err %v on server", err)
}

func (h *serverSessionHandlerUseAgentClient) getWorkerSess(sess common.ISession) *common.AgentSession {
	if h.agentSess == nil {
		h.agentSess = h.agentClient.BoundServerSession(sess, h.OnPacketFromAgentServer)
	}
	return h.agentSess
}

func (h *serverSessionHandlerUseAgentClient) OnPacketFromAgentServer(sess common.ISession, agentId int32, pak packet.IPacket) error {
	return sess.Send(pak.Data(), func() bool {
		return pak.MMType() != packet.MemoryManagementSystemGC
	}())
}

func createServerUseAgentClient(address string) *server.Server {
	agentClient, err := createAgentClient()
	if err != nil {
		log.Fatalf("create agent client err %v", err)
		return nil
	}
	s := server.NewServer(newServerSessionHandlerUseAgentClient, options.WithNewSessionHandlerFuncArgs(agentClient), options.WithTickSpan(100*time.Millisecond), options.WithSendListMode(ecommon.SendListMode))
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
	go func() {
		http.ListenAndServe("0.0.0.0:6060", nil)
	}()
	s.Serve()
}
