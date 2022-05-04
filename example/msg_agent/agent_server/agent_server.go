package main

import (
	"time"

	"github.com/huoshan017/gsnet/common"
	"github.com/huoshan017/gsnet/log"
	"github.com/huoshan017/gsnet/msg"

	acommon "github.com/huoshan017/gsnet/example/msg_agent/common"
)

type msgAgentServerHandler struct {
}

func newMsgAgentServerHandler(args ...any) msg.IMsgSessionEventHandler {
	return &msgAgentServerHandler{}
}

func (h *msgAgentServerHandler) OnConnected(sess *msg.MsgSession) {
	log.Infof("session %v connected to message agent server", sess.GetId())
}

func (h *msgAgentServerHandler) OnDisconnected(sess *msg.MsgSession, err error) {
	log.Infof("session %v disconnected from message agent server", sess.GetId())
}

func (h *msgAgentServerHandler) OnMsgHandle(sess *msg.MsgSession, msgid msg.MsgIdType, msgobj any) error {
	return sess.SendMsg(msgid, msgobj)
}

func (h *msgAgentServerHandler) OnTick(sess *msg.MsgSession, tick time.Duration) {
}

func (h *msgAgentServerHandler) OnError(err error) {
	log.Infof("occur err %v on message agent server", err)
}

func createMsgAgentServer(address string) *msg.MsgAgentServer {
	s := msg.NewProtobufMsgAgentServer(newMsgAgentServerHandler, nil, acommon.IdMsgMapper, common.WithTickSpan(time.Millisecond*10), common.WithSendListMode(acommon.SendListMode))
	if err := s.Listen(address); err != nil {
		log.Infof("agent server listen and serve err %v", err)
		return nil
	}
	log.Infof("listening %v", address)
	return s
}

func main() {
	s := createMsgAgentServer(acommon.MsgAgentServerAddress)
	if s == nil {
		return
	}
	defer s.End()
	log.Infof("agent server started")
	s.Start()
}
