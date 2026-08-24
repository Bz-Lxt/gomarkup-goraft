package raft

import (
	"context"

	"goraft/internal/observability"
	"goraft/internal/raftpb"
	"goraft/internal/rpc"
)

func (n *Node) campaign() {
	if !n.conf.IsVoter(n.id) {
		n.electionElapsed = 0
		return
	}
	n.becomeCandidate()
	n.emit(observability.EvtVoteRequest, 0, "request vote broadcast", map[string]any{
		"last_index": n.log.LastIndex(), "last_term": n.log.LastTerm(),
	})
	if n.voteGranted() >= n.conf.Quorum() {
		n.becomeLeader()
		return
	}
	args := raftpb.RequestVoteArgs{
		Term:         n.hs.CurrentTerm,
		CandidateID:  n.id,
		LastLogIndex: n.log.LastIndex(),
		LastLogTerm:  n.log.LastTerm(),
	}
	sendTerm := n.hs.CurrentTerm
	for _, id := range n.conf.Voters {
		if id == n.id {
			continue
		}
		to := id
		go n.sendVote(to, args, sendTerm)
	}
}

func (n *Node) sendVote(to raftpb.NodeID, args raftpb.RequestVoteArgs, sendTerm raftpb.Term) {
	if n.trans == nil {
		return
	}
	ctx, cancel := rpc.WithTimeout(context.Background(), n.cfg.RPCTimeout)
	defer cancel()
	reply, err := n.trans.RequestVote(ctx, to, args)
	n.do(func() {
		if n.hs.CurrentTerm != sendTerm || n.state != raftpb.StateCandidate {
			return
		}
		if err != nil {
			return
		}
		n.onVoteReply(to, reply)
	})
}

func (n *Node) onVoteReply(from raftpb.NodeID, reply raftpb.RequestVoteReply) {
	if reply.Term > n.hs.CurrentTerm {
		n.becomeFollower(reply.Term, "")
		return
	}
	if reply.Term != n.hs.CurrentTerm {
		return
	}
	if reply.VoteGranted {
		n.votes[from] = true
		n.emit(observability.EvtVoteGranted, 0, "vote granted", map[string]any{"from": from, "count": n.voteGranted()})
		if n.voteGranted() >= n.conf.Quorum() {
			n.becomeLeader()
		}
	} else {
		n.emit(observability.EvtVoteDenied, 0, "vote denied", map[string]any{"from": from})
	}
}

func (n *Node) voteGranted() int {
	c := 0
	for _, ok := range n.votes {
		if ok {
			c++
		}
	}
	return c
}

func (n *Node) handleRequestVote(args raftpb.RequestVoteArgs) raftpb.RequestVoteReply {
	reply := raftpb.RequestVoteReply{Term: n.hs.CurrentTerm}
	if args.Term < n.hs.CurrentTerm {
		n.emit(observability.EvtVoteDenied, 0, "stale term", map[string]any{"from": args.CandidateID})
		return reply
	}
	if args.Term > n.hs.CurrentTerm {
		n.becomeFollower(args.Term, "")
		reply.Term = n.hs.CurrentTerm
	}
	if n.hs.VotedFor != "" && n.hs.VotedFor != args.CandidateID {
		n.emit(observability.EvtVoteDenied, 0, "already voted", map[string]any{"voted_for": n.hs.VotedFor})
		return reply
	}
	if !n.logUpToDate(args.LastLogIndex, args.LastLogTerm) {
		n.emit(observability.EvtVoteDenied, 0, "log stale candidate", map[string]any{"from": args.CandidateID})
		return reply
	}
	n.hs.VotedFor = args.CandidateID
	_ = n.persistHardState()
	n.electionElapsed = 0
	reply.VoteGranted = true
	n.emit(observability.EvtVoteGranted, 0, "granted vote", map[string]any{"to": args.CandidateID})
	return reply
}

func (n *Node) logUpToDate(lastIndex raftpb.Index, lastTerm raftpb.Term) bool {
	ourTerm := n.log.LastTerm()
	ourIndex := n.log.LastIndex()
	if lastTerm != ourTerm {
		return lastTerm > ourTerm
	}
	return lastIndex >= ourIndex
}
