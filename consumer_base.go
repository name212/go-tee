// Copyright 2026
// license that can be found in the LICENSE file.

package gotee

type BaseConsumer struct {
	name   string
	closed *ClosedFlag
}

func NewBaseConsumer(n string) *BaseConsumer {
	return &BaseConsumer{
		closed: NewClosedFlag(),
		name: n,
	}
}

func (c *BaseConsumer) Name() string {
	return c.name
}

func (c *BaseConsumer) IsClosed() bool {
	return c.closed.IsClosed()
}

func (c *BaseConsumer) SetClosed() bool {
	return c.closed.SetClosed()
}

type privateBaseConsumer struct {
	base *BaseConsumer
}

func newPrivateBaseConsumer(name ...string) *privateBaseConsumer {
	nameForSet := CalculateStreamName(2, name...)
	return &privateBaseConsumer{
		base: NewBaseConsumer(nameForSet),
	}
}

func (c *privateBaseConsumer) Name() string {
	return c.base.Name()
}

func (c *privateBaseConsumer) isClosed() bool {
	return c.base.IsClosed()
}

func (c *privateBaseConsumer) setClosed() bool {
	return c.base.SetClosed()
}
