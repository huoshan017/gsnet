package common

import (
	"sync"

	"github.com/huoshan017/gsnet/packet"
)

var (
	nullWrapperSendData = wrapperSendData{}
	snodePool           *sync.Pool
)

func init() {
	snodePool = &sync.Pool{
		New: func() any {
			return &snode{}
		},
	}
}

func snodeGet() *snode {
	n := snodePool.Get().(*snode)
	n.next = nil
	return n
}

func snodePut(n *snode) {
	snodePool.Put(n)
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
	if s.head == nil {
		return nullWrapperSendData, false
	}
	n := s.head
	s.head = n.next
	snodePut(n)
	s.length -= 1
	if s.length == 0 {
		s.tail = nil
	}
	return n.value, true
}

func (s *slist) recycle() {
	n := s.head
	for n != nil {
		n.value.recycle()
		n = n.next
	}
}

type condSendList struct {
	cond     *sync.Cond
	sendList *slist
}

func newCondSendList() *condSendList {
	return &condSendList{
		cond:     sync.NewCond(&sync.Mutex{}),
		sendList: newSlist(),
	}
}

func (l *condSendList) pushBack(wd wrapperSendData) {
	l.cond.L.Lock()
	l.sendList.pushBack(wd)
	l.cond.L.Unlock()
	l.cond.Signal()
}

func (l *condSendList) popFront() (wrapperSendData, bool) {
	l.cond.L.Lock()
	defer l.cond.L.Unlock()
	for l.sendList.getLength() == 0 {
		l.cond.Wait()
	}
	return l.sendList.popFront()
}
