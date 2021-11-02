// Copyright 2021 Datawire. All rights reserved.
//
// This file contains documentation copied from Go 1.17.1 sync/mutex.go.
// This file contains code inspired by Go 1.17.1 sync/mutex.go and sync/cond.go.
//
// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE. file.

// Package dsync provides cancelable variants of synchronization primitives such as mutual exclusion
// locks.
//
// As much as we try to make dsync performant, it is and will remain 5x-10x slower than stdlib sync.
// That's OK, you should use them in different situations:
//
// If you're dealing with low-level concepts and care about things on CPU timescales: (1) You're
// doing locking for memory safety, not for application logic.  (2) This is all very fast, the code
// in your critical section can't "block", everything is fast enough that you actually do care about
// the time difference betweens sync and dsync.
//
// If you're dealing with human-level concepts and care about things on human-noticable timescales:
// (1) You probably should care about cancelation.  (2) On these timescales you don't care about the
// time difference between sync and dsync.
//
// stdlib sync has been serving both use-cases, but is really only well-equipped to serve the first
// one.  And that's been mostly OK, since in Go you should share by communicating and so you
// "should" be avoiding the second as much as possible.  But sometimes, (more than us Go nuts like
// to admit) it really is appropriate; so that's where dsync comes in.  It would be slick if both
// use-cases could be served by the same code, but the evidence right now is that just isn't
// realistic[1].
//
// A good rule-of-thumb for when to use sync vs dsync is to use sync when your critical section
// cannot block, and to use dsync when it can block.
//
// Here are the benchmarks comparing stdlib 'sync' with dsync.  These measurements were taken on my
// aging laptop with other things running and without doing anything like disabling CPU scaling; AKA
// they are garbage.  That's OK, even "garbage measurements" is good enough to show you "order of
// magnitude difference".
//
//  |                             |       <r> |         <r> |                                     |       <r> |           <r> |      <r> |
//  | sync test                   |  sync cnt |   sync rate | dsync test                          | dsync cnt |    dsync rate | slowdown |
//  |-----------------------------+-----------+-------------+-------------------------------------+-----------+---------------+----------|
//  | goos: linux                 |           |             | goos: linux                         |           |               |          |
//  | goarch: amd64               |           |             | goarch: amd64                       |           |               |          |
//  | pkg: sync                   |           |             | pkg: github.com/datawire/dlib/dsync |           |               |          |
//  | BenchmarkCond1              |           |             | BenchmarkCond1                      |           |               |          |
//  | BenchmarkCond1-4            |   4008776 |   287 ns/op | BenchmarkCond1-4                    |    702122 |    1636 ns/op |     570% |
//  | BenchmarkCond2              |           |             | BenchmarkCond2                      |           |               |          |
//  | BenchmarkCond2-4            |   1367472 |   886 ns/op | BenchmarkCond2-4                    |    313716 |    3253 ns/op |     367% |
//  | BenchmarkCond4              |           |             | BenchmarkCond4                      |           |               |          |
//  | BenchmarkCond4-4            |    616113 |  1926 ns/op | BenchmarkCond4-4                    |    209254 |    6430 ns/op |     333% |
//  | BenchmarkCond8              |           |             | BenchmarkCond8                      |           |               |          |
//  | BenchmarkCond8-4            |    434738 |  2709 ns/op | BenchmarkCond8-4                    |     70065 |   14720 ns/op |     543% |
//  | BenchmarkCond16             |           |             | BenchmarkCond16                     |           |               |          |
//  | BenchmarkCond16-4           |    241214 |  4934 ns/op | BenchmarkCond16-4                   |     13461 |   91578 ns/op |   1,856% |
//  | BenchmarkCond32             |           |             | BenchmarkCond32                     |           |               |          |
//  | BenchmarkCond32-4           |    104055 | 10464 ns/op | BenchmarkCond32-4                   |      1280 | 1032153 ns/op |   9,863% |
//  | BenchmarkMutexUncontended   |           |             | BenchmarkMutexUncontended           |           |               |          |
//  | BenchmarkMutexUncontended-4 | 142903719 |  8.44 ns/op | BenchmarkMutexUncontended-4         |   7308866 |     145 ns/op |   1,718% |
//  | BenchmarkMutex              |           |             | BenchmarkMutex                      |           |               |          |
//  | BenchmarkMutex-4            |  16354251 |  78.8 ns/op | BenchmarkMutex-4                    |    803186 |    2204 ns/op |   2,796% |
//  | BenchmarkMutexSlack         |           |             | BenchmarkMutexSlack                 |           |               |          |
//  | BenchmarkMutexSlack-4       |   8745819 |   154 ns/op | BenchmarkMutexSlack-4               |     10000 |  140796 ns/op |  91,425% |
//  | BenchmarkMutexWork          |           |             | BenchmarkMutexWork                  |           |               |          |
//  | BenchmarkMutexWork-4        |  12567664 |   101 ns/op | BenchmarkMutexWork-4                |   1085060 |    1131 ns/op |   1,119% |
//  | BenchmarkMutexWorkSlack     |           |             | BenchmarkMutexWorkSlack             |           |               |          |
//  | BenchmarkMutexWorkSlack-4   |   9121546 |   149 ns/op | BenchmarkMutexWorkSlack-4           |      7290 |  150538 ns/op | 101,032% |
//  | BenchmarkMutexNoSpin        |           |             | BenchmarkMutexNoSpin                |           |               |          |
//  | BenchmarkMutexNoSpin-4      |   1626068 |   748 ns/op | BenchmarkMutexNoSpin-4              |   1000000 |    2194 ns/op |     293% |
//  | BenchmarkMutexSpin          |           |             | BenchmarkMutexSpin                  |           |               |          |
//  | BenchmarkMutexSpin-4        |    379140 |  3254 ns/op | BenchmarkMutexSpin-4                |    315388 |    3678 ns/op |     113% |
//  | PASS                        |           |             | PASS                                |           |               |          |
//  | ok sync                     |   23.018s |             | ok github.com/datawire/dlib/dsync   |   33.981s |               |          |
//
// Those measurements are with Go 1.15.2.  I'm actually seeing large dsync speedups with Go 1.17,
// I'll be excited to see how they compare when we upgrade dlib's reference Go version to Go 1.17.
//
// [1]: That said, stdlib sync has access to runtime internals that dsync doesn't have access to.
// Would dsync be faster if it had access to those?  You bet it would!  Would it be as fast as
// stdlib sync?  IDK, but probably not.  I suspect that the biggest difference in performance from
// having access to the runtime internals would be putting less pressure on the memory-allocator/GC.
package dsync

