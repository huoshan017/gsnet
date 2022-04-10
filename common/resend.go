package common

import (
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/huoshan017/gsnet/common/packet"
)

type IResendEventHandler interface {
	OnSent(data interface{}, mmt packet.MemoryManagementType)
	OnAck(pak packet.IPacket) int32
	OnProcessed(n int16)
	OnUpdate(IConn) error
}

type IResendChecker interface {
	CanSend() bool
}

type IResendSender interface {
	Send(writer io.Writer) error
}

type ResendConfig struct {
	SentListLength int16
	AckSentSpan    time.Duration
	AckSentNum     int16
	UseLockFree    bool
}

type ResendData struct {
	config   ResendConfig
	sentList []struct {
		data interface{}
		mmt  packet.MemoryManagementType
	}
	locker     sync.Mutex
	sentList2  *resendList
	nProcessed int16
	ackTime    time.Time
}

type resendNode struct {
	value *struct {
		data interface{}
		mmt  packet.MemoryManagementType
	}
	next unsafe.Pointer
}

type resendList struct {
	head unsafe.Pointer
	tail unsafe.Pointer
	num  int32
}

func newResendList() *resendList {
	n := unsafe.Pointer(&struct {
		data interface{}
		mmt  packet.MemoryManagementType
	}{})
	return &resendList{head: n, tail: n}
}

func (l *resendList) getNum() int32 {
	return atomic.LoadInt32(&l.num)
}

func (l *resendList) pushBack(v *struct {
	data interface{}
	mmt  packet.MemoryManagementType
}) {
	n := &resendNode{value: v}
	for {
		tail := load(&l.tail)
		next := load(&tail.next)
		if tail == load(&l.tail) {
			if next == nil {
				if cas(&tail.next, next, n) {
					cas(&l.tail, tail, n)
					atomic.AddInt32(&l.num, 1)
					return
				}
			} else {
				cas(&l.tail, tail, next)
			}
		}
	}
}

func (l *resendList) popFront() *struct {
	data interface{}
	mmt  packet.MemoryManagementType
} {
	for {
		head := load(&l.head)
		tail := load(&l.tail)
		next := load(&head.next)
		if head == load(&l.head) {
			if head == tail {
				if next == nil {
					return nil
				}
				cas(&l.tail, tail, next)
			} else {
				v := next.value
				if cas(&l.head, head, next) {
					atomic.AddInt32(&l.num, -1)
					return v
				}
			}
		}
	}
}

func load(p *unsafe.Pointer) (n *resendNode) {
	return (*resendNode)(atomic.LoadPointer(p))
}

func cas(p *unsafe.Pointer, old, new *resendNode) (ok bool) {
	return atomic.CompareAndSwapPointer(p, unsafe.Pointer(old), unsafe.Pointer(new))
}

func NewResendData(config *ResendConfig) *ResendData {
	return &ResendData{
		sentList: make([]struct {
			data interface{}
			mmt  packet.MemoryManagementType
		}, 0),
		sentList2: newResendList(),
		config:    *config,
	}
}

// ResendData.OnSent call in different goroutine to OnAck
func (rd *ResendData) OnSent(data interface{}, mmt packet.MemoryManagementType) {
	if rd.config.UseLockFree {
		rd.sentList2.pushBack(&struct {
			data interface{}
			mmt  packet.MemoryManagementType
		}{data, mmt})
	} else {
		rd.locker.Lock()
		rd.sentList = append(rd.sentList, struct {
			data interface{}
			mmt  packet.MemoryManagementType
		}{data, mmt})
		rd.locker.Unlock()
	}
}

// ResendData.OnAck ack the packet
// the return value 0 means not ack packet, -1 means ack packet failed, 1 means success
func (rd *ResendData) OnAck(pak packet.IPacket) int32 {
	if pak.Type() != packet.PacketSentAck {
		return 0
	}

	d := *pak.Data()
	n := (int16(d[0])<<8)&0x7f00 | int16(d[1]&0xff)

	if rd.config.UseLockFree {
		num := int16(rd.sentList2.getNum())
		if n > num {
			errstr := fmt.Sprintf("acknum %v > length of sentlist %v", n, num)
			GetLogger().Fatalf(errstr)
			//panic(errstr)
			return -1
		}

		if n <= 0 {
			n = num
		}

		for i := int16(0); i < n; i++ {
			node := rd.sentList2.popFront()
			FreeSendData(node.mmt, node.data)
		}
	} else {
		if int(n) > len(rd.sentList) {
			errstr := fmt.Sprintf("acknum %v > length of sentlist %v", n, len(rd.sentList))
			GetLogger().Fatalf(errstr)
			return -1
		}

		if n <= 0 {
			n = int16(len(rd.sentList))
		}

		rd.locker.Lock()
		for i := int16(0); i < n; i++ {
			sd := rd.sentList[i]
			FreeSendData(sd.mmt, sd.data)
		}
		rd.sentList = rd.sentList[n:]
		rd.locker.Unlock()
	}

	return 1
}

func (d *ResendData) OnProcessed(n int16) {
	d.nProcessed += n
}

func (d *ResendData) OnUpdate(conn IConn) error {
	return d.update(conn)
}

func (d *ResendData) update(conn IConn) error {
	now := time.Now()
	if d.ackTime.IsZero() {
		d.ackTime = now
	}

	var err error
	if d.nProcessed <= 0 {
		return err
	}
	if d.nProcessed >= d.config.AckSentNum || now.Sub(d.ackTime) >= d.config.AckSentSpan {
		err = conn.Send(packet.PacketSentAck, []byte{byte(d.nProcessed >> 8), byte(d.nProcessed & 0xff)}, false)
		if err == nil || IsNoDisconnectError(err) {
			d.nProcessed = 0
			d.ackTime = now
		}
	}
	return err
}
