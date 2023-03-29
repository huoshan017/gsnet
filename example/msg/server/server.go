package main

import (
	"time"

	"github.com/huoshan017/gsnet/log"
	"github.com/huoshan017/gsnet/msg"
	"github.com/huoshan017/gsnet/options"
	"github.com/huoshan017/gsnet/test/tproto"

	"net/http"
	_ "net/http/pprof"

	acommon "github.com/huoshan017/gsnet/example/msg_agent/common"
)

type serverHandler struct {
}

func newServerHandler(args ...any) msg.IMsgSessionHandler {
	return &serverHandler{}
}

func (h *serverHandler) OnConnected(sess *msg.MsgSession) {
	log.Infof("session %v connected to server", sess.GetId())
}

func (h *serverHandler) OnReady(sess *msg.MsgSession) {
	log.Infof("session %v ready", sess.GetId())
}

func (h *serverHandler) OnDisconnected(sess *msg.MsgSession, err error) {
	log.Infof("session %v disconnected from server", sess.GetId())
}

func (h *serverHandler) OnMsgHandle(sess *msg.MsgSession, msgid msg.MsgIdType, msgobj any) error {
	var err error
	if msgid == acommon.MsgIdPing {
		m := msgobj.(*tproto.MsgPing)
		var response tproto.MsgPong
		response.Content = m.Content
		err = sess.SendMsg(acommon.MsgIdPong, &response)
	} else {
		log.Fatalf("unsupported message id %v", msgid)
	}
	return err
}

func (h *serverHandler) OnTick(sess *msg.MsgSession, tick time.Duration) {
}

func (h *serverHandler) OnError(err error) {
	log.Infof("occur err %v on server", err)
}

func createMsgServer(address string) (*msg.MsgServer, error) {
	var s = msg.NewProtobufMsgServer(newServerHandler, nil, acommon.IdMsgMapper, options.WithNetProto(options.NetProtoUDP))
	err := s.Listen(acommon.TestAddress)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func main() {
	s, err := createMsgServer(acommon.TestAddress)
	if err != nil {
		log.Infof("create msg server err: %v", err)
		return
	}

	defer func() {
		s.End()
	}()

	go func() {
		http.ListenAndServe("0.0.0.0:6060", nil)
	}()

	s.Serve()

	log.Infof("server started")
}
