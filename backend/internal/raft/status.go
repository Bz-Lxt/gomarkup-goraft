package raft

import "goraft/internal/raftpb"

type Status struct {
	ID          string             `json:"id"`
	State       string             `json:"state"`
	Term        uint64             `json:"term"`
	Leader      string             `json:"leader"`
	LeaderAddr  string             `json:"leader_addr"`
	CommitIndex uint64             `json:"commit_index"`
	LastApplied uint64             `json:"last_applied"`
	LastIndex   uint64             `json:"last_index"`
	LastTerm    uint64             `json:"last_term"`
	SnapIndex   uint64             `json:"snap_index"`
	VotedFor    string             `json:"voted_for"`
	Voters      []string           `json:"voters"`
	Learners    []string           `json:"learners"`
	NextIndex   map[string]uint64  `json:"next_index"`
	MatchIndex  map[string]uint64  `json:"match_index"`
	Mode        string             `json:"mode"`
	Dead        bool               `json:"dead"`
	ElectionTO  int                `json:"election_ticks"`
	LogLen      int                `json:"log_len"`
}

func (n *Node) Status() Status {
	var st Status
	n.do(func() { st = n.statusLocked() })
	return st
}

func (n *Node) statusLocked() Status {
	next := map[string]uint64{}
	match := map[string]uint64{}
	for id, p := range n.prg {
		next[string(id)] = uint64(p.NextIndex)
		match[string(id)] = uint64(p.MatchIndex)
	}
	dead := false
	if n.chaos != nil {
		dead = n.chaos.Dead()
	}
	return Status{
		ID:          string(n.id),
		State:       n.state.String(),
		Term:        uint64(n.hs.CurrentTerm),
		Leader:      string(n.leader),
		LeaderAddr:  n.LeaderAddr(),
		CommitIndex: uint64(n.commitIndex),
		LastApplied: uint64(n.lastApplied),
		LastIndex:   uint64(n.log.LastIndex()),
		LastTerm:    uint64(n.log.LastTerm()),
		SnapIndex:   uint64(n.log.SnapIndex()),
		VotedFor:    string(n.hs.VotedFor),
		Voters:      idsToStr(n.conf.Voters),
		Learners:    idsToStr(n.conf.Learners),
		NextIndex:   next,
		MatchIndex:  match,
		Mode:        string(n.cfg.Mode),
		Dead:        dead,
		ElectionTO:  n.randomizedElectionTimeout,
		LogLen:      n.log.Len(),
	}
}

func (n *Node) ConfigView() raftpb.Membership { return n.conf.Clone() }
