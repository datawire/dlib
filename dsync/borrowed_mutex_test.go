// MODIFIED: META: This file is a verbatim copy of Go 1.15.14 sync/mutex_test.go,
// MODIFIED: META: except for lines marked "MODIFIED".

// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// GOMAXPROCS=10 go test

package dsync_test // MODIFIED: FROM: package sync_test

import (
	"context" // MODIFIED: ADDED
	"fmt"
	"github.com/datawire/dlib/dlog"             // MODIFIED: ADDED
	. "github.com/datawire/dlib/dsync"          // MODIFIED: FROM: . "sync"
	"github.com/datawire/dlib/internal/testenv" // MODIFIED: FROM: "internal/testenv"
	"github.com/stretchr/testify/assert"        // MODIFIED: ADDED
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"
	"time"
)

/* // MODIFIED: ADDED
func HammerSemaphore(s *uint32, loops int, cdone chan bool) {
	for i := 0; i < loops; i++ {
		Runtime_Semacquire(s)
		Runtime_Semrelease(s, false, 0)
	}
	cdone <- true
}

func TestSemaphore(t *testing.T) {
	s := new(uint32)
	*s = 1
	c := make(chan bool)
	for i := 0; i < 10; i++ {
		go HammerSemaphore(s, 1000, c)
	}
	for i := 0; i < 10; i++ {
		<-c
	}
}

func BenchmarkUncontendedSemaphore(b *testing.B) {
	s := new(uint32)
	*s = 1
	HammerSemaphore(s, b.N, make(chan bool, 2))
}

func BenchmarkContendedSemaphore(b *testing.B) {
	b.StopTimer()
	s := new(uint32)
	*s = 1
	c := make(chan bool)
	defer runtime.GOMAXPROCS(runtime.GOMAXPROCS(2))
	b.StartTimer()

	go HammerSemaphore(s, b.N/2, c)
	go HammerSemaphore(s, b.N/2, c)
	<-c
	<-c
}
*/ // MODIFIED: ADDED

func HammerMutex(t testing.TB, m *Mutex, loops int, cdone chan bool) { // MODIFIED: FROM: func HammerMutex(m *Mutex, loops int, cdone chan bool) {
	ctx := dlog.NewTestContext(t, true) // MODIFIED: ADDED
	for i := 0; i < loops; i++ {
		assert.NoError(t, m.Lock(ctx)) // MODIFIED: FROM: m.Lock()
		m.Unlock()
	}
	cdone <- true
}

func TestMutex(t *testing.T) {
	if n := runtime.SetMutexProfileFraction(1); n != 0 {
		t.Logf("got mutexrate %d expected 0", n)
	}
	defer runtime.SetMutexProfileFraction(0)
	m := new(Mutex)
	c := make(chan bool)
	for i := 0; i < 10; i++ {
		go HammerMutex(t, m, 1000, c) // MODIFIED: FROM: go HammerMutex(m, 1000, c)
	}
	for i := 0; i < 10; i++ {
		<-c
	}
}

var misuseTests = []struct {
	name string
	f    func(context.Context) // MODIFIED: FROM: f func()
}{
	{
		"Mutex.Unlock",
		func(_ context.Context) { // MODIFIED: FROM: func() {
			var mu Mutex
			mu.Unlock()
		},
	},
	{
		"Mutex.Unlock2",
		func(ctx context.Context) { // MODIFIED: FROM: func() {
			var mu Mutex
			if err := mu.Lock(ctx); err != nil { // MODIFIED: FROM: mu.Lock()
				panic(err) // MODIFIED: ADDED
			} // MODIFIED: ADDED
			mu.Unlock()
			mu.Unlock()
		},
	},
	/* // MODIFIED: ADDED
	{
		"RWMutex.Unlock",
		func() {
			var mu RWMutex
			mu.Unlock()
		},
	},
	{
		"RWMutex.Unlock2",
		func() {
			var mu RWMutex
			mu.RLock()
			mu.Unlock()
		},
	},
	{
		"RWMutex.Unlock3",
		func() {
			var mu RWMutex
			mu.Lock()
			mu.Unlock()
			mu.Unlock()
		},
	},
	{
		"RWMutex.RUnlock",
		func() {
			var mu RWMutex
			mu.RUnlock()
		},
	},
	{
		"RWMutex.RUnlock2",
		func() {
			var mu RWMutex
			mu.Lock()
			mu.RUnlock()
		},
	},
	{
		"RWMutex.RUnlock3",
		func() {
			var mu RWMutex
			mu.RLock()
			mu.RUnlock()
			mu.RUnlock()
		},
	},
	*/ // MODIFIED: ADDED
}

func init() {
	ctx := context.Background() // MODIFIED: ADDED
	if len(os.Args) == 3 && os.Args[1] == "TESTMISUSE" {
		for _, test := range misuseTests {
			if test.name == os.Args[2] {
				func() {
					//defer func() { recover() }() // MODIFIED: FROM: defer func() { recover() }()
					test.f(ctx) // MODIFIED: FROM: test.f()
				}()
				fmt.Printf("test completed\n")
				os.Exit(0)
			}
		}
		fmt.Printf("unknown test\n")
		os.Exit(0)
	}
}

