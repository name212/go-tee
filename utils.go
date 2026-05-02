// Copyright 2026
// license that can be found in the LICENSE file.

package gotee

import (
	"fmt"
	"math/rand"
	"runtime"
	"strings"
	"time"
)

func CalculateStreamName(deep int, name ...string) string {
	if len(name) > 0 {
		return strings.Join(name, " ")
	}

	if deep < 0 {
		deep = 1
	}

	deep++

	randPrefix := rand.NewSource(time.Now().UnixNano()).Int63()

	_, f, line, ok := runtime.Caller(deep)
	if !ok {
		return fmt.Sprintf("%d: unknown", randPrefix)
	}

	return fmt.Sprintf("%d: %s:%d", randPrefix, f, line)
}

func CopyBytes(input []byte) []byte {
	res := make([]byte, len(input))
	copy(res, input)
	return res
}

func appendErr(resErr error, toAppend error) error {
	if toAppend == nil {
		return resErr
	}

	if resErr != nil {
		return fmt.Errorf("%w\n%w", resErr, toAppend)
	}

	return toAppend
}

func concatErrs(first, second error) error {
	return fmt.Errorf("%w: %w", first, second)
}
