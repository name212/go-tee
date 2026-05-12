// Copyright 2026
// license that can be found in the LICENSE file.

package gotee

import (
	"bufio"
	"sync"
)

var (
	_ PartsHandler = &linePartsWrapper{}
	_ LineHandler  = &FuncLineHandler{}
	_ LineHandler  = &StringsSliceLineHandler{}
)

type (
	FuncStr      func(string) error
	FuncStrNoErr func(string)
)

type LineHandler interface {
	Handle(l string) error
}

func NewLineConsumer(handler LineHandler, name ...string) *SplitConsumer {
	nameForSet := []string{ConsumerName(1, name...)}
	return newLineConsumer(handler, nameForSet...)
}

func NewCustomLineConsumer(partsHandler PartsHandler, name ...string) *SplitConsumer {
	nameForSet := []string{ConsumerName(1, name...)}
	return NewSplitConsumer(bufio.ScanLines, partsHandler, nameForSet...)
}

func NewFuncLineConsumer(handler FuncStr, name ...string) *SplitConsumer {
	nameForSet := []string{ConsumerName(1, name...)}
	return newLineConsumer(NewFuncLineHandler(handler), nameForSet...)
}

func NewFuncNoErrLineConsumer(handler FuncStrNoErr, name ...string) *SplitConsumer {
	nameForSet := []string{ConsumerName(1, name...)}
	return newLineConsumer(NewFuncNoErrLineHandler(handler), nameForSet...)
}

type StringsSliceLineHandler struct {
	mu    sync.Mutex
	lines []string
}

func NewStringsSliceLineHandler(capacity ...int) *StringsSliceLineHandler {
	resCap := 16
	if len(capacity) > 0 {
		resCap = capacity[0]
	}

	return &StringsSliceLineHandler{
		lines: make([]string, 0, resCap),
	}
}

func (h *StringsSliceLineHandler) Handle(l string) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.lines = append(h.lines, l)

	return nil
}

func (h *StringsSliceLineHandler) Lines() []string {
	h.mu.Lock()
	defer h.mu.Unlock()

	res := make([]string, len(h.lines))
	copy(res, h.lines)

	return res
}

type FuncLineHandler struct {
	handler FuncStr
}

func NewFuncLineHandler(handler FuncStr) *FuncLineHandler {
	return &FuncLineHandler{
		handler: handler,
	}
}

func NewFuncNoErrLineHandler(handler FuncStrNoErr) *FuncLineHandler {
	return &FuncLineHandler{
		handler: func(s string) error {
			handler(s)
			return nil
		},
	}
}

func (l *FuncLineHandler) Handle(s string) error {
	return l.handler(s)
}

func newLineConsumer(handler LineHandler, name ...string) *SplitConsumer {
	wrapper := newLinePartsWrapper(handler)
	return NewSplitConsumer(bufio.ScanLines, wrapper, name...)
}

type linePartsWrapper struct {
	handler LineHandler
}

func newLinePartsWrapper(handler LineHandler) *linePartsWrapper {
	return &linePartsWrapper{
		handler: handler,
	}
}

func (h *linePartsWrapper) Handle(part []byte, _ bool, _ bool) error {
	return h.handler.Handle(string(part))
}
