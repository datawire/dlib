package dtime

import (
	"context"
	"time"
)

// StdClock is a Clock implementation that the uses the real stdlib time.Now().
type StdClock struct{}

// Now implements Clock.
func (_ StdClock) Now() Time {
	return time.Now()
}

// At implements Clock.
func (_ StdClock) At(ctx context.Context, t Time, f func()) {
	ctx, cancel := context.WithCancel(ctx)
	timer := time.AfterFunc(time.Until(t), func() {
		cancel()
		f()
	})
	go func() {
		<-ctx.Done()
		timer.Stop()
	}()
}
