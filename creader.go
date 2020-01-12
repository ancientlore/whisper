package main

import "io"

type creader struct {
	io.Reader
	count int
}

func (c *creader) Read(buf []byte) (int, error) {
	n, err := c.Reader.Read(buf)
	c.count += n
	return n, err
}

// implement io.ByteReader so that gob doesn't buffer
func (c *creader) ReadByte() (byte, error) {
	var buf [1]byte
	_, err := c.Read(buf[:])
	return buf[0], err
}

func (c *creader) Count() int {
	return c.count
}
