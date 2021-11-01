// Copyright 2021 Datawire. All rights reserved.
//
// This file contains documentation copied from Go 1.17.1 sync/mutex.go.
// This file contains code inspired by Go 1.17.1 sync/cond.go.
//
// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE. file.

// Package dsync provides cancelable variants of synchronization primitives such as mutual exclusion
// locks.
package dsync

import (
	"context"
	"sync/atomic"
	"unsafe"
)

// A Locker represents an object that can be locked and unlocked, and follows the principal that any
// blocking operation should be cancelable with a Context.
type Locker interface {
	Lock(context.Context) error
	Unlock()
}

// A Mutex is a mutual exclusion lock.
//
// The zero value for a Mutex is an unlocked mutex.
//
// A Mutex must not be copied after first use.
type Mutex struct {
	ch unsafe.Pointer // *chan struct{}

	noCopy    noCopyRuntime
	noCopyVet noCopyVet //nolint:structcheck // embedded for `go vet` purposes, not actually used
}

// Lock locks m.
//
// If the lock is already in use, the calling goroutine blocks until either the mutex is available
// (and returns nil) or the Context is canceled (and returns ctx.Err()).
func (m *Mutex) Lock(ctx context.Context) error {
	if !m.noCopy.check() {
		panic("dsync.Mutex.Lock: mutex was copied after first use")
	}
	myCh := make(chan struct{})
	for {
		if swapped := atomic.CompareAndSwapPointer(&m.ch, nil, unsafe.Pointer(&myCh)); swapped {
			// Yay, we got the lock.
			return nil
		}
		theirCh := (*chan struct{})(atomic.LoadPointer(&m.ch))
		if theirCh == nil {
			// The lock got released in the time since we last tried to get it;
			// try again.
			continue
		}
		// Wait for the lock gets released.
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-(*theirCh):
		}
	}
}

// Unlock unlocks m.
//
// It is a runtime error (panic) if m is not locked on entry to Unlock.
//
// A locked Mutex is not associated with a particular goroutine.  It is allowed for one goroutine to
// lock a Mutex and then arrange for another goroutine to unlock it.
func (m *Mutex) Unlock() {
	if !m.noCopy.check() {
		panic("dsync.Mutex.Unlock: mutex was copied after first use")
	}
	// unlock it
	ch := (*chan struct{})(atomic.SwapPointer(&m.ch, nil))
	if ch == nil {
		panic("dsync.Mutex.Unlock: not locked")
	}
	// wake up listeners
	close(*ch)
}
