package raft

import (
	"testing"

	"goraft/internal/raftpb"
)

func TestLogSliceTermAndRestore(t *testing.T) {
	lg := NewLog()
	lg.Append(
		raftpb.Entry{Term: 1, Index: 1, Data: []byte("a")},
		raftpb.Entry{Term: 1, Index: 2, Data: []byte("b")},
		raftpb.Entry{Term: 2, Index: 3, Data: []byte("c")},
	)
	got := lg.Slice(2, 3)
	if len(got) != 2 || got[0].Index != 2 {
		t.Fatalf("slice=%v", got)
	}
	term, ok := lg.Term(0)
	if !ok || term != 0 {
		t.Fatal("index 0")
	}
	lg.Restore(3, 2, nil)
	if lg.LastIndex() != 3 || lg.LastTerm() != 2 || lg.Len() != 0 {
		t.Fatalf("restore last=%d term=%d len=%d", lg.LastIndex(), lg.LastTerm(), lg.Len())
	}
	if _, ok := lg.At(1); ok {
		t.Fatal("compacted entry still visible")
	}
}

func TestEncodeDecodeEntryRejectsZeroIndex(t *testing.T) {
	if _, err := encodeEntry(raftpb.Entry{Term: 1}); err == nil {
		t.Fatal("expected error")
	}
	if _, err := decodeEntry(nil); err == nil {
		t.Fatal("empty")
	}
	b, err := encodeEntry(raftpb.Entry{Term: 3, Index: 9, Type: raftpb.EntryNoop})
	if err != nil {
		t.Fatal(err)
	}
	e, err := decodeEntry(b)
	if err != nil || e.Index != 9 || e.Term != 3 {
		t.Fatalf("%+v %v", e, err)
	}
}

func TestMembershipHelpers(t *testing.T) {
	m := raftpb.Membership{Voters: []raftpb.NodeID{"n1", "n2", "n3"}, Learners: []raftpb.NodeID{"n4"}}
	if !m.IsVoter("n1") || !m.IsLearner("n4") || !m.Contains("n2") || m.Quorum() != 2 {
		t.Fatalf("%+v q=%d", m, m.Quorum())
	}
	c := m.Clone()
	c.Voters[0] = "xx"
	if m.Voters[0] != "n1" {
		t.Fatal("clone alias")
	}
	if len(m.All()) != 4 {
		t.Fatal(m.All())
	}
	if raftpb.StateLeader.String() != "leader" || raftpb.StateFollower.String() != "follower" {
		t.Fatal(raftpb.StateLeader.String())
	}
}
