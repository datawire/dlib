package dcontext_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/datawire/dlib/dcontext"
)

func isClosed(ch <-chan struct{}) bool {
	select {
	case <-ch:
		return true
	default:
		return false
	}
}

func TestWithoutCancel(t *testing.T) {
	type ctxKey struct{}

	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 0)
	defer cancel()
	ctx = context.WithValue(ctx, ctxKey{}, "foo")
	<-ctx.Done()

	// sanity check
	deadline, ok := ctx.Deadline()
	assert.True(t, ok)
	assert.True(t, deadline.Before(time.Now()))
	assert.True(t, isClosed(ctx.Done()))
	assert.Error(t, ctx.Err())
	assert.Equal(t, "foo", ctx.Value(ctxKey{}))

	ctx = dcontext.WithoutCancel(ctx)

	// the actual meaningful check
	deadline, ok = ctx.Deadline()
	assert.False(t, ok)
	assert.True(t, deadline.IsZero())
	assert.False(t, isClosed(ctx.Done()))
	assert.NoError(t, ctx.Err())
	assert.Equal(t, "foo", ctx.Value(ctxKey{}))
}

func TestNoSoftCancel(t *testing.T) {
	hardCtx, hardCancel := context.WithCancel(context.Background())
	softCtx, softCancel := context.WithCancel(dcontext.WithSoftness(hardCtx))
	noCancelCtx := dcontext.WithoutCancel(softCtx)

	// 0
	assert.NoError(t, noCancelCtx.Err())
	assert.NoError(t, dcontext.HardContext(noCancelCtx).Err())

	// 1
	softCancel()
	assert.NoError(t, noCancelCtx.Err())
	assert.NoError(t, dcontext.HardContext(noCancelCtx).Err())

	// 2
	hardCancel()
	assert.NoError(t, noCancelCtx.Err())
	assert.NoError(t, dcontext.HardContext(noCancelCtx).Err())
}
