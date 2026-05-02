// Copyright 2026
// license that can be found in the LICENSE file.

package gotee

var _ Consumer = &FuncConsumer{}

type (
	Func      func([]byte) error
	FuncNoErr func([]byte)
)

type FuncConsumer struct {
	*privateBaseConsumer
	handler Func
}

func NewFuncConsumer(h Func, name ...string) *FuncConsumer {
	return &FuncConsumer{
		handler:             h,
		privateBaseConsumer: newPrivateBaseConsumer(name...),
	}
}

func NewFuncNoErrConsumer(h FuncNoErr, name ...string) *FuncConsumer {
	nameForSet := CalculateStreamName(1, name...)
	return NewFuncConsumer(
		func(b []byte) error {
			h(b)
			return nil
		},
		nameForSet,
	)
}

func (c *FuncConsumer) Write(p []byte) (int, error) {
	if c.isClosed() {
		return 0, ErrClosed
	}

	if err := c.handler(p); err != nil {
		return 0, err
	}

	return len(p), nil
}

func (c *FuncConsumer) Close() error {
	c.setClosed()

	return nil
}
