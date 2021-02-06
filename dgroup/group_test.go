package dgroup

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// WithStacktraceForTesting overrides the stacktrace that would be
// logged by dgroup, to set it to something fixed, to make dgroup's
// unit tests simpler.
func WithStacktraceForTesting(ctx context.Context, trace string) context.Context {
	return context.WithValue(ctx, stacktraceForTestingCtxKey{}, trace)
}

func TestParentGroup(t *testing.T) {
	// The example tests the positive case, so just test the
	// negative case here.
	group := ParentGroup(context.Background())
	assert.Nil(t, group)
}
