package derror_test

import (
	"fmt"
	"runtime"
	"strings"
	"testing"

	"github.com/pkg/errors"

	"github.com/datawire/dlib/derror"
)

var thispackage, thisfile = func() (string, string) {
	pc, file, _, _ := runtime.Caller(0)
	name := runtime.FuncForPC(pc).Name()
	// name is "foo.bar/baz/pkg.func1.func2"; we want
	// "foo.bar/baz/pkg".  That is: We trim at the first dot after
	// the last slash.  This logic is similar to that from
	// github.com/pkg/errors.funcname().
	slash := strings.LastIndex(name, "/")
	dot := slash + strings.Index(name[slash:], ".")
	return name[:dot], file
}()

func TestPanicToError(t *testing.T) {
	checkErr := func(t *testing.T, err error) {
		if err == nil {
			t.Error("error: err is nil")
			return
		}

		var k, v string

		////////////////////////////////////////////////////////////////
		k = "err.Error()"
		v = err.Error()
		t.Logf("debug: %s: %q", k, v)
		if !strings.HasPrefix(v, "PANIC: ") {
			t.Errorf("error: %s doesn't look like a panic: %q", k, v)
		}
		if strings.Count(v, "PANIC") != 1 {
			t.Errorf("error: %s looks like it nested wrong: %q", k, v)
		}
		if strings.Contains(v, ".go") {
			t.Errorf("error: %s looks it includes a stack trace: %q", k, v)
		}
		////////////////////////////////////////////////////////////////
		k = "fmt.Sprintf(\"%s\", err)"
		v = fmt.Sprintf("%s", err)
		t.Logf("debug: %s: %q", k, v)
		if !strings.HasPrefix(v, "PANIC: ") {
			t.Errorf("error: %s doesn't look like a panic: %q", k, v)
		}
		if strings.Count(v, "PANIC") != 1 {
			t.Errorf("error: %s looks like it nested wrong: %q", k, v)
		}
		if strings.Contains(v, ".go") {
			t.Errorf("error: %s looks it includes a stack trace: %q", k, v)
		}
		str := v
		////////////////////////////////////////////////////////////////
		k = "fmt.Sprintf(\"%q\", err)"
		v = fmt.Sprintf("%q", err)
		t.Logf("debug: %s: %q", k, v)
		if !strings.HasPrefix(v, "\"") {
			t.Errorf("error: %s doesn't look quoted: %q", k, v)
		} else if !strings.HasPrefix(v, "\"PANIC: ") {
			t.Errorf("error: %s doesn't look like a panic: %q", k, v)
		}
		if v != fmt.Sprintf("%q", str) {
			t.Errorf("error: %s doesn't match fmt.Sprintf(\"%%s\", err):\n\t%%q: %q\n\t%%s: %s", k, v, str)
		}
		if strings.Count(v, "PANIC") != 1 {
			t.Errorf("error: %s looks like it nested wrong: %q", k, v)
		}
		if strings.Contains(v, ".go") {
			t.Errorf("error: %s looks it includes a stack trace: %q", k, v)
		}
		////////////////////////////////////////////////////////////////
		k = "fmt.Sprintf(\"%v\", err)"
		v = fmt.Sprintf("%v", err)
		t.Logf("debug: %s: %q", k, v)
		if !strings.HasPrefix(v, "PANIC: ") {
			t.Errorf("error: %s doesn't look like a panic: %q", k, v)
		}
		if strings.Count(v, "PANIC") != 1 {
			t.Errorf("error: %s looks like it nested wrong: %q", k, v)
		}
		if strings.Contains(v, ".go") {
			t.Errorf("error: %s looks it includes a stack trace: %q", k, v)
		}
		////////////////////////////////////////////////////////////////
		k = "fmt.Sprintf(\"%+v\", err)"
		v = fmt.Sprintf("%+v", err)
		t.Logf("debug: %s: %q", k, v)
		if !strings.HasPrefix(v, "PANIC: ") {
			t.Errorf("error: %s doesn't look like a panic: %q", k, v)
		}
		if strings.Count(v, "PANIC") != 1 {
			t.Errorf("error: %s looks like it nested wrong: %q", k, v)
		}
		if !strings.Contains(v, "panic_test.go") {
			t.Errorf("error: %s doesn't include a stack trace: %q", k, v)
		}
		if strings.Contains(v, ".PanicToError") {
			t.Errorf("error: %s doesn't trim enough of the stack trace: %q", k, v)
		}
		lines := strings.Split(v, "\n")
		if len(lines) <= 3 { // we check the first 3 lines, and assert that there are more
			t.Errorf("error: %s doesn't include enough of a stack trace: %q", k, v)
		}
		if !strings.HasPrefix(lines[1], thispackage+".") {
			t.Errorf("error: %s the stack trace doesn't start in package %q: %q", k, thispackage, v)
		}
		if !strings.HasPrefix(lines[2], "\t"+thisfile+":") {
			t.Errorf("error: %s the stack trace doesn't start in file %q: %q", k, thisfile, v)
		}
		////////////////////////////////////////////////////////////////
	}
	t.Run("nil", func(t *testing.T) {
		if derror.PanicToError(nil) != nil {
			t.Error("error: PanicToError(nil) should be nil")
		}
	})
	t.Run("non-error", func(t *testing.T) { checkErr(t, derror.PanicToError("foo")) })
	t.Run("plain-error", func(t *testing.T) { checkErr(t, derror.PanicToError(errors.New("err"))) })
	t.Run("wrapped-error", func(t *testing.T) {
		root := fmt.Errorf("x")
		err := derror.PanicToError(errors.Wrap(root, "wrapped"))
		checkErr(t, err)
		if errors.Cause(err) != root {
			t.Error("error: error has the wrong cause")
		}
	})
	t.Run("sigsegv", func(t *testing.T) {
		defer func() {
			checkErr(t, derror.PanicToError(recover()))
		}()
		var str *string
		fmt.Println(*str) //nolint:govet // this will panic
	})
	t.Run("panic-recover-panic", func(t *testing.T) {
		var a, b error
		defer func() {
			b = derror.PanicToError(recover())
			checkErr(t, b)
			if a != b {
				t.Errorf("error: error was wrapped extra times")
			}
		}()
		defer func() {
			a = derror.PanicToError(recover())
			panic(a)
		}()
		panic("root")
	})
}
