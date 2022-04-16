package packet

import (
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"io"
)

type CompressType int8

const (
	CompressNone CompressType = iota
	CompressZip  CompressType = 1
	CompressGzip CompressType = 2
)

type ICompressor interface {
	Compress([]byte) ([]byte, error)
	Decompress([]byte) ([]byte, error)
	Close() error
}

type ZipCompressor struct {
	writer      *zlib.Writer
	writeBuffer bytes.Buffer
	closed      bool
}

func NewZipCompressor() *ZipCompressor {
	c := &ZipCompressor{}
	c.writer = zlib.NewWriter(&c.writeBuffer)
	return c
}

func (c *ZipCompressor) Compress(data []byte) ([]byte, error) {
	c.writeBuffer.Reset()
	c.writer.Reset(&c.writeBuffer)
	_, err := c.writer.Write(data)
	if err != nil {
		return nil, err
	}
	c.writer.Flush()
	return c.writeBuffer.Bytes(), nil
}

func (c *ZipCompressor) Decompress(data []byte) ([]byte, error) {
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

func (c *ZipCompressor) Close() error {
	if c.closed {
		return nil
	}
	err := c.writer.Close()
	if err == nil {
		c.closed = true
	}
	return err
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

func (c *GzipCompressor) Decompress(data []byte) ([]byte, error) {
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