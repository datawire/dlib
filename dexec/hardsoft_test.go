package dexec_test

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"testing"
	"time"

	"github.com/datawire/dlib/dcontext"
	"github.com/datawire/dlib/dexec"
	"github.com/datawire/dlib/dlog"
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

func TestSoftCancel(t *testing.T) {
	ctx := dlog.NewTestContext(t, true)
	ctx, hardCancel := context.WithCancel(ctx)
	defer hardCancel()
	ctx = dcontext.WithSoftness(ctx)
	ctx, softCancel := context.WithCancel(ctx)

	output := &lineBuffer{
		lines: make(chan string, 50),
	}
	cmd := dexec.CommandContext(ctx, os.Args[0], "-test.run=TestSoftHelperProcess")
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
	if err.Error() != "signal: killed" {
		t.Errorf("unexpected error value: %v", err)
	}
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
	ctx := dlog.NewTestContext(t, true)
	ctx, hardCancel := context.WithCancel(ctx)
	defer hardCancel()
	ctx = dcontext.WithSoftness(ctx)
	ctx, softCancel := context.WithCancel(ctx)
	defer softCancel()

	cmd := dexec.CommandContext(ctx, "/")
	err := cmd.Start()
	if err == nil {
		t.Fatal("expected to get an error from Start()")
	}
	if err.Error() != `exec: "/": permission denied` {
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
}
