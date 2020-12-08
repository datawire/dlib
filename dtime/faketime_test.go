package dtime_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/datawire/dlib/dtime"
)

func check(t *testing.T, ft *dtime.FakeTime, what string, wantedSec int) {
	ssb := int(ft.TimeSinceBoot() / time.Second)

	if ssb != wantedSec {
		t.Errorf("%s: wanted %d seconds since boot, got %d", what, wantedSec, ssb)
	}
}

func TestFakeTime(t *testing.T) {
	ft := dtime.NewFakeTime()

	ftBoot := ft.BootTime()
	ftStart := ft.Now()

	if ftStart != ftBoot {
		t.Errorf("boot: current time %s and boot time %s don't match", ftStart, ftBoot)
	}

	check(t, ft, "at boot", 0)

	ft.StepSec(5)

	check(t, ft, "after StepSec(5)", 5)

	ft.StepSec(5)

	check(t, ft, "after 2*StepSec(5)", 10)

	ft.StepSec(-15)

	check(t, ft, "after StepSec(-15)", -5)

	ft.StepSec(15)

	check(t, ft, "after StepSec(15)", 10)

	ftEnd := ft.Now()

	ftDur := ftEnd.Sub(ftStart)

	if ftDur != 10*time.Second {
		t.Errorf("overall: wanted a 10-second duration, got %s", ftDur)
	}
}

func ExampleFakeTime() {
	ft := dtime.NewFakeTime()

	// At boot, ft.Now() and ft.BootTime() will be identical...
	if ft.Now() == ft.BootTime() {
		fmt.Println("Equal!")
	} else {
		fmt.Println("Whoa, Now and BootTime don't match at boot??")
	}

	// ...and TimeSinceBoot will be 0.
	fmt.Printf("%d\n", ft.TimeSinceBoot()/time.Second)

	// After that, we can declare that 10s have passed...
	ft.StepSec(10)

	// ...and we should see that in TimeSinceBoot.
	fmt.Printf("%d\n", ft.TimeSinceBoot()/time.Second)

	// But, of course, we're not limited to seconds.
	ft.Step(2 * time.Hour)
	fmt.Printf("%s\n", ft.TimeSinceBoot())

	// Output:
	// Equal!
	// 0
	// 10
	// 2h0m10s
}
