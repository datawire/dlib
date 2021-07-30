package dgroup_test

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"syscall"
	"testing"
)

func ensureProcessGroup() (keepGoing bool) {
	if os.Getenv("GO_IN_OWN_PROCESS_GROUP") == "1" {
		return true
	}

	pc, _, _, _ := runtime.Caller(1)
	qname := runtime.FuncForPC(pc).Name() // Returns "domain.tld/pkgpath.Function".
	dot := strings.LastIndex(qname, ".")  // Find the dot separating the pkg from the func.
	name := qname[dot+1:]                 // Split on that dot.

	cmd := exec.Command(os.Args[0], "-test.run=TestExampleHelper", "--", name)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
	}
	cmd.Env = append(os.Environ(),
		"GO_WANT_HELPER_PROCESS=1",
		"GO_IN_OWN_PROCESS_GROUP=1")

	if err := cmd.Run(); err != nil {
		fmt.Println("switch process group:", err)
	}
	return false
}

func TestExampleHelper(*testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	defer os.Exit(0)

	subcommands := map[string]func(){
		"Example_signalHandling1": Example_signalHandling1,
		"Example_signalHandling2": Example_signalHandling2,
		"Example_signalHandling3": Example_signalHandling3,
	}

	args := os.Args
	for len(args) > 0 {
		if args[0] == "--" {
			args = args[1:]
			break
		}
		args = args[1:]
	}
	if len(args) != 1 {
		fmt.Fprintf(os.Stderr, "No command\n")
		os.Exit(2)
	}

	subcommand, ok := subcommands[args[0]]
	if !ok {
		fmt.Fprintf(os.Stderr, "Unknown command %q\n", args[0])
		os.Exit(2)
	}

	subcommand()
}
