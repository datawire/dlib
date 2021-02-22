package dtime_test

import (
	"context"
	"fmt"
	"testing"

	dtime "github.com/datawire/dlib/dtime/v2"
)

func check(t *testing.T, fc *dtime.FakeClock, what string, wantedSec int) {
	ssb := int(fc.TimeSinceBoot() / dtime.Second)

	if ssb != wantedSec {
		t.Errorf("%s: wanted %d seconds since boot, got %d", what, wantedSec, ssb)
	}
}

func TestFakeClock(t *testing.T) {
	fc := dtime.NewFakeClock(dtime.Now(context.Background()))

	fcBoot := fc.BootTime()
	fcStart := fc.Now()

	if fcStart != fcBoot {
		t.Errorf("boot: current time %s and boot time %s don't match", fcStart, fcBoot)
	}

	check(t, fc, "at boot", 0)

	fc.StepSec(5)

	check(t, fc, "after StepSec(5)", 5)

	fc.StepSec(5)

	check(t, fc, "after 2*StepSec(5)", 10)

	fc.StepSec(-15)

	check(t, fc, "after StepSec(-15)", -5)

	fc.StepSec(15)

	check(t, fc, "after StepSec(15)", 10)

	fcEnd := fc.Now()

	fcDur := fcEnd.Sub(fcStart)

	if fcDur != 10*dtime.Second {
		t.Errorf("overall: wanted a 10-second duration, got %s", fcDur)
	}
}

func ExampleFakeClock() {
	fc := dtime.NewFakeClock(dtime.Now(context.Background()))

	// At boot, fc.Now() and fc.BootTime() will be identical...
	if fc.Now() == fc.BootTime() {
		fmt.Println("Equal!")
	} else {
		fmt.Println("Whoa, Now and BootTime don't match at boot??")
	}

	// ...and TimeSinceBoot will be 0.
	fmt.Printf("%d\n", fc.TimeSinceBoot()/dtime.Second)

	// After that, we can declare that 10s have passed...
	fc.StepSec(10)

	// ...and we should see that in TimeSinceBoot.
	fmt.Printf("%d\n", fc.TimeSinceBoot()/dtime.Second)

	// But, of course, we're not limited to seconds.
	fc.Step(2 * dtime.Hour)
	fmt.Printf("%s\n", fc.TimeSinceBoot())

	// Output:
	// Equal!
	// 0
	// 10
	// 2h0m10s
}
