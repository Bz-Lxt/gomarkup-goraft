package api

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	"goraft/internal/observability"
	"goraft/internal/rpc"
)

type chaosBody struct {
	Peer     string  `json:"peer"`
	DropRate float64 `json:"drop_rate"`
	DelayMS  int     `json:"delay_ms"`
	Blocked  bool    `json:"blocked"`
	Peers    []string `json:"peers"`
}

func (s *Server) handleChaosKill(w http.ResponseWriter, r *http.Request) {
	s.chaos.Kill()
	s.publishChaos("kill", "")
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "dead": true})
}

func (s *Server) handleChaosRevive(w http.ResponseWriter, r *http.Request) {
	s.chaos.Revive()
	s.publishChaos("revive", "")
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "dead": false})
}

func (s *Server) handleChaosPartition(w http.ResponseWriter, r *http.Request) {
	var body chaosBody
	if err := readBody(r, &body); err != nil {
		writeErr(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	targets := body.Peers
	if body.Peer != "" {
		targets = append(targets, body.Peer)
	}
	for _, p := range targets {
		s.chaos.SetPeer(p, rpc.Rule{Blocked: true})
	}
	s.publishChaos("partition", body.Peer)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "chaos": s.chaos.Snapshot()})
}

func (s *Server) handleChaosHeal(w http.ResponseWriter, r *http.Request) {
	s.chaos.Heal()
	s.publishChaos("heal", "")
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleChaosDelay(w http.ResponseWriter, r *http.Request) {
	var body chaosBody
	if err := readBody(r, &body); err != nil {
		writeErr(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	if body.Peer == "" {
		writeErr(w, http.StatusBadRequest, "bad_request", "peer required")
		return
	}
	s.chaos.SetPeer(body.Peer, rpc.Rule{Delay: time.Duration(body.DelayMS) * time.Millisecond, DropRate: body.DropRate})
	s.publishChaos("delay", body.Peer)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleChaosDrop(w http.ResponseWriter, r *http.Request) {
	s.handleChaosDelay(w, r)
}

func (s *Server) publishChaos(action, peer string) {
	ev := observability.NewEvent(s.cfg.ID, observability.EvtChaos, 0).
		WithMsg("warn", "chaos "+action).
		With("action", action).
		With("peer", peer)
	s.bus.Publish(ev)
}

func readBody(r *http.Request, dest any) error {
	b, err := io.ReadAll(io.LimitReader(r.Body, 1<<16))
	if err != nil {
		return err
	}
	if len(b) == 0 {
		return nil
	}
	return json.Unmarshal(b, dest)
}
