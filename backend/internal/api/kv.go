package api

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"goraft/internal/observability"
	"goraft/internal/statemachine"
)

type kvBody struct {
	Value    string `json:"value"`
	ClientID string `json:"client_id"`
	Seq      uint64 `json:"seq"`
}

func (s *Server) handlePutKV(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	if strings.TrimSpace(key) == "" {
		writeErr(w, http.StatusBadRequest, "bad_request", "key required")
		return
	}
	raw, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "bad_request", "read body")
		return
	}
	var body kvBody
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &body); err != nil {
			writeErr(w, http.StatusBadRequest, "bad_request", "invalid json")
			return
		}
	}
	if body.ClientID == "" {
		body.ClientID = r.Header.Get("X-Client-Id")
	}
	if body.Seq == 0 {
		body.Seq = uint64(time.Now().UnixNano())
	}
	cmd := statemachine.Command{Op: statemachine.OpPut, Key: key, Value: body.Value, ClientID: body.ClientID, Seq: body.Seq}
	s.propose(w, r, cmd)
}

func (s *Server) handleDeleteKV(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	cmd := statemachine.Command{Op: statemachine.OpDelete, Key: key, ClientID: r.Header.Get("X-Client-Id"), Seq: uint64(time.Now().UnixNano())}
	s.propose(w, r, cmd)
}

func (s *Server) propose(w http.ResponseWriter, r *http.Request, cmd statemachine.Command) {
	data, err := statemachine.Encode(cmd)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	traceID := observability.NewTraceID()
	s.tracer.Start(traceID, cmd.Op, cmd.Key)
	s.tracer.Span(traceID, "client_recv", s.cfg.ID, cmd.Key)
	idx, term, res, err := s.node.Propose(r.Context(), data, traceID)
	if err != nil {
		s.tracer.Finish(traceID)
		writeRedirect(w, err, string(s.node.Leader()), s.node.LeaderAddr())
		return
	}
	s.tracer.Span(traceID, "commit", s.cfg.ID, strconv.FormatUint(uint64(idx), 10))
	s.tracer.Span(traceID, "apply", s.cfg.ID, "")
	s.tracer.Span(traceID, "client_reply", s.cfg.ID, "")
	tr := s.tracer.Finish(traceID)
	writeJSON(w, http.StatusOK, map[string]any{
		"ok": true, "index": idx, "term": term, "result": res, "trace_id": traceID, "trace": tr,
	})
}

func (s *Server) handleGetKV(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	stale := r.URL.Query().Get("stale") == "true"
	if stale {
		v, ok := s.kv.Get(key)
		writeJSON(w, http.StatusOK, map[string]any{"key": key, "value": v, "found": ok, "stale": true, "channel": "observe"})
		return
	}
	traceID := observability.NewTraceID()
	s.tracer.Start(traceID, "get", key)
	val, err := s.node.LinearizableRead(r.Context(), traceID, func() any {
		v, ok := s.kv.Get(key)
		return map[string]any{"key": key, "value": v, "found": ok}
	})
	if err != nil {
		s.tracer.Finish(traceID)
		writeRedirect(w, err, string(s.node.Leader()), s.node.LeaderAddr())
		return
	}
	s.tracer.Finish(traceID)
	m, _ := val.(map[string]any)
	if m == nil {
		m = map[string]any{}
	}
	m["stale"] = false
	m["channel"] = "linearizable"
	m["trace_id"] = traceID
	writeJSON(w, http.StatusOK, m)
}

func (s *Server) handleListKV(w http.ResponseWriter, r *http.Request) {
	prefix := r.URL.Query().Get("prefix")
	writeJSON(w, http.StatusOK, map[string]any{
		"items":   s.kv.Prefix(prefix),
		"stale":   true,
		"channel": "observe",
	})
}
