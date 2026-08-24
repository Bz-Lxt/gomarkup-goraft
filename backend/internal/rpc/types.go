package rpc

import "goraft/internal/raftpb"

type RequestVoteArgs = raftpb.RequestVoteArgs
type RequestVoteReply = raftpb.RequestVoteReply
type AppendEntriesArgs = raftpb.AppendEntriesArgs
type AppendEntriesReply = raftpb.AppendEntriesReply
type InstallSnapshotArgs = raftpb.InstallSnapshotArgs
type InstallSnapshotReply = raftpb.InstallSnapshotReply
