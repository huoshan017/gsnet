package worker

import (
	"github.com/huoshan017/gsnet/common"
	"github.com/huoshan017/gsnet/packet"
	"github.com/huoshan017/gsnet/server"
)

type serverHandler struct {
	commonHandler
	packetHandle func(common.ISession, packet.IPacket) error
}

func (h *serverHandler) setPacketHandle(handle func(common.ISession, packet.IPacket) error) {
	h.packetHandle = handle
}

func (h *serverHandler) OnPacket(sess common.ISession, pak packet.IPacket) error {
	sessId := common.BufferToUint64(pak.Data()[:8])
	sessChannel := common.NewSessionChannel(sessId, sess)
	if pak.MMType() == packet.MemoryManagementSystemGC {
		p := packet.BytesPacket(pak.Data()[8:])
		pak = &p
	} else {
		ppak, o := pak.(*packet.Packet)
		if !o {
			return ErrPacketTypeNotSupported
		}
		ppak.Offset(8)
	}
	return h.packetHandle(sessChannel, pak)
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

type Server struct {
	*server.Server
}

func NewServer(newFunc server.NewSessionHandlerFunc, options ...common.Option) *Server {
	var rewriteNewFunc server.NewSessionHandlerFunc = func(args ...any) common.ISessionEventHandler {
		handler := newFunc(args...)
		return newServerHandler(handler)
	}
	return &Server{
		Server: server.NewServer(rewriteNewFunc, options...),
	}
}
