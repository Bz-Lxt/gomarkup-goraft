package observability

import (
	"sync"
)

type Subscriber func(Event)

type Bus struct {
	mu   sync.RWMutex
	subs []Subscriber
}

func NewBus() *Bus { return &Bus{} }

func (b *Bus) Subscribe(fn Subscriber) {
	if fn == nil {
		return
	}
	b.mu.Lock()
	b.subs = append(b.subs, fn)
	b.mu.Unlock()
}

func (b *Bus) Publish(ev Event) {
	b.mu.RLock()
	subs := append([]Subscriber(nil), b.subs...)
	b.mu.RUnlock()
	for _, fn := range subs {
		fn(ev)
	}
}
