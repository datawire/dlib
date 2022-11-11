package dsync

import (
	"sync"
)

var chPool = &sync.Pool{
	New: func() interface{} {
		return make(chan struct{}, 1)
	},
}

func makeCh() chan struct{} {
	return chPool.Get().(chan struct{})
}

type bcaster struct {
	mu        sync.Mutex
	listeners map[<-chan struct{}]chan struct{}
}

func (b *bcaster) Subscribe() <-chan struct{} {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.listeners == nil {
		b.listeners = make(map[<-chan struct{}]chan struct{})
	}
	ch := makeCh()
	b.listeners[ch] = ch
	return ch
}

func (b *bcaster) Unsubscribe(ch <-chan struct{}) {
	b.mu.Lock()
	defer b.mu.Unlock()

drain:
	for {
		select {
		case _ = <-ch:
		default:
			break drain
		}
	}

	other := b.listeners[ch]
	chPool.Put(other)
	delete(b.listeners, ch)
}

func (b *bcaster) Broadcast() {
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, ch := range b.listeners {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}
