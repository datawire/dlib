package dcontext

import (
	"context"
	"time"
)

type withoutCancel struct {
	context.Context
}

func (withoutCancel) Deadline() (deadline time.Time, ok bool) { return }
func (withoutCancel) Done() <-chan struct{}                   { return nil }
func (withoutCancel) Err() error                              { return nil }
func (c withoutCancel) String() string                        { return contextName(c.Context) + ".WithoutCancel" }

// WithoutCancel returns a copy of parent that inherits only values and not
// deadlines/cancellation/errors.  This is useful for implementing non-timed-out
// tasks during cleanup.
func WithoutCancel(parent context.Context) context.Context {
	return withoutCancel{parent}
}
