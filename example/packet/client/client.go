package main

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/huoshan017/gsnet/client"
	"github.com/huoshan017/gsnet/common"
	"github.com/huoshan017/gsnet/log"
	"github.com/huoshan017/gsnet/packet"

	ex_packet_common "github.com/huoshan017/gsnet/example/packet/common"
)

type testClientUseUpdateHandler struct {
	sendDataList *ex_packet_common.SendDataInfo
	compareNum   int32
}

func newTestClientUseUpdateHandler(args ...any) common.ISessionEventHandler {
	h := &testClientUseUpdateHandler{}
	h.sendDataList, _ = args[0].(*ex_packet_common.SendDataInfo)
	return h
}

func (h *testClientUseUpdateHandler) OnConnect(sess common.ISession) {
	log.Infof("connected")
}

func (h *testClientUseUpdateHandler) OnDisconnect(sess common.ISession, err error) {
	log.Infof("disconnected, err: %v", err)
}

func (h *testClientUseUpdateHandler) OnPacket(sess common.ISession, packet packet.IPacket) error {
	var (
		o bool
		e error
	)
	data := packet.Data()
	if o, e = h.sendDataList.CompareData(data, true); !o {
		err := fmt.Errorf("compare err: %v", e)
		panic(err)
	}
	h.compareNum += 1
	if h.compareNum >= 50000 {
		sess.Close()
	}
	log.Infof("compared %v", h.compareNum)
	return nil
}

func (h *testClientUseUpdateHandler) OnTick(sess common.ISession, tick time.Duration) {
}

func (h *testClientUseUpdateHandler) OnError(err error) {
	log.Infof("occur err: %v", err)
}

func createTestClientUseUpdate(userData any) *client.Client {
	// 启用tick处理
	return client.NewClient(newTestClientUseUpdateHandler(userData),
		client.WithRunMode(client.RunModeOnlyUpdate),
	)
}

var letters = []byte("abcdefghijklmnopqrstuvwxyz01234567890~!@#$%^&*()_+-={}[]|:;'<>?/.,")
var ran = rand.New(rand.NewSource(time.Now().UnixNano()))

func randBytes(n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[ran.Intn(len(letters))]
	}
	return b
}

func main() {
	sendNum := 10
	sd := ex_packet_common.CreateSendDataInfo(int32(sendNum))
	tc := createTestClientUseUpdate(sd)
	err := tc.Connect(ex_packet_common.TestAddress)
	if err != nil {
		log.Fatalf("test client connect err: %+v", err)
		return
	}
	defer tc.Close()

	for {
		err = tc.Update()
		if err != nil {
			log.Infof("test client update err %v", err)
			break
		}
		d := randBytes(30)
		err = tc.Send(d, false)
		if err != nil {
			log.Infof("test client send err: %+v", err)
			break
		}
		sd.AppendSendData(d)
		time.Sleep(time.Millisecond)
	}

	time.Sleep(time.Second)

	log.Infof("test done")
}
