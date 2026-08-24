package raft

import (
	"context"
	"math/rand"
	"sync"
	"time"

	"goraft/internal/config"
	"goraft/internal/logutil"
	"goraft/internal/observability"
	"goraft/internal/raftpb"
	"goraft/internal/rpc"
	"goraft/internal/snapshot"
	"goraft/internal/wal"
)

type StateMachine interface {
	Apply(index uint64, typ raftpb.EntryType, data []byte) (any, error)
	Snapshot() ([]byte, error)
	Restore(data []byte) error
}

type proposeResult struct {
	index raftpb.Index
	term  raftpb.Term
	res   any
	err   error
}

type pendingRead struct {
	index raftpb.Index
	acks  map[raftpb.NodeID]bool
	fn    func() any
	ch    chan readResult
	trace string
}

type readResult struct {
	val any
	err error
}

type Node struct {
	id  raftpb.NodeID
	cfg *config.Config

	wal   *wal.WAL
	snaps *snapshot.Store
	log   *Log
	sm    StateMachine

	hs     raftpb.HardState
	state  raftpb.State
	leader raftpb.NodeID

	commitIndex raftpb.Index
	lastApplied raftpb.Index

	prg map[raftpb.NodeID]*Progress

	votes                     map[raftpb.NodeID]bool
	electionElapsed           int
	heartbeatElapsed          int
	randomizedElectionTimeout int

	conf             raftpb.Membership
	pendingConfIndex raftpb.Index
	addrs            map[raftpb.NodeID]string

	reads     []pendingRead
	proposals map[raftpb.Index]chan proposeResult

	trans  Transport
	chaos  *rpc.Chaos
	bus    *observability.Bus
	tracer *observability.Tracer
	ring   *observability.Ring

	incoming *snapshot.Incoming

	callCh chan func()
	stop   chan struct{}
	wg     sync.WaitGroup
	once   sync.Once

	rng *rand.Rand
}

func New(cfg *config.Config, sm StateMachine, bus *observability.Bus, tracer *observability.Tracer, ring *observability.Ring, chaos *rpc.Chaos) (*Node, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	w, rec, err := wal.Open(cfg.DataDir+"/wal", cfg.WALSegmentBytes, cfg.GroupCommitWin)
	if err != nil {
		return nil, err
	}
	snaps, err := snapshot.Open(cfg.DataDir + "/snap")
	if err != nil {
		_ = w.Close()
		return nil, err
	}
	lg, hs, err := restoreLog(rec)
	if err != nil {
		_ = w.Close()
		return nil, err
	}
	if fileHS, err := loadHardStateFile(cfg.DataDir); err == nil && fileHS.CurrentTerm >= hs.CurrentTerm {
		hs = fileHS
	}
	if meta, data, err := snaps.Latest(); err == nil {
		if err := sm.Restore(data); err != nil {
			_ = w.Close()
			return nil, err
		}
		lg.Compact(raftpb.Index(meta.LastIncludedIndex), raftpb.Term(meta.LastIncludedTerm))
		if hs.Commit < raftpb.Index(meta.LastIncludedIndex) {
			hs.Commit = raftpb.Index(meta.LastIncludedIndex)
		}
	}
	voters := make([]raftpb.NodeID, 0, len(cfg.Peers))
	addrs := map[raftpb.NodeID]string{}
	for id, addr := range cfg.Peers {
		voters = append(voters, raftpb.NodeID(id))
		addrs[raftpb.NodeID(id)] = addr
	}
	n := &Node{
		id:        raftpb.NodeID(cfg.ID),
		cfg:       cfg,
		wal:       w,
		snaps:     snaps,
		log:       lg,
		sm:        sm,
		hs:        hs,
		state:     raftpb.StateFollower,
		commitIndex: hs.Commit,
		lastApplied: lg.SnapIndex(),
		prg:       map[raftpb.NodeID]*Progress{},
		votes:     map[raftpb.NodeID]bool{},
		conf:      raftpb.Membership{Voters: voters},
		addrs:     addrs,
		proposals: map[raftpb.Index]chan proposeResult{},
		chaos:     chaos,
		bus:       bus,
		tracer:    tracer,
		ring:      ring,
		callCh:    make(chan func(), 1024),
		stop:      make(chan struct{}),
		rng:       rand.New(rand.NewSource(time.Now().UnixNano())),
	}
	if n.lastApplied > n.commitIndex {
		n.lastApplied = n.commitIndex
	}
	n.resetRandomizedElectionTimeout()
	n.emit(observability.EvtLog, 0, "node restored", map[string]any{
		"term": n.hs.CurrentTerm, "commit": n.commitIndex, "last": n.log.LastIndex(),
	})
	return n, nil
}

func (n *Node) SetTransport(t Transport) { n.trans = t }

func (n *Node) ID() raftpb.NodeID { return n.id }

func (n *Node) Start() {
	n.wg.Add(1)
	go n.loop()
}

