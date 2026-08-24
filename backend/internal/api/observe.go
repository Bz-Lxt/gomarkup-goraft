package api

import (
	"net/http"
	"strconv"
)

func (s *Server) handleObserveState(w http.ResponseWriter, r *http.Request) {
	st := s.node.Status()
	writeJSON(w, http.StatusOK, map[string]any{
		"channel": "observe",
		"note":    "local dirty read, not linearizable",
		"status":  st,
		"kv_size": s.kv.Size(),
		"chaos":   s.chaos.Snapshot(),
	})
}

func (s *Server) handleObserveLogs(w http.ResponseWriter, r *http.Request) {
	after, _ := strconv.ParseInt(r.URL.Query().Get("after"), 10, 64)
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 200
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"events": s.ring.Snapshot(after, limit),
	})
}

func (s *Server) handleObserveTraces(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id != "" {
		writeJSON(w, http.StatusOK, s.tracer.Get(id))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"traces": s.tracer.Recent(40)})
}
