// Copyright 2026
// license that can be found in the LICENSE file.

package gotee

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"sync/atomic"

	"github.com/name212/gotee/internal"
)

const (
	DefaultBufSize = 16
)

var (
	ErrStopped = fmt.Errorf("stream was stopped")
	ErrClosed  = fmt.Errorf("already closed")
)

type (
	ConsumersErrors = map[string]error

	noValT   = struct{}
	stopChan = chan noValT
	outChan  = chan []byte
	errChan  = chan error
)

var (
	noVal = struct{}{}
)

type Consumer interface {
	io.WriteCloser
	Name() string
}

type Stream interface {
	Run(ctx context.Context) *Results
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

func (r *Results) HasAnyError() bool {
	if r.HasReadError() {
		return true
	}

	if r.HasConsumersErrors() {
		return true
	}

	return false
}

func (r *Results) GetError() error {
	if !r.HasAnyError() {
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
		res = appendErr(res, err)
	}

	return res
}

func (r *Results) Error() string {
	if err := r.GetError(); err != nil {
		return err.Error()
	}

	return ""
}

func NewStoppedResults() *Results {
	return &Results{
		ReadErr:       ErrStopped,
		ConsumersErrs: make(ConsumersErrors),
	}
}

func NewEmptyResults() *Results {
	return &Results{
		ReadErr:       nil,
		ConsumersErrs: make(ConsumersErrors),
	}
}

type baseStream struct {
	stopped *ClosedFlag
}

func newBaseStream() *baseStream {
	return &baseStream{
		stopped: NewClosedFlag(),
	}
}

func (s *baseStream) isStopped() bool {
	return s.stopped.IsClosed()
}

func (s *baseStream) setStopped() bool {
	return s.stopped.SetClosed()
}

type pipe struct {
	consumer Consumer

	writeCh outChan
	// end channel needs for
	// waiting that all data will write
	endCh stopChan

	closed atomic.Bool

	errMu sync.RWMutex
	err   error
}

func newPipe(consumer Consumer) *pipe {
	return &pipe{
		endCh:    make(stopChan, 2),
		writeCh:  make(outChan, 100),
		consumer: consumer,
	}
}

func (p *pipe) writeToPipe(buf []byte) {
	p.writeCh <- buf
}

func (p *pipe) start() {
	var sendError error

	logger := p.createLogger("WRITE_CYCLE")

	logger.Log("Start write")

	defer func() {
		// first close end channel
		// because if we have send error
		// we need close pipe
		// but we can get write error
		// in normal close operation
		// for not pass additional flag
		// we close channel first
		// for prevent deadlock
		logger.Log("Close endCh")
		close(p.endCh)

		if internal.IsNil(sendError) {
			logger.Log("Send err is nil")
			return
		}

		logger.Log("Send err is: %v", sendError)

		p.addErrorAndClose(
			fmt.Errorf(
				"send to consumer %s: %w",
				p.consumer.Name(),
				sendError,
			),
		)
	}()

	writeAllToConsumer := func(data []byte) error {
		logger.LogBuf(data, -1, "Write buf...")
		for len(data) > 0 {
			n, err := p.consumer.Write(data)
			if err != nil {
				logger.Log("Write buf got err: %v", err)
				return err
			}

			logger.Log("Written bytes %d", n)

			data = data[n:]
		}

		return nil
	}

	for data := range p.writeCh {
		err := writeAllToConsumer(data)

		if internal.IsNil(err) {
			continue
		}

		if !p.checkClosed(err) {
			sendError = err
		}

		return
	}
}

func (p *pipe) checkClosed(err error) bool {
	if errors.Is(err, ErrClosed) {
		return true
	}

	return false
}

func (p *pipe) addErrorAndClose(err error) {
	p.appendErr(err)

	// we can safe call close
	// in situation when we got write error
	// after normal close
	// because we compare and swap closed flag
	// in one operation
	// and if we in close procedure
	// closed flag already set and we
	// return from close function without
	// deadlock
	p.close()
}

func (p *pipe) isClosed() (bool, error) {
	return p.closed.Load(), p.getErr()
}

func (p *pipe) close() {
	logger := p.createLogger("CLOSE")
	needClose := p.closed.CompareAndSwap(false, true)
	if !needClose {
		logger.Log("Already closed")
		return
	}

	logger.Log("Send close to writeCh")
	close(p.writeCh)

	logger.Log("Waiting ending")

	<-p.endCh

	logger.Log("Close consumer")

	if err := p.consumer.Close(); err != nil {
		logger.Log("Close consumer err: %v", err)
		p.appendErr(fmt.Errorf("consumer close error: %w", err))
	}

	logger.Log("Closed!")
}

func (p *pipe) getErr() error {
	p.errMu.RLock()
	defer p.errMu.RUnlock()

	return p.err
}

func (p *pipe) appendErr(err error) {
	p.errMu.Lock()
	defer p.errMu.Unlock()

	p.err = appendErr(p.err, err)
}

func (p *pipe) createLogger(target string) debugLogger {
	return getDebugLogger("INTERNAL_PIPE", target, p.consumer.Name())
}
