// Copyright 2026
// license that can be found in the LICENSE file.

// based on go std library
// src/bufio/scan.go

package scan

import (
	"bufio"
	"fmt"
	"io"
	"unicode/utf8"

	"github.com/name212/gotee/internal"
)

const maxConsecutiveEmptyReads = 100

type TokenHandler interface {
	NewToken(token []byte, isLast bool)
}

var (
	ErrAlreadyDone = fmt.Errorf("[github.com/name212/gotee/internal/Scanner] already done")
)

type NonBlockScanner struct {
	split        bufio.SplitFunc // The function to split the tokens.
	maxTokenSize int             // Maximum size of a token; modified by tests.
	buf          []byte          // Buffer used as argument to split.
	start        int             // First non-processed byte in buf.
	end          int             // End of data in buf.
	empties      int             // Count of successive empty tokens.
	scanCalled   bool            // Scan has been called; buffer is in use.
	done         bool            // Scan has finished.

	handler TokenHandler
}

const (
	// MaxScanTokenSize is the maximum size used to buffer a token
	// unless the user provides an explicit buffer with [Scanner.Buffer].
	// The actual maximum token size may be smaller as the buffer
	// may need to include, for instance, a newline.
	MaxScanTokenSize = 64 * 1024

	startBufSize = 4096 // Size of initial allocation for buffer.
)

func NewNonBlockScanner(handler TokenHandler) *NonBlockScanner {
	if internal.IsNil(handler) {
		panic("TokenHandler is nil")
	}

	return &NonBlockScanner{
		split:        bufio.ScanLines,
		maxTokenSize: MaxScanTokenSize,
		handler:      handler,
	}
}

func (s *NonBlockScanner) Scan(consumed []byte) (bool, error) {
	if s.done {
		return false, ErrAlreadyDone
	}

	s.scanCalled = true

	if len(consumed) == 0 {
		return false, nil
	}

	toWrite := make([]byte, len(consumed))
	copy(toWrite, consumed)

	// Loop until we have input
	for len(toWrite) > 0 {
		if err := s.prepareBuffer(); err != nil {
			return false, err
		}

		toWrite = s.copyRemain(toWrite)

		for s.start < s.end {
			// See if we can get a token with what we already have.
			// If we've run out of data but have an error, give the split function
			// a chance to recover any remaining, possibly empty token.
			advance, token, err := s.split(s.buf[s.start:s.end], false)
			if err != nil {
				if err == bufio.ErrFinalToken {
					s.handler.NewToken(token, true)
					s.done = true
					// move advice to no return unhandled
					// passed token
					if finalAdvance := len(token); finalAdvance > 0 {
						_ = s.advance(finalAdvance)
					}
					return true, nil
				}

				// move for getting unhandled
				if aErr := s.advance(advance); aErr != nil {
					return false, fmt.Errorf("%w with %w", err, aErr)
				}

				return false, err
			}

			if err := s.advance(advance); err != nil {
				return false, err
			}

			if token != nil {
				s.handler.NewToken(token, false)
				s.empties = 0
			}

			if advance == 0 {
				s.empties++
				if s.empties > maxConsecutiveEmptyReads {
					return false, io.ErrNoProgress
				}
				break
			}

			s.empties = 0
		}
	}

	return false, nil
}

func (s *NonBlockScanner) Unhandled() []byte {
	if s.end == s.start || s.start > s.end {
		return nil
	}

	return s.buf[s.start:s.end]
}

func (s *NonBlockScanner) Buffer(buf []byte, max int) {
	if s.scanCalled {
		panic("Buffer called after Scan")
	}
	s.buf = buf[0:cap(buf)]
	s.maxTokenSize = max
}

func (s *NonBlockScanner) Split(split bufio.SplitFunc) {
	if s.scanCalled {
		panic("Split called after Scan")
	}
	s.split = split
}

func (s *NonBlockScanner) Cleanup() {
	s.buf = nil
}

func (s *NonBlockScanner) advance(n int) error {
	if n < 0 {
		return bufio.ErrNegativeAdvance
	}
	if n > s.end-s.start {
		return bufio.ErrAdvanceTooFar
	}
	s.start += n

	return nil
}

func (s *NonBlockScanner) prepareBuffer() error {
	// Must read more data.
	// First, shift data to beginning of buffer if there's lots of empty space
	// or space is needed.
	if s.start > 0 && (s.end == len(s.buf) || s.start > len(s.buf)/2) {
		copy(s.buf, s.buf[s.start:s.end])
		s.end -= s.start
		s.start = 0
	}

	// Is the buffer full? If so, resize.
	if s.end == len(s.buf) {
		// Guarantee no overflow in the multiplication below.
		const maxInt = int(^uint(0) >> 1)
		if len(s.buf) >= s.maxTokenSize || len(s.buf) > maxInt/2 {
			return bufio.ErrTooLong
		}
		newSize := len(s.buf) * 2
		if newSize == 0 {
			newSize = startBufSize
		}
		newSize = min(newSize, s.maxTokenSize)
		newBuf := make([]byte, newSize)
		copy(newBuf, s.buf[s.start:s.end])
		s.buf = newBuf
		s.end -= s.start
		s.start = 0
	}

	return nil
}

func (s *NonBlockScanner) copyRemain(toWrite []byte) []byte {
	lenToWrite := len(toWrite)

	toCopyBytes := len(s.buf) - s.end
	dest := s.buf[s.end:len(s.buf)]
	if toCopyBytes > lenToWrite {
		toCopyBytes = lenToWrite
	}

	copy(dest, toWrite[0:toCopyBytes])
	s.end += toCopyBytes

	if lenToWrite >= toCopyBytes {
		return toWrite[toCopyBytes:]
	}

	return nil
}

// PrivateMaxTokenSize
// using for testing only!
func (s *NonBlockScanner) PrivateMaxTokenSize(n int) {
	if n < utf8.UTFMax || n > 1e9 {
		panic("bad max token size")
	}
	if n < len(s.buf) {
		s.buf = make([]byte, n)
	}
	s.maxTokenSize = n
}
