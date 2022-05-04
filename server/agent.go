package server

import (
	"time"

	"github.com/huoshan017/gsnet/common"
	"github.com/huoshan017/gsnet/packet"
)

type commonHandler struct {
	connectHandle    func(common.ISession)
	readyHandle      func(common.ISession)
	disconnectHandle func(common.ISession, error)
	tickHandle       func(common.ISession, time.Duration)
	errorHandle      func(error)
}

func (h *commonHandler) setConnectHandle(handle func(common.ISession)) {
	h.connectHandle = handle
}

func (h *commonHandler) setReadyHandle(handle func(common.ISession)) {
	h.readyHandle = handle
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

func (h *commonHandler) OnReady(sess common.ISession) {
	if h.readyHandle != nil {
		h.readyHandle(sess)
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
	agentSess := common.NewAgentSession(agentId, sess)
	return h.packetHandle(agentSess, pak)
}

func newServerHandler(handler common.ISessionEventHandler) *serverHandler {
	sh := &serverHandler{}
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
