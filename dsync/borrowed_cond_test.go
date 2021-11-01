// MODIFIED: META: This file is a verbatim copy of Go 1.15.14 sync/cond_test.go,
// MODIFIED: META: except for lines marked "MODIFIED".

// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package dsync_test // MODIFIED: FROM: package sync_test

import (
	"github.com/datawire/dlib/dlog"      // MODIFIED: ADDED
	. "github.com/datawire/dlib/dsync"   // MODIFIED: FROM: . "sync"
	"github.com/stretchr/testify/assert" // MODIFIED: ADDED
	"os"                                 // MODIFIED: ADDED
	"reflect"
	"runtime"
	stdsync "sync" // MODIFIED: ADDED
	"testing"
	"time"
)

func TestCondSignal(t *testing.T) {
	ctx := dlog.NewTestContext(t, true) // MODIFIED: ADDED
	var m Mutex
	c := NewCond(&m)
	n := 2
	running := make(chan bool, n)
	awake := make(chan bool, n)
	for i := 0; i < n; i++ {
		go func() {
			assert.NoError(t, m.Lock(ctx)) // MODIFIED: FROM: m.Lock()
			running <- true
			assert.NoError(t, c.Wait(ctx)) // MODIFIED: FROM: c.Wait()
			awake <- true
			m.Unlock()
		}()
	}
	for i := 0; i < n; i++ {
		<-running // Wait for everyone to run.
	}
	for n > 0 {
		select {
		case <-awake:
			t.Fatal("goroutine not asleep")
		default:
		}
		assert.NoError(t, m.Lock(ctx)) // MODIFIED: FROM: m.Lock()
		c.Signal()
		m.Unlock()
		<-awake // Will deadlock if no goroutine wakes up
		select {
		case <-awake:
			t.Fatal("too many goroutines awake")
		default:
		}
		n--
	}
	c.Signal()
}

func TestCondSignalGenerations(t *testing.T) {
	ctx := dlog.NewTestContext(t, true) // MODIFIED: ADDED
	var wg stdsync.WaitGroup            // MODIFIED: ADDED
	var m Mutex
	c := NewCond(&m)
	n := 100
	running := make(chan bool, n)
	awake := make(chan int, n)
	wg.Add(n) // MODIFIED: ADDED
	for i := 0; i < n; i++ {
		go func(i int) {
			assert.NoError(t, m.Lock(ctx)) // MODIFIED: FROM: m.Lock()
			running <- true
			assert.NoError(t, c.Wait(ctx)) // MODIFIED: FROM: c.Wait()
			awake <- i
			m.Unlock()
			wg.Done() // MODIFIED: ADDED
		}(i)
		if i > 0 {
			a := <-awake
			if a != i-1 {
				t.Fatalf("wrong goroutine woke up: want %d, got %d", i-1, a)
			}
		}
		<-running
		assert.NoError(t, m.Lock(ctx)) // MODIFIED: FROM: m.Lock()
		c.Signal()
		m.Unlock()
	}
	wg.Wait() // MODIFIED: ADDED
}

func TestCondBroadcast(t *testing.T) {
	if testRace && os.Getenv("CI") != "" {
		// This test is quite slow with the race detector turned on.  Slow enough that it
		// causes problems for CI.
		t.SkipNow()
	}
	ctx := dlog.NewTestContext(t, true) // MODIFIED: ADDED
	var m Mutex
	c := NewCond(&m)
	n := 200
	running := make(chan int, n)
	awake := make(chan int, n)
	exit := false
	for i := 0; i < n; i++ {
		go func(g int) {
			assert.NoError(t, m.Lock(ctx)) // MODIFIED: FROM: m.Lock()
			for !exit {
				running <- g
				assert.NoError(t, c.Wait(ctx)) // MODIFIED: FROM: c.Wait()
				awake <- g
			}
			m.Unlock()
		}(i)
	}
	for i := 0; i < n; i++ {
		for i := 0; i < n; i++ {
			<-running // Will deadlock unless n are running.
		}
		if i == n-1 {
			assert.NoError(t, m.Lock(ctx)) // MODIFIED: FROM: m.Lock()
			exit = true
			m.Unlock()
		}
		select {
		case <-awake:
			t.Fatal("goroutine not asleep")
		default:
		}
		assert.NoError(t, m.Lock(ctx)) // MODIFIED: FROM: m.Lock()
		c.Broadcast()
		m.Unlock()
		seen := make([]bool, n)
		for i := 0; i < n; i++ {
			g := <-awake
			if seen[g] {
				t.Fatal("goroutine woke up twice")
			}
			seen[g] = true
		}
	}
	select {
	case <-running:
		t.Fatal("goroutine did not exit")
	default:
	}
	c.Broadcast()
}

