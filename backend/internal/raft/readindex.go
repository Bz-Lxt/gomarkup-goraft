package raft

import (
	"context"

	"goraft/internal/observability"
	"goraft/internal/raftpb"
)

func (n *Node) LinearizableRead(ctx context.Context, traceID string, fn func() any) (any, error) {
	if n.chaos != nil && n.chaos.Dead() {
		return nil, ErrDead
	}
	ch := make(chan readResult, 1)
	n.do(func() {
		if n.state != raftpb.StateLeader {
			ch <- readResult{err: ErrNotLeader}
			return
		}
		if !n.currentTermCommitted() {
			ch <- readResult{err: ErrLeaderNotReady}
			return
		}
		n.reads = append(n.reads, pendingRead{
			index: n.commitIndex,
			acks:  map[raftpb.NodeID]bool{n.id: true},
			fn:    fn,
			ch:    ch,
			trace: traceID,
		})
		n.emit(observability.EvtReadIndex, n.commitIndex, "read index barrier", map[string]any{"trace_id": traceID})
		n.bcastAppend()
		n.tryFinishReads()
	})
	select {
	case r := <-ch:
		return r.val, r.err
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-n.stop:
		return nil, ErrStopped
	}
}

func (n *Node) currentTermCommitted() bool {
	if n.commitIndex == 0 {
		return false
	}
	t, ok := n.log.Term(n.commitIndex)
	if !ok {
		return n.commitIndex <= n.log.SnapIndex() && n.hs.CurrentTerm > 0 && n.state == raftpb.StateLeader
	}
	return t == n.hs.CurrentTerm
}

func (n *Node) ackReads(from raftpb.NodeID) {
	for i := range n.reads {
		n.reads[i].acks[from] = true
	}
	n.tryFinishReads()
}

func (n *Node) tryFinishReads() {
	remain := n.reads[:0]
	for _, r := range n.reads {
		if n.lastApplied < r.index {
			remain = append(remain, r)
			continue
		}
		if len(r.acks) < n.conf.Quorum() {
			remain = append(remain, r)
			continue
		}
		var val any
		if r.fn != nil {
			val = r.fn()
		}
		r.ch <- readResult{val: val}
	}
	n.reads = remain
}
