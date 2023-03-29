package main

import (
	"bytes"
	"fmt"
	"math/rand"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	_ "net/http/pprof"

	"github.com/huoshan017/gsnet/log"
	"github.com/huoshan017/gsnet/msg"
	"github.com/huoshan017/gsnet/options"
	"github.com/huoshan017/gsnet/test/tproto"

	acommon "github.com/huoshan017/gsnet/example/msg_agent/common"
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

type testMsgClientUseUpdateHandler struct {
	index        int32
	sendDataList *sendDataInfo
	totalNum     int32
	compareNum   int32
}

func newTestMsgClientUseUpdateHandler(args ...any) *testMsgClientUseUpdateHandler {
	if len(args) < 3 {
		panic("At least need 3 arguments")
	}
	h := &testMsgClientUseUpdateHandler{}
	if len(args) > 0 {
		h.index = args[0].(int32)
	}
	if len(args) > 1 {
		h.sendDataList = args[1].(*sendDataInfo)
	}
	if len(args) > 2 {
		h.totalNum = args[2].(int32)
	}
	return h
}

func (h *testMsgClientUseUpdateHandler) OnConnected(sess *msg.MsgSession) {
	log.Infof("client %v connected", h.index)
}

func (h *testMsgClientUseUpdateHandler) OnReady(sess *msg.MsgSession) {
	log.Infof("client %v ready", h.index)
}

func (h *testMsgClientUseUpdateHandler) OnDisconnected(sess *msg.MsgSession, err error) {
	log.Infof("client %v disconnected, err: %v", h.index, err)
}

func (h *testMsgClientUseUpdateHandler) OnHandlePong(sess *msg.MsgSession, msgobj any) error {
	var (
		o bool
		e error
	)
	m := msgobj.(*tproto.MsgPong)
	data := []byte(m.Content)
	if o, e = h.sendDataList.compareData(data, true); !o {
		err := fmt.Errorf("client %v compare err: %v", h.index, e)
		panic(err)
	}
	h.compareNum += 1
	if h.compareNum >= h.totalNum {
		sess.Close()
	}
	//h.t.Logf("compared %v", h.compareNum)
	return nil
}

func (h *testMsgClientUseUpdateHandler) OnTick(sess *msg.MsgSession, tick time.Duration) {
}

func (h *testMsgClientUseUpdateHandler) OnError(err error) {
	log.Infof("client %v occur err: %v", h.index, err)
}

func createMsgClientUseUpdate(index int32, userData any, count int32, updateInterval time.Duration) *msg.MsgClient {
	// 启用tick处理
	handler := newTestMsgClientUseUpdateHandler(index, userData, count)
	c := msg.NewProtobufMsgClient(acommon.IdMsgMapper, options.WithRunMode(options.RunModeOnlyUpdate), options.WithSendListMode(acommon.SendListMode), options.WithNetProto(options.NetProtoUDP), options.WithKcpInterval(int32(updateInterval.Milliseconds())))
	c.SetConnectHandle(handler.OnConnected)
	c.SetReadyHandle(handler.OnReady)
	c.SetDisconnectHandle(handler.OnDisconnected)
	c.SetTickHandle(handler.OnTick)
	c.SetErrorHandle(handler.OnError)
	c.RegisterMsgHandle(acommon.MsgIdPong, handler.OnHandlePong)
	return c
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
	const (
		clientNum      int32 = 10000
		compareNum     int32 = 200
		updateInterval       = 30 * time.Millisecond
	)
	var (
		wg    sync.WaitGroup
		count int32
		ch    = make(chan int32, clientNum)
	)

	go func() {
		http.ListenAndServe("0.0.0.0:6061", nil)
	}()

	wg.Add(int(clientNum))

	go func() {
		for c := range ch {
			log.Infof("already complete %v", c)
			wg.Done()
		}
	}()

	for i := int32(0); i < clientNum; i++ {
		sd := createSendDataInfo(100)
		client := createMsgClientUseUpdate(i, sd, compareNum, updateInterval)
		go func(no int32) {
			defer func() {
				ch <- atomic.AddInt32(&count, 1)
			}()

			err := client.Connect(acommon.TestAddress)
			if err != nil {
				log.Infof("test client connect err %v", err)
				return
			}
			defer client.Close()

			//log.Infof("test client %v connected server", no)

			var (
				cn       int32
				request  tproto.MsgPing
				nextTime = time.Now().Add(updateInterval)
				ran      = rand.New(rand.NewSource(time.Now().UnixNano()))
			)

			err = update(client, ran, &cn, compareNum, &request, sd)
			for err == nil {
				d := time.Until(nextTime)
				if d > 0 {
					time.Sleep(d)
				} else if d < 0 {
					nextTime = time.Now().Add(updateInterval)
				}
				err = update(client, ran, &cn, compareNum, &request, sd)
			}
		}(i + 1)
	}

	wg.Wait()
}

func update(client *msg.MsgClient, ran *rand.Rand, cn *int32, compareNum int32, request *tproto.MsgPing, sd *sendDataInfo) error {
	err := client.Update()
	if err != nil {
		log.Infof("test client update err %v", err)
		return err
	}
	if *cn < compareNum {
		rn := ran.Intn(128) + 1
		d := randBytes(rn, ran)
		request.Content = string(d)
		err = client.Send(acommon.MsgIdPing, request)
		if err != nil {
			log.Infof("test client send err: %+v", err)
			return err
		}
		sd.appendSendData(d)
		*cn += 1
	}
	return nil
}
