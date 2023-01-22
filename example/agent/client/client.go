package main

import (
	"bytes"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/huoshan017/gsnet/client"
	"github.com/huoshan017/gsnet/common"
	"github.com/huoshan017/gsnet/log"
	"github.com/huoshan017/gsnet/options"
	"github.com/huoshan017/gsnet/packet"

	ecommon "github.com/huoshan017/gsnet/example/agent/common"
)

type sendDataInfo struct {
	list        [][]byte
	num         int32
	cnum        int32
	numCh       chan int32
	numChClosed bool
}

func createSendDataInfo(cnum int32) *sendDataInfo {
	return &sendDataInfo{
		list:  make([][]byte, 0),
		cnum:  cnum,
		numCh: make(chan int32, 1),
	}
}

// 发送goroutine中调用
func (info *sendDataInfo) appendSendData(data []byte) {
	info.list = append(info.list, data)
}

// 在逻辑goroutine中调用
func (info *sendDataInfo) compareData(data []byte, isForward bool) (bool, error) {
	if bytes.Equal(info.list[0], data) {
		if isForward {
			info.compareForward(false)
		}
		return true, nil
	}
	return false, fmt.Errorf("data %v compare info.list[0] %v failed", data, info.list[0])
}

func (info *sendDataInfo) compareForward(toLock bool) {
	info.list = info.list[1:]
	info.num += 1
	if !info.numChClosed && info.num >= info.cnum {
		info.numCh <- info.num
		close(info.numCh)
		info.numChClosed = true
	}
}

type testClientUseUpdateHandler struct {
	sendDataList *sendDataInfo
	totalNum     int32
	compareNum   int32
}

func newTestClientUseUpdateHandler(args ...any) common.ISessionHandler {
	if len(args) < 2 {
		panic("At least need 2 arguments")
	}
	h := &testClientUseUpdateHandler{}
	if len(args) > 0 {
		h.sendDataList, _ = args[0].(*sendDataInfo)
	}
	if len(args) > 1 {
		h.totalNum = args[1].(int32)
	}
	return h
}

func (h *testClientUseUpdateHandler) OnConnect(sess common.ISession) {
	log.Infof("connected")
}

func (h *testClientUseUpdateHandler) OnReady(sess common.ISession) {
	log.Infof("ready")
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
	if o, e = h.sendDataList.compareData(data, true); !o {
		err := fmt.Errorf("compare err: %v", e)
		panic(err)
	}
	h.compareNum += 1
	if h.compareNum >= h.totalNum {
		sess.Close()
	}
	//h.t.Logf("compared %v", h.compareNum)
	return nil
}

func (h *testClientUseUpdateHandler) OnTick(sess common.ISession, tick time.Duration) {
}

func (h *testClientUseUpdateHandler) OnError(err error) {
	log.Infof("occur err: %v", err)
}

func createTestClientUseUpdate(userData any, count int32) *client.Client {
	// 启用tick处理
	return client.NewClient(newTestClientUseUpdateHandler(userData, count), options.WithRunMode(options.RunModeOnlyUpdate))
}

var letters = []byte("abcdefghijklmnopqrstuvwxyz01234567890~!@#$%^&*()_+-={}[]|:;'<>?/.,")
var lettersLen = len(letters)

func randBytes(n int, ran *rand.Rand) []byte {
	b := make([]byte, n)
	for i := 0; i < len(b); i++ {
		r := ran.Int31n(int32(lettersLen))
		b[i] = letters[r]
	}
	return b
}

func main() {
	var (
		clientNum        = 5000
		compareNum int32 = 200
		wg         sync.WaitGroup
		count      int32
		ch         = make(chan int32, clientNum)
	)

	wg.Add(clientNum)

	go func() {
		for c := range ch {
			log.Infof("already complete %v", c)
			wg.Done()
		}
	}()

	for i := 0; i < clientNum; i++ {
		sd := createSendDataInfo(100)
		client := createTestClientUseUpdate(sd, compareNum)
		go func(no int) {
			defer func() {
				ch <- atomic.AddInt32(&count, 1)
			}()

			err := client.Connect(ecommon.TestAddress)
			if err != nil {
				log.Infof("test client connect err %v", err)
				return
			}
			defer client.Close()

			log.Infof("test client %v connected server", no)

			var cn int32
			ran := rand.New(rand.NewSource(time.Now().UnixNano()))
			for {
				err = client.Update()
				if err != nil {
					log.Infof("test client update err %v", err)
					break
				}
				if cn < compareNum {
					rn := ran.Intn(1024) + 1
					d := randBytes(rn, ran)
					err = client.Send(d, false)
					if err != nil {
						log.Infof("test client send err: %+v", err)
						break
					}
					sd.appendSendData(d)
					cn += 1
				}
				time.Sleep(time.Millisecond * 10)
			}
		}(i + 1)
	}

	wg.Wait()
}
