package dtime

import (
	"context"
	"sync"
)

// The FuncTimer type represents a single event (though it can be reused for another event by
// calling Reset).  A FuncTimer must be created with AfterFunc; the zero value is invalid.  When the
// FuncTimer expires, it will call the function passed to AfterFunc.
type FuncTimer struct {
	// Set at initialization
	ctx      context.Context
	async    bool
	name     string
	fireFn   func()
	cancelFn func()

	// Set at runtime
	outerMu   sync.Mutex
	innerMu   sync.Mutex // must hold outerMu before grabbing innerMu
	curCtx    context.Context
	curCancel context.CancelFunc
	waiting   bool
	waitDone  chan struct{}
}

func (t *FuncTimer) wait() {
	select {
	case <-t.ctx.Done():
		t.innerMu.Lock()
		t.curCancel()
	case <-t.curCtx.Done():
		t.innerMu.Lock()
	}
	if t.ctx.Err() != nil && t.cancelFn != nil {
		// We do this in a separate t.ctx.Err() check instead of on <-t.ctx.Done() because
		// <-t.ctx.Done() implies <-t.curCtx.Done(), so we can't rely which one actually
		// gets selected.
		t.cancelFn()
	}
	t.curCtx = nil
	t.curCancel = nil
	sendDone := !t.waiting
	t.waiting = false
	t.innerMu.Unlock()
	if sendDone {
		t.waitDone <- struct{}{}
	}
}

// ensureStopped makes sure that no wait() goroutines are running.
//
// t.innerMu MUST be held while calling ensureStopped.
func (t *FuncTimer) ensureStopped() {
	if t.waiting {
		t.waiting = false  // tell it that we want waitDone when it shuts down
		t.curCancel()      // tell it to shut down
		t.innerMu.Unlock() // let it shut down
		<-t.waitDone       // wait for it to shut down
		t.innerMu.Lock()   // resume normal operation
	}
}

func (t *FuncTimer) fire() {
	t.outerMu.Lock()
	defer t.outerMu.Unlock()
	t.innerMu.Lock()
	defer t.innerMu.Unlock()

	if !t.waiting {
		// Race between t.Stop/t.ctx.Done and clock.At()
		return
	}

	// Signal wait() to shut down.
	t.ensureStopped()

	if t.async {
		go t.fireFn()
	} else {
		t.fireFn()
	}
}

// AfterFunc waits for the Duration to elapse and then calls f in its own goroutine.  If the Context
// becomes Done before the Duration has elapsed, then f will never be called.  It returns a
// FuncTimer that can be used to cancel the call using its Stop method.
func AfterFunc(ctx context.Context, d Duration, f func()) *FuncTimer {
	if ctx == nil {
		ctx = context.Background()
	}
	t := &FuncTimer{
		ctx:      ctx,
		async:    true,
		name:     "FuncTimer",
		fireFn:   f,
		waitDone: make(chan struct{}),
	}
	t.Reset(d)
	return t
}

// Reset changes the FuncTimer to expire after Duration d.  It returns true if the timer had been
// active, false if the timer had expired or been stopped.
//
// Reset either reschedules when the function 'f' passed to AfterFunc(ctx, d, f) will run, in which
// case Reset returns true, or schedules f to run again, in which case it returns false.  When Reset
// returns false, Reset neither waits for the prior f to complete before returning nor does it
// guarantee that the subsequent goroutine running f does not run concurrently with the prior one.
// If the caller needs to know whether the prior execution of f is completed, it must coordinate
// with f explicitly.
func (t *FuncTimer) Reset(d Duration) (wasActive bool) {
	return t.reset(true, d)
}

func (t *FuncTimer) reset(activeAllowed bool, d Duration) (wasActive bool) {
	if t.ctx == nil {
		panic("dtime: Reset called on uninitialized " + t.name)
	}

	t.outerMu.Lock()
	defer t.outerMu.Unlock()
	t.innerMu.Lock()
	defer t.innerMu.Unlock()

	ret := t.waiting
	if ret {
		if activeAllowed {
			t.ensureStopped()
		} else {
			panic("dtime: Reset called on an active " + t.name)
		}
	}

	t.curCtx, t.curCancel = context.WithCancel(t.ctx)
	clock := getClock(t.ctx)
	clock.At(t.curCtx, clock.Now().Add(d), t.fire)
	t.waiting = true
	go t.wait()

	return ret
}

// Stop prevents the FuncTimer from firing.  It returns true if the call stops the FuncTimer, false
// if the FuncTimer has already expired or been stopped.
//
// If Stop returns false, then the ChanTimer has already expired and the function 'f' passed to
// AfterFunc(ctx, d, f) has been started in its own goroutine; Stop does not wait for f to complete
// before returning.  If the caller needs to know whether f is completed, it must coordinate with f
// explicitly.
//
// It is a no-op to call Stop on a FuncTimer that has a cancelled Context; in such a case Stop will
// return false.
func (t *FuncTimer) Stop() bool {
	if t.ctx == nil {
		panic("dtime: Stop called on uninitialized " + t.name)
	}

	t.outerMu.Lock()
	defer t.outerMu.Unlock()
	t.innerMu.Lock()
	defer t.innerMu.Unlock()

	// Signal wait() to shut down.
	t.ensureStopped()

	ret := t.waiting
	t.waiting = false
	return ret
}
