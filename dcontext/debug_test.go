package dcontext_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/datawire/dlib/dcontext"
)

func TestDebug(t *testing.T) {
	// Start with the base context.Background() and keep wrapping it, each
	// testcase creating another generation of grandchildren.  At each
	// generation, we make sure that it didn't throw away debug info from
	// previous generations.
	ctx := context.Background()
	var cancel context.CancelFunc
	// Because each testcase is adding to the chain, we have to run them in
	// order, so this is a []struct{Name; ...} instead of a
	// map[string]struct{...}.
	testcases := []struct {
		Name   string
		SetCtx func()
		Suffix string
	}{
		{
			Name: "WithCancel",
			SetCtx: func() {
				ctx, cancel = context.WithCancel(ctx)
				t.Cleanup(cancel)
			},
			Suffix: "WithCancel",
		},
		{"WithSoftness", func() { ctx = dcontext.WithSoftness(ctx) }, ""},
		{"HardContext", func() { ctx = dcontext.HardContext(ctx) }, "HardContext"},
		{"WithoutCancel", func() { ctx = dcontext.WithoutCancel(ctx) }, "WithoutCancel"},
	}
	t.Log(fmt.Sprint(ctx))
	for _, tc := range testcases {
		tc := tc // capture loop variable

		// It's a little weird to t.Run this because we must run all of
		// them in order, so the -run= flag to select specific subtests
		// won't really be useful.  But because we check before:= again
		// instead of just swapping before=after, later steps should be
		// resilient to breakage in earlier steps, so t.Run is nice
		// because it will tell us exactly which cases are failing.
		t.Run(tc.Name, func(t *testing.T) {
			before := fmt.Sprint(ctx)
			tc.SetCtx()
			after := fmt.Sprint(ctx)
			t.Log(after)
			if tc.Suffix == "" {
				assert.Truef(t, strings.HasPrefix(after, before+"."),
					"%s string does not start with %q: %q", tc.Name, before+".", after)
			} else {
				assert.Equal(t, before+"."+tc.Suffix, after)
			}
		})
	}
}
