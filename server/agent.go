package server

import (
	"time"

	"github.com/huoshan017/gsnet/common"
	"github.com/huoshan017/gsnet/options"
	"github.com/huoshan017/gsnet/packet"
)

type baseAgentHandler struct {
	connectHandle    func(common.ISession)
	readyHandle      func(common.ISession)
	disconnectHandle func(common.ISession, error)
	tickHandle       func(common.ISession, time.Duration)
	errorHandle      func(error)
}

func (h *baseAgentHandler) setConnectHandle(handle func(common.ISession)) {
	h.connectHandle = handle
}

func (h *baseAgentHandler) setReadyHandle(handle func(common.ISession)) {
	h.readyHandle = handle
}

func (h *baseAgentHandler) setDisconnectHandle(handle func(common.ISession, error)) {
	h.disconnectHandle = handle
}

func (h *baseAgentHandler) setTickHandle(handle func(common.ISession, time.Duration)) {
	h.tickHandle = handle
}

func (h *baseAgentHandler) setErrorHandle(handle func(error)) {
	h.errorHandle = handle
}

func (h *baseAgentHandler) OnConnect(sess common.ISession) {
	if h.connectHandle != nil {
		h.connectHandle(sess)
	}
}

func (h *baseAgentHandler) OnReady(sess common.ISession) {
	if h.readyHandle != nil {
		h.readyHandle(sess)
	}
}

func (h *baseAgentHandler) OnTick(sess common.ISession, tick time.Duration) {
	if h.tickHandle != nil {
		h.tickHandle(sess, tick)
	}
}

func (h *baseAgentHandler) OnDisconnect(sess common.ISession, err error) {
	if h.disconnectHandle != nil {
		h.disconnectHandle(sess, err)
	}
}

func (h *baseAgentHandler) OnError(err error) {
	if h.errorHandle != nil {
		h.errorHandle(err)
	}
}

type agentServerHandler struct {
	baseAgentHandler
	packetHandle func(common.ISession, packet.IPacket) error
	agentSess    *common.AgentSession
}

func (h *agentServerHandler) setPacketHandle(handle func(common.ISession, packet.IPacket) error) {
	h.packetHandle = handle
}

func (h *agentServerHandler) OnPacket(sess common.ISession, pak packet.IPacket) error {
	agentId := common.BufferToUint32(pak.Data()[:4])
	ppak, o := pak.(*packet.Packet)
	if o {
		ppak.Offset(4)
	} else {
		bpak, o := pak.(*packet.BytesPacket)
		if !o {
			return common.ErrPacketTypeNotSupported
		}
		p := packet.BytesPacket(bpak.Data()[4:])
		pak = &p
	}
	if h.agentSess == nil {
		h.agentSess = common.NewAgentSession(agentId, sess)
	} else {
		h.agentSess.Reset(agentId, sess)
	}
	return h.packetHandle(h.agentSess, pak)
}

func newServerHandler(handler common.ISessionHandler) *agentServerHandler {
	sh := &agentServerHandler{}
	sh.setConnectHandle(handler.OnConnect)
	sh.setReadyHandle(handler.OnReady)
	sh.setDisconnectHandle(handler.OnDisconnect)
	sh.setTickHandle(handler.OnTick)
	sh.setErrorHandle(handler.OnError)
	sh.setPacketHandle(handler.OnPacket)
	return sh
}

type AgentServer struct {
	*Server
}

func NewAgentServer(newFunc NewSessionHandlerFunc, options ...options.Option) *AgentServer {
	var rewriteNewFunc NewSessionHandlerFunc = func(args ...any) common.ISessionHandler {
		handler := newFunc(args...)
		return newServerHandler(handler)
	}
	return &AgentServer{
		Server: NewServer(rewriteNewFunc, options...),
	}
}

func NewAgentServerWithOptions(newFunc NewSessionHandlerFunc, options *options.ServerOptions) *AgentServer {
	var rewriteNewFunc NewSessionHandlerFunc = func(args ...any) common.ISessionHandler {
		handler := newFunc(args...)
		return newServerHandler(handler)
	}
	return &AgentServer{
		Server: NewServerWithOptions(rewriteNewFunc, options),
	}
}
