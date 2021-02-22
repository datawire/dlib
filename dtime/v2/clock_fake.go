package dtime

import (
	"context"
	"sort"
	"sync"
)

type cronjob struct {
	c context.Context
	f func()
}

// FakeClock is a Clock implementation that it keeps track of fake time for us, so that we don't
// have to rely on the real system clock.  This can make life during testing much, much easier --
// rather than needing to wait forever, you can control the passage of time however you like.
//
// To use FakeClock, use NewFakeClock to instantiate it, then Step (or StepSec) to change its
// current time.  FakeClock also remembers its boot time (the time when it was instantiated) so that
// you can meaningfully talk about how much fake time has passed since boot and, if necessary,
// relate fake times to actual system times.
type FakeClock struct {
	mu sync.Mutex

	bootTime    Time
	currentTime Time

	cronjobs map[Time][]cronjob
}

// NewFakeClock creates a new FakeClock structure.
func NewFakeClock(bootTime Time) *FakeClock {
	return &FakeClock{
		bootTime:    bootTime,
		currentTime: bootTime,
	}
}

func (f *FakeClock) gcJobs() {
	for ts, jobs := range f.cronjobs {
		changed := false
		for i := 0; i < len(jobs); i++ {
			if jobs[i].c.Err() != nil {
				copy(jobs[i:], jobs[i+1:])
				jobs = jobs[:len(jobs)-1]
				changed = true
			}
		}
		if len(jobs) == 0 {
			delete(f.cronjobs, ts)
		} else if changed {
			f.cronjobs[ts] = jobs
		}
	}
}

func (f *FakeClock) fireJobs() {
	var times []Time
	for ts := range f.cronjobs {
		if !ts.After(f.currentTime) {
			times = append(times, ts)
		}
	}
	sort.Slice(times, func(i, j int) bool {
		return times[i].Before(times[j])
	})
	for _, ts := range times {
		for _, job := range f.cronjobs[ts] {
			f := job.f
			go f()
		}
		delete(f.cronjobs, ts)
	}
}

// Step steps a FakeClock by the given duration.  Any duration may be used, with all the obvious
// concerns about stepping the fake clock into the past.
func (f *FakeClock) Step(d Duration) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.currentTime = f.currentTime.Add(d)
	f.gcJobs()
	f.fireJobs()
}

// StepSec steps a FakeClock by a given number of seconds. Any number of seconds is valid, with all
// the obvious concerns about stepping the fake clock into the past.
//
// This is a convenience to allow writing unit tests that don't have to have "* time.Second"
// scattered over and over and over again through everything.
func (f *FakeClock) StepSec(s int) {
	f.Step(Duration(s) * Second)
}

// BootTime returns the real system time at which the FakeClock was instantiated, in case it's
// needed.
//
// This is an accessor because we don't really want people changing the boot time after boot.
func (f *FakeClock) BootTime() Time {
	return f.bootTime
}

// TimeSinceBoot returns the amount of fake time that has passed since the FakeClock was
// instantiated.
func (f *FakeClock) TimeSinceBoot() Duration {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.currentTime.Sub(f.bootTime)
}

// Now implements Clock.
func (f *FakeClock) Now() Time {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.currentTime
}

// At implements Clock.
func (f *FakeClock) At(ctx context.Context, t Time, fn func()) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.cronjobs[t] = append(f.cronjobs[t], cronjob{
		c: ctx,
		f: fn,
	})
}
