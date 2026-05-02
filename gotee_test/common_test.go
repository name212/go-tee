// Copyright 2026
// license that can be found in the LICENSE file.

package gotee_test

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	tee "github.com/name212/gotee"
	"github.com/stretchr/testify/require"
)

var (
	_ tee.Consumer    = &testWriteCloserConsumer{}
	_ io.WriteCloser  = &testWriteCloser{}
	_ io.Writer       = &testWriteCloser{}
)

var (
	testsBaseDir = filepath.Join(os.TempDir(), "tests-go-tee")
)

type testWriteCloserConsumer struct {
	*tee.BaseConsumer

	mu       sync.Mutex
	buf      *bytes.Buffer
	writeErr error
	closeErr error
}

func newTestWriteCloserConsumer(name string) *testWriteCloserConsumer {
	return &testWriteCloserConsumer{
		BaseConsumer: tee.NewBaseConsumer(name),
		buf:          &bytes.Buffer{},
	}
}

func (c *testWriteCloserConsumer) Write(p []byte) (int, error) {
	if c.IsClosed() {
		return 0, tee.ErrClosed
	}

	if err := c.getWriteErr(); err != nil {
		return 0, err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	return c.buf.Write(p)
}

func (c *testWriteCloserConsumer) Close() error {
	if c.SetClosed() {
		return nil
	}

	return c.getCloseErr()
}

func (c *testWriteCloserConsumer) content() string {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.buf.String()
}

func (c *testWriteCloserConsumer) getWriteErr() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.writeErr
}

func (c *testWriteCloserConsumer) setWriteErr(err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.writeErr = err
}

func (c *testWriteCloserConsumer) getCloseErr() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.closeErr
}

func (c *testWriteCloserConsumer) setCloseErr(err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.closeErr = err
}

type testWriteCloser struct {
	base *testWriteCloserConsumer
}

func newTestWriteCloser() *testWriteCloser {
	return &testWriteCloser{
		base: newTestWriteCloserConsumer(""),
	}
}

func (c *testWriteCloser) Write(p []byte) (int, error) {
	return c.base.Write(p)
}

func (c *testWriteCloser) Close() error {
	return c.base.Close()
}

func randString(seed string) string {
	n := rand.NewSource(time.Now().UnixNano()).Int63()

	all := fmt.Sprintf("%s%d", seed, n)

	hash := md5.Sum([]byte(all))

	res := fmt.Sprintf("%x", hash)

	return fmt.Sprintf("%.10s", res)
}

func writeScript(t *testing.T, name, content string) string {
	err := os.MkdirAll(testsBaseDir, 0o777)
	require.NoError(t, err, "base tests dir %s should create", testsBaseDir)

	randStr := randString(content)

	fullName := fmt.Sprintf(
		"%s.%s.sh",
		name,
		randStr,
	)

	fullPath := filepath.Join(testsBaseDir, fullName)

	err = os.WriteFile(fullPath, []byte(content), 0o777)
	require.NoError(t, err, "script %s should write to %s", name, fullPath)

	return fullPath
}

func enableDebugLogs(t *testing.T) {
	t.Setenv("GO_TEE_ENABLE_DEBUG_LOG", "true")
	t.Setenv("GO_TEE_DEBUG_LOG_FULL_BUFF", "true")
}
