package rpc

import (
	"net/http"

	"goraft/internal/raftpb"
)

type RaftHandler interface {
	RequestVote(args raftpb.RequestVoteArgs) raftpb.RequestVoteReply
	AppendEntries(args raftpb.AppendEntriesArgs) raftpb.AppendEntriesReply
	InstallSnapshot(args raftpb.InstallSnapshotArgs) raftpb.InstallSnapshotReply
}

func Mount(mux *http.ServeMux, h RaftHandler, chaos *Chaos) {
	mux.HandleFunc("POST /raft/request-vote", func(w http.ResponseWriter, r *http.Request) {
		if chaos != nil && chaos.Dead() {
			http.Error(w, `{"error":"dead"}`, http.StatusServiceUnavailable)
			return
		}
		var args raftpb.RequestVoteArgs
		if err := readJSON(r.Body, maxRPCBody, &args); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if args.Term == 0 || args.CandidateID == "" {
			http.Error(w, "invalid request-vote", http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, h.RequestVote(args))
	})
	mux.HandleFunc("POST /raft/append-entries", func(w http.ResponseWriter, r *http.Request) {
		if chaos != nil && chaos.Dead() {
			http.Error(w, `{"error":"dead"}`, http.StatusServiceUnavailable)
			return
		}
		from := r.Header.Get("X-Raft-From")
		if chaos != nil && from != "" {
			drop, _ := chaos.Intercept(from)
			if drop {
				http.Error(w, `{"error":"partition"}`, http.StatusServiceUnavailable)
				return
			}
		}
		var args raftpb.AppendEntriesArgs
		if err := readJSON(r.Body, maxRPCBody, &args); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if args.Term == 0 || args.LeaderID == "" {
			http.Error(w, "invalid append-entries", http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, h.AppendEntries(args))
	})
	mux.HandleFunc("POST /raft/install-snapshot", func(w http.ResponseWriter, r *http.Request) {
		if chaos != nil && chaos.Dead() {
			http.Error(w, `{"error":"dead"}`, http.StatusServiceUnavailable)
			return
		}
		var args raftpb.InstallSnapshotArgs
		if err := readJSON(r.Body, maxRPCBody, &args); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if args.Term == 0 || args.LeaderID == "" {
			http.Error(w, "invalid install-snapshot", http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, h.InstallSnapshot(args))
	})
}
