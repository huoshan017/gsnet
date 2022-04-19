package packet

import (
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"io"

	"github.com/golang/snappy"
)

type CompressType int8

const (
	CompressNone   CompressType = iota
	CompressZlib   CompressType = 1
	CompressGzip   CompressType = 2
	CompressSnappy CompressType = 3
	CompressMax    CompressType = 4
)

func IsValidCompressType(typ CompressType) bool {
	if typ < CompressNone || typ >= CompressMax {
		return false
	}
	return true
}

type ICompressor interface {
	Compress([]byte) ([]byte, error)
	Close() error
}

type IDecompressor interface {
	Decompress([]byte) ([]byte, error)
	Close() error
}

type ZlibCompressor struct {
	writer      *zlib.Writer
	writeBuffer bytes.Buffer
	closed      bool
}

func NewZlibCompressor() *ZlibCompressor {
	c := &ZlibCompressor{}
	c.writer = zlib.NewWriter(&c.writeBuffer)
	return c
}

func (c *ZlibCompressor) Compress(data []byte) ([]byte, error) {
	c.writeBuffer.Reset()
	c.writer.Reset(&c.writeBuffer)
	_, err := c.writer.Write(data)
	if err != nil {
		return nil, err
	}
	c.writer.Flush()
	return c.writeBuffer.Bytes(), nil
}

func (c *ZlibCompressor) Close() error {
	if c.closed {
		return nil
	}
	err := c.writer.Close()
	if err == nil {
		c.closed = true
	}
	return err
}

type ZlibDecompressor struct {
}

func NewZlibDecompressor() *ZlibDecompressor {
	return &ZlibDecompressor{}
}

func (c *ZlibDecompressor) Decompress(data []byte) ([]byte, error) {
	b := bytes.NewReader(data)
	reader, err := zlib.NewReader(b)
	if err != nil {
		return nil, err
	}
	var rd bytes.Buffer
	io.Copy(&rd, reader)
	reader.Close()
	return rd.Bytes(), nil
}

func (c *ZlibDecompressor) Close() error {
	return nil
}

type GzipCompressor struct {
	writer      *gzip.Writer
	writeBuffer bytes.Buffer
	closed      bool
}

func NewGzipCompressor() *GzipCompressor {
	c := &GzipCompressor{}
	c.writer = gzip.NewWriter(&c.writeBuffer)
	return c
}

func (c *GzipCompressor) Compress(data []byte) ([]byte, error) {
	c.writeBuffer.Reset()
	c.writer.Reset(&c.writeBuffer)
	_, err := c.writer.Write(data)
	if err != nil {
		return nil, err
	}
	c.writer.Flush()
	return c.writeBuffer.Bytes(), nil
}

func (c *GzipCompressor) Close() error {
	if c.closed {
		return nil
	}
	err := c.writer.Close()
	if err == nil {
		c.closed = true
	}
	return err
}

type GzipDecompressor struct {
}

func NewGzipDecompressor() *GzipDecompressor {
	return &GzipDecompressor{}
}

func (c *GzipDecompressor) Decompress(data []byte) ([]byte, error) {
	b := bytes.NewReader(data)
	reader, err := gzip.NewReader(b)
	if err != nil {
		return nil, err
	}
	var rd bytes.Buffer
	io.Copy(&rd, reader)
	reader.Close()
	return rd.Bytes(), nil
}

func (c *GzipDecompressor) Close() error {
	return nil
}

type SnappyCompressor struct {
}

func NewSnappyCompressor() *SnappyCompressor {
	c := &SnappyCompressor{}
	return c
}

func (c *SnappyCompressor) Compress(data []byte) ([]byte, error) {
	return snappy.Encode(nil, data), nil
}

func (c *SnappyCompressor) Close() error {
	return nil
}

type SnappyDecompressor struct {
}

func NewSnappyDecompressor() *SnappyDecompressor {
	return &SnappyDecompressor{}
}

func (c *SnappyDecompressor) Decompress(data []byte) ([]byte, error) {
	return snappy.Decode(nil, data)
}

func (c *SnappyDecompressor) Close() error {
	return nil
}
