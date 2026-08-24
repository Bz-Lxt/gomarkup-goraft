package raft

import (
	"context"

	"goraft/internal/observability"
	"goraft/internal/raftpb"
	"goraft/internal/rpc"
)

func (n *Node) bcastAppend() {
	if n.state != raftpb.StateLeader {
		return
	}
	for _, id := range n.conf.All() {
		if id == n.id {
			continue
		}
		n.sendAppend(id)
	}
}

func (n *Node) sendAppend(to raftpb.NodeID) {
	if n.trans == nil {
		return
	}
	p := n.progressOf(to)
	if p.Snapshot {
		n.sendSnapshot(to)
		return
	}
	prev := p.NextIndex - 1
	if prev < n.log.SnapIndex() {
		n.sendSnapshot(to)
		return
	}
	prevTerm, ok := n.log.Term(prev)
	if !ok {
		n.sendSnapshot(to)
		return
	}
	ents := n.log.Slice(p.NextIndex, n.log.LastIndex())
	if len(ents) > 64 {
		ents = ents[:64]
	}
	args := raftpb.AppendEntriesArgs{
		Term:         n.hs.CurrentTerm,
		LeaderID:     n.id,
		PrevLogIndex: prev,
		PrevLogTerm:  prevTerm,
		Entries:      ents,
		LeaderCommit: n.commitIndex,
		Heartbeat:    len(ents) == 0,
	}
	sendTerm := n.hs.CurrentTerm
	nextHint := p.NextIndex + raftpb.Index(len(ents))
	if len(ents) > 0 {
		n.emit(observability.EvtAppendSend, p.NextIndex, "append send", map[string]any{
			"to": to, "n": len(ents), "prev": prev,
		})
	} else {
		n.emit(observability.EvtHeartbeat, 0, "heartbeat", map[string]any{"to": to})
	}
	go func() {
		ctx, cancel := rpc.WithTimeout(context.Background(), n.cfg.RPCTimeout)
		defer cancel()
		reply, err := n.trans.AppendEntries(ctx, to, args)
		n.do(func() {
			if n.hs.CurrentTerm != sendTerm || n.state != raftpb.StateLeader {
				return
			}
			if err != nil {
				return
			}
			n.onAppendReply(to, args, reply, nextHint)
		})
	}()
}

func (n *Node) onAppendReply(to raftpb.NodeID, args raftpb.AppendEntriesArgs, reply raftpb.AppendEntriesReply, nextHint raftpb.Index) {
	if reply.Term > n.hs.CurrentTerm {
		n.becomeFollower(reply.Term, "")
		return
	}
	p := n.progressOf(to)
	if reply.Success {
		if reply.MatchIndex > p.MatchIndex {
			p.MatchIndex = reply.MatchIndex
		} else if nextHint-1 > p.MatchIndex {
			p.MatchIndex = nextHint - 1
		}
		p.NextIndex = p.MatchIndex + 1
		n.emit(observability.EvtAppendAck, p.MatchIndex, "append ack", map[string]any{"from": to})
		n.ackReads(to)
		n.maybeCommit()
		if p.NextIndex <= n.log.LastIndex() {
			n.sendAppend(to)
		}
		n.maybePromoteLearner(to)
		return
	}
	n.emit(observability.EvtAppendReject, p.NextIndex, "append reject", map[string]any{
		"from": to, "conflict_index": reply.ConflictIndex, "conflict_term": reply.ConflictTerm,
	})
	if reply.ConflictTerm != 0 {
		last := n.lastIndexOfTerm(reply.ConflictTerm)
		if last > 0 {
			p.NextIndex = last + 1
		} else if reply.ConflictIndex > 0 {
			p.NextIndex = reply.ConflictIndex
		} else if p.NextIndex > 1 {
			p.NextIndex--
		}
	} else if reply.ConflictIndex > 0 {
		p.NextIndex = reply.ConflictIndex
	} else if p.NextIndex > 1 {
		p.NextIndex--
	}
	if p.NextIndex < n.log.SnapIndex()+1 {
		n.sendSnapshot(to)
		return
	}
	n.sendAppend(to)
}

