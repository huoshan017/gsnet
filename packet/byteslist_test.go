package packet

import (
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/huoshan017/gsnet/pool"
)

type chunk struct {
	rawbuf  *[]byte
	datalen int32
	isend   bool
}

var (
	chunkPool *sync.Pool
)

func init() {
	chunkPool = &sync.Pool{
		New: func() any {
			return &chunk{}
		},
	}
}

func getChunk() *chunk {
	c := chunkPool.Get().(*chunk)
	return c
}

func putChunk(c *chunk) {
	//if c.isend {
	//	pool.GetBuffPool().Free(c.rawbuf)
	//}
	c.rawbuf = nil
	c.datalen = 0
	c.isend = false
	chunkPool.Put(c)
}

func TestRing(t *testing.T) {
	var (
		bufSize int32 = 1024
	)
	r := rand.New(rand.NewSource(time.Now().Unix()))
	ring := newBytesNodeRing(100)

	for k := 0; k < 1000; k++ {
		c := r.Int31n(200) + 1
		for i := int32(0); i < c; i++ {
			node := getBytesNode()
			node.buf = pool.GetBuffPool().Alloc(bufSize)
			if !ring.PushBack(node) {
				break
			}
		}
		c = r.Int31n(200) + 1
		for i := int32(0); i < c; i++ {
			if node, o := ring.PopFront(); o {
				node.forceRelease(true)
				putBytesNode(node)
			} else {
				break
			}
		}
	}

	t.Logf("ring: %v", *ring)
}

var lettersLen = len(letters)
var (
	chunkSize   int32 = 2048
	maxDataSize int32 = 1024 * 5
	sendNum           = 1000000
	chanLength        = 50
)

func randBytes2Buf(ran *rand.Rand, buf []byte) {
	for i := 0; i < len(buf); i++ {
		r := ran.Int31n(int32(lettersLen))
		buf[i] = letters[r]
	}
}

func int16ToBytes(num int16, buf []byte) {
	buf[0] = byte(num & 0xff)
	buf[1] = byte(num >> 8)
}

func bytes2int16(buf []byte) int16 {
	return int16(buf[0]) + (int16(buf[1])<<8)&0x7f00
}

func makeHead(t *testing.T, sn int32, head []byte, lastChunk *chunk, buffOffset int32, ch chan *chunk) (*chunk, int32) {
	var (
		cn int32
	)
	for int(cn) < len(head) {
		if buffOffset < 0 {
			panic(fmt.Sprintf("when send head buffOffset(%v) must >= 0", buffOffset))
		}
		if lastChunk == nil || lastChunk.isend {
			lastChunk = getChunk()
			lastChunk.rawbuf = pool.GetBuffPool().Alloc(chunkSize)
		}
		var c = int32(copy((*lastChunk.rawbuf)[buffOffset:], head[cn:]))
		lastChunk.datalen += c
		cn += c
		buffOffset += c
		t.Logf("(%v) make head: lastChunk.rawbuf(%p) lastChunk.datacount(%v) buffOffset(%v)", sn, lastChunk.rawbuf, lastChunk.datalen, buffOffset)
		if buffOffset == int32(len(*lastChunk.rawbuf)) {
			lastChunk.isend = true
			ch <- lastChunk
			lastChunk = nil
			buffOffset = 0
		}
	}
	return lastChunk, buffOffset
}

func makeBody(t *testing.T, sn int32, r *rand.Rand, rn int32, lastChunk *chunk, buffOffset int32, ch chan *chunk) (*chunk, int32) {
	for rn > 0 {
		if buffOffset < 0 {
			panic(fmt.Sprintf("when send bytes buffOffset(%v) must >= 0", buffOffset))
		}
		if lastChunk == nil || lastChunk.isend {
			lastChunk = getChunk()
			lastChunk.rawbuf = pool.GetBuffPool().Alloc(chunkSize)
		}
		left := int32(len(*lastChunk.rawbuf)) - buffOffset
		if left >= rn {
			randBytes2Buf(r, (*lastChunk.rawbuf)[buffOffset:buffOffset+rn])
			lastChunk.datalen += rn
			buffOffset += rn
			rn = 0
		} else {
			randBytes2Buf(r, (*lastChunk.rawbuf)[buffOffset:])
			lastChunk.datalen += left
			buffOffset += left
			rn -= left
		}
		t.Logf("(%v) make body: lastChunk.rawbuf(%p) lastChunk.datacount(%v) buffOffset(%v)", sn, lastChunk.rawbuf, lastChunk.datalen, buffOffset)
		if buffOffset == int32(len(*lastChunk.rawbuf)) {
			lastChunk.isend = true
			ch <- lastChunk
			lastChunk = nil
			buffOffset = 0
		} else {
			rawbuf := lastChunk.rawbuf
			ch <- lastChunk
			lastChunk = getChunk()
			lastChunk.rawbuf = rawbuf
		}
	}
	return lastChunk, buffOffset
}

func TestBytesList(t *testing.T) {
	var (
		lastChunk  *chunk
		buffOffset int32
		head       [2]byte
		r          = rand.New(rand.NewSource(time.Now().Unix()))
		ch         = make(chan *chunk, chanLength)
	)
	go func() {
		var rl int32
		for i := int32(0); i < int32(sendNum); i++ {
			rl = r.Int31n(maxDataSize) + 1
			// head
			int16ToBytes(int16(rl), head[:])
			lastChunk, buffOffset = makeHead(t, i, head[:], lastChunk, buffOffset, ch)
			// data
			lastChunk, buffOffset = makeBody(t, i, r, rl, lastChunk, buffOffset, ch)
			//time.Sleep(time.Millisecond)
		}
		close(ch)
	}()

	var (
		rhead   [2]byte
		c       *chunk
		rbytes  Bytes
		ok      bool
		datalen int16
		n       int32
		bl      = NewBytesList(4)
	)

	readFunc := func() bool {
		if datalen == 0 {
			ok = bl.ReadBytesTo(rhead[:], true)
			if !ok {
				t.Logf("BytesList read head failed")
				return false
			}
			datalen = bytes2int16(rhead[:])
		}
		rbytes, ok = bl.ReadBytes(int32(datalen), true)
		if !ok {
			return false
		}
		rbytes.Release(true)
		datalen = 0
		n += 1
		return true
	}

	for n := 0; n < sendNum; {
		c, ok = <-ch
		if !ok {
			break
		}
		if !bl.PushBytes(c.rawbuf, c.datalen) {
			t.Logf("!!!!! bl.PushBytes c.rawbuf(%p) c.datalen(%v) failed", c.rawbuf, c.datalen)
		}
		putChunk(c)
		for readFunc() {
			n += 1
		}
	}

	t.Logf("BytesList remain data %+v", bl)

	bl.Release()
}
