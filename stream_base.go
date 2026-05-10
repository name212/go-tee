// Copyright 2026
// license that can be found in the LICENSE file.

package gotee

import (
	"fmt"

	"github.com/name212/gotee/internal"
)

const (
	DefaultBufSize = 16
)

var (
	ErrStopped = fmt.Errorf("stream was stopped")
	ErrClosed  = fmt.Errorf("already closed")
)

type baseStream struct {
	stopped    *ClosedFlag
	name       string
	beforeStop []BeforeStop
}

func newBaseStream() *baseStream {
	return &baseStream{
		stopped: NewClosedFlag(),
	}
}

func (s *baseStream) isStopped() bool {
	return s.stopped.IsClosed()
}

func (s *baseStream) WithName(n string) {
	s.name = n
}

func (s *baseStream) GetName() string {
	return s.name
}

func (s *baseStream) WithBeforeStop(bs ...BeforeStop) {
	s.beforeStop = bs
}

func (s *baseStream) setStopped() bool {
	return s.stopped.SetClosed()
}

func (s *baseStream) runBeforeStop(logger internal.Logger) {
	for indx, bs := range s.beforeStop {
		if internal.IsNil(bs) {
			continue
		}

		logger.Log("Run before stop %d", indx)
		bs()
	}
}
