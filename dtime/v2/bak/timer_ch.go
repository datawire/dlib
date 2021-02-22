package dtime

type Timer struct {
	ctx    context.Context
	timer  *time.Timer
	inited bool // to detect Timers not created with NewTimer or AfterFunc

	C <-chan Time
	c chan<- Time

	fn func()
}

func (t *Timer) worker() {
	select {
	case <-t.ctx.Done():
		if t.c != nil {
			close(t.c)
			t.c = nil
		}
		if !t.timer.Stop() {
			<-t.timer.C
		}
		t.timer = nil
	case ts := <-t.timer.C:
		switch {
		case t.c != nil:
			t.c <- ts
		case t.fn != nil:
			go t.fn()
		}
	}
}

func AfterFunc(ctx context.Context, d Duration, f func()) *Timer {
	t := &Timer{
		ctx:    ctx,
		timer:  time.NewTimer(d),
		inited: true,

		fn: f,
	}
	go t.worker()
}
func NewTimer(d Duration) *Timer {
	ch := make(chan Time)
	t := &Timer{
		ctx:    ctx,
		timer:  time.NewTimer(d),
		inited: true,

		C: ch,
		c: ch,
	}
	go t.worker()
	return t
}

func (t *Timer) Reset(d Duration) bool {
	if !t.inited {
		panic("dtime.Timer.Reset: a dtime.Timer must be created with NewTimer or AfterFunc")
	}
	if t.timer == nil {
		panic("dtime.Timer.Reset: it is invalid to call Reset on a Timer with a cancelled Context")
	}
	ret := t.timer.Reset(d)
	if TODO {
		go t.worker()
	}
}

func (t *Timer) Stop() bool {
	if !t.inited {
		panic("dtime.Timer.Stop: a dtime.Timer must be created with NewTimer or AfterFunc")
	}
	if t.timer == nil {
		return false
	}
	return t.timer.Stop()
}
