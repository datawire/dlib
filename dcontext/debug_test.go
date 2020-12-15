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
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	baseStr := fmt.Sprint(ctx)
	ctx = dcontext.WithSoftness(ctx)

	softStr := fmt.Sprint(ctx)
	hardStr := fmt.Sprint(dcontext.HardContext(ctx))

	assert.Truef(t, strings.HasPrefix(softStr, baseStr+"."),
		"Soft Context string does not start with %q: %q", baseStr+".", softStr)
	assert.Equal(t,
		softStr+".HardContext",
		hardStr)
}
