package kcp

import (
	"context"
	"math/rand"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"net/http"
	_ "net/http/pprof"

	"github.com/huoshan017/gsnet/common"
	"github.com/huoshan017/gsnet/options"
	"github.com/huoshan017/gsnet/packet"
	kcp "github.com/huoshan017/kcpgo"
)

var letters = []byte("abcdefghijklmnopqrstuvwxyz01234567890~!@#$%^&*()_+-={}[]|:;'<>?/.,")
var lettersLen = len(letters)

func randBytes2Buf(ran *rand.Rand, buf []byte) {
	for i := 0; i < len(buf); i++ {
		r := ran.Int31n(int32(lettersLen))
		buf[i] = letters[r]
	}
}

func newClientConn(t *testing.T, conn net.Conn, cidx int32, wg *sync.WaitGroup, completeNum *int32) *KConn {
	const (
		totalNum int32 = 100
	)
	var ops = options.NewOptions()
	ops.SetPacketPool(packet.GetDefaultPacketPool())
	kconn := NewKConn(conn, common.NewPacketCodec(ops), ops)
	go kconn.Run()
	go func() {
		var (
			ctx    context.Context
			cancel context.CancelFunc
			pak    packet.IPacket
			id     int32
			err    error
		)
		ctx, cancel = context.WithCancel(context.Background())
		var (
			sendNum      int32
			buffer       [8192]byte
			bnum, lnum   int32
			totalSent    int32
			totalReceive int32
			ran          = rand.New(rand.NewSource(time.Now().UnixMilli()))
		)
		defer wg.Done()
		for err == nil {
			pak, id, err = kconn.Wait(ctx, nil)
			if err != nil {
				cm := atomic.AddInt32(completeNum, 1)
				t.Logf("client wait err: %v, client num %v", err, cm)
				continue
			}
			if pak != nil {
				totalReceive += int32(len(pak.Data()))
				//t.Logf("client %v received %v total bytes", cidx, totalReceive)
				if totalReceive >= totalSent && sendNum >= totalNum {
					cm := atomic.AddInt32(completeNum, 1)
					t.Logf("client done %v", cm)
					break
				}
			} else if id < 0 {
				if sendNum < totalNum {
					if lnum == 0 {
						bnum = 1 + ran.Int31n(2000)
						randBytes2Buf(ran, buffer[:bnum])
					}
					lnum += 1
					if lnum >= 10 {
						lnum = 0
					}
					err = kconn.Send(packet.PacketNormalData, buffer[:bnum], true)
					if err != nil {
						cm := atomic.AddInt32(completeNum, 1)
						t.Logf("client %v send data err: %v, client num %v", cidx, err, cm)
						continue
					}
					totalSent += bnum
					sendNum += 1
					//t.Logf("client %v sent %v total bytes", cidx, totalSent)
				}
			}
			if pak != nil {
				packet.GetDefaultPacketPool().Put(pak)
			}
		}
		kconn.Close()
		cancel()
	}()
	return kconn
}

func newServerConn(t *testing.T, conn net.Conn, sidx int32) *KConn {
	var (
		recvNum   int32
		totalSent int32
	)
	var ops = options.NewOptions()
	ops.SetPacketPool(packet.GetDefaultPacketPool())
	kconn := NewKConn(conn, common.NewPacketCodec(ops), ops)
	go kconn.Run()
	go func() {
		var (
			ctx    context.Context
			cancel context.CancelFunc
			pak    packet.IPacket
			err    error
		)
		ctx, cancel = context.WithCancel(context.Background())
		for err == nil {
			pak, _, err = kconn.Wait(ctx, nil)
			if err != nil {
				t.Logf("server wait err: %v", err)
				continue
			}
			if err == nil && pak != nil {
				recvNum += 1
				err = kconn.Send(packet.PacketNormalData, pak.Data(), true)
				if err != nil {
					t.Logf("server send(count:%v) data err: %v", recvNum, err)
					continue
				}
				totalSent += int32(len(pak.Data()))
				//t.Logf("server %v sent total bytes %v", sidx, totalSent)
			}
			if pak != nil {
				packet.GetDefaultPacketPool().Put(pak)
			}
		}
		kconn.Close()
		cancel()
	}()
	return kconn
}

func testKConn(t *testing.T, reuseAddr, reusePort bool) {
	SetMBufferSize(16384)
	var (
		sops options.ServerOptions
		cops options.ClientOptions
	)
	if reuseAddr {
		sops.SetReuseAddr(true)
	}
	if reusePort {
		sops.SetReusePort(true)
	}

	kcp.UserMtuBufferFunc(getKcpMtuBuffer, putKcpMtuBuffer)
	acceptor, err := createAcceptor(t, "127.0.0.1:9000", &sops)
	if err != nil {
		t.Errorf("create acceptor err: %v", err)
		return
	}
	defer acceptor.Close()

	go func() {
		http.ListenAndServe("0.0.0.0:6060", nil)
	}()

	const (
		clientNum = 5
	)
	var (
		closeCh = make(chan struct{})
	)

	var wg sync.WaitGroup
	wg.Add(clientNum)
	go func() {
		var (
			connCh = acceptor.GetNewConnChan()
			run    = true
			c      int32
		)
		for run {
			select {
			case conn, o := <-connCh:
				if o {
					kc := newServerConn(t, conn, c)
					c += 1
					t.Logf("new KConn arrived %p, count %v", kc, c)
				}
			case <-closeCh:
				run = false
			}
		}
	}()

	var completeNum int32
	for i := int32(0); i < clientNum; i++ {
		conn, err := DialUDP("127.0.0.1:9000", &cops.Options)
		if err != nil {
			wg.Done()
			cm := atomic.AddInt32(&completeNum, 1)
			t.Logf("dial udp err: %v, client num %v", err, cm)
			continue
		}
		t.Logf("client(%v) connected server", conn.LocalAddr())
		newClientConn(t, conn, i, &wg, &completeNum)
	}

	wg.Wait()

	t.Logf("complete!!!")
}

func TestKConn(t *testing.T) {
	testKConn(t, false, false)
}

func TestKConnReuseAddr(t *testing.T) {
	testKConn(t, true, false)
}

func TestKConnReusePort(t *testing.T) {
	testKConn(t, false, true)
}

func TestKConnReuseAddrPort(t *testing.T) {
	testKConn(t, true, true)
}
