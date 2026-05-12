// Copyright 2026
// license that can be found in the LICENSE file.

package gotee

import "sync/atomic"

type ClosedFlag struct {
	closed atomic.Bool
}

func NewClosedFlag() *ClosedFlag {
	return &ClosedFlag{}
}

func (c *ClosedFlag) IsClosed() bool {
	return c.closed.Load()
}

func (c *ClosedFlag) SetClosed() bool {
	shouldClose := c.closed.CompareAndSwap(false, true)
	return !shouldClose
}
