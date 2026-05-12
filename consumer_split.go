// Copyright 2026
// license that can be found in the LICENSE file.

package gotee

import (
	"bufio"

	"github.com/name212/gotee/internal"
	"github.com/name212/gotee/scan"
)

var (
	_ Consumer = &SplitConsumer{}
)

type PartsHandler interface {
	Handle(part []byte, last bool, scanErr bool) error
}

type SplitConsumer struct {
	*privateBaseConsumer

	flushed *ClosedFlag

	scanner *scan.NonBlockScanner
	handler *scannerHandler
}

func NewSplitConsumer(split bufio.SplitFunc, handler PartsHandler, name ...string) *SplitConsumer {
	if internal.IsNil(split) {
		split = bufio.ScanLines
	}

	tokenHandler := newScannerHandler(handler)

	scanner := scan.NewNonBlockScanner(tokenHandler)
	scanner.Split(split)

	return &SplitConsumer{
		privateBaseConsumer: newPrivateBaseConsumer(name...),
		scanner:             scanner,
		handler:             tokenHandler,
		flushed:             NewClosedFlag(),
	}
}

func (c *SplitConsumer) Write(p []byte) (int, error) {
	if c.isClosed() {
		return 0, ErrClosed
	}

	receiveLastToken, scanErr := c.scanner.Scan(p)
	if err := c.handler.getErr(); err != nil {
		return 0, err
	}

	hasScanErr := !internal.IsNil(scanErr)
	if receiveLastToken || hasScanErr {
		flushErr := internal.AppendErr(scanErr, c.flush(true, internal.IsNil(scanErr)))
		// not handle error because already flush
		_ = c.Close()
		l := 0
		if !hasScanErr {
			l = len(p)
		}
		return l, internal.AppendErr(flushErr, ErrClosed)
	}

	return len(p), nil
}

func (c *SplitConsumer) flush(last bool, writeErr bool) error {
	if c.flushed.SetClosed() {
		return nil
	}

	unhandled := c.scanner.Unhandled()
	if len(unhandled) > 0 {
		return c.handler.partsHandler.Handle(CopyBytes(unhandled), last, writeErr)
	}

	return nil
}

func (c *SplitConsumer) Close() error {
	if c.setClosed() {
		return nil
	}

	if err := c.flush(false, false); err != nil {
		return err
	}

	return nil
}

type scannerHandler struct {
	partsHandler PartsHandler
	err          error
}

func newScannerHandler(partsHandler PartsHandler) *scannerHandler {
	return &scannerHandler{
		partsHandler: partsHandler,
	}
}

func (h *scannerHandler) NewToken(token []byte, isLast bool) {
	if h.err != nil {
		return
	}

	if err := h.partsHandler.Handle(CopyBytes(token), isLast, false); err != nil {
		h.err = err
	}

}

func (h *scannerHandler) getErr() error {
	return h.err
}