func TestRace(t *testing.T) {
	ctx := dlog.NewTestContext(t, true) // MODIFIED: ADDED
	x := 0
	c := NewCond(&Mutex{})
	done := make(chan bool)
	go func() {
		assert.NoError(t, c.L.Lock(ctx)) // MODIFIED: FROM: c.L.Lock()
		x = 1
		assert.NoError(t, c.Wait(ctx)) // MODIFIED: FROM: c.Wait()
		if x != 2 {
			t.Error("want 2")
		}
		x = 3
		c.Signal()
		c.L.Unlock()
		done <- true
	}()
	go func() {
		assert.NoError(t, c.L.Lock(ctx)) // MODIFIED: FROM: c.L.Lock()
		for {
			if x == 1 {
				x = 2
				c.Signal()
				break
			}
			c.L.Unlock()
			runtime.Gosched()
			assert.NoError(t, c.L.Lock(ctx)) // MODIFIED: FROM: c.L.Lock()
		}
		c.L.Unlock()
		done <- true
	}()
	go func() {
		assert.NoError(t, c.L.Lock(ctx)) // MODIFIED: FROM: c.L.Lock()
		for {
			if x == 2 {
				assert.NoError(t, c.Wait(ctx)) // MODIFIED: FROM: c.Wait()
				if x != 3 {
					t.Error("want 3")
				}
				break
			}
			if x == 3 {
				break
			}
			c.L.Unlock()
			runtime.Gosched()
			assert.NoError(t, c.L.Lock(ctx)) // MODIFIED: FROM: c.L.Lock()
		}
		c.L.Unlock()
		done <- true
	}()
	<-done
	<-done
	<-done
}

func TestCondSignalStealing(t *testing.T) {
	ctx := dlog.NewTestContext(t, true) // MODIFIED: ADDED
	var wg stdsync.WaitGroup            // MODIFIED: ADDED
	wg.Add(1000)                        // MODIFIED: ADDED
	for iters := 0; iters < 1000; iters++ {
		var m Mutex
		cond := NewCond(&m)

		// Start a waiter.
		ch := make(chan struct{})
		go func() {
			assert.NoError(t, m.Lock(ctx)) // MODIFIED: FROM: m.Lock()
			ch <- struct{}{}
			assert.NoError(t, cond.Wait(ctx)) // MODIFIED: FROM: cond.Wait()
			m.Unlock()

			ch <- struct{}{}
		}()

		<-ch
		assert.NoError(t, m.Lock(ctx)) // MODIFIED: FROM: m.Lock()
		m.Unlock()

		// We know that the waiter is in the cond.Wait() call because we
		// synchronized with it, then acquired/released the mutex it was
		// holding when we synchronized.
		//
		// Start two goroutines that will race: one will broadcast on
		// the cond var, the other will wait on it.
		//
		// The new waiter may or may not get notified, but the first one
		// has to be notified.
		done := false
		go func() {
			cond.Broadcast()
		}()

		go func() {
			assert.NoError(t, m.Lock(ctx)) // MODIFIED: FROM: m.Lock()
			for !done {
				assert.NoError(t, cond.Wait(ctx)) // MODIFIED: FROM: cond.Wait()
			}
			m.Unlock()
			wg.Done() // MODIFIED: ADDED
		}()

		// Check that the first waiter does get signaled.
		select {
		case <-ch:
		case <-time.After(2 * time.Second):
			t.Fatalf("First waiter didn't get broadcast.")
		}

		// Release the second waiter in case it didn't get the
		// broadcast.
		assert.NoError(t, m.Lock(ctx)) // MODIFIED: FROM: m.Lock()
		done = true
		m.Unlock()
		cond.Broadcast()
	}
	wg.Wait() // MODIFIED: ADDED
}

func TestCondCopy(t *testing.T) {
	defer func() {
		err := recover()
		exp := "dsync.Cond.Signal: cond was copied after first use" // MODIFIED: ADDED
		if err == nil || err.(string) != exp {                      // MODIFIED: FROM: if err == nil || err.(string) != "sync.Cond is copied" {
			t.Fatalf("got %q, expect %q", err, exp) // MODIFIED: FROM: t.Fatalf("got %v, expect sync.Cond is copied", err)
		}
	}()
	c := Cond{L: &Mutex{}}
	c.Signal()
	var c2 Cond
	reflect.ValueOf(&c2).Elem().Set(reflect.ValueOf(&c).Elem()) // c2 := c, hidden from vet
	c2.Signal()
}

func BenchmarkCond1(b *testing.B) {
	benchmarkCond(b, 1)
}

func BenchmarkCond2(b *testing.B) {
	benchmarkCond(b, 2)
}

func BenchmarkCond4(b *testing.B) {
	benchmarkCond(b, 4)
}

func BenchmarkCond8(b *testing.B) {
	benchmarkCond(b, 8)
}

func BenchmarkCond16(b *testing.B) {
	benchmarkCond(b, 16)
}

func BenchmarkCond32(b *testing.B) {
	benchmarkCond(b, 32)
}

func benchmarkCond(b *testing.B, waiters int) {
	ctx := dlog.NewTestContext(b, true) // MODIFIED: ADDED
	c := NewCond(&Mutex{})
	done := make(chan bool)
	id := 0

	for routine := 0; routine < waiters+1; routine++ {
		go func() {
			for i := 0; i < b.N; i++ {
				assert.NoError(b, c.L.Lock(ctx)) // MODIFIED: FROM: c.L.Lock()
				if id == -1 {
					c.L.Unlock()
					break
				}
				id++
				if id == waiters+1 {
					id = 0
					c.Broadcast()
				} else {
					assert.NoError(b, c.Wait(ctx)) // MODIFIED: FROM: c.Wait()
				}
				c.L.Unlock()
			}
			assert.NoError(b, c.L.Lock(ctx)) // MODIFIED: FROM: c.L.Lock()
			id = -1
			c.Broadcast()
			c.L.Unlock()
			done <- true
		}()
	}
	for routine := 0; routine < waiters+1; routine++ {
		<-done
	}
}
