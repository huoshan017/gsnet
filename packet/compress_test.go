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

func testCompressor(compressor ICompressor, t *testing.T) {
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
		od, err = compressor.Decompress(d)
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

func TestZlibCompressor(t *testing.T) {
	zc := NewZipCompressor()
	defer zc.Close()
	testCompressor(zc, t)
}

func TestGzipCompressor(t *testing.T) {
	gc := NewGzipCompressor()
	defer gc.Close()
	testCompressor(gc, t)
}

func TestBzipCompressor(t *testing.T) {

}
