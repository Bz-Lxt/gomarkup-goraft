package api

import "net/http"

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	st := s.node.Status()
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":     !st.Dead,
		"id":     st.ID,
		"state":  st.State,
		"term":   st.Term,
		"leader": st.Leader,
		"mode":   st.Mode,
	})
}
