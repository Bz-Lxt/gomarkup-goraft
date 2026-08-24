package observability

import (
	"crypto/rand"
	"encoding/hex"
	"sync"

	"goraft/internal/clock"
)

type Span struct {
	Name     string `json:"name"`
	NodeID   string `json:"node_id"`
	TSUnixUS int64  `json:"ts_unix_us"`
	TS       string `json:"ts"`
	Detail   string `json:"detail,omitempty"`
	Duration int64  `json:"duration_us"`
}

type Trace struct {
	TraceID string `json:"trace_id"`
	Key     string `json:"key,omitempty"`
	Op      string `json:"op,omitempty"`
	Spans   []Span `json:"spans"`
	Done    bool   `json:"done"`
}

type Tracer struct {
	mu     sync.Mutex
	active map[string]*Trace
	done   *RingTraces
}

type RingTraces struct {
	mu   sync.RWMutex
	buf  []*Trace
	head int
	size int
	cap  int
}

func NewTracer() *Tracer {
	return &Tracer{active: map[string]*Trace{}, done: NewRingTraces(256)}
}

func NewTraceID() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return "tr_" + hex.EncodeToString(b[:])
}

func (t *Tracer) Start(id, op, key string) *Trace {
	if id == "" {
		id = NewTraceID()
	}
	tr := &Trace{TraceID: id, Op: op, Key: key, Spans: []Span{}}
	t.mu.Lock()
	t.active[id] = tr
	t.mu.Unlock()
	return tr
}

func (t *Tracer) Span(id, name, node, detail string) {
	now := clock.Now()
	sp := Span{
		Name:     name,
		NodeID:   node,
		TSUnixUS: now.UnixMicro(),
		TS:       clock.FormatPrecise(now),
		Detail:   detail,
	}
	t.mu.Lock()
	tr := t.active[id]
	if tr != nil {
		if n := len(tr.Spans); n > 0 {
			sp.Duration = sp.TSUnixUS - tr.Spans[n-1].TSUnixUS
		}
		tr.Spans = append(tr.Spans, sp)
	}
	t.mu.Unlock()
}

func (t *Tracer) Finish(id string) *Trace {
	t.mu.Lock()
	tr := t.active[id]
	if tr != nil {
		tr.Done = true
		delete(t.active, id)
		t.done.Push(tr)
	}
	t.mu.Unlock()
	return tr
}

func (t *Tracer) Get(id string) *Trace {
	t.mu.Lock()
	tr := t.active[id]
	t.mu.Unlock()
	if tr != nil {
		return tr
	}
	return t.done.Find(id)
}

func (t *Tracer) Recent(n int) []*Trace {
	return t.done.Latest(n)
}

func NewRingTraces(n int) *RingTraces {
	if n <= 0 {
		n = 64
	}
	return &RingTraces{buf: make([]*Trace, n), cap: n}
}

func (r *RingTraces) Push(tr *Trace) {
	r.mu.Lock()
	r.buf[r.head] = tr
	r.head = (r.head + 1) % r.cap
	if r.size < r.cap {
		r.size++
	}
	r.mu.Unlock()
}

func (r *RingTraces) Latest(n int) []*Trace {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if n <= 0 || n > r.size {
		n = r.size
	}
	out := make([]*Trace, 0, n)
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

func (r *RingTraces) Find(id string) *Trace {
	r.mu.RLock()
	defer r.mu.RUnlock()
	start := 0
	if r.size == r.cap {
		start = r.head
	}
	for i := 0; i < r.size; i++ {
		tr := r.buf[(start+i)%r.cap]
		if tr != nil && tr.TraceID == id {
			return tr
		}
	}
	return nil
}
