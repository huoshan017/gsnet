package kcp

import (
	"io"
	"net"
	"sync"
	"sync/atomic"

	"github.com/huoshan017/gsnet/pool"
)

const (
	mBufferStateFree       = iota
	mBufferStateUsing      = 1
	mBufferStateCanRecycle = 2
)

type mBuffer struct {
	buf   *[]byte
	used  int32
	ref   int32
	state int32
}

func (b mBuffer) buffer() []byte {
	return (*b.buf)[b.used:]
}

func (b *mBuffer) use(n int32) bool {
	if b.left() < n {
		return false
	}
	b.used += n
	atomic.AddInt32(&b.ref, 1)
	return true
}

// 最後一個slice，儅調用mBufferSlice.finish后mBuffer將被回收
func (b *mBuffer) lastSlice() (mBufferSlice, bool) {
	left := b.left()
	if left <= 0 {
		return mBufferSlice{}, false
	}
	atomic.AddInt32(&b.ref, 1)
	return mBufferSlice{slice: nil, buffer: b}, true
}

// 准備回收mBuffer
func (b *mBuffer) finish() bool {
	// 引用计数和可回收标记标记判断
	if atomic.AddInt32(&b.ref, -1) <= 0 && b.canRecycle() {
		return true
	}
	return false
}

func (b *mBuffer) left() int32 {
	return int32(len(*b.buf)) - b.used
}

func (b *mBuffer) clear() {
	b.buf = nil
	b.used = 0
	atomic.StoreInt32(&b.ref, 0)
	atomic.StoreInt32(&b.state, mBufferStateFree)
}

func (b *mBuffer) canRecycle() bool {
	return atomic.LoadInt32(&b.state) == mBufferStateCanRecycle
}

func (b *mBuffer) markRecycle() {
	atomic.StoreInt32(&b.state, mBufferStateCanRecycle)
}

type mBufferSlice struct {
	slice  []byte
	buffer *mBuffer
}

func (s mBufferSlice) getData() []byte {
	return s.slice
}

func (s mBufferSlice) finish(recycle func(*mBuffer)) {
	if s.buffer.finish() && recycle != nil {
		recycle(s.buffer)
	}
}

func Read2MBuffer(reader io.Reader, buf *mBuffer) (mBufferSlice, error) {
	b := buf.buffer()
	n, e := reader.Read(buf.buffer())
	if e != nil {
		return mBufferSlice{}, e
	}
	buf.use(int32(n))
	return mBufferSlice{slice: b[:n], buffer: buf}, nil
}

func ReadFrom2MBuffer(conn net.PacketConn, buf *mBuffer) (mBufferSlice, net.Addr, error) {
	ob := buf.buffer()
	n, addr, e := conn.ReadFrom(buf.buffer())
	if e != nil {
		return mBufferSlice{}, nil, e
	}
	buf.use(int32(n))
	return mBufferSlice{slice: ob[:n], buffer: buf}, addr, nil
}

const (
	defaultMBufferSize = 8192
	minMBufferSize     = 4096
)

var (
	mbufferPool sync.Pool
	mbufferSize int32
)

func init() {
	mbufferPool = sync.Pool{
		New: func() any {
			return &mBuffer{}
		},
	}
}

func getMBuffer() *mBuffer {
	buf := mbufferPool.Get().(*mBuffer)
	if mbufferSize == 0 {
		buf.buf = pool.GetBuffPool().Alloc(defaultMBufferSize)
	} else {
		buf.buf = pool.GetBuffPool().Alloc(mbufferSize)
	}
	return buf
}

func putMBuffer(buf *mBuffer) {
	pbuf := buf.buf
	buf.clear()
	mbufferPool.Put(buf)
	pool.GetBuffPool().Free(pbuf)
}

func SetMBufferSize(size int32) {
	if size < minMBufferSize {
		size = minMBufferSize
	}
	mbufferSize = size
}
