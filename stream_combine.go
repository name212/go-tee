// Copyright 2026
// license that can be found in the LICENSE file.

package gotee

import (
	"context"
	"fmt"
	"sync"

	"github.com/name212/gotee/internal"
)

var _ Stream = &CombineStream{}

type CombineStream struct {
	*baseStream
	streams []Stream
}

func NewCombineStream(streams ...Stream) (*CombineStream, error) {
	if len(streams) == 0 {
		return nil, fmt.Errorf("no passed streams to combine stream")
	}

	return &CombineStream{
		baseStream: newBaseStream(),
		streams:    append([]Stream{}, streams...),
	}, nil
}

func (s *CombineStream) Run(ctx context.Context) *Results {
	if s.isStopped() {
		return newStoppedResults()
	}

	streamsCount := len(s.streams)

	results := make([]Results, streamsCount)

	wg := sync.WaitGroup{}
	wg.Add(streamsCount)

	for i, curStream := range s.streams {
		go func(indx int, stream Stream) {
			defer wg.Done()

			res := stream.Run(ctx)

			if res != nil {
				results[indx].ReadErr = res.ReadErr
				results[indx].ConsumersErrs = res.ConsumersErrs
			}

		}(i, curStream)
	}

	wg.Wait()

	s.Stop()

	var resReadErr error
	resConsumersErrors := make(ConsumersErrors)

	for i, res := range results {
		if res.ReadErr != nil {
			resReadErr = internal.AppendErr(resReadErr, fmt.Errorf("stream %d read err: %w", i, res.ReadErr))
		}

		for c, cErr := range res.ConsumersErrs {
			nameForSet := c
			_, ok := resConsumersErrors[c]
			if ok {
				nameForSet = fmt.Sprintf("stream %d consumer %s", i, c)
			}

			resConsumersErrors[nameForSet] = cErr
		}
	}

	r := &Results{
		ReadErr:       resReadErr,
		ConsumersErrs: resConsumersErrors,
	}

	if r.HasLeastOneError() {
		return r
	}

	return nil
}

func (s *CombineStream) Stop() {
	if s.setStopped() {
		return
	}

	logger := s.createLogger("STOP")

	s.runBeforeStop(logger)

	for indx, st := range s.streams {
		logger.Log("Stopping stream %d", indx)
		st.Stop()
	}
}

func (s *CombineStream) createLogger(target string) internal.Logger {
	return internal.GetDebugLogger("COMBINE_STREAM", s.GetName(), target)
}
