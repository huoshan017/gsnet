package packet

import (
	"bytes"
	"fmt"
	"math/rand"
	"testing"
)

var letters = []byte("abcdefghijklmnopqrstuvwxyz01234567890~!@#$%^&*()_+-={}[]|:;'<>?/.,")

func randBytes(n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return b
}

func testCompressor(compressor ICompressor, decompressor IDecompressor, t *testing.T) {
	for i := 0; i < 10; i++ {
		bs := randBytes(50)
		t.Logf("%v: \r\noriginal bytes %v", i+1, bs)

		d, err := compressor.Compress(bs)
		if err != nil {
			t.Errorf("zlib compressor compress err: %v", err)
			return
		}

		t.Logf("compressed data: %v", d)

		var od []byte
		od, err = decompressor.Decompress(d)
		if err != nil {
			t.Errorf("zlib compressor decomporess err: %v", err)
			return
		}
		if !bytes.Equal(od, bs) {
			panic(fmt.Errorf("compare failed: %v    compare to     %v", od, bs))
		}
		t.Logf("result string %v", od)
	}
}

func benchmarkCompressor(compressor ICompressor, decompressor IDecompressor, b *testing.B) {
	bs := randBytes(50)
	d, err := compressor.Compress(bs)
	if err != nil {
		b.Errorf("zlib compressor compress err: %v", err)
		return
	}

	var od []byte
	od, err = decompressor.Decompress(d)
	if err != nil {
		b.Errorf("zlib compressor decomporess err: %v", err)
		return
	}
	if !bytes.Equal(od, bs) {
		panic(fmt.Errorf("compare failed: %v    compare to     %v", od, bs))
	}
}

func TestZlibCompressor(t *testing.T) {
	zc := NewZlibCompressor()
	zdc := NewZlibDecompressor()
	defer zc.Close()
	defer zdc.Close()
	testCompressor(zc, zdc, t)
}

func TestGzipCompressor(t *testing.T) {
	gc := NewGzipCompressor()
	defer gc.Close()
	gdc := NewGzipDecompressor()
	defer gdc.Close()
	testCompressor(gc, gdc, t)
}

func TestSnappyCompressor(t *testing.T) {
	sc := NewSnappyCompressor()
	defer sc.Close()
	sdc := NewSnappyDecompressor()
	defer sdc.Close()
	testCompressor(sc, sdc, t)
}

func BenchmarkZlibCompressor(b *testing.B) {
	zc := NewZlibCompressor()
	defer zc.Close()
	zdc := NewZlibDecompressor()
	defer zdc.Close()
	for i := 0; i < b.N; i++ {
		benchmarkCompressor(zc, zdc, b)
	}
}

func BenchmarkGzipCompressor(b *testing.B) {
	gc := NewGzipCompressor()
	defer gc.Close()
	gdc := NewGzipDecompressor()
	defer gdc.Close()
	for i := 0; i < b.N; i++ {
		benchmarkCompressor(gc, gdc, b)
	}
}

func BenchmarkSnappyCompressor(b *testing.B) {
	sc := NewSnappyCompressor()
	defer sc.Close()
	sdc := NewSnappyDecompressor()
	defer sdc.Close()
	for i := 0; i < b.N; i++ {
		benchmarkCompressor(sc, sdc, b)
	}
}
