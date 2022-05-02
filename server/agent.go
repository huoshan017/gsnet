package server

import (
	"time"

	"github.com/huoshan017/gsnet/common"
	"github.com/huoshan017/gsnet/packet"
)

type commonHandler struct {
	connectHandle    func(common.ISession)
	disconnectHandle func(common.ISession, error)
	tickHandle       func(common.ISession, time.Duration)
	errorHandle      func(error)
}

func (h *commonHandler) setConnectHandle(handle func(common.ISession)) {
	h.connectHandle = handle
}

func (h *commonHandler) setDisconnectHandle(handle func(common.ISession, error)) {
	h.disconnectHandle = handle
}

func (h *commonHandler) setTickHandle(handle func(common.ISession, time.Duration)) {
	h.tickHandle = handle
}

func (h *commonHandler) setErrorHandle(handle func(error)) {
	h.errorHandle = handle
}

func (h *commonHandler) OnConnect(sess common.ISession) {
	if h.connectHandle != nil {
		h.connectHandle(sess)
	}
}

func (h *commonHandler) OnTick(sess common.ISession, tick time.Duration) {
	if h.tickHandle != nil {
		h.tickHandle(sess, tick)
	}
}

func (h *commonHandler) OnDisconnect(sess common.ISession, err error) {
	if h.disconnectHandle != nil {
		h.disconnectHandle(sess, err)
	}
}

func (h *commonHandler) OnError(err error) {
	if h.errorHandle != nil {
		h.errorHandle(err)
	}
}

type serverHandler struct {
	commonHandler
	packetHandle func(common.ISession, packet.IPacket) error
}

func (h *serverHandler) setPacketHandle(handle func(common.ISession, packet.IPacket) error) {
	h.packetHandle = handle
}

func (h *serverHandler) OnPacket(sess common.ISession, pak packet.IPacket) error {
	agentId := common.BufferToUint32(pak.Data()[:4])
	agentSess := common.NewAgentSession(agentId, sess)
	if pak.MMType() == packet.MemoryManagementSystemGC {
		p := packet.BytesPacket(pak.Data()[4:])
		pak = &p
	} else {
		ppak, o := pak.(*packet.Packet)
		if !o {
			return common.ErrPacketTypeNotSupported
		}
		ppak.Offset(4)
	}
	return h.packetHandle(agentSess, pak)
}

func newServerHandler(handler common.ISessionEventHandler) *serverHandler {
	sh := &serverHandler{}
	sh.setConnectHandle(handler.OnConnect)
	sh.setDisconnectHandle(handler.OnDisconnect)
	sh.setTickHandle(handler.OnTick)
	sh.setErrorHandle(handler.OnError)
	sh.setPacketHandle(handler.OnPacket)
	return sh
}

type AgentServer struct {
	*Server
}

func NewAgentServer(newFunc NewSessionHandlerFunc, options ...common.Option) *AgentServer {
	var rewriteNewFunc NewSessionHandlerFunc = func(args ...any) common.ISessionEventHandler {
		handler := newFunc(args...)
		return newServerHandler(handler)
	}
	return &AgentServer{
		Server: NewServer(rewriteNewFunc, options...),
	}
}

func NewAgentServerWithOptions(newFunc NewSessionHandlerFunc, options *ServerOptions) *AgentServer {
	var rewriteNewFunc NewSessionHandlerFunc = func(args ...any) common.ISessionEventHandler {
		handler := newFunc(args...)
		return newServerHandler(handler)
	}
	return &AgentServer{
		Server: NewServerWithOptions(rewriteNewFunc, options),
	}
}
