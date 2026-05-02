// Copyright 2026
// license that can be found in the LICENSE file.

package gotee_test

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
	"testing"

	tee "github.com/name212/gotee"
	"github.com/stretchr/testify/require"
)

const (
	execTestDefaultBufKey = "buf"
)

func TestExec(t *testing.T) {
	t.Setenv("GO_TEE_ENABLE_DEBUG_LOG", "true")
	t.Setenv("GO_TEE_DEBUG_LOG_FULL_BUFF", "true")

	returnOneBufConsumer := func(tst *testExec, name string) []tee.Consumer {
		buf := &bytes.Buffer{}
		tst.consumersData[execTestDefaultBufKey] = buf
		consumer := tee.NewBufferConsumer(buf, name)
		return []tee.Consumer{consumer}
	}

	stdOutOnlyTest := []testExec{
		{
			name: "one buffer consumer",
			stdoutConsumers: func(tst *testExec) []tee.Consumer {
				return returnOneBufConsumer(tst, "tst_one_buf")
			},
			script: scriptOnlyStdOut,
			assert: func(t *testing.T, tst *testExec, results *tee.Results, err error) {
				assertExecResults(t, results)
				assertExecError(t, err, false)
				assertSingleBuffer(t, tst, `First string
Second string
Third string
`)
			},
		},

		{
			name: "multiple consumers",
			stdoutConsumers: func(tst *testExec) []tee.Consumer {
				return returnOneBufConsumer(tst, "tst_one_buf")
			},
			script: scriptOnlyStdOut,
			assert: func(t *testing.T, tst *testExec, results *tee.Results, err error) {
				assertExecResults(t, results)
				assertExecError(t, err, false)
				assertSingleBuffer(t, tst, `First string
Second string
Third string
`)
			},
		},
	}

	t.Run("stout only", func(t *testing.T) {
		for _, tt := range stdOutOnlyTest {
			tt.run(t)
		}
	})
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

func assertSingleBuffer(t *testing.T, tst *testExec, expected ...string) {
	buf := tst.consumersData[execTestDefaultBufKey]
	assertBuffer(t, buf, expected...)
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
