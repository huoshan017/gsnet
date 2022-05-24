package main

import (
	"flag"
	"math/rand"
	"time"

	"github.com/huoshan017/gsnet/common"
	"github.com/huoshan017/gsnet/log"
	"github.com/huoshan017/gsnet/msg"

	fc "github.com/huoshan017/gsnet/example/framework/common"
	tproto "github.com/huoshan017/gsnet/test/tproto"
)

var (
	maxLatency int32 = 100 // milliseconds
)

type msgAgentServerHandler struct {
	rand *rand.Rand
}

func newMsgAgentServerHandler(args ...any) msg.IMsgSessionEventHandler {
	return &msgAgentServerHandler{}
}

func (h *msgAgentServerHandler) OnConnected(sess *msg.MsgSession) {
	h.rand = rand.New(rand.NewSource(time.Now().UnixNano()))
	log.Infof("session %v connected to message agent server", sess.GetId())
}

func (h *msgAgentServerHandler) OnReady(sess *msg.MsgSession) {
	log.Infof("session %v ready", sess.GetId())
}

func (h *msgAgentServerHandler) OnDisconnected(sess *msg.MsgSession, err error) {
	log.Infof("session %v disconnected from message agent server, err %v", sess.GetId(), err)
}

func (h *msgAgentServerHandler) OnMsgHandle(sess *msg.MsgSession, msgid msg.MsgIdType, msgobj any) error {
	var err error
	if msgid == fc.MsgIdPing {
		//time.Sleep(time.Duration(h.rand.Int31n(maxLatency)) * time.Millisecond)
		m := msgobj.(*tproto.MsgPing)
		var response tproto.MsgPong
		response.Content = m.Content
		err = sess.SendMsg(fc.MsgIdPong, &response)
	} else {
		log.Fatalf("unsupported message id %v", msgid)
	}
	return err
}

func (h *msgAgentServerHandler) OnTick(sess *msg.MsgSession, tick time.Duration) {
}

func (h *msgAgentServerHandler) OnError(err error) {
	log.Infof("occur err %v on message agent server", err)
}

func createMsgAgentServer(address string) *msg.MsgAgentServer {
	s := msg.NewProtobufMsgAgentServer(newMsgAgentServerHandler, nil, fc.IdMsgMapper, common.WithTickSpan(time.Millisecond*10))
	if err := s.Listen(address); err != nil {
		log.Infof("agent server listen and serve err %v", err)
		return nil
	}
	log.Infof("listening %v", address)
	return s
}

func main() {
	var addressIndex int
	flag.IntVar(&addressIndex, "address_index", 0, "backend address index")
	flag.Parse()
	s := createMsgAgentServer(fc.BackendAddress[addressIndex])
	if s == nil {
		return
	}
	defer s.End()
	log.Infof("agent server started")
	s.Serve()
}
