package dexec_test

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"text/template"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	"github.com/datawire/dlib/dexec"
	"github.com/datawire/dlib/dlog"
)

type logWriter struct {
	tb testing.TB
	w  io.Writer
}

func (w *logWriter) Write(p []byte) (int, error) {
	w.tb.Logf("dlog: %s", p)
	return w.w.Write(p)
}

func newCapturingContext(tb testing.TB, w io.Writer) context.Context {
	logger := logrus.New()
	logger.SetOutput(&logWriter{tb, w})
	logger.SetFormatter(&logrus.TextFormatter{
		DisableTimestamp: true,
	})
	ctx := context.Background()
	ctx = dlog.WithLogger(ctx, dlog.WrapLogrus(logger))
	ctx, cancel := context.WithCancel(ctx)
	tb.Cleanup(cancel)
	return ctx
}

func TestOutputErrors(t *testing.T) {
	testcases := map[string]struct {
		InputFunc   func(*dexec.Cmd) ([]byte, error)
		InputStdout io.Writer
		InputStderr io.Writer

		ExpectedStream string
		ExpectedBytes  []byte
		ExpectedErr    string
	}{
		"Output-nil-nil": {
			InputFunc:      (*dexec.Cmd).Output,
			ExpectedStream: "stdout",

			InputStdout:   nil,
			InputStderr:   nil,
			ExpectedBytes: []byte("this is stdout\n"),
			ExpectedErr:   "",
		},
		"Output-set-nil": {
			InputFunc:      (*dexec.Cmd).Output,
			ExpectedStream: "stdout",

			InputStdout:   &strings.Builder{},
			InputStderr:   nil,
			ExpectedBytes: nil,
			ExpectedErr:   "exec: Stdout already set",
		},
		"Output-nil-set": {
			InputFunc:      (*dexec.Cmd).Output,
			ExpectedStream: "stdout",

			InputStdout:   nil,
			InputStderr:   &strings.Builder{},
			ExpectedBytes: []byte("this is stdout\n"),
			ExpectedErr:   "",
		},
		"Output-set-set": {
			InputFunc:      (*dexec.Cmd).Output,
			ExpectedStream: "stdout",

			InputStdout:   &strings.Builder{},
			InputStderr:   &strings.Builder{},
			ExpectedBytes: nil,
			ExpectedErr:   "exec: Stdout already set",
		},

		"CombinedOutput-nil-nil": {
			InputFunc:      (*dexec.Cmd).CombinedOutput,
			ExpectedStream: "stdout+stderr",

			InputStdout:   nil,
			InputStderr:   nil,
			ExpectedBytes: []byte("this is stdout\n"),
			ExpectedErr:   "",
		},
		"CombinedOutput-set-nil": {
			InputFunc:      (*dexec.Cmd).CombinedOutput,
			ExpectedStream: "stdout+stderr",

			InputStdout:   &strings.Builder{},
			InputStderr:   nil,
			ExpectedBytes: nil,
			ExpectedErr:   "exec: Stdout already set",
		},
		"CombinedOutput-nil-set": {
			InputFunc:      (*dexec.Cmd).CombinedOutput,
			ExpectedStream: "stdout+stderr",

			InputStdout:   nil,
			InputStderr:   &strings.Builder{},
			ExpectedBytes: nil,
			ExpectedErr:   "exec: Stderr already set",
		},
		"CombinedOutput-set-set": {
			InputFunc:      (*dexec.Cmd).CombinedOutput,
			ExpectedStream: "stdout+stderr",

			InputStdout:   &strings.Builder{},
			InputStderr:   &strings.Builder{},
			ExpectedBytes: nil,
			ExpectedErr:   "exec: Stdout already set",
		},
	}

	tmpl, err := template.New("expected.log.txt").Parse(`` +
		`level=info msg="started command [\"` + os.Args[0] + `\" \"-test.run=TestLoggingHelperProcess\"]" dexec.pid={{ .PID }}` + "\n" +
		`level=info dexec.err=EOF dexec.pid={{ .PID }} dexec.stream=stdin` + "\n" +
		`level=info dexec.data="this is stdout\n" dexec.pid={{ .PID }} dexec.stream={{ .Stream }}` + "\n" +
		`level=info msg="finished successfully: exit status 0" dexec.pid={{ .PID }}` + "\n" +
		``)
	if err != nil {
		t.Fatal(err)
	}

	for tcName, tcData := range testcases {
		tcData := tcData
		t.Run(tcName, func(t *testing.T) {
			var actualLog strings.Builder
			ctx := newCapturingContext(t, &actualLog)

			cmd := dexec.CommandContext(ctx, os.Args[0], "-test.run=TestLoggingHelperProcess")
			cmd.Env = []string{"GO_WANT_HELPER_PROCESS=1"}
			cmd.Stdout = tcData.InputStdout
			cmd.Stderr = tcData.InputStderr

			actualBytes, actualErr := tcData.InputFunc(cmd)

			assert.Equal(t, tcData.ExpectedBytes, actualBytes)
			if tcData.ExpectedErr == "" {
				assert.NoError(t, actualErr)
				var expectedLog strings.Builder
				err = tmpl.Execute(&expectedLog, map[string]interface{}{
					"PID":    cmd.ProcessState.Pid(),
					"Stream": tcData.ExpectedStream,
				})
				if assert.NoError(t, err) {
					assert.Equal(t, expectedLog.String(), actualLog.String())
				}
			} else {
				assert.EqualError(t, actualErr, tcData.ExpectedErr)
				assert.Equal(t, "", actualLog.String())
			}
		})
	}
}

