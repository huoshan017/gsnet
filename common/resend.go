package common

import (
	"fmt"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/gogo/protobuf/proto"

	"github.com/huoshan017/gsnet/options"
	"github.com/huoshan017/gsnet/packet"
	"github.com/huoshan017/gsnet/protocol"
)

const (
	MaxCacheSentPacket int32 = 256
	MaxSentPacketCount int32 = 65536
)

var (
	ErrPeerRecvNumMustEqualToAckNum = func(peerRecvNum, ackNum int32) error {
		return fmt.Errorf("gsnet: peer received packet num %v must equal to ack num %v", peerRecvNum, ackNum)
	}
)

type IResendEventHandler interface {
	OnSent(data any, pt_mmt int32)
	OnAck(pak packet.IPacket) error
	OnProcessed(packet.IPacket)
	OnUpdate(IConn) error
}

type sendNode struct {
	value *wrapperSendData
	next  unsafe.Pointer
}

type SendList struct {
	head unsafe.Pointer
	tail unsafe.Pointer
	num  int32
}

func newSendList() *SendList {
	n := unsafe.Pointer(&sendNode{})
	return &SendList{head: n, tail: n}
}

func (l *SendList) getNum() int32 {
	return atomic.LoadInt32(&l.num)
}

func (l *SendList) pushBack(v *wrapperSendData) {
	n := &sendNode{value: v}
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

func (l *SendList) popFront() *wrapperSendData {
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

func load(p *unsafe.Pointer) (n *sendNode) {
	return (*sendNode)(atomic.LoadPointer(p))
}

func cas(p *unsafe.Pointer, old, new *sendNode) (ok bool) {
	return atomic.CompareAndSwapPointer(p, unsafe.Pointer(old), unsafe.Pointer(new))
}

// 重传数据类
type ResendData struct {
	config           options.ResendConfig
	sentList         *SendList
	ackTotalNum      int32 // 已确认自己发送数据包的总数量
	peerRecvTotalNum int32 // 对面已接收自己发送数据包的总数量，对方的确认包中带来的
	recvTotalNum     int32 // 已接收对方数据包的总数量
	sendRecvCount    int32 // 可发送确认包时的接收数量，没发送一次重置清零，一般不超过100
	ackTime          time.Time
	disposed         bool
}

func NewResendData(config *options.ResendConfig) *ResendData {
	return &ResendData{
		sentList: newSendList(),
		config:   *config,
	}
}

// ResendData.OnSent call in different goroutine to OnAck
func (rd *ResendData) OnSent(data any, pt_mmt int32) {
	rd.sentList.pushBack(&wrapperSendData{data, pt_mmt})
}

// ResendData.OnAck ack the packet
// the return value 0 means not ack packet, -1 means ack packet failed, 1 means success
func (rd *ResendData) OnAck(pak packet.IPacket) error {
	if pak.Type() != packet.PacketSentAck {
		return nil
	}

	d := pak.Data()
	var sentAck protocol.SentAck
	err := proto.Unmarshal(d, &sentAck)
	if err != nil {
		return err
	}

	sentNum := rd.sentList.getNum()
	if sentAck.RecvCount > sentNum {
		return fmt.Errorf("acknum %v > length of sentlist %v", sentAck.RecvCount, sentNum)
	}

	for i := int32(0); i < sentAck.RecvCount; i++ {
		node := rd.sentList.popFront()
		FreeSendData(node.getMMT(), node.data)
	}

	// 对面已收到的总数量
	rd.peerRecvTotalNum = sentAck.RecvTotalNum

	// 已确认的总数量
	rd.ackTotalNum += sentAck.RecvCount
	if rd.ackTotalNum > MaxSentPacketCount {
		rd.ackTotalNum -= MaxSentPacketCount
	}

	if rd.peerRecvTotalNum != rd.ackTotalNum {
		return ErrPeerRecvNumMustEqualToAckNum(rd.peerRecvTotalNum, rd.ackTotalNum)
	}

	return nil
}

func (rd *ResendData) OnProcessed(pak packet.IPacket) {
	if pak.Type() != packet.PacketNormalData {
		return
	}
	rd.sendRecvCount += 1
	rd.recvTotalNum += 1
	if rd.recvTotalNum > MaxSentPacketCount {
		rd.recvTotalNum -= MaxSentPacketCount
	}
}

func (rd *ResendData) OnUpdate(conn IConn) error {
	return rd.update(conn)
}

func (rd *ResendData) Dispose() {
	if rd.disposed {
		return
	}
	n := rd.sentList.getNum()
	for i := int32(0); i < n; i++ {
		sd := rd.sentList.popFront()
		if sd != nil {
			FreeSendData(sd.getMMT(), sd.data)
		}
	}
	rd.disposed = true
}

func (rd ResendData) getAckTotalNum() int32 {
	return rd.ackTotalNum
}

func (rd ResendData) getSendRecvCount() int32 {
	return rd.sendRecvCount
}

func (rd ResendData) getRecvTotalNum() int32 {
	return rd.recvTotalNum
}

func (rd *ResendData) update(conn IConn) error {
	now := time.Now()
	if rd.ackTime.IsZero() {
		rd.ackTime = now
	}

	if rd.sendRecvCount <= 0 {
		return nil
	}

	var (
		data []byte
		err  error
	)

	if rd.sendRecvCount >= rd.config.AckSentNum || now.Sub(rd.ackTime) >= rd.config.AckSentSpan {
		var sentAck = &protocol.SentAck{
			RecvCount:    rd.sendRecvCount,
			RecvTotalNum: rd.recvTotalNum,
		}
		if data, err = proto.Marshal(sentAck); err == nil {
			err = conn.Send(packet.PacketSentAck, data, false)
			if err == nil || IsNoDisconnectError(err) {
				rd.sendRecvCount = 0
				rd.ackTime = now
			}
		}
	}

	return err
}

type ReconnectInfo struct {
	sessId     uint64
	sessKey    uint64
	resendData *ResendData
}

func (ri ReconnectInfo) GetSessionId() uint64 {
	return atomic.LoadUint64(&ri.sessId)
}

func (ri ReconnectInfo) GetSessionKey() uint64 {
	return atomic.LoadUint64(&ri.sessKey)
}

func (ri ReconnectInfo) GetResendData() *ResendData {
	var rp = (*unsafe.Pointer)(unsafe.Pointer(&ri.resendData))
	return (*ResendData)(atomic.LoadPointer(rp))
}

func (ri *ReconnectInfo) Set(id uint64, key uint64, resendData *ResendData) {
	atomic.StoreUint64(&ri.sessId, id)
	atomic.StoreUint64(&ri.sessKey, key)
	var rp = (*unsafe.Pointer)(unsafe.Pointer(&ri.resendData))
	atomic.StorePointer(rp, unsafe.Pointer(resendData))
}
