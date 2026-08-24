package raft

import (
	"encoding/json"
	"fmt"

	"goraft/internal/observability"
	"goraft/internal/raftpb"
)

func (n *Node) applyConfigBytes(data []byte) {
	var m raftpb.Membership
	if err := json.Unmarshal(data, &m); err != nil {
		return
	}
	if len(m.Voters) == 0 {
		return
	}
	n.conf = m.Clone()
	for _, id := range n.conf.All() {
		if n.prg[id] == nil {
			n.prg[id] = &Progress{NextIndex: n.log.LastIndex() + 1}
		}
	}
	n.emit(observability.EvtMembership, 0, "membership applied", map[string]any{
		"voters": n.conf.Voters, "learners": n.conf.Learners,
	})
}

func (n *Node) ChangeMembership(add raftpb.NodeID, addr string, remove raftpb.NodeID, asLearner bool) error {
	var err error
	n.do(func() {
		err = n.changeMembershipLocked(add, addr, remove, asLearner)
	})
	return err
}

func (n *Node) changeMembershipLocked(add raftpb.NodeID, addr string, remove raftpb.NodeID, asLearner bool) error {
	if n.state != raftpb.StateLeader {
		return ErrNotLeader
	}
	if n.pendingConfIndex > n.commitIndex {
		return ErrConfInProgress
	}
	next := n.conf.Clone()
	if add != "" {
		if next.Contains(add) {
			return fmt.Errorf("member %s already exists", add)
		}
		if addr != "" {
			n.addrs[add] = addr
			if ht, ok := n.trans.(*HTTPTransport); ok {
				ht.addrs[add] = addr
			}
		}
		if asLearner {
			next.Learners = append(next.Learners, add)
		} else {
			next.Voters = append(next.Voters, add)
		}
	}
	if remove != "" {
		next.Voters = filterID(next.Voters, remove)
		next.Learners = filterID(next.Learners, remove)
	}
	if len(next.Voters) == 0 {
		return fmt.Errorf("refusing to remove last voter")
	}
	b, err := json.Marshal(next)
	if err != nil {
		return err
	}
	n.appendData(raftpb.EntryConfig, b, "")
	n.maybeCommit()
	n.bcastAppend()
	return nil
}

func filterID(in []raftpb.NodeID, drop raftpb.NodeID) []raftpb.NodeID {
	out := make([]raftpb.NodeID, 0, len(in))
	for _, id := range in {
		if id != drop {
			out = append(out, id)
		}
	}
	return out
}

func (n *Node) maybePromoteLearner(id raftpb.NodeID) {
	if !n.conf.IsLearner(id) || n.state != raftpb.StateLeader {
		return
	}
	if n.pendingConfIndex > n.commitIndex {
		return
	}
	p := n.prg[id]
	if p == nil || p.MatchIndex < n.log.LastIndex() {
		return
	}
	next := n.conf.Clone()
	next.Learners = filterID(next.Learners, id)
	next.Voters = append(next.Voters, id)
	b, err := json.Marshal(next)
	if err != nil {
		return
	}
	n.appendData(raftpb.EntryConfig, b, "")
	n.maybeCommit()
	n.bcastAppend()
}
