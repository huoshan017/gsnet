package pool

import (
	"math/rand"
	"testing"
	"time"
)

func TestBufferPool(t *testing.T) {
	rand.Seed(time.Now().Unix() + time.Now().UnixNano())
	for n := 0; n < 100; n++ {
		go func() {
			for i := 0; i < 100000; i++ {
				n := rand.Int31n(128*1024 + 1)
				if n == 0 {
					continue
				}
				buf := GetBuffPool().Alloc(n)
				b := (*buf)[:n]
				b[0] = 1
				GetBuffPool().Free(buf)
			}
		}()
	}
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
