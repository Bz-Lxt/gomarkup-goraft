package wal

import (
	"sync"
	"time"
)

type waiter struct {
	errc chan error
}

func (w *WAL) flusher(window time.Duration) {
	defer w.wg.Done()
	if window <= 0 {
		window = 2 * time.Millisecond
	}
	t := time.NewTicker(window)
	defer t.Stop()
	for {
		select {
		case <-w.stop:
			w.flushLocked()
			return
		case <-w.dirty:
			w.flushLocked()
		case <-t.C:
			w.flushLocked()
		}
	}
}

func (w *WAL) flushLocked() {
	w.mu.Lock()
	if w.closed || w.active == nil || len(w.waiters) == 0 {
		w.mu.Unlock()
		return
	}
	waiters := w.waiters
	w.waiters = nil
	sg := w.active
	w.mu.Unlock()

	err := sg.sync()

	w.mu.Lock()
	if err == nil {
		w.lastSync = sg.size
	}
	w.mu.Unlock()

	for _, wt := range waiters {
		wt.errc <- err
	}
}

func (w *WAL) notifyDirty() {
	select {
	case w.dirty <- struct{}{}:
	default:
	}
}

func (w *WAL) addWaiter() *waiter {
	wt := &waiter{errc: make(chan error, 1)}
	w.waiters = append(w.waiters, wt)
	return wt
}

// CloseOnce documents the Close contract used by tests.
type closeState struct {
	once sync.Once
	err  error
}
