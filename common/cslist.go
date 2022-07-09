package common

import (
	"sync"
	"sync/atomic"

	"github.com/huoshan017/gsnet/packet"
)

var (
	nullWrapperSendData = wrapperSendData{}
	snodePool           *sync.Pool
	snodeMemMode        = 0
)

func init() {
	snodePool = &sync.Pool{
		New: func() any {
			return &snode{}
		},
	}
}

func snodeGet() *snode {
	var n *snode
	if snodeMemMode == 0 {
		n = &snode{}
	} else {
		n = snodePool.Get().(*snode)
		n.next = nil
	}
	return n
}

func snodePut(n *snode) {
	if snodeMemMode != 0 {
		snodePool.Put(n)
	}
}

// wrapperSendData send data wrapper
type wrapperSendData struct {
	data   any
	pt_mmt int32
}

func mergePacketTypeMMTAndResend(pt packet.PacketType, mmt packet.MemoryManagementType, resend bool) int32 {
	var r = func() int32 {
		if resend {
			return 1
		}
		return 0
	}()
	return (int32(pt) << 16 & 0x00ff0000) | (int32(mmt) << 8 & 0xff00) | r
}

func getPacketType(pt_mmt int32) packet.PacketType {
	return packet.PacketType((pt_mmt >> 16) & 0xff)
}

// wrapperSendData.getData transfer any data to the right type
func (sd *wrapperSendData) getData() ([]byte, *[]byte, [][]byte, []*[]byte) {
	return GetSendData(sd.data)
}

func (sd wrapperSendData) getPacketType() packet.PacketType {
	return getPacketType(sd.pt_mmt)
}

func (sd wrapperSendData) getMMT() packet.MemoryManagementType {
	return packet.MemoryManagementType(sd.pt_mmt >> 8 & 0xff)
}

func (sd wrapperSendData) getResend() bool {
	return sd.pt_mmt&0xff > 0
}

// only free to returns from wrapperSendData.getData with same instance
func (sd *wrapperSendData) toFree(b []byte, pb *[]byte, ba [][]byte, pba []*[]byte) {
	var mmt = sd.getMMT()
	FreeSendData2(mmt, b, pb, ba, pba)
}

// wrapperSendData.recycle recycle free send data to pool
func (sd *wrapperSendData) recycle() {
	if sd.data == nil {
		panic("gsnet: wrapper send data nil")
	}
	b, pb, ba, pba := sd.getData()
	sd.toFree(b, pb, ba, pba)
}

type snode struct {
	value wrapperSendData
	next  *snode
}

type slist struct {
	head   *snode
	tail   *snode
	length int32
	maxLen int32
}

func newSlist() *slist {
	return newSlistWithLength(0)
}

func newSlistWithLength(length int32) *slist {
	return &slist{
		maxLen: length,
	}
}

func (s slist) getLength() int32 {
	return s.length
}

func (s *slist) pushBack(wd wrapperSendData) bool {
	if s.maxLen > 0 && s.length >= s.maxLen {
		return false
	}
	n := snodeGet()
	n.value = wd
	if s.head == nil {
		s.head = n
		s.tail = s.head
	} else {
		s.tail.next = n
		s.tail = n
	}
	s.length += 1
	return true
}

func (s *slist) popFront() (wrapperSendData, bool) {
	wd, o := s.peekFront()
	if o {
		s.deleteFront()
	}
	return wd, o
}

func (s *slist) peekFront() (wrapperSendData, bool) {
	if s.head == nil {
		return nullWrapperSendData, false
	}
	return s.head.value, true
}

func (s *slist) deleteFront() {
	if s.head == nil {
		return
	}
	n := s.head
	s.head = n.next
	snodePut(n)
	s.length -= 1
	if s.length == 0 {
		s.tail = nil
	}
}

func (s *slist) recycle() {
	n := s.head
	for n != nil {
		n.value.recycle()
		snodePut(n)
		n = n.next
	}
	s.length = 0
}

// 发送队列接口
type ISendList interface {
	PushBack(wrapperSendData) error
	PopFront() (wrapperSendData, bool)
	Close()
	Finalize()
}

type condSendList struct {
	cond     *sync.Cond
	sendList *slist
	quit     int32
}

func newCondSendList(maxLength int32) *condSendList {
	return &condSendList{
		cond:     sync.NewCond(&sync.Mutex{}),
		sendList: newSlistWithLength(maxLength),
	}
}

func (l *condSendList) PushBack(wd wrapperSendData) error {
	if atomic.LoadInt32(&l.quit) == 1 {
		return ErrConnClosed
	}
	l.cond.L.Lock()
	defer l.cond.L.Unlock()
	if !l.sendList.pushBack(wd) {
		return ErrSendListFull
	}
	l.cond.Signal()
	return nil
}

func (l *condSendList) PopFront() (wrapperSendData, bool) {
	if atomic.LoadInt32(&l.quit) == 1 {
		return nullWrapperSendData, false
	}
	l.cond.L.Lock()
	defer l.cond.L.Unlock()
	for l.sendList.getLength() == 0 {
		l.cond.Wait()
		if atomic.LoadInt32(&l.quit) == 1 {
			return nullWrapperSendData, false
		}
	}
	return l.sendList.popFront()
}

func (l *condSendList) Finalize() {
	l.cond.L.Lock()
	defer l.cond.L.Unlock()
	l.sendList.recycle()
}

func (l *condSendList) Close() {
	atomic.StoreInt32(&l.quit, 1)
	l.cond.Signal()
}

type newSendListFunc func(int32) ISendList

var (
	newSendListFuncMap map[int32]newSendListFunc = map[int32]newSendListFunc{
		0: func(maxLength int32) ISendList { return newCondSendList(maxLength) },
		1: func(maxLength int32) ISendList { return newUnlimitedChan(maxLength) },
	}
)
