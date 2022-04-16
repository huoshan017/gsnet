package common

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/huoshan017/gsnet/common/packet"
)

const (
	MaxCacheSentPacket int16 = 256
)

type IResendEventHandler interface {
	OnSent(data any, mmt packet.MemoryManagementType) bool
	OnAck(pak packet.IPacket) int32
	OnProcessed(n int16)
	OnUpdate(IConn) error
}

type IResendChecker interface {
	CanSend() bool
}

type IResender interface {
	Resend(sess ISession) error
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
		data any
		mmt  packet.MemoryManagementType
	}
	locker           sync.Mutex
	sentList2        *resendList
	nextSentIndex    int16
	nextAckSentIndex int16
	nProcessed       int16
	nextRecvIndex    int16
	ackTime          time.Time
	disposed         bool
}

type resendNode struct {
	value *struct {
		data any
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
		data any
		mmt  packet.MemoryManagementType
	}{})
	return &resendList{head: n, tail: n}
}

func (l *resendList) getNum() int32 {
	return atomic.LoadInt32(&l.num)
}

func (l *resendList) pushBack(v *struct {
	data any
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
	data any
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
			data any
			mmt  packet.MemoryManagementType
		}, 0),
		sentList2: newResendList(),
		config:    *config,
	}
}

func (rd *ResendData) CanSend() bool {
	var can bool
	if rd.config.UseLockFree {
		if rd.sentList2.getNum() < 255 {
			can = true
		}
	} else {
		if len(rd.sentList) < 255 {
			can = true
		}
	}
	return can
}

// ResendData.OnSent call in different goroutine to OnAck
func (rd *ResendData) OnSent(data any, mmt packet.MemoryManagementType) bool {
	var sent bool
	if rd.config.UseLockFree {
		if rd.sentList2.getNum() < int32(MaxCacheSentPacket) {
			rd.sentList2.pushBack(&struct {
				data any
				mmt  packet.MemoryManagementType
			}{data, mmt})
			sent = true
		}
	} else {
		rd.locker.Lock()
		if len(rd.sentList) < int(MaxCacheSentPacket) {
			rd.sentList = append(rd.sentList, struct {
				data any
				mmt  packet.MemoryManagementType
			}{data, mmt})
			sent = true
		}
		rd.locker.Unlock()
	}
	if sent {
		rd.nextSentIndex += 1
		if rd.nextSentIndex >= int16(MaxCacheSentPacket) {
			rd.nextSentIndex = 0
		}
	}
	return sent
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

	rd.nextAckSentIndex = (rd.nextAckSentIndex + n) % MaxCacheSentPacket

	return 1
}

func (rd *ResendData) OnProcessed(n int16) {
	rd.nProcessed += n
	rd.nextRecvIndex = (rd.nextRecvIndex + n) % MaxCacheSentPacket
}

func (rd *ResendData) OnUpdate(conn IConn) error {
	return rd.update(conn)
}

func (rd *ResendData) Dispose() {
	if rd.disposed {
		return
	}
	if rd.config.UseLockFree {
		n := rd.sentList2.getNum()
		for i := int32(0); i < n; i++ {
			sd := rd.sentList2.popFront()
			FreeSendData(sd.mmt, sd.data)
		}
	} else {
		n := len(rd.sentList)
		for i := int16(0); i < int16(n); i++ {
			sd := rd.sentList[i]
			FreeSendData(sd.mmt, sd.data)
		}
		rd.sentList = rd.sentList[:0]
	}
	rd.disposed = true
}

func (rd *ResendData) update(conn IConn) error {
	now := time.Now()
	if rd.ackTime.IsZero() {
		rd.ackTime = now
	}

	var err error
	if rd.nProcessed <= 0 {
		return err
	}
	if rd.nProcessed >= rd.config.AckSentNum || now.Sub(rd.ackTime) >= rd.config.AckSentSpan {
		err = conn.Send(packet.PacketSentAck, []byte{byte(rd.nProcessed >> 8), byte(rd.nProcessed & 0xff)}, false)
		if err == nil || IsNoDisconnectError(err) {
			rd.nProcessed = 0
			rd.ackTime = now
		}
	}
	return err
}
