package main

import "io"

// creader is a reader that counts how many bytes have been read.
// It was created to be used with gob and groupcache so that we can
// encode results as well as return a byte stream.
type creader struct {
	io.Reader
	count int
}

// Read implements io.Reader.
func (c *creader) Read(buf []byte) (int, error) {
	n, err := c.Reader.Read(buf)
	c.count += n
	return n, err
}

// ReadByte implements io.ByteReader so that gob doesn't buffer.
func (c *creader) ReadByte() (byte, error) {
	var buf [1]byte
	_, err := c.Read(buf[:])
	return buf[0], err
}

// Count returns the number of bytes read.
func (c *creader) Count() int {
	return c.count
}
