// Copyright 2026
// license that can be found in the LICENSE file.

package gotee

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/name212/gotee/internal"
)

var _ Stream = &TeeStream{}

type TeeStream struct {
	*baseStream

	input     io.Reader
	bufSize   int
	consumers []Consumer

	innerStopCh stopChan

	name string
}

func NewTeeStream(input io.Reader, consumers ...Consumer) (*TeeStream, error) {
	if len(consumers) == 0 {
		return nil, fmt.Errorf("empty consumers list")
	}

	return &TeeStream{
		baseStream:  newBaseStream(),
		input:       input,
		consumers:   append([]Consumer{}, consumers...),
		innerStopCh: make(stopChan, 1),
		bufSize:     DefaultBufSize,
	}, nil
}

func (s *TeeStream) WithBufSize(size int) *TeeStream {
	if size > 0 {
		s.bufSize = size
	}

	return s
}

func (s *TeeStream) WithName(n string) *TeeStream {
	s.name = n

	return s
}

func (s *TeeStream) Run(ctx context.Context) *Results {
	if s.isStopped() {
		return NewStoppedResults()
	}

	stopCh := make(stopChan, 2)
	errCh := make(errChan)
	outCh := make(outChan)

	pipes := make([]*pipe, 0, len(s.consumers))
	for _, c := range s.consumers {
		p := newPipe(c)
		pipes = append(pipes, p)
		go p.start()
	}

	logger := s.createLogger("RUN")
	loggerSendAll := s.createLogger("SEND_ALL")

	sendToAll := func(b []byte) error {
		pipesCount := len(pipes)
		loggerSendAll.LogBuf(b, -1, "Send buf to %d", pipesCount)

		errConsumers := 0
		closedConsumers := 0
		sended := 0

		for _, p := range pipes {
			closed, err := p.isClosed()
			if closed {
				closedConsumers++
				if err != nil {
					errConsumers++
				}
				continue
			}

			sended++
			p.writeToPipe(b)
		}

		loggerSendAll.Log(
			"sends %d: done %d; closed %d, errors: %d",
			pipesCount,
			sended,
			closedConsumers,
			errConsumers,
		)

		if errConsumers == pipesCount {
			return fmt.Errorf("all consumers have errors")
		}

		if closedConsumers == pipesCount {
			return fmt.Errorf("all consumers closed")
		}

		return nil
	}

	logger.Log("Start read")

	go s.startRead(outCh, stopCh, errCh)

	var readErr error
OuterLoop:
	for {
		select {
		case <-ctx.Done():
			logger.Log("Got ctx done")
			if err := ctx.Err(); err != nil {
				readErr = fmt.Errorf("handle context error: %w", err)
			}
			s.Stop()
			break OuterLoop
		case err := <-errCh:
			logger.Log("Got read err: %v", err)
			readErr = err
			break OuterLoop
		case <-stopCh:
			logger.Log("Got stop")
			break OuterLoop
		case buf := <-outCh:
			logger.LogBuf(buf, -1, "Got buf from outCh")
			if err := sendToAll(buf); err != nil {
				readErr = err
				break OuterLoop
			}
		}
	}

	consumersErrs := make(ConsumersErrors)

	for _, p := range pipes {
		closed, err := p.isClosed()
		if !closed {
			logger.Log("Inner pipe for not closed. Close...", p.consumer.Name())
			p.close()
		}

		if err != nil {
			consumersErrs[p.consumer.Name()] = err
		}
	}

	logger.Log("Send stop to read cycle")

	s.Stop()

	r := &Results{
		ReadErr:       readErr,
		ConsumersErrs: consumersErrs,
	}

	if r.HasAnyError() {
		logger.Log("Has any errors")
		return r
	}

	logger.Log("Done!")

	return nil
}

func (s *TeeStream) Stop() {
	if s.setStopped() {
		return
	}

	s.createLogger("STOP").Log("Send stop signal to reader cycle")

	s.innerStopCh <- noVal
}

func (s *TeeStream) startRead(outCh outChan, stopCh stopChan, errCh errChan) {
	buf := make([]byte, s.bufSize)

	sendStop := func() {
		stopCh <- noVal
	}

	logger := s.createLogger("READ_CYCLE")

	for {
		n, err := s.input.Read(buf)
		if n > 0 {
			logger.LogBuf(buf, n, "Receive buf")
			toSend := make([]byte, n)
			copy(toSend, buf[:n])
			outCh <- toSend
		}

		if internal.IsNil(err) {
			if s.isReceiveStop() {
				logger.Log("Got stop signal. Send stop to Run")
				sendStop()
				return
			}

			continue
		}

		if s.isEndRead(err) {
			logger.Log("End read. Send stop to Run")
			sendStop()
		} else {
			logger.Log("End read. Got error: %v", err)
			errCh <- err
		}

		return
	}
}

func (s *TeeStream) isReceiveStop() bool {
	select {
	case <-s.innerStopCh:
		return true
	default:
		return false
	}
}

func (s *TeeStream) isEndRead(err error) bool {
	if errors.Is(err, io.EOF) {
		return true
	}

	if errors.Is(err, io.ErrClosedPipe) {
		return true
	}

	return false
}

func (p *TeeStream) createLogger(target string) debugLogger {
	return getDebugLogger("TEE_STREAM", p.name, target)
}
