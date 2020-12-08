package dtime_test

import (
	"fmt"
	"time"

	"github.com/datawire/dlib/dtime"
)

// This example uses a dtime.FakeTime to change the behavior of
// dtime.Now, allowing explicit control of the passage of time.
func ExampleNow() {
	ft := dtime.NewFakeTime()
	dtime.SetNow(ft.Now)

	// At the start, ft.Now and dtime.Now should give the same answer.
	start := ft.Now()
	now := dtime.Now()
	fmt.Printf("%d\n", int(now.Sub(start)))

	// If we step ft by five minutes, dtime.Now should reflect that.
	ft.Step(5 * time.Minute)
	now = dtime.Now()
	fmt.Printf("%d\n", int(now.Sub(start)/time.Second))

	// When all is said and done, ft.TimeSinceBoot() should also tell
	// us that we've stepped ft by five minutes.
	fmt.Printf("%s\n", ft.TimeSinceBoot())

	// Output:
	// 0
	// 300
	// 5m0s
}
