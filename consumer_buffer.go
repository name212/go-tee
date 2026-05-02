// Copyright 2026
// license that can be found in the LICENSE file.

package gotee

import (
	"bytes"
)

var _ Consumer = &BufferConsumer{}

type BufferConsumer struct {
	*privateBaseConsumer
	buf *bytes.Buffer
}

func NewBufferConsumer(buf *bytes.Buffer, name ...string) *BufferConsumer {
	return &BufferConsumer{
		buf:                 buf,
		privateBaseConsumer: newPrivateBaseConsumer(name...),
	}
}

func NewDefaultBufferConsumer(name ...string) *BufferConsumer {
	return &BufferConsumer{
		buf:                 &bytes.Buffer{},
		privateBaseConsumer: newPrivateBaseConsumer(name...),
	}
}

func (c *BufferConsumer) Buffer() *bytes.Buffer {
	return c.buf
}

func (c *BufferConsumer) Write(p []byte) (int, error) {
	if c.isClosed() {
		return 0, ErrClosed
	}

	return c.buf.Write(p)
}

func (c *BufferConsumer) Close() error {
	c.setClosed()
	return nil
}
