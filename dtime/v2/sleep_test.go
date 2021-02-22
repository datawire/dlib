package dtime_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/datawire/dlib/dlog"
	dtime "github.com/datawire/dlib/dtime/v2"
)

func assertDurationEq(t testing.TB, expected, actual, slop dtime.Duration, msgAndArgs ...interface{}) bool {
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
		Arg         dtime.Duration
		CancelAfter dtime.Duration
		Expected    dtime.Duration
	}{
		"negative":    {Arg: -1 * dtime.Hour, Expected: 0},
		"zero":        {Arg: 0, Expected: 0},
		"canceled":    {Arg: 1 * dtime.Hour, CancelAfter: 1 * dtime.Second, Expected: 1 * dtime.Second},
		"normal":      {Arg: 1 * dtime.Second, Expected: 1 * dtime.Second},
		"late-cancel": {Arg: 1 * dtime.Second, CancelAfter: 1 * dtime.Hour, Expected: 1 * dtime.Second},
	}
	for tcname, tcinfo := range testcases {
		t.Run(tcname, func(t *testing.T) {
			ctx := dlog.NewTestContext(t, false)
			if tcinfo.CancelAfter > 0 {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, tcinfo.CancelAfter)
				defer cancel()
			}
			start := dtime.Now(ctx)
			dtime.Sleep(ctx, tcinfo.Arg)
			actual := dtime.Since(ctx, start)
			assertDurationEq(t, tcinfo.Expected, actual,
				dtime.Second/100)
		})
	}
}
