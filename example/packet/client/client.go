package main

import (
	"flag"
	"fmt"
	"math/rand"
	"os"
	"sync"
	"sync/atomic"
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

var compared int32

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
	if atomic.CompareAndSwapInt32(&compared, 0, 1) {
		log.Infof("compared %v", h.compareNum)
	}
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

func randBytes(n int, ran *rand.Rand) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[ran.Intn(len(letters))]
	}
	return b
}

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("args num invalid")
		return
	}
	var clientNum int
	flag.IntVar(&clientNum, "client_num", 1, "client number")
	flag.Parse()

	log.Infof("client num %v", clientNum)

	var leftNum int32 = int32(clientNum)
	var wg sync.WaitGroup
	wg.Add(clientNum)
	for i := 0; i < clientNum; i++ {
		go func() {
			defer func() {
				wg.Done()
				atomic.AddInt32(&leftNum, -1)
				log.Infof("left client %v", leftNum)
			}()

			sd := ex_packet_common.CreateSendDataInfo(10)
			tc := createTestClientUseUpdate(sd)
			err := tc.Connect(ex_packet_common.TestAddress)
			if err != nil {
				log.Infof("test client connect err: %+v", err)
				return
			}
			defer tc.Close()

			ran := rand.New(rand.NewSource(time.Now().UnixNano()))
			for {
				err = tc.Update()
				if err != nil {
					log.Infof("test client update err %v", err)
					break
				}
				d := randBytes(30, ran)
				err = tc.Send(d, true)
				if err != nil {
					log.Infof("test client send err: %+v", err)
					break
				}
				sd.AppendSendData(d)
				time.Sleep(time.Millisecond)
			}
		}()
	}

	wg.Wait()

	log.Infof("test done")
}
