// Copyright 2026
// license that can be found in the LICENSE file.

package gotee_test

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"testing"

	tee "github.com/name212/gotee"
	"github.com/stretchr/testify/require"
)

const (
	execTestDefaultBufKey    = "buf"
	execTestDefaultLineKey   = "line"
	execTestDefaultWriterKey = "writer"
)

func TestExec(t *testing.T) {
	suit := newTestExecSuit(t)

	suit.enableDebug(true)
	suit.runOnlyStdOnlyTest()

	var (
		errStdoutOnlyWriter       = fmt.Errorf("errStdoutOnlyWriter")
		errStdoutOnlyWriterSecond = fmt.Errorf("secondErrStdoutOnlyWriter")
	)

	stdOutOnlyTest := []testExec{
		{
			name: "one buffer consumer",
			stdoutConsumers: func(tst *testExec) []tee.Consumer {
				return returnDefaultBufConsumer(tst, "tst_one_buf")
			},
			script: scriptOnlyStdOut,
			assert: func(t *testing.T, tst *testExec, results *tee.Results, err error) {
				assertExecResults(t, results)
				assertExecError(t, err, false)
				assertDefaultBuffer(t, tst, `First string
Second string
Third string
`)
			},
		},

		{
			name: "multiple consumers",
			stdoutConsumers: func(tst *testExec) []tee.Consumer {
				consumers := returnDefaultBufConsumer(tst, "tst_multiple_buf")
				consumers = append(consumers, returnDefaultLineConsumer(tst, "tst_multiple_line")...)
				return append(consumers, returnDefaultWriterConsumer(tst, "tst_multiple_writer")...)
			},
			script: scriptOnlyStdOut,
			assert: func(t *testing.T, tst *testExec, results *tee.Results, err error) {
				assertExecResults(t, results)
				assertExecError(t, err, false)
				bufExpected := `First string
Second string
Third string
`
				assertDefaultBuffer(t, tst, bufExpected)
				assertDefaultLinesHandler(t, tst, []string{
					"First string",
					"Second string",
					"Third string",
				}...)
				assertDefaultWriterConsumer(t, tst, bufExpected)
			},
		},

		{
			name: "multiple consumers stdout with stderr",
			stdoutConsumers: func(tst *testExec) []tee.Consumer {
				consumers := returnDefaultBufConsumer(tst, "tst_both_multiple_buf")
				consumers = append(consumers, returnDefaultLineConsumer(tst, "tst__both_multiple_line")...)
				return append(consumers, returnDefaultWriterConsumer(tst, "tst_both_multiple_writer")...)
			},
			script: scriptStdOutAndErr,
			assert: func(t *testing.T, tst *testExec, results *tee.Results, err error) {
				assertExecResults(t, results)
				assertExecError(t, err, false)
				bufExpected := `First string
Second string
Third string
`
				assertDefaultBuffer(t, tst, bufExpected)
				assertDefaultLinesHandler(t, tst, []string{
					"First string",
					"Second string",
					"Third string",
				}...)
				assertDefaultWriterConsumer(t, tst, bufExpected)
			},
		},

		{
			name: "multiple consumers only stderr",
			stdoutConsumers: func(tst *testExec) []tee.Consumer {
				consumers := returnDefaultBufConsumer(tst, "tst_both_multiple_buf")
				consumers = append(consumers, returnDefaultLineConsumer(tst, "tst_both_multiple_line")...)
				return append(consumers, returnDefaultWriterConsumer(tst, "tst_both_multiple_writer")...)
			},
			script: scriptOnlyStdErr,
			assert: func(t *testing.T, tst *testExec, results *tee.Results, err error) {
				assertExecResults(t, results)
				assertExecError(t, err, false)

				assertDefaultBuffer(t, tst)
				assertDefaultLinesHandler(t, tst)
				assertDefaultWriterConsumer(t, tst)
			},
		},

		{
			name: "multiple consumers with error",
			stdoutConsumers: func(tst *testExec) []tee.Consumer {
				consumers := returnDefaultErrWriterConsumer(tst, "stdout_err_writer", func(b []byte) ([]byte, error) {
					cut := []byte("Second")
					if bytes.Contains(b, cut) {
						t.Logf("Return error %v", errStdoutOnlyWriter)
						return cut, errStdoutOnlyWriter
					}
					return cut, nil
				})

				consumers = append(consumers, returnDefaultLineConsumer(tst, "stdout_err_writer_line_all")...)

				return consumers
			},
			script: scriptStdOutAndErr,
			assert: func(t *testing.T, tst *testExec, results *tee.Results, err error) {
				assertExecResults(t, results, errStdoutOnlyWriter.Error())
				assertExecError(t, err, false)

				assertDefaultWriterConsumer(t, tst, "First string\n")
				assertDefaultLinesHandler(t, tst, []string{
					"First string",
					"Second string",
					"Third string",
				}...)
			},
		},

		{
			name: "multiple consumers with error multiple error sleep",
			stdoutConsumers: func(tst *testExec) []tee.Consumer {
				consumers := returnDefaultErrWriterConsumer(tst, "stdout_err_writer_first", func(b []byte) ([]byte, error) {
					cut := []byte("Second")
					if bytes.Contains(b, cut) {
						t.Logf("Return error for first %v", errStdoutOnlyWriter)
						return cut, errStdoutOnlyWriter
					}
					return cut, nil
				})

				second := newErrWriterConsumer(tst, "stdout_err_writer_second", "second", func(b []byte) ([]byte, error) {
					cut := []byte("string")
					if bytes.Contains(b, cut) {
						t.Logf("Return error for second %v", errStdoutOnlyWriterSecond)
						return cut, errStdoutOnlyWriterSecond
					}
					return cut, nil
				})

				consumers = append(consumers, second)

				consumers = append(consumers, returnDefaultLineConsumer(tst, "stdout_err_writer_line_all_mul")...)

				return consumers
			},
			script: scriptStdOutAndErrWithSleep(3),
			assert: func(t *testing.T, tst *testExec, results *tee.Results, err error) {
				assertExecResults(
					t,
					results,
					errStdoutOnlyWriter.Error(),
					errStdoutOnlyWriterSecond.Error(),
				)
				assertExecError(t, err, false)

				assertDefaultWriterConsumer(t, tst, "First string\n")
				assertDefaultLinesHandler(t, tst, []string{
					"First string",
					"Second string",
					"Third string",
				}...)

				assertWriterConsumer(t, tst.consumersData["second"], "First ")
			},
		},
	}

	t.Run("stout only", func(t *testing.T) {
		for indx, tt := range stdOutOnlyTest {
			if suit.checkStdOnlyTestSkip(t, indx, tt.name) {
				continue
			}

			tt.run(t)
		}
	})
}

