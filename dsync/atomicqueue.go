package dsync

import (
	"sync"
)

// TODO(lukeshu): Rework atomicQueue to use sync/atomic rather than sync.Mutex.
//
// But don't juse use atomic to implement a spin lock, the benchmarks tell me that actually performs
// much worse:
//
//     type spinMutex struct {
//         v int32
//     }
//
//     func (mu *spinMutex) Lock() {
//         for !atomic.CompareAndSwapInt32(&mu.v, 0, 1) {
//         }
//     }
//
//     func (mu *spinMutex) Unlock() {
//         if !atomic.CompareAndSwapInt32(&mu.v, 1, 0) {
//             panic("dsync.spinMutex.Unlock: not locked")
//         }
//     }

// atomicQueue is a FIFO queue that is thread-safe.
type atomicQueue struct {
	mu   sync.Mutex
	root atomicQueueEntry
	len  int
}

type atomicQueueEntry struct {
	next, prev *atomicQueueEntry
}

func (q *atomicQueue) init() {
	if q.root.next == nil {
		q.root.next = &q.root
		q.root.prev = &q.root
	}
}

// does NOT remove the entry
func (q *atomicQueue) Get() *atomicQueueEntry {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.init()

	if q.len == 0 {
		return nil
	}
	return q.root.next
}

func (q *atomicQueue) Add(e *atomicQueueEntry) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.init()

	e.prev = q.root.prev
	e.next = &q.root
	e.prev.next = e
	e.next.prev = e
	q.len++
}

func (q *atomicQueue) Remove(e *atomicQueueEntry) int {
	q.mu.Lock()
	defer q.mu.Unlock()
	if e.next == nil {
		return q.len
	}
	q.init()

	e.prev.next = e.next
	e.next.prev = e.prev
	e.next = nil
	e.prev = nil
	q.len--
	return q.len
}
