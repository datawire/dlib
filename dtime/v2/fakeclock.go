package dtime

import (
	"time"
)

// A FakeClock keeps track of fake time for us, so that we don't have to rely on the real system
// clock. This can make life during testing much, much easier -- rather than needing to wait
// forever, you can control the passage of time however you like.
//
// To use FakeClock, use NewFakeClock to instantiate it, then Step (or StepSec) to change its
// current time. FakeClock also remembers its boot time (the time when it was instantiated) so that
// you can meaningfully talk about how much fake time has passed since boot and, if necessary,
// relate fake times to actual system times.
type FakeClock struct {
	bootTime    time.Time
	currentTime time.Time
}

// NewFakeClock creates a new FakeClock structure, booted at the current time.  Once instantiated,
// its Now method is a drop-in replacement for time.Now.
func NewFakeClock() *FakeClock {
	ft := &FakeClock{}

	ft.bootTime = time.Now()
	ft.currentTime = ft.bootTime

	return ft
}

// Step steps a FakeClock by the given duration. Any duration may be used, with all the obvious
// concerns about stepping the fake clock into the past.
func (f *FakeClock) Step(d time.Duration) {
	f.currentTime = f.currentTime.Add(d)
}

// StepSec steps a FakeClock by a given number of seconds. Any number of seconds is valid, with all
// the obvious concerns about stepping the fake clock into the past.
//
// This is a convenience to allow writing unit tests that don't have to have "* time.Second"
// scattered over and over and over again through everything.
func (f *FakeClock) StepSec(s int) {
	f.Step(time.Duration(s) * time.Second)
}

// BootTime returns the real system time at which the FakeClock was instantiated, in case it's
// needed.
//
// This is an accessor because we don't really want people changing the boot time after boot.
func (f *FakeClock) BootTime() time.Time {
	return f.bootTime
}

// Now returns the current fake time. It is a drop-in replacement for time.Now, and is particularly
// suitable for use with dtime.SetNow and dtime.Now.
func (f *FakeClock) Now() time.Time {
	return f.currentTime
}

// TimeSinceBoot returns the amount of fake time that has passed since the FakeClock was
// instantiated.
func (f *FakeClock) TimeSinceBoot() time.Duration {
	return f.currentTime.Sub(f.bootTime)
}
