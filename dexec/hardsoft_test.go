package dexec_test

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/datawire/dlib/dcontext"
	"github.com/datawire/dlib/dexec"
)

type lineBuffer struct {
	partial []byte
	lines   chan string
}

func (b *lineBuffer) Write(p []byte) (int, error) {
	n := len(p)
	b.partial = append(b.partial, p...)
	for {
		nl := bytes.IndexByte(b.partial, '\n')
		if nl < 0 {
			break
		}
		line := b.partial[:nl+1]
		b.partial = b.partial[nl+1:]
		b.lines <- string(line)
	}
	return n, nil
}

var errKilled = "signal: killed"

func init() {
	if runtime.GOOS == "windows" {
		errKilled = "exit status 1"
	}
}

// newInterruptableSysProcAttr gets overridden in hardsoft_windows_test.go
var newInterruptableSysProcAttr = func() *syscall.SysProcAttr { return nil }

func TestSoftCancel(t *testing.T) {
	log := &strings.Builder{}
	ctx := newCapturingContext(t, log)
	ctx, hardCancel := context.WithCancel(ctx)
	defer hardCancel()
	ctx = dcontext.WithSoftness(ctx)
	ctx, softCancel := context.WithCancel(ctx)

	output := &lineBuffer{
		lines: make(chan string, 50),
	}
	cmd := dexec.CommandContext(ctx, os.Args[0], "-test.run=TestSoftHelperProcess")
	cmd.SysProcAttr = newInterruptableSysProcAttr()
	cmd.Env = []string{"GO_WANT_HELPER_PROCESS=1"}
	cmd.Stdout = output
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}

	// give it a chance to set up the signal handler
	line := <-output.lines
	if line != "started\n" {
		t.Fatalf("didn't get expected output: %q", line)
	}

	// send SIGINT
	softCancel()
	line = <-output.lines
	if line != "caught signal: interrupt\n" {
		t.Logf("didn't get expected output: %q", line)
	}

	// send SIGKILL
	hardCancel()
	err := cmd.Wait()
	if err == nil {
		t.Fatal("expected to get an error from Wait()")
	}
	if _, ok := err.(*dexec.ExitError); !ok {
		t.Errorf("error is of the wrong type: %[1]T(%[1]v)", err)
	}
	if err.Error() != errKilled {
		t.Errorf("unexpected error value: %v", err)
	}

	assert.Equal(t, fmt.Sprintf(``+
		`level=info msg="started command [%[1]s \"-test.run=TestSoftHelperProcess\"]" dexec.pid=%[2]d`+"\n"+
		`level=info dexec.err=EOF dexec.pid=%[2]d dexec.stream=stdin`+"\n"+
		`level=info dexec.data="started\n" dexec.pid=%[2]d dexec.stream=stdout`+"\n"+
		`level=info msg="sending SIGINT"`+"\n"+
		`level=info dexec.data="caught signal: interrupt\n" dexec.pid=%[2]d dexec.stream=stdout`+"\n"+
		`level=info msg="sending SIGKILL"`+"\n"+
		`level=info msg="finished with error: `+errKilled+`" dexec.pid=%[2]d`+"\n"+
		``, quote15(os.Args[0]), cmd.ProcessState.Pid()),
		log.String())
}

func TestSoftCancelUnsupported(t *testing.T) {
	ctx := dcontext.WithSoftness(context.Background())

	cmd := dexec.CommandContext(ctx, os.Args[0], "-test.run=TestHelperProcess", "--", "echo", "foo")
	cmd.Env = []string{"GO_WANT_HELPER_PROCESS=1"}
	err := cmd.Start()
	switch runtime.GOOS {
	case "windows":
		if err == nil {
			t.Error("expected an error from cmd.Start() but did not get one")
		} else if err.Error() != "dexec.Cmd.Start: on GOOS=windows it is an error to use soft cancellation without CREATE_NEW_PROCESS_GROUP" {
			t.Errorf("unexpected error value: %v", err)
		}
	default:
		if err != nil {
			t.Fatal(err)
		}
	}

	if err == nil {
		if err := cmd.Wait(); err != nil {
			t.Fatal(err)
		}
	}
}

