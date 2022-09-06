package kcp

import (
	"io"
	"net"
	"sync"
	"sync/atomic"

	"github.com/huoshan017/gsnet/pool"
)

type mBuffer struct {
	buf  *[]byte
	used int32
	ref  int32
}

func (b *mBuffer) left() int32 {
	return int32(len(*b.buf)) - b.used
}

func (b *mBuffer) clear() {
	b.buf = nil
	b.used = 0
	b.ref = 0
}

type mBufferSlice struct {
	slice  []byte
	buffer *mBuffer
}

func (s mBufferSlice) getData() []byte {
	return s.slice
}

func (s *mBufferSlice) skip(n int32) bool {
	if n > int32(len(s.slice)) {
		return false
	}
	s.slice = s.slice[n:]
	return true
}

func (s *mBufferSlice) read(buf []byte) int32 {
	n := int32(copy(s.slice, buf))
	s.slice = s.slice[n:]
	return n
}

func (s mBufferSlice) finish(recycle func(*mBuffer)) {
	if atomic.AddInt32(&s.buffer.ref, -1) == 0 && recycle != nil {
		recycle(s.buffer)
	}
}

func Read2MBuffer(reader io.Reader, buf *mBuffer) (mBufferSlice, error) {
	n, e := reader.Read((*buf.buf)[buf.used:])
	if e != nil {
		return mBufferSlice{}, e
	}
	oldUsed := (*buf).used
	buf.used += int32(n)
	atomic.AddInt32(&buf.ref, 1)
	return mBufferSlice{slice: (*buf.buf)[oldUsed:buf.used], buffer: buf}, nil
}

func ReadFromUDP2MBuffer(conn *net.UDPConn, buf *mBuffer) (mBufferSlice, *net.UDPAddr, error) {
	n, addr, e := conn.ReadFromUDP((*buf.buf)[buf.used:])
	if e != nil {
		return mBufferSlice{}, nil, e
	}
	oldUsed := (*buf).used
	buf.used += int32(n)
	atomic.AddInt32(&buf.ref, 1)
	return mBufferSlice{slice: (*buf.buf)[oldUsed:buf.used], buffer: buf}, addr, nil
}

const (
	defaultMBufferSize = 8192
	minMBufferSize     = 2048
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

func getMBufferWithSize(size int32) *mBuffer {
	if size < minMBufferSize {
		size = minMBufferSize
	}
	buf := mbufferPool.Get().(*mBuffer)
	buf.buf = pool.GetBuffPool().Alloc(size)
	return buf
}

func putMBuffer(buf *mBuffer) {
	pool.GetBuffPool().Free(buf.buf)
	buf.clear()
	mbufferPool.Put(buf)
}

func SetMBufferSize(size int32) {
	if size < minMBufferSize {
		size = minMBufferSize
	}
	mbufferSize = size
}
