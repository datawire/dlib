package dcontext_test

import (
	"context"
	"testing"

	"github.com/datawire/dlib/dcontext"
)

type valkey struct{}

func TestContextIdentity(t *testing.T) {
	ctx := context.Background()
	if ctx != dcontext.HardContext(ctx) {
		t.Fatalf("background context %+v treated differently than hard context %+v", ctx, dcontext.HardContext(ctx))
	}
	softCtx := dcontext.WithSoftness(ctx)
	if ctx == softCtx {
		t.Fatalf("background context %+v treated same as soft context %+v", ctx, softCtx)
	}
	ctx = context.WithValue(softCtx, valkey{}, 0)
	hardCtx := dcontext.HardContext(ctx)
	if ctx == hardCtx {
		t.Fatalf("soft context %+v treated same as hard context %+v", ctx, hardCtx)
	}
	if hardCtx != dcontext.HardContext(hardCtx) {
		t.Fatalf("hard context %+v treated differently than hard context %+v", hardCtx, dcontext.HardContext(hardCtx))
	}
}