func (n *Node) lastIndexOfTerm(term raftpb.Term) raftpb.Index {
	for i := n.log.LastIndex(); i > n.log.SnapIndex(); i-- {
		t, ok := n.log.Term(i)
		if ok && t == term {
			return i
		}
		if ok && t < term {
			break
		}
	}
	return 0
}

func (n *Node) maybeCommit() {
	if n.state != raftpb.StateLeader {
		return
	}
	for idx := n.log.LastIndex(); idx > n.commitIndex; idx-- {
		t, ok := n.log.Term(idx)
		if !ok || t != n.hs.CurrentTerm {
			continue
		}
		if n.replicated(idx) >= n.conf.Quorum() {
			n.commitIndex = idx
			_ = n.wal.AppendCommit(uint64(idx))
			n.emit(observability.EvtCommit, idx, "commit advanced", map[string]any{"commit": idx})
			n.maybeApply()
			return
		}
	}
}

func (n *Node) replicated(idx raftpb.Index) int {
	c := 0
	for _, id := range n.conf.Voters {
		if id == n.id {
			if n.log.LastIndex() >= idx {
				c++
			}
			continue
		}
		if p := n.prg[id]; p != nil && p.MatchIndex >= idx {
			c++
		}
	}
	return c
}

func (n *Node) handleAppendEntries(args raftpb.AppendEntriesArgs) raftpb.AppendEntriesReply {
	reply := raftpb.AppendEntriesReply{Term: n.hs.CurrentTerm}
	if args.Term < n.hs.CurrentTerm {
		return reply
	}
	if args.Term > n.hs.CurrentTerm || n.state != raftpb.StateFollower {
		n.becomeFollower(args.Term, args.LeaderID)
	} else {
		n.leader = args.LeaderID
		n.electionElapsed = 0
	}
	reply.Term = n.hs.CurrentTerm

	if args.PrevLogIndex > n.log.LastIndex() {
		reply.ConflictIndex = n.log.LastIndex() + 1
		reply.ConflictTerm = 0
		return reply
	}
	if args.PrevLogIndex > 0 {
		t, ok := n.log.Term(args.PrevLogIndex)
		if !ok || t != args.PrevLogTerm {
			reply.ConflictTerm = t
			reply.ConflictIndex = n.firstIndexOfTerm(t)
			if reply.ConflictIndex == 0 {
				reply.ConflictIndex = args.PrevLogIndex
			}
			return reply
		}
	}
	for _, e := range args.Entries {
		if exist, ok := n.log.At(e.Index); ok && exist.Term != e.Term {
			n.log.TruncateFrom(e.Index)
		}
		if _, ok := n.log.At(e.Index); !ok {
			n.log.Append(e)
			_ = n.persistEntries([]raftpb.Entry{e}, true)
			if e.Type == raftpb.EntryConfig {
				n.applyConfigBytes(e.Data)
			}
		}
	}
	if args.LeaderCommit > n.commitIndex {
		last := n.log.LastIndex()
		n.commitIndex = args.LeaderCommit
		if n.commitIndex > last {
			n.commitIndex = last
		}
		n.emit(observability.EvtCommit, n.commitIndex, "follower commit", map[string]any{"commit": n.commitIndex})
		n.maybeApply()
	}
	reply.Success = true
	reply.MatchIndex = n.log.LastIndex()
	return reply
}

func (n *Node) firstIndexOfTerm(term raftpb.Term) raftpb.Index {
	if term == 0 {
		return n.log.LastIndex() + 1
	}
	var first raftpb.Index
	for i := n.log.SnapIndex() + 1; i <= n.log.LastIndex(); i++ {
		t, ok := n.log.Term(i)
		if ok && t == term {
			if first == 0 {
				first = i
			}
		}
	}
	return first
}
