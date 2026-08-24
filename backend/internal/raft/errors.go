package raft

import "errors"

var (
	ErrNotLeader       = errors.New("raft: not leader")
	ErrStopped         = errors.New("raft: stopped")
	ErrDead            = errors.New("raft: node injected down")
	ErrLeaderNotReady  = errors.New("raft: leader not ready for linearizable read")
	ErrConfInProgress  = errors.New("raft: membership change already in progress")
	ErrInvalidCommand  = errors.New("raft: invalid command")
	ErrCompacted       = errors.New("raft: index compacted")
	ErrUnavailable     = errors.New("raft: cluster unavailable")
	ErrTimeout         = errors.New("raft: request timeout")
	ErrStepDown        = errors.New("raft: stepped down")
)
