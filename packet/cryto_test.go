package packet

import (
	"bytes"
	"fmt"
	"testing"
)

var (
	aesKey = GenAesKey()
	desKey = GenDesKey()
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
	for i := 0; i < 100; i++ {
		testCrypto(ae, ad, t, nil)
	}
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
	for i := 0; i < 100; i++ {
		testCrypto(de, dd, t, nil)
	}
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
