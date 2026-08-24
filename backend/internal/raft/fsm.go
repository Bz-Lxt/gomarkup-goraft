package raft

import (
	"goraft/internal/observability"
	"goraft/internal/raftpb"
)

func (n *Node) becomeFollower(term raftpb.Term, leader raftpb.NodeID) {
	prev := n.state
	if term > n.hs.CurrentTerm {
		n.hs.CurrentTerm = term
		n.hs.VotedFor = ""
		_ = n.persistHardState()
		n.emit(observability.EvtTermChange, 0, "term advanced", map[string]any{"term": term})
	}
	n.state = raftpb.StateFollower
	n.leader = leader
	n.votes = map[raftpb.NodeID]bool{}
	n.electionElapsed = 0
	n.heartbeatElapsed = 0
	n.resetRandomizedElectionTimeout()
	if prev != n.state {
		n.emit(observability.EvtStateChange, 0, "became follower", map[string]any{
			"from": prev.String(), "to": n.state.String(), "leader": leader,
		})
	}
	n.failPending(ErrStepDown)
}

func (n *Node) becomeCandidate() {
	prev := n.state
	n.state = raftpb.StateCandidate
	n.leader = ""
	n.hs.CurrentTerm++
	n.hs.VotedFor = n.id
	_ = n.persistHardState()
	n.votes = map[raftpb.NodeID]bool{n.id: true}
	n.electionElapsed = 0
	n.resetRandomizedElectionTimeout()
	n.emit(observability.EvtTermChange, 0, "campaign", map[string]any{"term": n.hs.CurrentTerm})
	n.emit(observability.EvtStateChange, 0, "became candidate", map[string]any{
		"from": prev.String(), "to": n.state.String(),
	})
}

func (n *Node) becomeLeader() {
	prev := n.state
	n.state = raftpb.StateLeader
	n.leader = n.id
	n.resetProgress()
	n.heartbeatElapsed = 0
	n.emit(observability.EvtStateChange, 0, "became leader", map[string]any{
		"from": prev.String(), "to": n.state.String(),
	})
	n.appendData(raftpb.EntryNoop, nil, "")
	n.maybeCommit()
	n.bcastAppend()
}

func (n *Node) resetRandomizedElectionTimeout() {
	span := n.cfg.ElectionTicksMax - n.cfg.ElectionTicksMin + 1
	if span <= 0 {
		span = 1
	}
	n.randomizedElectionTimeout = n.cfg.ElectionTicksMin + n.rng.Intn(span)
}

func (n *Node) failPending(err error) {
	for idx, ch := range n.proposals {
		ch <- proposeResult{index: idx, err: err}
		delete(n.proposals, idx)
	}
	for _, r := range n.reads {
		r.ch <- readResult{err: err}
	}
	n.reads = nil
}
