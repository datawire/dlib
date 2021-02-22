package dtime

import (
	"context"
)

// The ChanTimer type represents a single event (though it can be reused for another event by
// calling Reset).  When the ChanTimer expires, the current time will be sent on C.  A ChanTimer
// must be created with NewTimer; the zero value is invalid.
type ChanTimer struct {
	C       <-chan Time
	c       chan<- Time
	fnTimer FuncTimer
}

func (t *ChanTimer) sendTime() {
	select {
	case t.c <- Now(t.fnTimer.ctx):
	default:
	}
}

func (t *ChanTimer) closeCh() {
	if t.c != nil {
		close(t.c)
		t.c = nil
	}
}

// NewTimer creates a new ChanTimer that will either send the current time on its channel after at
// least Duration d, or will close its channel after Context ctx is cancelled.
//
// If ctx becomes Done after the ChanTimer has expired or has been stopped with the Stop method,
// then the channel will not be closed.  However, in such a case the channel will be closed
// immediately upon a subsequent call to the Reset method.
func NewTimer(ctx context.Context, d Duration) *ChanTimer {
	if ctx == nil {
		ctx = context.Background()
	}
	ch := make(chan Time, 1)
	t := &ChanTimer{
		C: ch,
		c: ch,
	}
	t.fnTimer = FuncTimer{
		ctx:      ctx,
		async:    false,
		name:     "ChanTimer",
		fireFn:   t.sendTime,
		cancelFn: t.closeCh,
		waitDone: make(chan struct{}),
	}
	t.Reset(d)
	return t
}

// Reset changes the ChanTimer to expire after Duration d.
//
// Reset may only be be invoked on stopped or expired ChanTimers with drained channels; it will
// panic if called on a ChanTimer that is active.
//
// If a program has already received a value from t.C, the timer is known to have expired and the
// channel drained, so t.Reset can be used directly.  If a program has not yet received a value from
// t.C, however, the timer must be stopped and—if Stop reports that the timer expired before being
// stopped—the channel explicitly drained:
//
// 	if !t.Stop() {
// 		<-t.C
// 	}
// 	t.Reset(d)
//
// This should not be done concurrent to other receives from the ChanTimer's channel.
//
// It is a no-op to call Reset on a ChanTimer that has a cancelled Context.
func (t *ChanTimer) Reset(d Duration) {
	t.fnTimer.reset(false, d)
}

// Stop prevents the ChanTimer from firing.  It returns true if the call stops the ChanTimer, false
// if the ChanTimer has already expired or been stopped.  Stop does not close the channel, to allow
// it to be resued by calling the ChanTimer's Reset method.
//
// To ensure the channel is empty after a call to Stop, check the return value and drain the
// channel. For example, assuming the program has not received from t.C already:
//
// 	if !t.Stop() {
// 		<-t.C
// 	}
//
// This cannot be done concurrent to other receives from the ChanTimer's channel or other calls to
// the ChanTimer's Stop method.
//
// It is a no-op to call Stop on a ChanTimer that has a cancelled Context; in such a case Stop will
// return false.
func (t *ChanTimer) Stop() bool {
	return t.fnTimer.Stop()
}
