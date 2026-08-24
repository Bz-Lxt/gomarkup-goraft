package clock

import (
	"sync"
	"time"
)

// Beijing is GMT+8, the only civil timezone used by this process.
var Beijing = time.FixedZone("CST", 8*3600)

func Now() time.Time {
	return time.Now().In(Beijing)
}

func NowUnixMicro() int64 {
	return Now().UnixMicro()
}

// FormatDisplay returns yyyy-MM-dd HH:mm:ss for UI / log display.
func FormatDisplay(t time.Time) string {
	return t.In(Beijing).Format("2006-01-02 15:04:05")
}

func FormatPrecise(t time.Time) string {
	return t.In(Beijing).Format("2006-01-02 15:04:05.000000")
}

func FormatNow() string {
	return FormatDisplay(Now())
}

func FormatNowPrecise() string {
	return FormatPrecise(Now())
}

// Clock abstracts time for tests.
type Clock interface {
	Now() time.Time
	Since(t time.Time) time.Duration
	NewTicker(d time.Duration) Ticker
}

type Ticker interface {
	C() <-chan time.Time
	Stop()
}

type realClock struct{}

func Real() Clock { return realClock{} }

func (realClock) Now() time.Time { return Now() }

func (realClock) Since(t time.Time) time.Duration {
	return Now().Sub(t.In(Beijing))
}

func (realClock) NewTicker(d time.Duration) Ticker {
	t := time.NewTicker(d)
	return &realTicker{t: t}
}

type realTicker struct{ t *time.Ticker }

func (t *realTicker) C() <-chan time.Time { return t.t.C }
func (t *realTicker) Stop()               { t.t.Stop() }

// Manual is a test clock.
type Manual struct {
	mu   sync.Mutex
	now  time.Time
	tick chan time.Time
}

func NewManual(start time.Time) *Manual {
	if start.IsZero() {
		start = time.Date(2026, 8, 24, 15, 0, 0, 0, Beijing)
	}
	return &Manual{now: start.In(Beijing), tick: make(chan time.Time, 64)}
}

func (m *Manual) Now() time.Time {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.now
}

func (m *Manual) Since(t time.Time) time.Duration {
	return m.Now().Sub(t)
}

func (m *Manual) Advance(d time.Duration) {
	m.mu.Lock()
	m.now = m.now.Add(d)
	now := m.now
	m.mu.Unlock()
	select {
	case m.tick <- now:
	default:
	}
}

func (m *Manual) NewTicker(d time.Duration) Ticker {
	return &manualTicker{ch: m.tick, d: d}
}

type manualTicker struct {
	ch <-chan time.Time
	d  time.Duration
}

func (t *manualTicker) C() <-chan time.Time { return t.ch }
func (t *manualTicker) Stop()               {}
