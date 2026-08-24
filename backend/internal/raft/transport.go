package raft

import (
	"context"

	"goraft/internal/raftpb"
	"goraft/internal/rpc"
)

type Transport interface {
	RequestVote(ctx context.Context, to raftpb.NodeID, args raftpb.RequestVoteArgs) (raftpb.RequestVoteReply, error)
	AppendEntries(ctx context.Context, to raftpb.NodeID, args raftpb.AppendEntriesArgs) (raftpb.AppendEntriesReply, error)
	InstallSnapshot(ctx context.Context, to raftpb.NodeID, args raftpb.InstallSnapshotArgs) (raftpb.InstallSnapshotReply, error)
}

type HTTPTransport struct {
	client *rpc.Client
	addrs  map[raftpb.NodeID]string
}

func NewHTTPTransport(client *rpc.Client, addrs map[raftpb.NodeID]string) *HTTPTransport {
	return &HTTPTransport{client: client, addrs: addrs}
}

func (t *HTTPTransport) addr(to raftpb.NodeID) (string, bool) {
	a, ok := t.addrs[to]
	return a, ok && a != ""
}

func (t *HTTPTransport) RequestVote(ctx context.Context, to raftpb.NodeID, args raftpb.RequestVoteArgs) (raftpb.RequestVoteReply, error) {
	addr, ok := t.addr(to)
	if !ok {
		return raftpb.RequestVoteReply{}, ErrUnavailable
	}
	return t.client.RequestVote(ctx, addr, string(to), args)
}

func (t *HTTPTransport) AppendEntries(ctx context.Context, to raftpb.NodeID, args raftpb.AppendEntriesArgs) (raftpb.AppendEntriesReply, error) {
	addr, ok := t.addr(to)
	if !ok {
		return raftpb.AppendEntriesReply{}, ErrUnavailable
	}
	return t.client.AppendEntries(ctx, addr, string(to), args)
}

func (t *HTTPTransport) InstallSnapshot(ctx context.Context, to raftpb.NodeID, args raftpb.InstallSnapshotArgs) (raftpb.InstallSnapshotReply, error) {
	addr, ok := t.addr(to)
	if !ok {
		return raftpb.InstallSnapshotReply{}, ErrUnavailable
	}
	return t.client.InstallSnapshot(ctx, addr, string(to), args)
}

type MemoryTransport struct {
	nodes map[raftpb.NodeID]*Node
	drop  map[string]bool
}

func NewMemoryTransport() *MemoryTransport {
	return &MemoryTransport{nodes: map[raftpb.NodeID]*Node{}, drop: map[string]bool{}}
}

func (m *MemoryTransport) Register(n *Node) {
	m.nodes[n.id] = n
}

func (m *MemoryTransport) Partition(a, b raftpb.NodeID, on bool) {
	m.drop[string(a)+"->"+string(b)] = on
	m.drop[string(b)+"->"+string(a)] = on
}

func (m *MemoryTransport) blocked(from, to raftpb.NodeID) bool {
	return m.drop[string(from)+"->"+string(to)]
}

func (m *MemoryTransport) RequestVote(ctx context.Context, to raftpb.NodeID, args raftpb.RequestVoteArgs) (raftpb.RequestVoteReply, error) {
	n := m.nodes[to]
	if n == nil || m.blocked(args.CandidateID, to) {
		return raftpb.RequestVoteReply{}, ErrUnavailable
	}
	return n.RequestVote(args), nil
}

func (m *MemoryTransport) AppendEntries(ctx context.Context, to raftpb.NodeID, args raftpb.AppendEntriesArgs) (raftpb.AppendEntriesReply, error) {
	n := m.nodes[to]
	if n == nil || m.blocked(args.LeaderID, to) {
		return raftpb.AppendEntriesReply{}, ErrUnavailable
	}
	return n.AppendEntries(args), nil
}

func (m *MemoryTransport) InstallSnapshot(ctx context.Context, to raftpb.NodeID, args raftpb.InstallSnapshotArgs) (raftpb.InstallSnapshotReply, error) {
	n := m.nodes[to]
	if n == nil || m.blocked(args.LeaderID, to) {
		return raftpb.InstallSnapshotReply{}, ErrUnavailable
	}
	return n.InstallSnapshot(args), nil
}
