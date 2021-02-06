package dtime

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/datawire/dlib/dlog"
)

func assertDurationEq(t testing.TB, expected, actual, slop time.Duration, msgAndArgs ...interface{}) bool {
	t.Helper()

	diff := expected - actual
	if diff < 0 {
		diff = -diff
	}

	if diff > slop {
		return assert.Fail(t,
			fmt.Sprintf("Expected duration to be within %v of %v, but was %v", slop, expected, actual),
			msgAndArgs...)
	}

	return true
}

func TestSleep(t *testing.T) {

	testcases := map[string]struct {
		Arg         time.Duration
		CancelAfter time.Duration
		Expected    time.Duration
		Hook        func()
	}{
		"negative":    {Arg: -1 * time.Hour, Expected: 0},
		"zero":        {Arg: 0, Expected: 0},
		"canceled":    {Arg: 1 * time.Hour, CancelAfter: 1 * time.Second, Expected: 1 * time.Second},
		"normal":      {Arg: 1 * time.Second, Expected: 1 * time.Second},
		"late-cancel": {Arg: 1 * time.Second, CancelAfter: 1 * time.Hour, Expected: 1 * time.Second},
		"race": {Arg: 11 * (time.Second / 10), CancelAfter: 1 * time.Second,
			Hook:     func() { time.Sleep(time.Second / 2) },
			Expected: 3 * (time.Second / 2)},
	}
	for tcname, tcinfo := range testcases {
		t.Run(tcname, func(t *testing.T) {
			ctx := dlog.NewTestContext(t, false)
			if tcinfo.CancelAfter > 0 {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, tcinfo.CancelAfter)
				defer cancel()
			}
			if tcinfo.Hook != nil {
				ctx = context.WithValue(ctx, sleepTestHookCtxKey{}, tcinfo.Hook)
			}
			start := time.Now()
			Sleep(ctx, tcinfo.Arg)
			actual := time.Since(start)
			assertDurationEq(t, tcinfo.Expected, actual,
				time.Second/100)
		})
	}
}