func TestHardCancel(t *testing.T) {
	log := &strings.Builder{}
	ctx := newCapturingContext(t, log)
	ctx, hardCancel := context.WithCancel(ctx)
	defer hardCancel()
	ctx = dcontext.WithSoftness(ctx)

	output := &lineBuffer{
		lines: make(chan string, 50),
	}
	cmd := dexec.CommandContext(ctx, os.Args[0], "-test.run=TestSoftHelperProcess")
	cmd.SysProcAttr = newInterruptableSysProcAttr()
	cmd.Env = []string{"GO_WANT_HELPER_PROCESS=1"}
	cmd.Stdout = output
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}

	// give it a chance to set up the signal handler
	line := <-output.lines
	if line != "started\n" {
		t.Fatalf("didn't get expected output: %q", line)
	}

	// send SIGKILL
	hardCancel()
	err := cmd.Wait()
	if err == nil {
		t.Fatal("expected to get an error from Wait()")
	}
	if _, ok := err.(*dexec.ExitError); !ok {
		t.Errorf("error is of the wrong type: %[1]T(%[1]v)", err)
	}
	if err.Error() != errKilled {
		t.Errorf("unexpected error value: %v", err)
	}

	assert.Equal(t, fmt.Sprintf(``+
		`level=info msg="started command [%[1]s \"-test.run=TestSoftHelperProcess\"]" dexec.pid=%[2]d`+"\n"+
		`level=info dexec.err=EOF dexec.pid=%[2]d dexec.stream=stdin`+"\n"+
		`level=info dexec.data="started\n" dexec.pid=%[2]d dexec.stream=stdout`+"\n"+
		`level=info msg="sending SIGKILL"`+"\n"+
		`level=info msg="finished with error: `+errKilled+`" dexec.pid=%[2]d`+"\n"+
		``, quote15(os.Args[0]), cmd.ProcessState.Pid()),
		log.String())
}

func TestSoftHelperProcess(*testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	defer os.Exit(0)

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	fmt.Println("started")

	for sig := range sigs {
		fmt.Println("caught signal:", sig)
	}
}

func TestSoftCancelCantStart(t *testing.T) {
	log := &strings.Builder{}
	ctx := newCapturingContext(t, log)
	ctx, hardCancel := context.WithCancel(ctx)
	defer hardCancel()
	ctx = dcontext.WithSoftness(ctx)
	ctx, softCancel := context.WithCancel(ctx)
	defer softCancel()

	cmd := dexec.CommandContext(ctx, "/")
	cmd.SysProcAttr = newInterruptableSysProcAttr()
	err := cmd.Start()
	if err == nil {
		t.Fatal("expected to get an error from Start()")
	}
	expErr := "permission denied"
	if runtime.GOOS == "windows" {
		expErr = "file does not exist"
	}
	if err.Error() != `exec: "/": `+expErr {
		t.Errorf("unexpected error value: %v", err)
	}

	// The main thing that this test is checking for is ensuring that that the cancel handler
	// doesn't get set up if Start() fails.  So we're going to try to trigger it, which if it
	// did get set up will cause a panic.

	softCancel()                // Trigger a soft cancel
	time.Sleep(1 * time.Second) // Give the cancel handler a chance to run

	hardCancel()                // Trigger a hard cancel
	time.Sleep(1 * time.Second) // Give the cancel handler a chance to run

	// We didn't panic, so the test passes.
	assert.Equal(t, "", log.String())
}

func TestSoftCancelDeadContext(t *testing.T) {
	log := &strings.Builder{}
	ctx := newCapturingContext(t, log)
	ctx = dcontext.WithSoftness(ctx)
	ctx, softCancel := context.WithCancel(ctx)
	softCancel()

	cmd := dexec.CommandContext(ctx, os.Args[0])
	cmd.SysProcAttr = newInterruptableSysProcAttr()
	err := cmd.Start()
	if err == nil {
		t.Fatal("expected to get an error from Start()")
	}
	if err.Error() != `context canceled` {
		t.Errorf("unexpected error value: %v", err)
	}

	assert.Equal(t, "", log.String())
}

func TestSoftCancelSelfExit(t *testing.T) {
	log := &strings.Builder{}
	ctx := newCapturingContext(t, log)
	ctx = dcontext.WithSoftness(ctx)

	cmd := dexec.CommandContext(ctx, os.Args[0], "-test.run=TestHelperProcess", "--", "echo", "foo")
	cmd.SysProcAttr = newInterruptableSysProcAttr()
	cmd.Env = []string{"GO_WANT_HELPER_PROCESS=1"}

	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, fmt.Sprintf(``+
		`level=info msg="started command [%[1]s \"-test.run=TestHelperProcess\" \"--\" \"echo\" \"foo\"]" dexec.pid=%[2]d`+"\n"+
		`level=info dexec.err=EOF dexec.pid=%[2]d dexec.stream=stdin`+"\n"+
		`level=info dexec.data="foo\n" dexec.pid=%[2]d dexec.stream=stdout+stderr`+"\n"+
		`level=info msg="finished successfully: exit status 0" dexec.pid=%[2]d`+"\n"+
		``, quote15(os.Args[0]), cmd.ProcessState.Pid()),
		log.String())
}
