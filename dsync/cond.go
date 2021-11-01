// Copyright 2021 Datawire. All rights reserved.
//
// This file contains documentation copied from and code inspired by Go 1.17.1 sync/cond.go.
//
// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE. file.

package dsync

import (
	"context"
	"sync"
)

// Cond implements a condition variable, a rendezvous point for goroutines waiting for or announcing
// the occurrence of an event.
//
// Each Cond has an associated Locker L (often a *Mutex), which must be held when changing the
// condition and when calling the Wait method.
//
// A Cond must not be copied after first use.
type Cond struct {
	// L is held while observing or changing the condition
	L Locker

	mu        sync.Mutex
	listeners map[chan struct{}]struct{}

	noCopy    noCopyRuntime
	noCopyVet noCopyVet //nolint:structcheck // embedded for `go vet` purposes, not actually used
}

// NewCond returns a new Cond with Locker l.
//
// This is just a convenience function for
//
//     &Cond{L: l}
//
func NewCond(l Locker) *Cond {
	return &Cond{L: l}
}

// Broadcast wakes all goroutines waiting on c.
//
// It is allowed but not required for the caller to hold c.L during the call.
func (c *Cond) Broadcast() {
	if !c.noCopy.check() {
		panic("dsync.Cond.Broadcast: cond was copied after first use")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	for ch := range c.listeners {
		close(ch)
		delete(c.listeners, ch)
	}
}

// Signal wakes one goroutine waiting on c, if there is any.
//
// It is allowed but not required for the caller to hold c.L during the call.
func (c *Cond) Signal() {
	if !c.noCopy.check() {
		panic("dsync.Cond.Signal: cond was copied after first use")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	for ch := range c.listeners {
		close(ch)
		delete(c.listeners, ch)
		return
	}
}

func (c *Cond) listen() chan struct{} {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.listeners == nil {
		c.listeners = make(map[chan struct{}]struct{})
	}
	ch := make(chan struct{})
	c.listeners[ch] = struct{}{}
	return ch
}

// Wait atomically unlocks c.L and suspends execution of the calling goroutine.  After later
// resuming execution, Wait locks c.L before returning.  Unlike in other systems, Wait cannot return
// unless awoken by Broadcast or Signal.
//
// Because c.L is not locked when Wait first resumes, the caller typically cannot assume that the
// condition is true when Wait returns.  Instead, the caller should Wait in a loop:
//
//    if err := c.L.Lock(); err != nil {
//        return err
//    }
//    for !condition() {
//        if err := c.Wait(ctx); err != nil {
//            // note: c.L is not held if err != nil
//            return err
//        }
//    }
//    ... make use of condition ...
//    c.L.Unlock()
//
func (c *Cond) Wait(ctx context.Context) error {
	if !c.noCopy.check() {
		panic("dsync.Cond.Wait: cond was copied after first use")
	}
	ch := c.listen()
	c.L.Unlock()
	select {
	case <-ctx.Done():
		c.mu.Lock()
		delete(c.listeners, ch)
		c.mu.Unlock()
		return ctx.Err()
	case <-ch:
		return c.L.Lock(ctx)
	}
}
