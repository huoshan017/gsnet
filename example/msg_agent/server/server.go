package main

import (
	"net/http"
	"time"

	"github.com/huoshan017/gsnet/common"
	"github.com/huoshan017/gsnet/log"
	"github.com/huoshan017/gsnet/msg"
	"github.com/huoshan017/gsnet/test/tproto"

	_ "net/http/pprof"

	acommon "github.com/huoshan017/gsnet/example/msg_agent/common"
)

func createMsgAgentClient() (*msg.MsgAgentClient, error) {
	c := msg.NewProtobufMsgAgentClient(acommon.IdMsgMapper, common.WithSendListMode(acommon.SendListMode))
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

func (h *serverHandlerUseMsgAgentClient) OnReady(sess *msg.MsgSession) {
	h.agentSess = h.msgAgentClient.BoundServerSession(sess, h.OnMsgFromAgentServer)
	log.Infof("session %v ready", sess.GetId())
}

func (h *serverHandlerUseMsgAgentClient) OnDisconnected(sess *msg.MsgSession, err error) {
	if h.agentSess != nil {
		h.msgAgentClient.UnboundServerSession(sess, h.agentSess)
	}
	log.Infof("session %v disconnected from server", sess.GetId())
}

func (h *serverHandlerUseMsgAgentClient) OnMsgHandle(sess *msg.MsgSession, msgid msg.MsgIdType, msgobj any) error {
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

func (h *serverHandlerUseMsgAgentClient) OnTick(sess *msg.MsgSession, tick time.Duration) {
}

func (h *serverHandlerUseMsgAgentClient) OnError(err error) {
	log.Infof("occur err %v on server", err)
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
	s := msg.NewProtobufMsgServer(newServerHandlerUseMsgAgentClient, []any{msgAgentClient}, acommon.IdMsgMapper, common.WithSendListMode(acommon.SendListMode))
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
	go func() {
		http.ListenAndServe("0.0.0.0:6060", nil)
	}()
	s.Serve()
}
