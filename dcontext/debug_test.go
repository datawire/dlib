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
	ctx := context.Background()
	var cancel context.CancelFunc
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
		tc := tc
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
