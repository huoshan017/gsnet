package packet

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/des"
	"fmt"
	"math/rand"
)

type EncryptionType int8

const (
	EncryptionNone EncryptionType = iota
	EncryptionAes  EncryptionType = 1
	EncryptionDes  EncryptionType = 2
	EncryptionMax  EncryptionType = 3
)

func IsValidEncryptionType(typ EncryptionType) bool {
	if typ < EncryptionNone || typ >= EncryptionMax {
		return false
	}
	return true
}

type IEncrypter interface {
	Encrypt([]byte) ([]byte, error)
}

type IDecrypter interface {
	Decrypt([]byte) ([]byte, error)
}

type AesEncrypter struct {
	blockSize          int
	blockModeEncrypter cipher.BlockMode
}

func NewAesEncrypter(key []byte) (*AesEncrypter, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	blockSize := block.BlockSize()
	return &AesEncrypter{
		blockSize:          blockSize,
		blockModeEncrypter: cipher.NewCBCEncrypter(block, key[:blockSize]),
	}, nil
}

func (c *AesEncrypter) Encrypt(data []byte) ([]byte, error) {
	origData := pkcs7Padding(data, c.blockSize)
	cryted := make([]byte, len(origData))
	c.blockModeEncrypter.CryptBlocks(cryted, origData)
	return cryted, nil
}

type AesDecrypter struct {
	blockSize          int
	blockModeDecrypter cipher.BlockMode
}

func NewAesDecrypter(key []byte) (*AesDecrypter, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	blockSize := block.BlockSize()
	return &AesDecrypter{
		blockSize:          blockSize,
		blockModeDecrypter: cipher.NewCBCDecrypter(block, key[:blockSize]),
	}, nil
}

func (c *AesDecrypter) Decrypt(data []byte) ([]byte, error) {
	orig := make([]byte, len(data))
	c.blockModeDecrypter.CryptBlocks(orig, data)
	orig = pkcs7Unpadding(orig)
	return orig, nil
}

func pkcs7Padding(cipherText []byte, blockSize int) []byte {
	padding := blockSize - len(cipherText)%blockSize
	padText := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(cipherText, padText...)
}

func pkcs7Unpadding(origData []byte) []byte {
	length := len(origData)
	unpadding := int(origData[length-1])
	return origData[:(length - unpadding)]
}

type DesEncrypter struct {
	block     cipher.Block
	blockSize int
}

func NewDesEncrypter(key []byte) (*DesEncrypter, error) {
	block, err := des.NewCipher(key)
	if err != nil {
		return nil, err
	}
	blockSize := block.BlockSize()
	return &DesEncrypter{
		block:     block,
		blockSize: blockSize,
	}, nil
}

func (c *DesEncrypter) Encrypt(data []byte) ([]byte, error) {
	data = zeroPadding(data, c.blockSize)
	if len(data)%c.blockSize != 0 {
		return nil, fmt.Errorf("crypto des need a multiple of the blocksize, len data %v, block size %v", len(data), c.blockSize)
	}
	out := make([]byte, len(data))
	dst := out
	for len(data) > 0 {
		c.block.Encrypt(dst, data[:c.blockSize])
		data = data[c.blockSize:]
		dst = dst[c.blockSize:]
	}
	//log.Infof("gsnet: des encrypted get data %v", out)
	return out, nil
}

type DesDecrypter struct {
	block     cipher.Block
	blockSize int
}

func NewDesDecrypter(key []byte) (*DesDecrypter, error) {
	block, err := des.NewCipher(key)
	if err != nil {
		return nil, err
	}
	blockSize := block.BlockSize()
	return &DesDecrypter{
		block:     block,
		blockSize: blockSize,
	}, nil
}

func (c *DesDecrypter) Decrypt(data []byte) ([]byte, error) {
	//log.Infof("gsnet: des to descrypt data %v", data)
	out := make([]byte, len(data))
	dst := out
	if len(data)%c.blockSize != 0 {
		return nil, fmt.Errorf("crypto des input not full blocks, len data %v, block size %v", len(data), c.blockSize)
	}
	for len(data) > 0 {
		c.block.Decrypt(dst, data[:c.blockSize])
		data = data[c.blockSize:]
		dst = dst[c.blockSize:]
	}
	return zeroUnpadding(out), nil
}

func zeroPadding(cipherText []byte, blockSize int) []byte {
	padding := blockSize - len(cipherText)%blockSize
	padText := bytes.Repeat([]byte{0}, padding)
	return append(cipherText, padText...)
}

func zeroUnpadding(origData []byte) []byte {
	return bytes.TrimFunc(origData, func(r rune) bool {
		return r == rune(0)
	})
}

var keyletters = []byte("abcdefghijklmnopqrstuvwxyz01234567890~!@#$%^&*()_+-={}[]|:;'<>?/.,")

func GenAesKey(r *rand.Rand) []byte {
	b := make([]byte, 16)
	for i := range b {
		b[i] = keyletters[r.Intn(len(keyletters))]
	}
	return b
}

func GenDesKey(r *rand.Rand) []byte {
	b := make([]byte, 8)
	for i := range b {
		b[i] = keyletters[r.Intn(len(keyletters))]
	}
	return b
}

func GenCryptoKey(et EncryptionType, r *rand.Rand) []byte {
	if et == EncryptionAes {
		return GenAesKey(r)
	} else if et == EncryptionDes {
		return GenDesKey(r)
	} else {
		return nil
	}
}