type testExecSuit struct {
	mu         sync.Mutex
	hasSkipped bool

	root *testing.T
}

func newTestExecSuit(root *testing.T) *testExecSuit {
	return &testExecSuit{
		root: root,
	}
}

func (s *testExecSuit) setHasSkipped() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.hasSkipped {
		return
	}

	s.root.Cleanup(func() {
		msg := "FAILED"

		if !s.root.Failed() {
			msg = "!! OK But have skipped tests !!"
		}

		s.root.Error(msg)
	})
}

func (s *testExecSuit) enableDebug(enable bool) {
	if enable {
		enableDebugLogs(s.root)
	}
}

func (s *testExecSuit) runOnlyStdOnlyTest(numbers ...int) {
	if len(numbers) == 0 {
		return
	}

	strs := make([]string, 0, len(numbers))
	for _, n := range numbers {
		strs = append(strs, fmt.Sprintf("%d", n))
	}

	s.root.Setenv("RUN_STD_ONLY_TEST", strings.Join(strs, ","))
}

func (s *testExecSuit) checkStdOnlyTestSkip(t *testing.T, indx int, name string) bool {
	runOnlyStdOutOnlyTest := os.Getenv("RUN_STD_ONLY_TEST")
	if runOnlyStdOutOnlyTest == "" {
		return false
	}

	numbersStrs := strings.Split(runOnlyStdOutOnlyTest, ",")

	toRun := make(map[int]struct{})
	for _, s := range numbersStrs {
		s = strings.TrimSpace(s)
		runTst, err := strconv.Atoi(s)
		if err != nil {
			continue
		}

		toRun[runTst] = struct{}{}
	}

	if _, ok := toRun[indx]; !ok {
		t.Logf("!!!!!! Skip std only test %s because run only %v !!!!", name, toRun)
		s.setHasSkipped()
		return true
	}

	return false
}

