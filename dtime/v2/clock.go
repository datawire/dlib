// Package dtime provides functionality for mocking time and cancelling sleep.
//
// dtime.Now(ctx) (and friends: Since and Until) is equivalent to time.Now() except that you can
// override it using WithClock, in order to have control over time for testing.
//
// dtime.Sleep(ctx, d) (and friends: Tick, NewTicker, After, AfterFunc, and NewTimer) is equivalent to
// time.Sleep(d) except that (1) it can be cancelled early via Context cancellation, freeing all
// resources earlier, and (2) that you can override the timing by using WithClock, in order to have
// control over time for testing.
//
// dtime is mostly a drop-in replacement for stdlib "time", except that (1) the functions mentioned
// above (Now, Since, Until, Tick, NewTicker, After, AfterFunc, and NewTimer) have an extra
// context.Context argument, (2) *time.Timer has been replaced by two separate *dtime.ChanTimer and
// *dtime.FuncTimer types, and (3) unlike *time.Timer.Reset() and *dtime.FuncTimer.Reset(),
// *dtime.ChanTimer.Reset() does not have a return value.
//
// dtime.FakeClock is a class that provides explicit control over a "fake" Clock, again for testing.
// The simplest pattern here is to instantiate a FakeTime, use its Step or StepSec methods to
// control when time passes, and pass it to WithClock.
package dtime

import (
	"context"
)

// Clock is the type you must implement and pass to WithClock if you would like to spoof the system
// clock.  StdClock{} is the actual system clock, and FakeClock is a handy mock clock that you can
// use instead of implementing your own.
type Clock interface {
	// Now returns the current local Time.
	Now() Time

	// At arranges for a function to be called at a given Time, unless the Context is cancelled
	// first.  If the given Time is before Now(), then the function is called immediately.  If
	// the Context is canceled before the Time is reached, then the function is not called.
	//
	// Either the time being reached or the Context being cancelled will release all allocated
	// resources for garbage collection.
	At(context.Context, Time, func())
}

type clockCtxKey struct{}

func getClock(ctx context.Context) Clock {
	var clock Clock = StdClock{}
	if untyped := ctx.Value(clockCtxKey{}); untyped != nil {
		clock = untyped.(Clock)
	}
	return clock
}

// Now returns the current local time.
func Now(ctx context.Context) Time {
	return getClock(ctx).Now()
}

// WithClock changes the Clock used by dtime functions that are passed the resulting Context.
func WithClock(ctx context.Context, clock Clock) context.Context {
	return context.WithValue(ctx, clockCtxKey{}, clock)
}
