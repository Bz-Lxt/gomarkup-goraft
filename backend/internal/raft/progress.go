package raft

import "goraft/internal/raftpb"

type Progress struct {
	NextIndex  raftpb.Index `json:"next_index"`
	MatchIndex raftpb.Index `json:"match_index"`
	Inflights  int          `json:"inflights"`
	Paused     bool         `json:"paused"`
	Snapshot   bool         `json:"snapshot"`
}

func (n *Node) resetProgress() {
	n.prg = map[raftpb.NodeID]*Progress{}
	last := n.log.LastIndex()
	for _, id := range n.conf.All() {
		p := &Progress{NextIndex: last + 1}
		if id == n.id {
			p.MatchIndex = last
			p.NextIndex = last + 1
		}
		n.prg[id] = p
	}
}

func (n *Node) progressOf(id raftpb.NodeID) *Progress {
	p := n.prg[id]
	if p == nil {
		p = &Progress{NextIndex: n.log.LastIndex() + 1}
		n.prg[id] = p
	}
	return p
}
