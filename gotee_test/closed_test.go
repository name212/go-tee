// Copyright 2026
// license that can be found in the LICENSE file.

package gotee_test

import (
	"testing"

	tee "github.com/name212/gotee"
	"github.com/stretchr/testify/require"
)

func TestClosedFlag(t *testing.T) {
	assertClosed(t, tee.NewClosedFlag())
}

func assertClosed(t *testing.T, c closed) {
	beforeClose := c.IsClosed()
	require.False(t, beforeClose, "should not closed before first SetClose")

	firstStop := c.SetClosed()
	require.False(t, firstStop, "first SetClosed should return that should close")

	afterClose := c.IsClosed()
	require.True(t, afterClose, "should return closed after first SetClose")

	secondStop := c.SetClosed()
	require.True(t, secondStop, "second SetClosed should return that should not close")

	afterSecondClose := c.IsClosed()
	require.True(t, afterSecondClose, "should return closed after second SetClose")
}

type closed interface {
	IsClosed() bool
	SetClosed() bool
}
