package main

import (
	"net/http"
	"time"

	"github.com/huoshan017/gsnet/log"
	"github.com/huoshan017/gsnet/msg"
	"github.com/huoshan017/gsnet/options"
	"github.com/huoshan017/gsnet/test/tproto"

	_ "net/http/pprof"

	acommon "github.com/huoshan017/gsnet/example/msg_agent/common"
)

func createMsgAgent() (*msg.MsgAgent, error) {
	c := msg.NewProtobufMsgAgent(acommon.IdMsgMapper, options.WithSendListMode(acommon.SendListMode))
	if err := c.Dial(acommon.MsgAgentServerAddress); err != nil {
		return nil, err
	}
	log.Infof("msg agent client connected")
	c.SetConnectHandle(func(sess *msg.MsgSession) {
		log.Infof("msg agent client (sess %v) connected to server", sess.GetId())
	})
	c.SetReadyHandle(func(sess *msg.MsgSession) {
		log.Infof("msg agent client (sess %v) ready", sess.GetId())
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

type serverHandlerUseMsgAgent struct {
	msgAgent  *msg.MsgAgent
	agentSess *msg.MsgAgentSession
}

func newServerHandlerUseMsgAgent(args ...any) msg.IMsgSessionHandler {
	msgAgent := args[0].(*msg.MsgAgent)
	return &serverHandlerUseMsgAgent{msgAgent: msgAgent}
}

func (h *serverHandlerUseMsgAgent) OnConnected(sess *msg.MsgSession) {
	log.Infof("session %v connected to server", sess.GetId())
}

func (h *serverHandlerUseMsgAgent) OnReady(sess *msg.MsgSession) {
	h.agentSess = h.msgAgent.BoundServerSession(sess, h.OnMsgFromAgentServer)
	log.Infof("session %v ready", sess.GetId())
}

func (h *serverHandlerUseMsgAgent) OnDisconnected(sess *msg.MsgSession, err error) {
	if h.agentSess != nil {
		h.msgAgent.UnboundServerSession(sess, h.agentSess)
	}
	log.Infof("session %v disconnected from server", sess.GetId())
}

func (h *serverHandlerUseMsgAgent) OnMsgHandle(sess *msg.MsgSession, msgid msg.MsgIdType, msgobj any) error {
	if h.agentSess == nil {
		log.Fatalf("why dont get message agent session")
		return nil
	}

	var err error
	if msgid == acommon.MsgIdPing {
		m := msgobj.(*tproto.MsgPing)
		var response tproto.MsgPong
		response.Content = m.Content
		err = h.agentSess.Send(acommon.MsgIdPong, &response)
	} else {
		log.Fatalf("unsupported message id %v", msgid)
	}
	return err
}

func (h *serverHandlerUseMsgAgent) OnTick(sess *msg.MsgSession, tick time.Duration) {
}

func (h *serverHandlerUseMsgAgent) OnError(err error) {
	log.Infof("occur err %v on server", err)
}

func (h *serverHandlerUseMsgAgent) OnMsgFromAgentServer(sess *msg.MsgSession, agentId int32, msgid msg.MsgIdType, msgobj any) error {
	return sess.SendMsg(msgid, msgobj)
}

func createServerUseMsgAgent(address string) *msg.MsgServer {
	msgAgent, err := createMsgAgent()
	if err != nil {
		log.Fatalf("create agent client err %v", err)
		return nil
	}
	s := msg.NewProtobufMsgServer(newServerHandlerUseMsgAgent, []any{msgAgent}, acommon.IdMsgMapper, options.WithSendListMode(acommon.SendListMode))
	if err = s.Listen(address); err != nil {
		log.Infof("test server listen err %v", err)
		return nil
	}
	return s
}

func main() {
	s := createServerUseMsgAgent(acommon.TestAddress)
	if s == nil {
		return
	}
	defer s.End()
	go func() {
		http.ListenAndServe("0.0.0.0:6060", nil)
	}()
	s.Serve()
}
