package raft

import (
	"testing"

	"goraft/internal/raftpb"
)

func TestLogMatchAndTruncate(t *testing.T) {
	lg := NewLog()
	lg.Append(
		raftpb.Entry{Term: 1, Index: 1},
		raftpb.Entry{Term: 1, Index: 2},
		raftpb.Entry{Term: 2, Index: 3},
	)
	if lg.LastIndex() != 3 || lg.LastTerm() != 2 {
		t.Fatalf("last %d/%d", lg.LastIndex(), lg.LastTerm())
	}
	lg.TruncateFrom(2)
	if lg.LastIndex() != 1 {
		t.Fatalf("after truncate %d", lg.LastIndex())
	}
	lg.Compact(1, 1)
	if lg.Len() != 0 || lg.SnapIndex() != 1 {
		t.Fatalf("compact %d len=%d", lg.SnapIndex(), lg.Len())
	}
}

func TestLogUpToDateRule(t *testing.T) {
	h := startCluster(t, []string{"n1"})
	n := h.waitLeader(t)
	if !n.logUpToDate(n.log.LastIndex(), n.log.LastTerm()) {
		t.Fatal("self log should be up to date")
	}
	if n.logUpToDate(0, 0) && n.log.LastIndex() > 0 {
		t.Fatal("empty log should not beat current")
	}
}
