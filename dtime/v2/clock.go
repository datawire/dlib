// Package dtime provides functionality for mocking time and cancelling sleep.
//
// dtime.Now(ctx) is equivalent to time.Now() except that you can override it using WithClock, in
// order to have control over time for testing.
//
// dtime.FakeTime is a class that provides explicit control over a "fake" clock, again for testing.
// The simplest pattern here is to instantiate a FakeTime, use its Step or StepSec methods to
// control when time passes, and pass its Now method to WithClock.
//
// dtime.Sleep(ctx, d) is like time.Sleep(d), but it does the right thing when the Context is
// cancelled.
package dtime

import (
	"context"
	"time"
)

type clockCtxKey struct{}

// Now returns the current local time.  It is possible to override the clock that the time is read
// from by using the WithClock function.  instead of time.Now, your program will continue to
// function exactly as it did before.
func Now(ctx context.Context) time.Time {
	clock := time.Now
	if untyped := ctx.Value(clockCtxKey{}); untyped != nil {
		clock = untyped.(func() time.Time)
	}
	return clock()
}

// WithClock changes the clock used by dtime functions that are passed the resulting Context.
func WithClock(ctx context.Context, clock func() time.Time) context.Context {
	return context.WithValue(ctx, clockCtxKey{}, clock)
}
