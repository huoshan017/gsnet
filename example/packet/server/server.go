package main

import (
	"fmt"
	"time"

	"github.com/huoshan017/gsnet/common"
	ex_packet_common "github.com/huoshan017/gsnet/example/packet/common"
	"github.com/huoshan017/gsnet/log"
	"github.com/huoshan017/gsnet/packet"
	"github.com/huoshan017/gsnet/server"
)

type testServerHandler struct {
	state int32 // 1 表示服务器模式  2 表示客户端模式
}

func newTestServerHandler(args ...any) common.ISessionEventHandler {
	h := &testServerHandler{}
	return h
}

func (h *testServerHandler) OnConnect(sess common.ISession) {
	log.Infof("new client(session_id: %v) connected", sess.GetId())
}

func (h *testServerHandler) OnDisconnect(sess common.ISession, err error) {
	log.Infof("client(session_id: %v) disconnected, err: %v", sess.GetId(), err)
}

func (h *testServerHandler) OnPacket(sess common.ISession, packet packet.IPacket) error {
	log.Infof("session %v OnPacket packet %v", sess.GetId(), packet.Data())
	err := sess.Send(packet.Data(), true)
	if err != nil {
		str := fmt.Sprintf("OnData with session %v send err: %v", sess.GetId(), err)
		log.Infof(str)
	}
	return err
}

func (h *testServerHandler) OnTick(sess common.ISession, tick time.Duration) {

}

func (h *testServerHandler) OnError(err error) {
	log.Infof("server occur err: %v @@@ @@@", err)
}

func createTestServer() *server.Server {
	return server.NewServer(newTestServerHandler,
		common.WithReadBuffSize(10*4096),
		common.WithWriteBuffSize(5*4096),
		common.WithPacketCompressType(packet.CompressSnappy),
		common.WithPacketEncryptionType(packet.EncryptionDes),
	)
}

func main() {
	s := createTestServer()
	err := s.Listen(ex_packet_common.TestAddress)
	if err != nil {
		log.Fatalf("server for test client listen err: %+v", err)
		return
	}
	defer s.End()

	s.Start()

	log.Infof("server for test client running")
}
