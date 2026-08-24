package raft

import (
	"goraft/internal/observability"
	"goraft/internal/raftpb"
)

func (n *Node) maybeApply() {
	for n.lastApplied < n.commitIndex {
		next := n.lastApplied + 1
		if next <= n.log.SnapIndex() {
			n.lastApplied = n.log.SnapIndex()
			if n.lastApplied > n.commitIndex {
				n.lastApplied = n.commitIndex
			}
			continue
		}
		e, ok := n.log.At(next)
		if !ok {
			break
		}
		var res any
		var err error
		if n.sm != nil && e.Type != raftpb.EntryNoop {
			applied, applyErr := n.sm.Apply(uint64(e.Index), e.Type, e.Data)
			err = applyErr
			if err == nil {
				res = applied
			}
		}
		n.lastApplied = next
		n.emit(observability.EvtApply, next, "applied", map[string]any{"type": e.Type})
		if ch, ok := n.proposals[next]; ok {
			ch <- proposeResult{index: next, term: e.Term, res: res, err: err}
			delete(n.proposals, next)
		}
	}
	n.tryFinishReads()
}
