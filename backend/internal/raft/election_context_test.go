package raft_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"goraft/internal/config"
	"goraft/internal/raft"
	"goraft/internal/raftpb"
	"goraft/internal/statemachine"
)

type delayedVoteTransport struct {
	started chan struct{}
	release chan struct{}
	calls   atomic.Int32
}

func (t *delayedVoteTransport) RequestVote(ctx context.Context, _ raftpb.NodeID, args raftpb.RequestVoteArgs) (raftpb.RequestVoteReply, error) {
	if t.calls.Add(1) != 1 {
		return raftpb.RequestVoteReply{Term: args.Term}, nil
	}
	close(t.started)
	select {
	case <-ctx.Done():
		return raftpb.RequestVoteReply{}, ctx.Err()
	case <-t.release:
		if err := ctx.Err(); err != nil {
			return raftpb.RequestVoteReply{}, err
		}
		return raftpb.RequestVoteReply{Term: args.Term, VoteGranted: true}, nil
	}
}

func (t *delayedVoteTransport) AppendEntries(ctx context.Context, _ raftpb.NodeID, args raftpb.AppendEntriesArgs) (raftpb.AppendEntriesReply, error) {
	if err := ctx.Err(); err != nil {
		return raftpb.AppendEntriesReply{}, err
	}
	return raftpb.AppendEntriesReply{
		Term:       args.Term,
		Success:    true,
		MatchIndex: args.PrevLogIndex + raftpb.Index(len(args.Entries)),
	}, nil
}

func (t *delayedVoteTransport) InstallSnapshot(ctx context.Context, _ raftpb.NodeID, args raftpb.InstallSnapshotArgs) (raftpb.InstallSnapshotReply, error) {
	if err := ctx.Err(); err != nil {
		return raftpb.InstallSnapshotReply{}, err
	}
	return raftpb.InstallSnapshotReply{Term: args.Term}, nil
}

func TestElectionCompletesWithDelayedVoteReplies(t *testing.T) {
	cfg := &config.Config{
		ID:                "n1",
		HTTPAddr:          ":0",
		AdvertiseAddr:     "n1",
		DataDir:           t.TempDir(),
		Peers:             map[string]string{"n1": "n1", "n2": "n2"},
		Mode:              config.ModeDemo,
		LogLevel:          "error",
		TickInterval:      2 * time.Millisecond,
		HeartbeatTicks:    5,
		ElectionTicksMin:  100,
		ElectionTicksMax:  100,
		WALSegmentBytes:   1 << 20,
		SnapshotLogN:      100,
		SnapshotBytes:     1 << 20,
		SnapshotChunkSize: 1024,
		RPCTimeout:        2 * time.Second,
		GroupCommitWin:    time.Millisecond,
	}
	n, err := raft.New(cfg, statemachine.NewKV(), nil, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	transport := &delayedVoteTransport{
		started: make(chan struct{}),
		release: make(chan struct{}),
	}
	n.SetTransport(transport)
	n.Start()
	t.Cleanup(n.Stop)

	select {
	case <-transport.started:
	case <-time.After(time.Second):
		t.Fatal("node did not start a remote vote request")
	}

	_ = n.Status()
	close(transport.release)

	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if st := n.Status(); st.State == "leader" {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("node did not elect a leader after a delayed vote reply: %+v", n.Status())
}
