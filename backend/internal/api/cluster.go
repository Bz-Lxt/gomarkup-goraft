package api

import (
	"net/http"

	"goraft/internal/raftpb"
)

func (s *Server) handleCluster(w http.ResponseWriter, r *http.Request) {
	st := s.node.Status()
	writeJSON(w, http.StatusOK, map[string]any{
		"self":   st,
		"peers":  s.cfg.Peers,
		"mode":   s.cfg.Mode,
		"voters": st.Voters,
	})
}

type memberBody struct {
	ID      string `json:"id"`
	Addr    string `json:"addr"`
	Learner bool   `json:"learner"`
}

func (s *Server) handleAddMember(w http.ResponseWriter, r *http.Request) {
	var body memberBody
	if err := readBody(r, &body); err != nil || body.ID == "" {
		writeErr(w, http.StatusBadRequest, "bad_request", "id required")
		return
	}
	err := s.node.ChangeMembership(raftpb.NodeID(body.ID), body.Addr, "", true)
	if err != nil {
		writeRedirect(w, err, string(s.node.Leader()), s.node.LeaderAddr())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "id": body.ID, "learner": true})
}

func (s *Server) handleRemoveMember(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	err := s.node.ChangeMembership("", "", raftpb.NodeID(id), false)
	if err != nil {
		writeRedirect(w, err, string(s.node.Leader()), s.node.LeaderAddr())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "removed": id})
}
