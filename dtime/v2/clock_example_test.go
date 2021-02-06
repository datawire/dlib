package dtime_test

import (
	"context"
	"fmt"
	"time"

	dtime "github.com/datawire/dlib/dtime/v2"
)

// This example uses a dtime.FakeTime to change the behavior of
// dtime.Now, allowing explicit control of the passage of time.
func ExampleNow() {
	ctx := context.Background()

	fc := dtime.NewFakeClock()
	ctx = dtime.WithClock(ctx, fc.Now)

	// At the start, fc.Now and dtime.Now should give the same answer.
	start := fc.Now()
	now := dtime.Now(ctx)
	fmt.Printf("%d\n", int(now.Sub(start)))

	// If we step fc by five minutes, dtime.Now should reflect that.
	fc.Step(5 * time.Minute)
	now = dtime.Now(ctx)
	fmt.Printf("%d\n", int(now.Sub(start)/time.Second))

	// When all is said and done, fc.TimeSinceBoot() should also tell
	// us that we've stepped fc by five minutes.
	fmt.Printf("%s\n", fc.TimeSinceBoot())

	// Output:
	// 0
	// 300
	// 5m0s
}