import (
	"context"
	"sync/atomic"
	"time"
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

	// Similar to stdlib sync.Mutex fairness, we have 2 modes: normal and starvation.
	//
	// No matter what, when you wait for a lock, you enter yourself in to a FIFO queue.
	//
	// In normal mode, that FIFO queue doesn't actually affect who gets the lock.  Instead, who
	// gets the lock is whichever goroutine the scheduler happens to decide to schedule.
	//
	// In starvation mode, it uses the FIFO queue.
	//
	// Waiters switch the Mutex to starvation mode when they are stuck waiting for more than
	// `starvationThresholdNs` (1ms).  When a waiter aquires the lock, it switches it back to
	// normal mode if either there are no more waiters or if it aquired the lock in less than
	// `starvationThresholdNs`.
	//
	// See the comments in stdlib sync.Mutex for why we do this.  The `TestMutexFairness`
	// demonstrates why FIFO mode is important: `go test -count=1 -run=TestMutexFairness` should
	// run the test in <15ms; but without starvation mode it takes more like 2s to 5s.  The
	// comments in stdlib sync.Mutex explain why non-FIFO mode is important.
	starving int32 // 0=normal, 1=starvation
	queue    atomicQueue

	noCopy    noCopyRuntime
	noCopyVet noCopyVet //nolint:structcheck // embedded for `go vet` purposes, not actually used
}

const starvationThresholdNs = 1e6 // 1ms

func runtime_nano() int64 {
	return time.Now().UnixNano()
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
	var entry atomicQueueEntry
	var waitStartTime int64

	for {
		if atomic.LoadInt32(&m.starving) == 0 || m.queue.Get() == &entry { // mode==normal || we're-next-in-the-queue
			// Try to grab the lock.
			if swapped := atomic.CompareAndSwapPointer(&m.ch, nil, unsafe.Pointer(&myCh)); swapped {
				// Yay, we got the lock.
				itemsStillQueued := m.queue.Remove(&entry)
				if itemsStillQueued == 0 || waitStartTime == 0 || runtime_nano()-waitStartTime < starvationThresholdNs {
					atomic.StoreInt32(&m.starving, 0)
				}
				return nil
			}
		}
		// Prepare to wait for the lock to get released.
		theirCh := (*chan struct{})(atomic.LoadPointer(&m.ch))
		if waitStartTime == 0 {
			waitStartTime = runtime_nano()
			m.queue.Add(&entry)
		} else if waitStartTime > starvationThresholdNs {
			atomic.StoreInt32(&m.starving, 1)
		}
		if theirCh == nil {
			// The lock got released in the time since we tried to get it; try again.
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
