package dtime

import (
	"context"
	"time"
)

// sleepTestHookCtxKey lets us attach a callback function to the Context, such that Sleep will call
// the function, and allow us insert a pause in order to more reliably test a certain race
// condition.
type sleepTestHookCtxKey struct{}

// Sleep pauses the current goroutine for at least the duration d, or until the Context is done,
// whichever happens first.
//
// You may be thinking, why not just do:
//
//     select {
//     case <-ctx.Done():
//     case <-time.After(d):
//     }
//
// well, time.After can't get garbage collected until the timer expires, even if the Context is
// done.  What this function provides is properly stopping the timer so that it can be garbage
// collected sooner.
//
// https://medium.com/@oboturov/golang-time-after-is-not-garbage-collected-4cbc94740082
//
// BUG(lukeshu): Sleep does not respect WithClock.
func Sleep(ctx context.Context, d time.Duration) {
	if d <= 0 {
		return
	}
	timer := time.NewTimer(d)
	select {
	case <-ctx.Done():
		if untyped := ctx.Value(sleepTestHookCtxKey{}); untyped != nil {
			hook := untyped.(func())
			hook()
		}
		if !timer.Stop() {
			<-timer.C
		}
	case <-timer.C:
	}
}
