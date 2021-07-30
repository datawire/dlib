package dcontext_test

import (
	"context"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/datawire/dlib/dcontext"
)

func TestWithoutCancel(t *testing.T) {
	isClosed := func(ch <-chan struct{}) bool {
		select {
		case <-ch:
			return true
		default:
			return false
		}
	}
	type ctxKey struct{}

	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 0)
	defer cancel()
	ctx = context.WithValue(ctx, ctxKey{}, "foo")
	<-ctx.Done()

	// sanity check
	deadline, ok := ctx.Deadline()
	assert.True(t, ok)
	switch runtime.GOOS {
	case "windows":
		// The Windows clock has low resolution, we might get the same
		// time again.
		assert.True(t, !deadline.After(time.Now()))
	default:
		assert.True(t, deadline.Before(time.Now()))
	}
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
