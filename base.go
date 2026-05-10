// Copyright 2026
// license that can be found in the LICENSE file.

package gotee

import (
	"context"
	"fmt"
	"io"

	"github.com/name212/gotee/internal"
)

const (
	DefaultConsumerBufferedWrites = 100
)

type (
	ConsumersErrors = map[string]error
	BeforeStop      func()
)

type Consumer interface {
	io.WriteCloser
	Name() string
}

type Stream interface {
	Run(ctx context.Context) *Results
	WithBeforeStop(...BeforeStop)
	WritesBufferedCount() int
	Stop()
}

type Results struct {
	ReadErr       error
	ConsumersErrs ConsumersErrors
}

func (r *Results) HasReadError() bool {
	return r.ReadErr != nil
}

func (r *Results) HasConsumersErrors() bool {
	return len(r.ConsumersErrs) > 0
}

func (r *Results) HasLeastOneError() bool {
	if r.HasReadError() {
		return true
	}

	if r.HasConsumersErrors() {
		return true
	}

	return false
}

func (r *Results) GetError() error {
	if !r.HasLeastOneError() {
		return nil
	}

	var res error

	if r.HasReadError() {
		res = fmt.Errorf("read error: %w", r.ReadErr)
	}

	if !r.HasConsumersErrors() {
		return res
	}

	for c, err := range r.ConsumersErrs {
		err = fmt.Errorf("consumer '%s' error: %w", c, err)
		res = internal.AppendErr(res, err)
	}

	return res
}

func (r *Results) Error() string {
	if err := r.GetError(); err != nil {
		return err.Error()
	}

	return ""
}

func newStoppedResults() *Results {
	return &Results{
		ReadErr:       ErrStopped,
		ConsumersErrs: make(ConsumersErrors),
	}
}

func newEmptyResults() *Results {
	return &Results{
		ReadErr:       nil,
		ConsumersErrs: make(ConsumersErrors),
	}
}

type (
	noValT   = struct{}
	stopChan = chan noValT
	outChan  = chan []byte
	errChan  = chan error
)

var (
	noVal = struct{}{}
)