func TestLogging(t *testing.T) {
	testcases := map[string]struct {
		InputStdout           io.Writer
		InputDisableLogging   bool
		InputDisableIOLogging bool
		ExpectedOutput        string
	}{
		"default": {
			InputStdout: &strings.Builder{},
			ExpectedOutput: `` +
				`level=info msg="started command [\"` + os.Args[0] + `\" \"-test.run=TestLoggingHelperProcess\"]" dexec.pid={{ .PID }}` + "\n" +
				`level=info dexec.err=EOF dexec.pid={{ .PID }} dexec.stream=stdin` + "\n" +
				`level=info dexec.data="this is stdout\n" dexec.pid={{ .PID }} dexec.stream=stdout` + "\n" +
				`level=info msg="finished successfully: exit status 0" dexec.pid={{ .PID }}` + "\n",
		},
		"DisableLogging": {
			InputStdout:         &strings.Builder{},
			InputDisableLogging: true,
			ExpectedOutput:      "",
		},
		"DisableIOLogging": {
			InputStdout:           &strings.Builder{},
			InputDisableIOLogging: true,
			ExpectedOutput: `` +
				`level=info msg="started command [` + quote15(os.Args[0]) + ` \"-test.run=TestLoggingHelperProcess\"]" dexec.pid={{ .PID }}` + "\n" +
				`level=info msg="finished successfully: exit status 0" dexec.pid={{ .PID }}` + "\n",
		},
	}
	for tcName, tcData := range testcases {
		tcData := tcData
		t.Run(tcName, func(t *testing.T) {
			var actualLog strings.Builder
			ctx := newCapturingContext(t, &actualLog)

			cmd := dexec.CommandContext(ctx, os.Args[0], "-test.run=TestLoggingHelperProcess")
			cmd.Env = []string{"GO_WANT_HELPER_PROCESS=1"}
			cmd.Stdout = tcData.InputStdout
			cmd.DisableLogging = tcData.InputDisableLogging
			cmd.DisableIOLogging = tcData.InputDisableIOLogging

			assert.NoError(t, cmd.Run())

			if tmpl, err := template.New("expected.txt").Parse(tcData.ExpectedOutput); assert.NoError(t, err) {
				var expectedLog strings.Builder
				err = tmpl.Execute(&expectedLog, map[string]interface{}{
					"PID": cmd.ProcessState.Pid(),
				})
				if assert.NoError(t, err) {
					assert.Equal(t, expectedLog.String(), actualLog.String())
				}
			}
		})
	}
}

func TestLoggingHelperProcess(*testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	defer os.Exit(0)

	fmt.Println("this is stdout")
}
