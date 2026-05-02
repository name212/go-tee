// Copyright 2026
// license that can be found in the LICENSE file.

package gotee

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
)

type Cleaner func() error

type Piper interface {
	StdoutPipe() (io.ReadCloser, error)
	StderrPipe() (io.ReadCloser, error)
}

type Command interface {
	Piper
	fmt.Stringer
	Start() error
	Wait() error
}

var (
	ErrCleanAfterRun         = fmt.Errorf("cannot clean after run cmd")
	ErrRunCmd                = fmt.Errorf("cannot run cmd")
	ErrCreateStreamBeforeRun = fmt.Errorf("cannot create stream before run cmd")
)

type (
	RunCmdOpts struct {
		stdoutConsumers []Consumer
		stderrConsumers []Consumer
		bufSize         int
		name            string
	}

	RunCmdOpt func(*RunCmdOpts)
)

func RunCmdWithStdout(consumers ...Consumer) RunCmdOpt {
	return func(o *RunCmdOpts) {
		if len(consumers) == 0 {
			return
		}

		o.stdoutConsumers = append(o.stdoutConsumers, consumers...)
	}
}

func RunCmdWithStderr(consumers ...Consumer) RunCmdOpt {
	return func(o *RunCmdOpts) {
		if len(consumers) == 0 {
			return
		}

		o.stderrConsumers = append(o.stderrConsumers, consumers...)
	}
}

func RunCmdWithBufSize(size int) RunCmdOpt {
	return func(o *RunCmdOpts) {
		if size > 0 {
			o.bufSize = size
		}
	}
}

func RunCmdWithName(name string) RunCmdOpt {
	return func(o *RunCmdOpts) {
		o.name = name
	}
}

func RunCmd(ctx context.Context, cmd Command, opts ...RunCmdOpt) (*Results, error) {
	returnErr := func(err error) (*Results, error) {
		return NewEmptyResults(), err
	}

	cloneOpts := make([]RunCmdOpt, len(opts))
	copy(cloneOpts, opts)

	cloneOpts = append(cloneOpts, RunCmdWithName(cmd.String()))

	stream, cleaner, err := NewStreamForCmd(cmd, cloneOpts...)
	if err != nil {
		return returnErr(concatErrs(ErrCreateStreamBeforeRun, err))
	}

	resCh := make(chan *Results, 1)

	go func() {
		res := stream.Run(ctx)
		resCh <- res
	}()

	cleanupAndReturnErr := func(err error) (*Results, error) {
		stream.Stop()

		cleanErr := cleaner()
		if cleanErr != nil {
			err = appendErr(err, concatErrs(ErrCleanAfterRun, cleanErr))
		}

		return NewEmptyResults(), err
	}

	if err := cmd.Start(); err != nil {
		return cleanupAndReturnErr(fmt.Errorf("%w cannot start: %w", ErrRunCmd, err))
	}

	if err := cmd.Wait(); err != nil {
		return cleanupAndReturnErr(fmt.Errorf("%w cannot wait: %w", ErrRunCmd, err))
	}

	stream.Stop()
	results := <-resCh

	close(resCh)

	if err := cleaner(); err != nil {
		return results, concatErrs(ErrCleanAfterRun, err)
	}

	return results, nil
}

func NewStreamForCmd(cmd Piper, opts ...RunCmdOpt) (*CombineStream, Cleaner, error) {
	createErr := func(f string, args ...any) (*CombineStream, Cleaner, error) {
		return nil, noCleaner, fmt.Errorf(f, args...)
	}

	optsToSet := &RunCmdOpts{
		bufSize: DefaultBufSize,
	}

	for _, o := range opts {
		o(optsToSet)
	}

	stdoutConsumers := optsToSet.stdoutConsumers
	stderrConsumers := optsToSet.stderrConsumers

	if len(stdoutConsumers) == 0 && len(stderrConsumers) == 0 {
		return createErr("stdout and/or sterr consumers not passed")
	}

	streams := make([]Stream, 0, 2)
	closers := make(map[string]io.Closer)

	createTeeStream := func(r io.Reader, consumers []Consumer, name string) (*TeeStream, error) {
		st, err := NewTeeStream(r, consumers...)
		if err != nil {
			return nil, err
		}

		if optsToSet.bufSize > 0 {
			st.WithBufSize(optsToSet.bufSize)
		}

		streamName := fmt.Sprintf("%s:%s", optsToSet.name, name)
		return st.WithName(streamName), nil
	}

	if len(stdoutConsumers) > 0 {
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			return createErr("create stdout pipe failed: %w", err)
		}

		const stdoutName = "stdout"

		st, err := createTeeStream(stdout, stdoutConsumers, stdoutName)
		if err != nil {
			return createErr("cannot create TeeStream for stdout: %w", err)
		}

		streams = append(streams, st)
		closers[stdoutName] = stdout
	}

	if len(stderrConsumers) > 0 {
		stderr, err := cmd.StderrPipe()
		if err != nil {
			return createErr("stderr pipe failed: %w", err)
		}

		const stderrName = "stderr"

		st, err := createTeeStream(stderr, stderrConsumers, stderrName)
		if err != nil {
			return createErr("cannot create TeeStream for stderr: %w", err)
		}

		streams = append(streams, st)
		closers[stderrName] = stderr
	}

	combine, err := NewCombineStream(streams...)
	if err != nil {
		return createErr("cannot create combine stream: %w", err)
	}

	cleaner := func() error {
		var resErr error

		for name, cl := range closers {
			if err := cl.Close(); err != nil {
				if !cmdPipeClosed(err) {
					resErr = appendErr(resErr, fmt.Errorf("cannot close pipe for %s: %w", name, err))
				}
			}
		}

		return resErr
	}

	return combine, cleaner, nil
}

func cmdPipeClosed(err error) bool {
	if errors.Is(err, os.ErrClosed) {
		return true
	}

	return false
}

func noCleaner() error {
	return nil
}
