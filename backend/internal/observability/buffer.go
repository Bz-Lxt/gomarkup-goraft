package observability

import "sync"

type Ring struct {
	mu   sync.RWMutex
	buf  []Event
	head int
	size int
	cap  int
}

func NewRing(n int) *Ring {
	if n <= 0 {
		n = 4096
	}
	return &Ring{buf: make([]Event, n), cap: n}
}

func (r *Ring) Push(ev Event) {
	r.mu.Lock()
	r.buf[r.head] = ev
	r.head = (r.head + 1) % r.cap
	if r.size < r.cap {
		r.size++
	}
	r.mu.Unlock()
}

func (r *Ring) Snapshot(afterUS int64, limit int) []Event {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if limit <= 0 || limit > r.size {
		limit = r.size
	}
	out := make([]Event, 0, limit)
	start := 0
	if r.size == r.cap {
		start = r.head
	}
	for i := 0; i < r.size; i++ {
		ev := r.buf[(start+i)%r.cap]
		if ev.TSUnixUS <= afterUS {
			continue
		}
		out = append(out, ev)
		if len(out) >= limit {
			break
		}
	}
	return out
}

func (r *Ring) Latest(n int) []Event {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if n <= 0 || n > r.size {
		n = r.size
	}
	out := make([]Event, 0, n)
	start := 0
	if r.size == r.cap {
		start = r.head
	}
	from := r.size - n
	for i := from; i < r.size; i++ {
		out = append(out, r.buf[(start+i)%r.cap])
	}
	return out
}
