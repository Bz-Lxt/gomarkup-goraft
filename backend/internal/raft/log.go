package raft

import (
	"encoding/json"
	"fmt"

	"goraft/internal/raftpb"
)

type Log struct {
	snapIndex raftpb.Index
	snapTerm  raftpb.Term
	ents      []raftpb.Entry
}

func NewLog() *Log { return &Log{} }

func (l *Log) LastIndex() raftpb.Index {
	if n := len(l.ents); n > 0 {
		return l.ents[n-1].Index
	}
	return l.snapIndex
}

func (l *Log) LastTerm() raftpb.Term {
	if n := len(l.ents); n > 0 {
		return l.ents[n-1].Term
	}
	return l.snapTerm
}

func (l *Log) SnapIndex() raftpb.Index { return l.snapIndex }
func (l *Log) SnapTerm() raftpb.Term   { return l.snapTerm }
func (l *Log) Len() int                { return len(l.ents) }

func (l *Log) Term(i raftpb.Index) (raftpb.Term, bool) {
	if i == 0 {
		return 0, true
	}
	if i == l.snapIndex {
		return l.snapTerm, true
	}
	if i < l.snapIndex {
		return 0, false
	}
	off := int(i - l.snapIndex - 1)
	if off < 0 || off >= len(l.ents) {
		return 0, false
	}
	return l.ents[off].Term, true
}

func (l *Log) At(i raftpb.Index) (raftpb.Entry, bool) {
	if i <= l.snapIndex {
		return raftpb.Entry{}, false
	}
	off := int(i - l.snapIndex - 1)
	if off < 0 || off >= len(l.ents) {
		return raftpb.Entry{}, false
	}
	return l.ents[off], true
}

func (l *Log) Slice(lo, hi raftpb.Index) []raftpb.Entry {
	if hi < lo {
		return nil
	}
	if lo <= l.snapIndex {
		lo = l.snapIndex + 1
	}
	last := l.LastIndex()
	if hi > last {
		hi = last
	}
	if lo > hi {
		return nil
	}
	out := make([]raftpb.Entry, 0, hi-lo+1)
	for i := lo; i <= hi; i++ {
		e, ok := l.At(i)
		if !ok {
			break
		}
		out = append(out, e)
	}
	return out
}

func (l *Log) Append(ents ...raftpb.Entry) {
	for _, e := range ents {
		l.ents = append(l.ents, e)
	}
}

func (l *Log) TruncateFrom(index raftpb.Index) {
	if index <= l.snapIndex {
		l.ents = nil
		return
	}
	off := int(index - l.snapIndex - 1)
	if off < 0 {
		l.ents = nil
		return
	}
	if off >= len(l.ents) {
		return
	}
	l.ents = l.ents[:off]
}

func (l *Log) Compact(index raftpb.Index, term raftpb.Term) {
	if index < l.snapIndex {
		return
	}
	off := int(index - l.snapIndex)
	if off > 0 && off <= len(l.ents) {
		l.ents = append([]raftpb.Entry(nil), l.ents[off:]...)
	} else if index >= l.LastIndex() {
		l.ents = nil
	}
	l.snapIndex = index
	l.snapTerm = term
}

func (l *Log) Restore(index raftpb.Index, term raftpb.Term, ents []raftpb.Entry) {
	l.snapIndex = index
	l.snapTerm = term
	l.ents = append([]raftpb.Entry(nil), ents...)
}

func encodeEntry(e raftpb.Entry) ([]byte, error) {
	if e.Index == 0 {
		return nil, fmt.Errorf("entry index must be > 0")
	}
	return json.Marshal(e)
}

func decodeEntry(b []byte) (raftpb.Entry, error) {
	var e raftpb.Entry
	if len(b) == 0 {
		return e, fmt.Errorf("empty entry payload")
	}
	if err := json.Unmarshal(b, &e); err != nil {
		return e, err
	}
	if e.Index == 0 {
		return e, fmt.Errorf("decoded entry missing index")
	}
	return e, nil
}
