package raftpb

type NodeID string
type Term uint64
type Index uint64

type State uint8

const (
	StateFollower State = iota
	StateCandidate
	StateLeader
	StateLearner
)

func (s State) String() string {
	switch s {
	case StateFollower:
		return "follower"
	case StateCandidate:
		return "candidate"
	case StateLeader:
		return "leader"
	case StateLearner:
		return "learner"
	default:
		return "unknown"
	}
}

type EntryType uint8

const (
	EntryNormal EntryType = iota
	EntryConfig
	EntryNoop
)

type Entry struct {
	Term  Term      `json:"term"`
	Index Index     `json:"index"`
	Type  EntryType `json:"type"`
	Data  []byte    `json:"data"`
}

type HardState struct {
	CurrentTerm Term   `json:"current_term"`
	VotedFor    NodeID `json:"voted_for"`
	Commit      Index  `json:"commit"`
}

type Membership struct {
	Voters   []NodeID `json:"voters"`
	Learners []NodeID `json:"learners"`
}

type RequestVoteArgs struct {
	Term         Term   `json:"term"`
	CandidateID  NodeID `json:"candidate_id"`
	LastLogIndex Index  `json:"last_log_index"`
	LastLogTerm  Term   `json:"last_log_term"`
	TraceID      string `json:"trace_id"`
}

type RequestVoteReply struct {
	Term        Term `json:"term"`
	VoteGranted bool `json:"vote_granted"`
}

type AppendEntriesArgs struct {
	Term         Term    `json:"term"`
	LeaderID     NodeID  `json:"leader_id"`
	PrevLogIndex Index   `json:"prev_log_index"`
	PrevLogTerm  Term    `json:"prev_log_term"`
	Entries      []Entry `json:"entries"`
	LeaderCommit Index   `json:"leader_commit"`
	TraceID      string  `json:"trace_id"`
	Heartbeat    bool    `json:"heartbeat"`
}

type AppendEntriesReply struct {
	Term          Term  `json:"term"`
	Success       bool  `json:"success"`
	ConflictIndex Index `json:"conflict_index"`
	ConflictTerm  Term  `json:"conflict_term"`
	MatchIndex    Index `json:"match_index"`
}

type InstallSnapshotArgs struct {
	Term              Term   `json:"term"`
	LeaderID          NodeID `json:"leader_id"`
	LastIncludedIndex Index  `json:"last_included_index"`
	LastIncludedTerm  Term   `json:"last_included_term"`
	Offset            uint64 `json:"offset"`
	Data              []byte `json:"data"`
	Done              bool   `json:"done"`
	Voters            []NodeID `json:"voters"`
	Learners          []NodeID `json:"learners"`
	TraceID           string `json:"trace_id"`
}

type InstallSnapshotReply struct {
	Term Term `json:"term"`
}

func (m Membership) Clone() Membership {
	out := Membership{
		Voters:   append([]NodeID(nil), m.Voters...),
		Learners: append([]NodeID(nil), m.Learners...),
	}
	return out
}

func (m Membership) IsVoter(id NodeID) bool {
	for _, v := range m.Voters {
		if v == id {
			return true
		}
	}
	return false
}

func (m Membership) IsLearner(id NodeID) bool {
	for _, v := range m.Learners {
		if v == id {
			return true
		}
	}
	return false
}

func (m Membership) Contains(id NodeID) bool {
	return m.IsVoter(id) || m.IsLearner(id)
}

func (m Membership) Quorum() int {
	n := len(m.Voters)
	if n == 0 {
		return 1
	}
	return n/2 + 1
}

func (m Membership) All() []NodeID {
	out := append([]NodeID(nil), m.Voters...)
	out = append(out, m.Learners...)
	return out
}
