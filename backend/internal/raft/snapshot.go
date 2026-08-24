package raft

import (
	"context"

	"goraft/internal/observability"
	"goraft/internal/raftpb"
	"goraft/internal/rpc"
	"goraft/internal/snapshot"
)

func (n *Node) maybeSnapshot() {
	if n.log.Len() < n.cfg.SnapshotLogN {
		return
	}
	if n.lastApplied <= n.log.SnapIndex() {
		return
	}
	data, err := n.sm.Snapshot()
	if err != nil {
		return
	}
	idx := n.lastApplied
	term, ok := n.log.Term(idx)
	if !ok {
		term = n.log.SnapTerm()
	}
	meta := snapshot.Meta{
		LastIncludedIndex: uint64(idx),
		LastIncludedTerm:  uint64(term),
		Voters:            idsToStr(n.conf.Voters),
		Learners:          idsToStr(n.conf.Learners),
	}
	n.emit(observability.EvtSnapshotStart, idx, "snapshot start", nil)
	if err := n.snaps.Save(meta, data); err != nil {
		return
	}
	n.log.Compact(idx, term)
	_ = n.wal.TruncatePrefix(n.walNextKeep())
	n.emit(observability.EvtSnapshotDone, idx, "snapshot done", map[string]any{"bytes": len(data)})
}

func (n *Node) walNextKeep() uint64 {
	return 1
}

func idsToStr(in []raftpb.NodeID) []string {
	out := make([]string, len(in))
	for i, id := range in {
		out[i] = string(id)
	}
	return out
}

func (n *Node) sendSnapshot(to raftpb.NodeID) {
	p := n.progressOf(to)
	p.Snapshot = true
	meta, data, err := n.snaps.Latest()
	if err != nil {
		p.Snapshot = false
		return
	}
	chunks := snapshot.Split(data, n.cfg.SnapshotChunkSize)
	sendTerm := n.hs.CurrentTerm
	go func() {
		for _, ch := range chunks {
			args := raftpb.InstallSnapshotArgs{
				Term:              sendTerm,
				LeaderID:          n.id,
				LastIncludedIndex: raftpb.Index(meta.LastIncludedIndex),
				LastIncludedTerm:  raftpb.Term(meta.LastIncludedTerm),
				Offset:            ch.Offset,
				Data:              ch.Data,
				Done:              ch.Done,
				Voters:            n.conf.Voters,
				Learners:          n.conf.Learners,
			}
			ctx, cancel := rpc.WithTimeout(context.Background(), n.cfg.RPCTimeout)
			reply, err := n.trans.InstallSnapshot(ctx, to, args)
			cancel()
			if err != nil {
				n.do(func() { n.progressOf(to).Snapshot = false })
				return
			}
			if reply.Term > sendTerm {
				n.do(func() { n.becomeFollower(reply.Term, "") })
				return
			}
		}
		n.do(func() {
			p := n.progressOf(to)
			p.Snapshot = false
			p.MatchIndex = raftpb.Index(meta.LastIncludedIndex)
			p.NextIndex = p.MatchIndex + 1
			n.sendAppend(to)
		})
	}()
}

func (n *Node) handleInstallSnapshot(args raftpb.InstallSnapshotArgs) raftpb.InstallSnapshotReply {
	reply := raftpb.InstallSnapshotReply{Term: n.hs.CurrentTerm}
	if args.Term < n.hs.CurrentTerm {
		return reply
	}
	if args.Term > n.hs.CurrentTerm {
		n.becomeFollower(args.Term, args.LeaderID)
	}
	n.leader = args.LeaderID
	n.electionElapsed = 0
	if n.incoming == nil {
		in, err := n.snaps.BeginInstall(snapshot.Meta{
			LastIncludedIndex: uint64(args.LastIncludedIndex),
			LastIncludedTerm:  uint64(args.LastIncludedTerm),
			Voters:            idsToStr(args.Voters),
			Learners:          idsToStr(args.Learners),
		})
		if err != nil {
			return reply
		}
		n.incoming = in
	}
	if err := n.incoming.Write(args.Offset, args.Data, args.Done); err != nil {
		n.incoming.Abort()
		n.incoming = nil
		return reply
	}
	if !args.Done {
		return reply
	}
	_, data, err := n.snaps.Latest()
	if err == nil {
		_ = n.sm.Restore(data)
	}
	n.log.Restore(args.LastIncludedIndex, args.LastIncludedTerm, nil)
	if n.commitIndex < args.LastIncludedIndex {
		n.commitIndex = args.LastIncludedIndex
	}
	n.lastApplied = args.LastIncludedIndex
	if len(args.Voters) > 0 {
		n.conf = raftpb.Membership{Voters: args.Voters, Learners: args.Learners}
	}
	n.incoming = nil
	n.emit(observability.EvtSnapshotInstall, args.LastIncludedIndex, "snapshot installed", nil)
	return reply
}