type testExec struct {
	name            string
	stdoutConsumers func(*testExec) []tee.Consumer
	stderrConsumers func(*testExec) []tee.Consumer
	bufSize         int
	consumersData   map[string]any
	assert          func(*testing.T, *testExec, *tee.Results, error)
	runInGorutine   func(*testing.T, *testExec)
	script          string
	scriptArgs      []string
}

func (tt *testExec) run(t *testing.T) {
	t.Run(tt.name, func(t *testing.T) {
		require.NotEmpty(t, tt.script, "script should passed")
		require.NotNil(t, tt.assert, "assert should passed")

		replaces := []string{
			" ",
			"/",
			"\\",
			":",
			`"`,
			`'`,
			`*`,
			`?`,
			`$`,
			`#`,
		}

		scriptName := tt.name
		for _, toReplace := range replaces {
			scriptName = strings.ReplaceAll(scriptName, toReplace, "_")
		}

		scriptPath := writeScript(t, scriptName, tt.script)

		tt.consumersData = make(map[string]any)

		opts := make([]tee.RunCmdOpt, 0, 3)

		if tt.stdoutConsumers != nil {
			opts = append(opts, tee.RunCmdWithStdout(tt.stdoutConsumers(tt)...))
		}

		if tt.stderrConsumers != nil {
			opts = append(opts, tee.RunCmdWithStderr(tt.stderrConsumers(tt)...))
		}

		if tt.bufSize > 0 {
			opts = append(opts, tee.RunCmdWithBufSize(tt.bufSize))
		}

		cmd := exec.Command(scriptPath, tt.scriptArgs...)
		if tt.runInGorutine != nil {
			go func() {
				tt.runInGorutine(t, tt)
			}()
		}

		results, err := tee.RunCmd(t.Context(), cmd, opts...)

		tt.assert(t, tt, results, err)
	})
}

func newBufConsumer(tst *testExec, name, bufKey string) tee.Consumer {
	buf := &bytes.Buffer{}
	tst.consumersData[bufKey] = buf
	a := 0
	a++
	return tee.NewBufferConsumer(buf, name)
}

func returnDefaultBufConsumer(tst *testExec, name string) []tee.Consumer {
	return []tee.Consumer{newBufConsumer(tst, name, execTestDefaultBufKey)}
}

func newLineConsumer(tst *testExec, name, handlerKey string) tee.Consumer {
	lineHandler := tee.NewStringsSliceLineHandler()
	tst.consumersData[handlerKey] = lineHandler
	return tee.NewLineConsumer(lineHandler, name)
}

func returnDefaultLineConsumer(tst *testExec, name string) []tee.Consumer {
	return []tee.Consumer{newLineConsumer(tst, name, execTestDefaultLineKey)}
}

func newWriterConsumer(tst *testExec, name, writerKey string) tee.Consumer {
	consumer := newTestWriteCloserConsumer(name)
	tst.consumersData[writerKey] = consumer
	return consumer
}

func returnDefaultWriterConsumer(tst *testExec, name string) []tee.Consumer {
	return []tee.Consumer{newWriterConsumer(tst, name, execTestDefaultWriterKey)}
}

func newErrWriterConsumer(tst *testExec, name, key string, checker func([]byte) ([]byte, error)) tee.Consumer {
	consumer := newTestWriteCloserConsumer(name)
	consumer.setWriteErrChecker(checker)
	tst.consumersData[key] = consumer

	return consumer
}

func returnDefaultErrWriterConsumer(tst *testExec, name string, checker func([]byte) ([]byte, error)) []tee.Consumer {
	return []tee.Consumer{newErrWriterConsumer(tst, name, execTestDefaultWriterKey, checker)}
}

