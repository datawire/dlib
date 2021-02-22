package dtime

import (
	"sync"
	"time"
)

type Timer struct {
	C  <-chan Time
	c  chan<- Time
	fn func()

	mu          sync.Mutex
	inited      bool
	ctx         context.Context
	timer       *time.Timer
	stopWaiting chan struct{}
	waiting     bool
}

func (t *Timer) wait() {
	select {
	case <-ctx.Done():
		t.mu.Lock()
		defer t.mu.Unlock()
		t.timer.Stop()
		if t.c != nil {
			close(t.c)
		}
		t.c = nil
		t.timer = nil
		t.stopWaiting = nil
	case <-t.stopWaiting:
		t.mu.Lock()
		defer t.mu.Unlock()
		t.waiting = false
	}
}

func (t *Timer) fire() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.timer == nil {
		return
	}
	t.stopWaiting <- struct{}{}

	if t.c != nil {
		go func() { t.c <- Now(t.ctx) }()
	}
	if t.fn != nil {
		go fn()
	}
}

func AfterFunc(ctx context.Context, d Duration, f func()) *Timer {
	t := &Timer{
		fn: f,

		inited:      true,
		ctx:         ctx,
		stopWaiting: make(chan struct{}),
		waiting:     true,
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	t.timer = time.AfterFunc(d, t.fire())
	go t.wait()
	return t
}

func NewTimer(d Duration) *Timer {
	ch := make(chan Time)
	t := &Timer{
		C: ch,
		c: ch,

		inited:      true,
		ctx:         ctx,
		stopWaiting: make(chan struct{}),
		waiting:     true,
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	t.timer = time.AfterFunc(d, t.fire())
	go t.wait()
	return t
}

func (t *Timer) Reset(d Duration) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	if !t.inited {
		panic("dtime: Reset called on uninitialized Timer")
	}
	if t.timer == nil {
		panic("dtime: Reset called on Timer with cancelled Context")
	}
	if t.ctx.Err() != nil {
		panic("dtime: Reset called on Timer with cancelled Context")
	}
	if !t.waiting {
		t.waiting = true
		go t.wait()
	}
	return t.timer.Reset(d)
}

func (t *Timer) Stop() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	if !t.inited {
		panic("dtime: Stop called on uninitialized Timer")
	}
	if t.timer == nil {
		return false
	}
	t.stopWaiting <- struct{}{}
	return t.timer.Stop()
}
