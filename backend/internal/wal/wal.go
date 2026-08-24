package wal

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type WAL struct {
	dir      string
	maxSeg   int64
	mu       sync.Mutex
	active   *segment
	nextID   uint64
	waiters  []*waiter
	dirty    chan struct{}
	stop     chan struct{}
	wg       sync.WaitGroup
	closed   bool
	closeSt  closeState
	lastSync int64
}

func Open(dir string, maxSeg int64, flushWindow time.Duration) (*WAL, Recovered, error) {
	if maxSeg <= 0 {
		maxSeg = 64 << 20
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, Recovered{}, err
	}
	rec, lastID, err := recoverDir(dir)
	if err != nil {
		return nil, Recovered{}, err
	}
	w := &WAL{
		dir:    dir,
		maxSeg: maxSeg,
		nextID: lastID,
		dirty:  make(chan struct{}, 1),
		stop:   make(chan struct{}),
	}
	if lastID == 0 {
		w.nextID = 1
		sg, err := openSegment(dir, 1, true)
		if err != nil {
			return nil, Recovered{}, err
		}
		w.active = sg
	} else {
		sg, err := openSegment(dir, lastID, false)
		if err != nil {
			return nil, Recovered{}, err
		}
		w.active = sg
		w.nextID = lastID
	}
	w.wg.Add(1)
	go w.flusher(flushWindow)
	return w, rec, nil
}

func (w *WAL) Append(rec Record) error {
	return w.AppendSync(rec, false)
}

func (w *WAL) AppendSync(rec Record, wait bool) error {
	if rec.Type == 0 {
		return fmt.Errorf("%w: missing record type", ErrCorrupt)
	}
	w.mu.Lock()
	if w.closed {
		w.mu.Unlock()
		return ErrClosed
	}
	if w.active.size >= w.maxSeg {
		if err := w.rotateLocked(); err != nil {
			w.mu.Unlock()
			return err
		}
	}
	if err := w.active.append(rec); err != nil {
		w.mu.Unlock()
		return err
	}
	var wt *waiter
	if wait {
		wt = w.addWaiter()
	}
	w.mu.Unlock()
	w.notifyDirty()
	if !wait {
		return nil
	}
	return <-wt.errc
}

func (w *WAL) Sync() error {
	w.mu.Lock()
	if w.closed {
		w.mu.Unlock()
		return ErrClosed
	}
	wt := w.addWaiter()
	w.mu.Unlock()
	w.notifyDirty()
	return <-wt.errc
}

func (w *WAL) AppendHardState(payload []byte) error {
	return w.AppendSync(Record{Type: RecHardState, Payload: payload}, true)
}

func (w *WAL) AppendEntry(payload []byte, sync bool) error {
	return w.AppendSync(Record{Type: RecEntry, Payload: payload}, sync)
}

func (w *WAL) AppendCommit(index uint64) error {
	var b [8]byte
	binary.LittleEndian.PutUint64(b[:], index)
	return w.Append(Record{Type: RecCommit, Payload: b[:]})
}

func (w *WAL) rotateLocked() error {
	if err := w.active.sync(); err != nil {
		return err
	}
	if err := w.active.close(); err != nil {
		return err
	}
	w.nextID++
	sg, err := openSegment(w.dir, w.nextID, true)
	if err != nil {
		return err
	}
	w.active = sg
	return nil
}

func (w *WAL) TruncatePrefix(keepFromSegment uint64) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	ids, err := listSegmentIDs(w.dir)
	if err != nil {
		return err
	}
	for _, id := range ids {
		if id >= w.nextID {
			break
		}
		if id < keepFromSegment {
			_ = os.Remove(filepath.Join(w.dir, segmentName(id)))
		}
	}
	return nil
}

func (w *WAL) Dir() string { return w.dir }

func (w *WAL) Close() error {
	w.closeSt.once.Do(func() {
		w.mu.Lock()
		w.closed = true
		w.mu.Unlock()
		select {
		case <-w.stop:
		default:
			close(w.stop)
		}
		w.wg.Wait()
		w.mu.Lock()
		if w.active != nil {
			_ = w.active.sync()
			w.closeSt.err = w.active.close()
			w.active = nil
		}
		w.mu.Unlock()
	})
	return w.closeSt.err
}
