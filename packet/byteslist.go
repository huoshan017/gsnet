package packet

import (
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/huoshan017/gsnet/pool"
)

const (
	nodeStateNoShare         = 0 // 未共享
	nodeStateSharing         = 1 // 正在共享
	nodeStateSharingReadEnd  = 2 // 共享结束
	nodeStateSharingForceEnd = 3 // 共享强制结束
)

type bytesNode struct {
	buf             *[]byte
	offset, datalen int32
	state           int32
	refcount        int32
}

func (bn *bytesNode) init(buf *[]byte, datalen int32) {
	bn.buf = buf
	bn.datalen = datalen
}

func (bn bytesNode) dataLength() int32 {
	return bn.datalen
}

func (bn bytesNode) left() int32 {
	return int32(len(*bn.buf)) - (bn.offset + bn.datalen)
}

func (bn *bytesNode) rightShift(buf *[]byte, slen int32) {
	if buf != bn.buf {
		panic(fmt.Sprintf("bytesNode right shift buf(%v) is not equal to param buf(%v)", bn.buf, buf))
	}
	left := bn.left()
	if slen > left {
		panic(fmt.Sprintf("bytesNode right shift length %v greater to left space of buf %v", slen, left))
	}
	bn.datalen += slen
}

func (bn *bytesNode) readTo(buf []byte, concurrent bool) int32 {
	if bn.datalen == 0 {
		return 0
	}
	n := copy(buf, (*bn.buf)[bn.offset:bn.offset+bn.datalen])
	bn.offset += int32(n)
	bn.datalen -= int32(n)
	if concurrent && int(bn.offset) == len(*bn.buf) && bn.datalen == 0 {
		atomic.CompareAndSwapInt32(&bn.state, nodeStateSharing, nodeStateSharingReadEnd)
	}
	return int32(n)
}

func (bn *bytesNode) readSub(length int32, concurrent bool) int32 {
	if bn.datalen < length {
		return -1
	}

	var offset = bn.offset
	bn.offset += length
	bn.datalen -= length
	if concurrent {
		atomic.AddInt32(&bn.refcount, 1)
	}
	return offset
}

func (bn *bytesNode) usedOnce() {
	atomic.AddInt32(&bn.refcount, -1)
}

func (bn bytesNode) isUsedout() bool {
	return int(bn.offset) == len(*bn.buf) && bn.datalen == 0
}

func (bn *bytesNode) isReleased() bool {
	return bn.buf == nil && bn.offset == 0 && bn.datalen == 0
}

func (bn *bytesNode) checkRelease(concurrent bool) bool {
	if !concurrent || atomic.LoadInt32(&bn.refcount) <= 0 {
		if bn.buf != nil && bn.offset == int32(len(*bn.buf)) && bn.datalen == 0 {
			pool.GetBuffPool().Free(bn.buf)
			bn.buf = nil
			bn.offset = 0
			return true
		}
	}
	return false
}

func (bn *bytesNode) forceRelease(concurrent bool) {
	if !concurrent || atomic.LoadInt32(&bn.refcount) <= 0 {
		if bn.buf != nil {
			pool.GetBuffPool().Free(bn.buf)
			bn.buf = nil
			bn.offset = 0
			bn.datalen = 0
		}
	}
}

var (
	bytesNodePool *sync.Pool
)

func init() {
	bytesNodePool = &sync.Pool{
		New: func() any {
			return &bytesNode{}
		},
	}
}

func getBytesNode() *bytesNode {
	return bytesNodePool.Get().(*bytesNode)
}

func putBytesNode(node *bytesNode) {
	bytesNodePool.Put(node)
}

type bytesNodeRing struct {
	ring        []*bytesNode
	read, count int32
}

func newBytesNodeRing(length int32) *bytesNodeRing {
	ring := make([]*bytesNode, length)
	return &bytesNodeRing{ring: ring}
}

func newBytesNodeRingObj(length int32) bytesNodeRing {
	ring := make([]*bytesNode, length)
	return bytesNodeRing{ring: ring}
}

func (r *bytesNodeRing) Clear() {
	for i := r.read; i < r.read+r.count; i++ {
		idx := int(i) % len(r.ring)
		if r.ring[idx] != nil {
			r.ring[idx].forceRelease(false)
			r.ring[idx] = nil
		}
	}
	r.read = 0
	r.count = 0
}

func (r bytesNodeRing) Count() int32 {
	return r.count
}

func (r bytesNodeRing) Length() int32 {
	return int32(len(r.ring))
}

func (r bytesNodeRing) IsEmpty() bool {
	return r.count == 0
}

func (r bytesNodeRing) IsFull() bool {
	return int(r.count) >= len(r.ring)
}

func (r *bytesNodeRing) PushBack(d *bytesNode) bool {
	if r.count >= int32(len(r.ring)) {
		return false
	}
	r.count += 1
	r.ring[int(r.read+r.count-1)%len(r.ring)] = d
	return true
}

func (r *bytesNodeRing) PopFront() (*bytesNode, bool) {
	if r.count <= 0 {
		return nil, false
	}
	d := r.ring[r.read]
	r.ring[r.read] = nil
	r.read += 1
	if r.read >= int32(len(r.ring)) {
		r.read = 0
	}
	r.count -= 1
	return d, true
}

