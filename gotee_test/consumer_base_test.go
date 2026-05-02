// Copyright 2026
// license that can be found in the LICENSE file.

package gotee_test

import (
	"testing"

	tee "github.com/name212/gotee"
)

func TestBaseConsumerClose(t *testing.T) {
	c := tee.NewBaseConsumer("name")

	assertClosed(t, c)
}