func assertDefaultBuffer(t *testing.T, tst *testExec, expected ...string) {
	buf := tst.consumersData[execTestDefaultBufKey]
	assertBuffer(t, buf, expected...)
}

func assertDefaultLinesHandler(t *testing.T, tst *testExec, expectedLines ...string) {
	handler := tst.consumersData[execTestDefaultLineKey]
	assertStringLineHandler(t, handler, expectedLines...)
}

func assertDefaultWriterConsumer(t *testing.T, tst *testExec, expectedLines ...string) {
	handler := tst.consumersData[execTestDefaultWriterKey]
	assertWriterConsumer(t, handler, expectedLines...)
}

func assertWriterConsumer(t *testing.T, rawConsumer any, expected ...string) {
	consumer, ok := rawConsumer.(*testWriteCloserConsumer)
	require.True(t, ok, "should be testWriteCloserConsumer")

	require.True(t, consumer.IsClosed(), "consumer should be closed")

	testBuffer := &bytes.Buffer{}
	testBuffer.WriteString(consumer.content())

	assertBuffer(t, testBuffer, expected...)
}

func assertStringLineHandler(t *testing.T, rawLine any, expectedLines ...string) {
	handler, ok := rawLine.(*tee.StringsSliceLineHandler)
	require.True(t, ok, "should be StringsSliceLineHandler")

	consumedLines := handler.Lines()
	require.Len(t, consumedLines, len(expectedLines), "lines handler should contains all lines")

	for indx, expected := range expectedLines {
		require.Equal(t, expected, consumedLines[indx], "incorrect consumed line %d", indx)
	}
}

func assertBuffer(t *testing.T, rawBuf any, expected ...string) {
	buf, ok := rawBuf.(*bytes.Buffer)
	require.True(t, ok, "should be buffer")

	content := buf.String()

	switch len(expected) {
	case 0:
		require.Empty(t, content, "buffer should not contains any")
		return
	case 1:
		require.Equal(t, expected[0], content, "buffer should equal")
	default:
		for _, e := range expected {
			require.Contains(t, content, e, "should contains buffer")
		}
	}
}

func assertExecResults(t *testing.T, r *tee.Results, contains ...string) {
	if len(contains) > 0 {
		require.NotNil(t, r, "results should not nil")
		errStr := r.Error()
		for _, c := range contains {
			require.Contains(t, errStr, c, "results should contain err")
		}

		return
	}

	if r != nil {
		require.Nil(t, r, "results should be nil got %s", r.Error())
	}

	require.Nil(t, r, "results should be nil")
}

func assertExecError(t *testing.T, err error, shouldBe bool) {
	if shouldBe {
		require.Error(t, err, "exec should have error")
		return
	}

	require.NoError(t, err, "exec should not have error")
}

var (
	scriptOnlyStdOut = `#!/usr/bin/env bash
echo "First string"
echo "Second string"
echo "Third string"
`

	scriptOnlyStdErr = `#!/usr/bin/env bash
echo "First err" >&2
echo "Second err" >&2
echo "Third err" >&2
`

	scriptStdOutAndErr = `#!/usr/bin/env bash
echo "First string"
echo "First err" >&2
echo "Second string"
echo "Second err" >&2
echo "Third string"
echo "Third err" >&2
`
	scriptStdOutAndErrWithErrExit = `#!/usr/bin/env bash
echo "First string"
echo "First err" >&2
echo "Second string"
echo "Second err" >&2
exit 1
`
)

func scriptStdOutAndErrWithSleep(seconds int) string {
	return fmt.Sprintf(`#!/usr/bin/env bash
echo "First string"
echo "First err" >&2
echo "Second string"
sleep %d
echo "Second err" >&2
echo "Third string"
echo "Third err" >&2
`, seconds)
}

func scriptStdOutAndErrWithSleepInEnd(seconds int) string {
	return fmt.Sprintf(`#!/usr/bin/env bash
echo "First string"
echo "First err" >&2
echo "Second string"
echo "Second err" >&2
echo "Third string"
echo "Third err" >&2
sleep %d
`, seconds)
}