func TestMutexMisuse(t *testing.T) {
	testenv.MustHaveExec(t)
	for _, test := range misuseTests {
		out, err := exec.Command(os.Args[0], "TESTMISUSE", test.name).CombinedOutput()
		if err == nil || !strings.Contains(string(out), "not locked") { // MODIFIED: FROM: if err == nil || !strings.Contains(string(out), "unlocked") {
			t.Errorf("%s: did not find failure with message about unlocked lock: %s\n%s\n", test.name, err, out)
		}
	}
}

func TestMutexFairness(t *testing.T) {
	ctx := dlog.NewTestContext(t, true) // MODIFIED: ADDED
	var mu Mutex
	stop := make(chan bool)
	defer close(stop)
	go func() {
		for {
			assert.NoError(t, mu.Lock(ctx)) // MODIFIED: FROM: mu.Lock()
			time.Sleep(100 * time.Microsecond)
			mu.Unlock()
			select {
			case <-stop:
				return
			default:
			}
		}
	}()
	done := make(chan bool)
	go func() {
		for i := 0; i < 10; i++ {
			time.Sleep(100 * time.Microsecond)
			assert.NoError(t, mu.Lock(ctx)) // MODIFIED: FROM: mu.Lock()
			mu.Unlock()
		}
		done <- true
	}()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatalf("can't acquire Mutex in 10 seconds")
	}
}

func BenchmarkMutexUncontended(b *testing.B) {
	type PaddedMutex struct {
		Mutex
		pad [128]uint8
	}
	ctx := dlog.NewTestContext(b, true) // MODIFIED: ADDED
	b.RunParallel(func(pb *testing.PB) {
		var mu PaddedMutex
		for pb.Next() {
			assert.NoError(b, mu.Lock(ctx)) // MODIFIED: FROM: mu.Lock()
			mu.Unlock()
		}
	})
}

func benchmarkMutex(b *testing.B, slack, work bool) {
	var mu Mutex
	if slack {
		b.SetParallelism(10)
	}
	ctx := dlog.NewTestContext(b, true) // MODIFIED: ADDED
	b.RunParallel(func(pb *testing.PB) {
		foo := 0
		for pb.Next() {
			assert.NoError(b, mu.Lock(ctx)) // MODIFIED: FROM: mu.Lock()
			mu.Unlock()
			if work {
				for i := 0; i < 100; i++ {
					foo *= 2
					foo /= 2
				}
			}
		}
		_ = foo
	})
}

func BenchmarkMutex(b *testing.B) {
	benchmarkMutex(b, false, false)
}

func BenchmarkMutexSlack(b *testing.B) {
	benchmarkMutex(b, true, false)
}

func BenchmarkMutexWork(b *testing.B) {
	benchmarkMutex(b, false, true)
}

func BenchmarkMutexWorkSlack(b *testing.B) {
	benchmarkMutex(b, true, true)
}

func BenchmarkMutexNoSpin(b *testing.B) {
	// This benchmark models a situation where spinning in the mutex should be
	// non-profitable and allows to confirm that spinning does not do harm.
	// To achieve this we create excess of goroutines most of which do local work.
	// These goroutines yield during local work, so that switching from
	// a blocked goroutine to other goroutines is profitable.
	// As a matter of fact, this benchmark still triggers some spinning in the mutex.
	var m Mutex
	var acc0, acc1 uint64
	b.SetParallelism(4)
	ctx := dlog.NewTestContext(b, true) // MODIFIED: ADDED
	b.RunParallel(func(pb *testing.PB) {
		c := make(chan bool)
		var data [4 << 10]uint64
		for i := 0; pb.Next(); i++ {
			if i%4 == 0 {
				assert.NoError(b, m.Lock(ctx)) // MODIFIED: FROM: m.Lock()
				acc0 -= 100
				acc1 += 100
				m.Unlock()
			} else {
				for i := 0; i < len(data); i += 4 {
					data[i]++
				}
				// Elaborate way to say runtime.Gosched
				// that does not put the goroutine onto global runq.
				go func() {
					c <- true
				}()
				<-c
			}
		}
	})
}

func BenchmarkMutexSpin(b *testing.B) {
	// This benchmark models a situation where spinning in the mutex should be
	// profitable. To achieve this we create a goroutine per-proc.
	// These goroutines access considerable amount of local data so that
	// unnecessary rescheduling is penalized by cache misses.
	var m Mutex
	var acc0, acc1 uint64
	ctx := dlog.NewTestContext(b, true) // MODIFIED: ADDED
	b.RunParallel(func(pb *testing.PB) {
		var data [16 << 10]uint64
		for i := 0; pb.Next(); i++ {
			assert.NoError(b, m.Lock(ctx)) // MODIFIED: FROM: m.Lock()
			acc0 -= 100
			acc1 += 100
			m.Unlock()
			for i := 0; i < len(data); i += 4 {
				data[i]++
			}
		}
	})
}
