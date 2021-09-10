package dcontext_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/datawire/dlib/dcontext"
)

func TestContextIdentity(t *testing.T) {
	// fmtCtx(indent, ctx) is like fmt.Sprintf("%v", ctx) except with
	// "\n"+indent inserted strategicallyto correspond with each wc/H/S step
	// below.
	fmtCtx := func(indent string, ctx context.Context) string {
		out := ""
		rest := fmt.Sprintf("%v", ctx)
		depth := 0
		for {
			next := strings.IndexAny(rest, ".()")
			if next < 0 {
				out += rest
				break
			}
			switch rest[next] {
			case '.':
				out += rest[:next+1]
				rest = rest[next+1:]
				if depth == 0 && strings.Contains(out, "WithCancel") && !strings.HasPrefix(rest, "WithCancel") {
					out += "\n" + indent
				}
			case '(':
				depth++
				out += rest[:next+1]
				rest = rest[next+1:]
			case ')':
				depth--
				out += rest[:next+1]
				rest = rest[next+1:]
			}
		}
		return out
	}
	assertCtxEqual := func(a, b context.Context) {
		t.Helper()
		if a != b {
			t.Fatalf("error: expected contexts to be equal:\n\ta: %s\n\tb: %s",
				fmtCtx("\t\t", a),
				fmtCtx("\t\t", b))
		}
	}
	assertCtxNotEqual := func(a, b context.Context) {
		t.Helper()
		if a == b {
			t.Fatalf("error: expected contexts to be unequal:\n\ta: %v\n\tb: likewise", a)
		}
	}
	assertChEqual := func(a, b context.Context) {
		t.Helper()
		if a.Done() != b.Done() {
			t.Fatalf("error: expected channels to be equal:\n\ta: %v\n\t   %s\n\tb: %v\n\t   %s",
				a.Done(), fmtCtx("\t\t", a),
				b.Done(), fmtCtx("\t\t", b))
		}
	}
	assertChNotEqual := func(a, b context.Context) {
		t.Helper()
		if a.Done() == b.Done() {
			t.Fatalf("error: expected channels to be unequal:\n\ta: %v\n\t%s\n\tb: %v\n\t%s",
				a.Done(), fmtCtx("\t\t", a),
				b.Done(), fmtCtx("\t\t", b))
		}
	}
	wc := func(ctx context.Context) context.Context {
		ctx, cancel := context.WithCancel(ctx)
		t.Cleanup(cancel)
		return ctx
	}
	H := func(s context.Context) context.Context { return dcontext.HardContext(s) }
	S := func(h context.Context) context.Context { return wc(dcontext.WithSoftness(h)) }
	log := func(name string, ctx context.Context) {
		t.Logf("debug: %-8s done=%12v ctx=%s", name, ctx.Done(), fmtCtx("\t\t", ctx))
	}

	var (
		ctx      = wc(context.Background()) // 0
		ctxH     = H(ctx)                   // 1
		ctxS     = S(ctx)                   // 2
		ctxSH    = H(ctxS)                  // 3
		ctxSH2   = H(ctxS)                  // 4
		ctxSHH   = H(ctxSH)                 // 5
		ctxSS    = S(ctxS)                  // 6
		ctxSS2   = S(ctxS)                  // 7
		ctxSSH   = H(ctxSS)                 // 8
		ctxSSHH  = H(ctxSSH)                // 9
		ctxSSHHH = H(ctxSSHH)               // 10
		ctxSSS   = S(ctxSS)                 // 11
	)

	// 0
	log("ctx", ctx)
	assertChEqual(ctx, ctx)

	// 1
	log("ctxH", ctxH)
	assertChEqual(ctxH, ctxH)
	assertCtxEqual(ctx, ctxH)

	// 2
	log("ctxS", ctxS)
	assertChEqual(ctxS, ctxS)
	assertCtxNotEqual(ctx, ctxS)
	assertChNotEqual(ctx, ctxS)

	// 3
	log("ctxSH", ctxSH)
	assertChEqual(ctxSH, ctxSH)
	assertCtxNotEqual(ctxS, ctxSH)
	assertChNotEqual(ctxS, ctxSH)
	assertCtxNotEqual(ctx, ctxSH)
	assertChEqual(ctx, ctxSH)

	// 4
	log("ctxSH2", ctxSH2)
	assertChEqual(ctxSH2, ctxSH2)
	assertCtxEqual(ctxSH, ctxSH2)

	// 5
	log("ctxSHH", ctxSHH)
	assertChEqual(ctxSHH, ctxSHH)
	assertCtxEqual(ctxSH, ctxSHH)

	// 6
	log("ctxSS", ctxSS)
	assertChEqual(ctxSS, ctxSS)
	assertCtxNotEqual(ctxS, ctxSS)
	assertChNotEqual(ctxS, ctxSS)

	// 7
	log("ctxSS2", ctxSS2)
	assertChEqual(ctxSS2, ctxSS2)
	assertCtxNotEqual(ctxSS, ctxSS2)
	assertChNotEqual(ctxSS, ctxSS2)

	// 8
	log("ctxSSH", ctxSSH)
	assertChEqual(ctxSSH, ctxSSH)
	assertCtxNotEqual(ctxSS, ctxSSH)
	assertChNotEqual(ctxSS, ctxSSH)
	assertCtxNotEqual(ctxS, ctxSSH)
	assertChEqual(ctxS, ctxSSH)

	// 9
	log("ctxSSHH", ctxSSHH)
	assertChEqual(ctxSSHH, ctxSSHH)
	assertCtxNotEqual(ctxSSH, ctxSSHH)
	assertChNotEqual(ctxSS, ctxSSH)
	assertCtxNotEqual(ctxS, ctxSSH)
	assertChEqual(ctxS, ctxSSH)

	// 10
	log("ctxSSHHH", ctxSSHHH)
	assertChEqual(ctxSSHHH, ctxSSHHH)
	assertCtxEqual(ctxSSHH, ctxSSHHH)

	// 11
	log("ctxSSS", ctxSSS)
	assertChEqual(ctxSSS, ctxSSS)
	assertCtxNotEqual(ctxSS, ctxSSS)
	assertChNotEqual(ctxSS, ctxSSS)
}
