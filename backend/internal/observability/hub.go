package observability

import "sync"

// Fanout is a lock-safe broadcaster used by the WS layer.
type Fanout struct {
	mu   sync.RWMutex
	subs map[int]chan Event
	next int
}

func NewFanout() *Fanout {
	return &Fanout{subs: map[int]chan Event{}}
}

func (f *Fanout) Add() (int, <-chan Event) {
	ch := make(chan Event, 256)
	f.mu.Lock()
	f.next++
	id := f.next
	f.subs[id] = ch
	f.mu.Unlock()
	return id, ch
}

func (f *Fanout) Remove(id int) {
	f.mu.Lock()
	if ch, ok := f.subs[id]; ok {
		delete(f.subs, id)
		close(ch)
	}
	f.mu.Unlock()
}

func (f *Fanout) Broadcast(ev Event) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	for _, ch := range f.subs {
		select {
		case ch <- ev:
		default:
		}
	}
}
