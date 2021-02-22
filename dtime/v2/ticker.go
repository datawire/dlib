package dtime

import (
	"context"
	"sync"
)

// A Ticker holds a channel that delivers “ticks” of a clock at intervals.
type Ticker struct {
	C <-chan Time // The channel on which the ticks are delivered.
	c chan<- Time

	mu    sync.Mutex
	start Time
	d     Duration
	i     int64

	timer *FuncTimer
}

func (t *Ticker) fire() {
	t.mu.Lock()
	defer t.mu.Unlock()
	select {
	case t.c <- Now(t.timer.ctx):
	default:
	}
	// Schedule the next tick based on t.start rather than 'Now()' to avoid drift over time.
	t.i++
	t.timer.Reset(Until(t.timer.ctx, t.start.Add(t.d*Duration(t.i+1))))
}

// NewTicker returns a new Ticker containing a channel that will send the time on the channel after
// each tick.  The period of the ticks is specified by the Duration argument.  The Ticker will
// adjust the time interval or drop ticks to make up for slow receivers.  The Duration d must be
// greater than zero; if not, NewTicker will panic.  The Context becoming Done implicitly calls the
// Ticker's Stop method.  To release associated resources, either call the Ticker's Stop method, or
// cancel the Context.
func NewTicker(ctx context.Context, d Duration) *Ticker {
	if d <= 0 {
		panic("dtime: non-positive interval for NewTicker")
	}

	ch := make(chan Time, 1)
	t := &Ticker{
		C: ch,
		c: ch,

		start: Now(ctx),
		d:     d,
	}
	t.timer = AfterFunc(ctx, d, t.fire)
	return t
}

// Reset stops a ticker and resets its period to the specified duration.  The next tick will arrive
// after the new period elapses.  It is a no-op to call Reset on a Ticker with a Context that is
// Done.
func (t *Ticker) Reset(d Duration) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.start = Now(t.timer.ctx)
	t.d = d
	t.i = 0
	t.timer.Reset(Until(t.timer.ctx, t.start.Add(d*Duration(t.i+1))))
}

// Stop turns off a ticker. After Stop, no more ticks will be sent. Stop does not close the channel,
// to prevent a concurrent goroutine reading from the channel from seeing an erroneous "tick", and
// to enable reusing the object by calling the Reset method.
func (t *Ticker) Stop() {
	t.timer.Stop()
}
