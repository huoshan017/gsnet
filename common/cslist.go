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

// wrapperSendData.getData transfer any data to the right type
func (sd *wrapperSendData) getData() ([]byte, *[]byte, [][]byte, []*[]byte) {
	return GetSendData(sd.data)
}

func (sd wrapperSendData) getPacketType() packet.PacketType {
	return packet.PacketType((sd.pt_mmt >> 16) & 0xffff)
}

func (sd wrapperSendData) getMMT() packet.MemoryManagementType {
	return packet.MemoryManagementType(sd.pt_mmt & 0xffff)
}

// only free to returns from wrapperSendData.getData with same instance
func (sd *wrapperSendData) toFree(b []byte, pb *[]byte, ba [][]byte, pba []*[]byte) {
	mmt := sd.getMMT()
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
}

func newSlist() *slist {
	return &slist{}
}

func (s slist) getLength() int32 {
	return s.length
}

func (s *slist) pushBack(wd wrapperSendData) {
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
	PushBack(wrapperSendData) bool
	PopFront() (wrapperSendData, bool)
	Close()
	Finalize()
}

type condSendList struct {
	cond     *sync.Cond
	sendList *slist
	quit     int32
}

func newCondSendList() *condSendList {
	return &condSendList{
		cond:     sync.NewCond(&sync.Mutex{}),
		sendList: newSlist(),
	}
}

func (l *condSendList) PushBack(wd wrapperSendData) bool {
	if atomic.LoadInt32(&l.quit) == 1 {
		return false
	}
	l.cond.L.Lock()
	defer l.cond.L.Unlock()
	l.sendList.pushBack(wd)
	l.cond.Signal()
	return true
}

func (l *condSendList) PopFront() (wrapperSendData, bool) {
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

type newSendListFunc func() ISendList

var (
	newSendListFuncArray []newSendListFunc = []newSendListFunc{
		func() ISendList { return newCondSendList() },
		func() ISendList { return newUnlimitedChan() },
	}
)