func (n *Node) Stop() {
	n.once.Do(func() {
		close(n.stop)
		n.wg.Wait()
		_ = n.wal.Close()
	})
}

func (n *Node) loop() {
	defer n.wg.Done()
	tick := time.NewTicker(n.cfg.TickInterval)
	defer tick.Stop()
	for {
		select {
		case <-n.stop:
			return
		case <-tick.C:
			if n.chaos != nil && n.chaos.Dead() {
				continue
			}
			n.tick()
		case fn := <-n.callCh:
			fn()
		}
	}
}

func (n *Node) do(fn func()) {
	done := make(chan struct{})
	select {
	case <-n.stop:
		return
	case n.callCh <- func() {
		fn()
		close(done)
	}:
	}
	select {
	case <-done:
	case <-n.stop:
	}
}

func (n *Node) tick() {
	switch n.state {
	case raftpb.StateLeader:
		n.heartbeatElapsed++
		if n.heartbeatElapsed >= n.cfg.HeartbeatTicks {
			n.heartbeatElapsed = 0
			n.bcastAppend()
		}
	default:
		n.electionElapsed++
		if n.conf.IsVoter(n.id) && n.electionElapsed >= n.randomizedElectionTimeout {
			n.campaign()
		}
	}
	n.maybeApply()
	n.maybeSnapshot()
}

func (n *Node) Propose(ctx context.Context, data []byte, traceID string) (raftpb.Index, raftpb.Term, any, error) {
	if n.chaos != nil && n.chaos.Dead() {
		return 0, 0, nil, ErrDead
	}
	ch := make(chan proposeResult, 1)
	n.do(func() {
		if n.state != raftpb.StateLeader {
			ch <- proposeResult{err: ErrNotLeader}
			return
		}
		if n.tracer != nil && traceID != "" {
			n.tracer.Span(traceID, "leader_accept", string(n.id), "propose")
		}
		idx := n.appendData(raftpb.EntryNormal, data, traceID)
		n.proposals[idx] = ch
		n.maybeCommit()
		n.bcastAppend()
	})
	select {
	case r := <-ch:
		return r.index, r.term, r.res, r.err
	case <-ctx.Done():
		return 0, 0, nil, ctx.Err()
	case <-n.stop:
		return 0, 0, nil, ErrStopped
	}
}

func (n *Node) RequestVote(args raftpb.RequestVoteArgs) raftpb.RequestVoteReply {
	var reply raftpb.RequestVoteReply
	n.do(func() { reply = n.handleRequestVote(args) })
	return reply
}

func (n *Node) AppendEntries(args raftpb.AppendEntriesArgs) raftpb.AppendEntriesReply {
	var reply raftpb.AppendEntriesReply
	n.do(func() { reply = n.handleAppendEntries(args) })
	return reply
}

func (n *Node) InstallSnapshot(args raftpb.InstallSnapshotArgs) raftpb.InstallSnapshotReply {
	var reply raftpb.InstallSnapshotReply
	n.do(func() { reply = n.handleInstallSnapshot(args) })
	return reply
}

func (n *Node) appendData(typ raftpb.EntryType, data []byte, traceID string) raftpb.Index {
	e := raftpb.Entry{
		Term:  n.hs.CurrentTerm,
		Index: n.log.LastIndex() + 1,
		Type:  typ,
		Data:  data,
	}
	n.log.Append(e)
	_ = n.persistEntries([]raftpb.Entry{e}, true)
	if n.prg[n.id] != nil {
		n.prg[n.id].MatchIndex = e.Index
		n.prg[n.id].NextIndex = e.Index + 1
	}
	if n.tracer != nil && traceID != "" {
		n.tracer.Span(traceID, "wal_append", string(n.id), "")
		n.tracer.Span(traceID, "wal_fsync", string(n.id), "")
	}
	n.emit(observability.EvtWALAppend, e.Index, "wal append", map[string]any{
		"index": e.Index, "type": typ, "trace_id": traceID,
	})
	if typ == raftpb.EntryConfig {
		n.applyConfigBytes(data)
		n.pendingConfIndex = e.Index
	}
	return e.Index
}

func (n *Node) emit(typ string, index raftpb.Index, msg string, payload map[string]any) {
	ev := observability.NewEvent(string(n.id), typ, uint64(n.hs.CurrentTerm)).WithMsg("info", msg)
	for k, v := range payload {
		ev = ev.With(k, v)
	}
	if index > 0 {
		ev = ev.With("index", index)
	}
	if n.bus != nil {
		n.bus.Publish(ev)
	}
	if n.ring != nil {
		n.ring.Push(ev)
	}
	if typ == observability.EvtHeartbeat {
		return
	}
	logutil.Debug(msg, logutil.Node(string(n.id)), logutil.Term(uint64(n.hs.CurrentTerm)), logutil.Event(typ))
}

func (n *Node) Leader() raftpb.NodeID { return n.leader }

func (n *Node) LeaderAddr() string {
	if n.leader == "" {
		return ""
	}
	return n.addrs[n.leader]
}
