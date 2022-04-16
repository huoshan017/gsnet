package pool

import (
	"math/rand"
	"testing"
	"time"
)

func TestBufferPool(t *testing.T) {
	dataCh := make(chan *[]byte, 1000)
	endCh := make(chan struct{})
	rand.Seed(time.Now().Unix() + time.Now().UnixNano())
	go func() {
		for i := 0; i < 10000000; i++ {
			n := rand.Int31n(128*1024 + 1)
			if n == 0 {
				continue
			}
			buf := GetBuffPool().Alloc(n)
			dataCh <- buf
		}
		close(dataCh)
	}()
	go func() {
		for buf := range dataCh {
			b := *buf
			b[0] = 1
			GetBuffPool().Free(buf)
		}
		close(endCh)
	}()
	<-endCh
}

func BenchmarkBufferPool(b *testing.B) {
	rand.Seed(time.Now().Unix() + time.Now().UnixNano())
	for n := 0; n < b.N; n++ {
		nn := rand.Int31n(128*1024 + 1)
		if nn == 0 {
			continue
		}
		buf := GetBuffPool().Alloc(nn)
		(*buf)[0] = 1
		GetBuffPool().Free(buf)
	}
}

func BenchmarkNoBufferPool(b *testing.B) {
	rand.Seed(time.Now().Unix() + time.Now().UnixNano())
	for n := 0; n < b.N; n++ {
		nn := rand.Int31n(128*1024 + 1)
		if nn == 0 {
			continue
		}
		buf := make([]byte, nn)
		buf[0] = 1
		buf = nil
	}
}

func BenchmarkBufferPoolParallel(b *testing.B) {
	b.RunParallel(func(p *testing.PB) {
		for p.Next() {
			nn := rand.Int31n(128*1024 + 1)
			if nn > 0 {
				buf := GetBuffPool().Alloc(nn)
				(*buf)[0] = 1
				GetBuffPool().Free(buf)
			}
		}
	})
}
