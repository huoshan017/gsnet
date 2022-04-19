package packet

import (
	"bytes"
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"
)

var (
	r      = rand.New(rand.NewSource(time.Now().UnixNano()))
	aesKey = GenAesKey(r)
	desKey = GenDesKey(r)
)

func testCrypto(encrypter IEncrypter, decrypter IDecrypter, t *testing.T, b *testing.B) {
	bs := randBytes(50)

	ds, err := encrypter.Encrypt(bs)
	if err != nil {
		if t != nil {
			t.Errorf("encrypt err: %v", err)
		}
		if b != nil {
			b.Errorf("encrypt err: %v", err)
		}
	}

	var os []byte
	os, err = decrypter.Decrypt(ds)
	if err != nil {
		if t != nil {
			t.Errorf("decrypt err: %v", err)
		}
		if b != nil {
			b.Errorf("decrypt err: %v", err)
		}
	}

	if !bytes.Equal(bs, os) {
		panic(fmt.Errorf(""))
	}
}

func testCryptoMultiGoroutine(encrypter IEncrypter, decrypter IDecrypter, t *testing.T) {
	type data struct {
		os []byte
		cs []byte
	}
	ch := make(chan data, 10)

	var (
		wg  sync.WaitGroup
		num int = 5000000
	)

	wg.Add(num)
	go func() {
		for i := 0; i < num; i++ {
			origStr := randBytes(50)
			crypStr, err := encrypter.Encrypt(origStr)
			if err != nil {
				if t != nil {
					t.Errorf("encrypt err: %v", err)
				}
			}
			ch <- data{os: origStr, cs: crypStr}
			wg.Done()
			//time.Sleep(time.Microsecond * 10)
		}
	}()

	go func() {
		for d := range ch {
			var (
				bs  []byte
				err error
			)
			bs, err = decrypter.Decrypt(d.cs)
			if err != nil {
				if t != nil {
					t.Errorf("decrypt err: %v", err)
				}
			}

			if !bytes.Equal(d.os, bs) {
				panic(fmt.Errorf(""))
			}
		}
	}()

	wg.Wait()

	t.Logf("test crypto with multi goroutine done")
}

func TestAesCrypto(t *testing.T) {
	ae, err := NewAesEncrypter(aesKey)
	if err != nil {
		t.Errorf("create aes encrypter err %v", err)
		return
	}
	ad, err := NewAesDecrypter(aesKey)
	if err != nil {
		t.Errorf("create aes decrypter err %v", err)
		return
	}
	for i := 0; i < 10000; i++ {
		testCrypto(ae, ad, t, nil)
	}
}

func TestAesCryptoMultiGoroutine(t *testing.T) {
	ae, err := NewAesEncrypter(aesKey)
	if err != nil {
		t.Errorf("create aes encrypter err %v", err)
		return
	}
	ad, err := NewAesDecrypter(aesKey)
	if err != nil {
		t.Errorf("create aes decrypter err %v", err)
		return
	}
	testCryptoMultiGoroutine(ae, ad, t)
}

func BenchmarkAesCrypto(b *testing.B) {
	ae, err := NewAesEncrypter(aesKey)
	if err != nil {
		b.Errorf("create aes encrypter err %v", err)
		return
	}
	ad, err := NewAesDecrypter(aesKey)
	if err != nil {
		b.Errorf("create aes decrypter err %v", err)
		return
	}
	for i := 0; i < b.N; i++ {
		testCrypto(ae, ad, nil, b)
	}
}

func TestDesCrypto(t *testing.T) {
	de, err := NewDesEncrypter(desKey)
	if err != nil {
		t.Errorf("create des encrypter err %v", err)
		return
	}
	dd, err := NewDesDecrypter(desKey)
	if err != nil {
		t.Errorf("create des decrypter err %v", err)
		return
	}
	for i := 0; i < 10000; i++ {
		testCrypto(de, dd, t, nil)
	}
}

func TestDesCryptoMultiGoroutine(t *testing.T) {
	de, err := NewDesEncrypter(desKey)
	if err != nil {
		t.Errorf("create des encrypter err %v", err)
		return
	}
	dd, err := NewDesDecrypter(desKey)
	if err != nil {
		t.Errorf("create des decrypter err %v", err)
		return
	}
	testCryptoMultiGoroutine(de, dd, t)
}

func BenchmarkDesCrypto(b *testing.B) {
	de, err := NewDesEncrypter(desKey)
	if err != nil {
		b.Errorf("create des encrypter err %v", err)
		return
	}
	dd, err := NewDesDecrypter(desKey)
	if err != nil {
		b.Errorf("create des decrypter err %v", err)
	}
	for i := 0; i < b.N; i++ {
		testCrypto(de, dd, nil, b)
	}
}
