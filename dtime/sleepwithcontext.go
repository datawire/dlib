package dtime

import (
	"context"
	"time"

	dtimev2 "github.com/datawire/dlib/dtime/v2"
)

// SleepWithContext pauses the current goroutine for at least the duration d, or
// until the Context is done, whichever happens first.
//
// You may be thinking, why not just do:
//
//     select {
//     case <-ctx.Done():
//     case <-time.After(d):
//     }
//
// well, time.After can't get garbage collected until the timer
// expires, even if the Context is done.  What this function provides
// is properly stopping the timer so that it can be garbage collected
// sooner.
//
// https://medium.com/@oboturov/golang-time-after-is-not-garbage-collected-4cbc94740082
func SleepWithContext(ctx context.Context, d time.Duration) {
	dtimev2.Sleep(ctx, d)
}
