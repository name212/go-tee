// Copyright 2026
// license that can be found in the LICENSE file.

package gotee

import "io"

var (
	_ Consumer = &WriteCloserConsumer{}
	_ Consumer = &WriterConsumer{}
)

type WriteCloserConsumer struct {
	*privateBaseConsumer
	writer io.WriteCloser
}

func NewWriteCloserConsumer(w io.WriteCloser, name ...string) *WriteCloserConsumer {
	return &WriteCloserConsumer{
		writer:              w,
		privateBaseConsumer: newPrivateBaseConsumer(name...),
	}
}

func (c *WriteCloserConsumer) Write(p []byte) (int, error) {
	if c.isClosed() {
		return 0, ErrClosed
	}

	return c.writer.Write(p)
}

func (c *WriteCloserConsumer) Close() error {
	if c.setClosed() {
		return nil
	}

	err := c.writer.Close()

	return err
}

type WriterConsumer struct {
	*privateBaseConsumer
	writer io.Writer
}

func NewWriterConsumer(w io.Writer, name ...string) *WriterConsumer {
	return &WriterConsumer{
		writer:       w,
		privateBaseConsumer: newPrivateBaseConsumer(name...),
	}
}

func (c *WriterConsumer) Write(p []byte) (int, error) {
	if c.isClosed() {
		return 0, ErrClosed
	}

	return c.writer.Write(p)
}

func (c *WriterConsumer) Close() error {
	c.setClosed()

	return nil
}
