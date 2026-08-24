package rpc

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"goraft/internal/raftpb"
)

type stub struct {
	vote raftpb.RequestVoteReply
}

func (s stub) RequestVote(args raftpb.RequestVoteArgs) raftpb.RequestVoteReply {
	return s.vote
}
func (s stub) AppendEntries(raftpb.AppendEntriesArgs) raftpb.AppendEntriesReply {
	return raftpb.AppendEntriesReply{Success: true, Term: 1}
}
func (s stub) InstallSnapshot(raftpb.InstallSnapshotArgs) raftpb.InstallSnapshotReply {
	return raftpb.InstallSnapshotReply{Term: 1}
}

func TestRPCRoundtripAndChaosDrop(t *testing.T) {
	h := stub{vote: raftpb.RequestVoteReply{Term: 2, VoteGranted: true}}
	ch := NewChaos()
	mux := http.NewServeMux()
	Mount(mux, h, ch)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	pool := NewPool(time.Second)
	cli := NewClient(pool, ch, "n1")
	ctx := context.Background()
	addr := strings.TrimPrefix(srv.URL, "http://")
	rep, err := cli.RequestVote(ctx, addr, "n2", raftpb.RequestVoteArgs{Term: 2, CandidateID: "n1"})
	if err != nil || !rep.VoteGranted {
		t.Fatalf("vote %v %v", rep, err)
	}
	ch.SetPeer("n2", Rule{Blocked: true})
	if _, err := cli.RequestVote(ctx, addr, "n2", raftpb.RequestVoteArgs{Term: 2, CandidateID: "n1"}); err == nil {
		t.Fatal("expected chaos drop")
	}
}

func TestDecodeRejectsUnknownField(t *testing.T) {
	var args raftpb.RequestVoteArgs
	err := readJSON(strings.NewReader(`{"term":1,"candidate_id":"n1","boom":true}`), 1024, &args)
	if err == nil {
		t.Fatal("expected unknown field error")
	}
}
