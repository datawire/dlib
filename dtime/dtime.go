// Package dtime is DEPRECATED, use dtime/v2 instead.
//
// Externally-consumable things provided here:
//
// dtime.Now() is equivalent to time.Now(), except that you can
// override it, if necessary, to have control over time for testing.
//
// dtime.FakeTime is a class that provides explicit control over a
// "fake" clock, again for testing. The simplest pattern here is to
// instantiate a FakeTime, use its Step or StepSec methods to control
// when time passes, and use its Now method instead of time.Now to
// get the time.
//
// dtime.SleepWithContext is like time.Sleep(), but it bails early
// and releases the resources if the Context gets cancelled.
package dtime

import (
	"context"
	"time"

	dtimev2 "github.com/datawire/dlib/dtime/v2"
)

var globals = struct { //nolint:gochecknoglobals // avoiding a global is why we had to create dtime/v2; can't get rid of it with this API
	ctx context.Context
}{
	ctx: context.Background(),
}

// Now is a clock function. It starts out as an alias to time.Now,
// so if you simply use dtime.Now instead of time.Now, your program
// will continue to function exactly as it did before.
//
// The power of dtime.Now is that you can use dtime.SetNow to swap
// in a different clock function for testing, so that you have
// explicit control over the passage of time. dtime.FakeTime is an
// obvious choice here, as shown in the example.
func Now() time.Time {
	return dtimev2.Now(globals.ctx)
}

type clock func() time.Time

func (c clock) Now() time.Time {
	return c()
}

func (_ clock) At(ctx context.Context, t time.Time, f func()) {
	dtimev2.StdClock{}.At(ctx, t, f)
}

// SetNow overrides the definition of dtime.Now.
//
// Note that overriding dtime.Now will (obviously) override it for the
// entire process. Note also that it is generally a bad idea to swap
// the clock in the middle of a program run and expect sane things to
// happen, if your program pays any attention to the clock at all.
func SetNow(newNow func() time.Time) {
	globals.ctx = dtimev2.WithClock(context.Background(), clock(newNow))
}
