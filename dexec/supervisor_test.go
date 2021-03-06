package dexec_test

import (
	"context"
	"regexp"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"

	exec "github.com/datawire/dlib/dexec"
	"github.com/datawire/dlib/dlog"
)

func TestMustCapture(t *testing.T) {
	result, err := exec.CommandContext(dlog.NewTestContext(t, true), "echo", "this", "is", "a", "test").Output()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if string(result) != "this is a test\n" {
		t.Errorf("unexpected result: %v", result)
	}
}

func TestCaptureError(t *testing.T) {
	_, err := exec.CommandContext(dlog.NewTestContext(t, true), "nosuchcommand").Output()
	if err == nil {
		t.Errorf("expected an error")
	}
}

func TestCaptureExitError(t *testing.T) {
	_, err := exec.CommandContext(dlog.NewTestContext(t, true), "test", "1", "==", "0").Output()
	if err == nil {
		t.Errorf("expected an error")
	}
}

func TestCaptureInput(t *testing.T) {
	cmd := exec.CommandContext(dlog.NewTestContext(t, true), "cat")
	cmd.Stdin = strings.NewReader("hello")
	output, err := cmd.Output()
	if err != nil {
		t.Errorf("unexpected error")
	}
	if string(output) != "hello" {
		t.Errorf("expected hello, got %v", output)
	}
}

func TestCommandRun(t *testing.T) {
	err := exec.CommandContext(dlog.NewTestContext(t, true), "ls").Run()
	if err != nil {
		t.Errorf("unexpted error: %v", err)
	}
}

func TestCommandRunLogging(t *testing.T) {
	logoutput := new(strings.Builder)
	ctx := dlog.WithLogger(context.Background(),
		dlog.WrapLogrus(&logrus.Logger{
			Out: logoutput,
			Formatter: &logrus.TextFormatter{
				DisableTimestamp: true,
				SortingFunc:      dlog.DefaultFieldSort,
			},
			Hooks: make(logrus.LevelHooks),
			Level: logrus.DebugLevel,
		}))

	// The "cat" in the command is important, otherwise the
	// ordering of the "stdin < EOF" and the "stdout+stderr > 1"
	// lines could go either way.
	cmd := exec.CommandContext(ctx, "bash", "-c", "cat; for i in $(seq 1 3); do echo $i; sleep 0.2; done")
	if err := cmd.Run(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	//nolint:lll
	expectedLines := []string{
		`level=info dexec.pid=XXPIDXX msg="started command [\"bash\" \"-c\" \"cat; for i in $(seq 1 3); do echo $i; sleep 0.2; done\"]"`,
		`level=info dexec.pid=XXPIDXX dexec.stream=stdin dexec.err=EOF`,
		`level=info dexec.pid=XXPIDXX dexec.stream=stdout+stderr dexec.data="1\n"`,
		`level=info dexec.pid=XXPIDXX dexec.stream=stdout+stderr dexec.data="2\n"`,
		`level=info dexec.pid=XXPIDXX dexec.stream=stdout+stderr dexec.data="3\n"`,
		`level=info dexec.pid=XXPIDXX msg="finished successfully: exit status 0"`,
		``,
	}
	receivedLines := strings.Split(
		regexp.MustCompile("dexec.pid=[0-9]+").
			ReplaceAllString(logoutput.String(), "dexec.pid=XXPIDXX"),
		"\n")
	if len(receivedLines) != len(expectedLines) {
		t.Log("log output didn't have the correct number of lines:")
		for i, line := range expectedLines {
			t.Logf("expected line %d: %q", i, line)
		}
		for i, line := range receivedLines {
			t.Logf("received line %d: %q", i, line)
		}
		t.FailNow()
	}
	for i, expectedLine := range expectedLines {
		receivedLine := receivedLines[i]
		if receivedLine != expectedLine {
			t.Errorf("log output line %d didn't match expectations:\n"+
				"expected: %q\n"+
				"received: %q\n",
				i, expectedLine, receivedLine)
		}
	}
}