func (r *bytesNodeRing) RemoveFront() bool {
	if r.count <= 0 {
		return false
	}
	r.ring[r.read] = nil
	r.read += 1
	if r.read >= int32(len(r.ring)) {
		r.read = 0
	}
	r.count -= 1
	return true
}

func (r bytesNodeRing) Front() (*bytesNode, bool) {
	if r.count <= 0 {
		return nil, false
	}
	return r.ring[r.read], true
}

func (r bytesNodeRing) Back() (*bytesNode, bool) {
	if r.count <= 0 {
		return nil, false
	}
	return r.ring[(r.read+r.count-1)%int32(len(r.ring))], true
}

type Bytes struct {
	ref     any
	offset  int32
	datalen int32
}

func (b Bytes) Data() []byte {
	var (
		bn     *bytesNode
		rawbuf *[]byte
		ok     bool
	)
	if bn, ok = b.ref.(*bytesNode); ok {
		return (*bn.buf)[b.offset : b.offset+b.datalen]
	} else if rawbuf, ok = b.ref.(*[]byte); ok {
		return *rawbuf
	}
	return nil
}

func (b Bytes) PData() *[]byte {
	return b.ref.(*[]byte)
}

func (b Bytes) IsShared() bool {
	if _, ok := b.ref.(*bytesNode); ok {
		return true
	}
	return false
}

func (b *Bytes) Release(concurrent bool) bool {
	var (
		bn     *bytesNode
		rawbuf *[]byte
		ok     bool
	)
	if bn, ok = b.ref.(*bytesNode); ok {
		bn.usedOnce()
		res := bn.checkRelease(concurrent)
		if res {
			putBytesNode(bn)
		}
		return res
	} else if rawbuf, ok = b.ref.(*[]byte); ok {
		pool.GetBuffPool().Free(rawbuf)
		return true
	}
	return false
}

var (
	nilBytes Bytes
)

type IBytesList interface {
	BytesLeft() int32
	PushBytes(buf *[]byte, datalen int32) bool
	ReadBytesTo(buf []byte, concurrent bool) bool
	ReadBytes(length int32, concurrent bool) (Bytes, bool)
}

type BytesList struct {
	ring       bytesNodeRing
	totalBytes int32
}

func NewBytesList(length int32) BytesList {
	return BytesList{ring: newBytesNodeRingObj(length)}
}

func NewBytesListConcurrent(length int32) BytesList {
	return BytesList{ring: newBytesNodeRingObj(length)}
}

func (bl *BytesList) Release() {
	bl.ring.Clear()
	bl.totalBytes = 0
}

func (bl BytesList) BytesLeft() int32 {
	return bl.totalBytes
}

func (bl *BytesList) PushBytes(buf *[]byte, datalen int32) bool {
	if buf == nil {
		panic("nil buf is nil")
	}

	if int(datalen) > len(*buf) {
		panic("datalen cant great to buf length")
	}

	var (
		node *bytesNode
		ok   bool
	)

	if node, ok = bl.ring.Front(); ok {
		for node.isReleased() {
			bl.ring.RemoveFront()
			putBytesNode(node)
			node, ok = bl.ring.Front()
			if !ok {
				break
			}
		}
	}

	node, ok = bl.ring.Back()
	if !ok || node.left() == 0 {
		if bl.ring.IsFull() {
			return false
		}
		node = getBytesNode()
		node.init(buf, datalen)
		bl.ring.PushBack(node)
	} else {
		node.rightShift(buf, datalen)
	}

	bl.totalBytes += datalen

	return true
}

func (bl *BytesList) ReadBytesTo(buf []byte, concurrent bool) bool {
	return bl.readTo(buf, concurrent)
}

func (bl *BytesList) ReadBytes(length int32, concurrent bool) (Bytes, bool) {
	if length == 0 || bl.totalBytes < length {
		return nilBytes, false
	}

	node, ok := bl.ring.Front()
	if !ok {
		panic("cant get front to BytesList")
	}

	if node.dataLength() >= length {
		offset := node.readSub(length, concurrent)
		bl.totalBytes -= length
		if node.isUsedout() {
			bl.ring.RemoveFront()
		}
		return Bytes{ref: node, offset: offset, datalen: length}, true
	}

	buf := pool.GetBuffPool().Alloc(length)
	bl.readTo(*buf, concurrent)
	return Bytes{ref: buf}, true
}

func (bl *BytesList) readTo(buf []byte, concurrent bool) bool {
	if int(bl.totalBytes) < len(buf) {
		return false
	}

	var (
		rn int32
	)
	for int(rn) < len(buf) {
		node, ok := bl.ring.Front()
		if ok {
			r := node.readTo(buf[rn:], concurrent)
			if r > 0 {
				rn += r
			}
			if node.checkRelease(concurrent) {
				bl.ring.RemoveFront()
				putBytesNode(node)
			}
		} else {
			panic("cant get front for BytesList")
		}
	}

	bl.totalBytes -= int32(len(buf))
	return true
}
