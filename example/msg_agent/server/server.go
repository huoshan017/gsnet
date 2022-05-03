package main

import (
	"time"

	"github.com/huoshan017/gsnet/common"
	"github.com/huoshan017/gsnet/log"
	"github.com/huoshan017/gsnet/msg"
	"github.com/huoshan017/gsnet/test/tproto"

	acommon "github.com/huoshan017/gsnet/example/msg_agent/common"
)

func createMsgAgentClient() (*msg.MsgAgentClient, error) {
	c := msg.NewProtobufMsgAgentClient(acommon.IdMsgMapper)
	if err := c.Dial(acommon.MsgAgentServerAddress); err != nil {
		return nil, err
	}
	log.Infof("msg agent client connected")
	c.SetConnectHandle(func(sess *msg.MsgSession) {
		log.Infof("msg agent client (sess %v) connected to server", sess.GetId())
	})
	c.SetDisconnectHandle(func(sess *msg.MsgSession, err error) {
		log.Infof("msg agent client (sess %v) disconnected from server", sess.GetId())
	})
	c.SetTickHandle(func(sess *msg.MsgSession, tick time.Duration) {
	})
	c.SetErrorHandle(func(err error) {
		log.Infof("msg agent client ocurr err %v", err)
	})
	return c, nil
}

type serverHandlerUseMsgAgentClient struct {
	msgAgentClient *msg.MsgAgentClient
	agentSess      *msg.MsgAgentSession
}

func newServerHandlerUseMsgAgentClient(args ...any) msg.IMsgSessionEventHandler {
	msgAgentClient := args[0].(*msg.MsgAgentClient)
	return &serverHandlerUseMsgAgentClient{msgAgentClient: msgAgentClient}
}

func (h *serverHandlerUseMsgAgentClient) OnConnected(sess *msg.MsgSession) {
	log.Infof("session %v connected to server", sess.GetId())
}

func (h *serverHandlerUseMsgAgentClient) OnDisconnected(sess *msg.MsgSession, err error) {
	log.Infof("session %v disconnected from server", sess.GetId())
}

func (h *serverHandlerUseMsgAgentClient) OnMsgHandle(sess *msg.MsgSession, msgid msg.MsgIdType, msgobj any) error {
	h.agentSess = h.getAgentSess(sess)
	if h.agentSess == nil {
		log.Fatalf("why dont get message agent session")
		return nil
	}
	if msgid == acommon.MsgIdPing {
		m := msgobj.(*tproto.MsgPing)
		var response tproto.MsgPong
		response.Content = m.Content
		return h.agentSess.Send(acommon.MsgIdPong, &response)
	} else {
		log.Fatalf("unsupported message id %v", msgid)
	}
	return nil
}

func (h *serverHandlerUseMsgAgentClient) OnTick(sess *msg.MsgSession, tick time.Duration) {
}

func (h *serverHandlerUseMsgAgentClient) OnError(err error) {
	log.Infof("occur err %v on server", err)
}

func (h *serverHandlerUseMsgAgentClient) getAgentSess(sess *msg.MsgSession) *msg.MsgAgentSession {
	if h.agentSess == nil {
		h.agentSess = h.msgAgentClient.BoundHandleAndGetAgentSession(sess, h.OnMsgFromAgentServer)
	}
	return h.agentSess
}

func (h *serverHandlerUseMsgAgentClient) OnMsgFromAgentServer(sess *msg.MsgSession, msgid msg.MsgIdType, msgobj any) error {
	return sess.SendMsg(msgid, msgobj)
}

func createServerUseMsgAgentClient(address string) *msg.MsgServer {
	msgAgentClient, err := createMsgAgentClient()
	if err != nil {
		log.Fatalf("create agent client err %v", err)
		return nil
	}
	s := msg.NewProtobufMsgServer(newServerHandlerUseMsgAgentClient, []any{msgAgentClient}, acommon.IdMsgMapper, common.WithTickSpan(100*time.Millisecond))
	if err = s.Listen(address); err != nil {
		log.Infof("test server listen err %v", err)
		return nil
	}
	return s
}

func main() {
	s := createServerUseMsgAgentClient(acommon.TestAddress)
	if s == nil {
		return
	}
	defer s.End()
	s.Start()
}
