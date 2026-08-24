package api

import (
	"net/http"
	"time"

	"goraft/internal/config"
	"goraft/internal/observability"
	"goraft/internal/raft"
	"goraft/internal/rpc"
	"goraft/internal/statemachine"
)

type Server struct {
	cfg    *config.Config
	node   *raft.Node
	kv     *statemachine.KV
	chaos  *rpc.Chaos
	bus    *observability.Bus
	tracer *observability.Tracer
	ring   *observability.Ring
	fan    *observability.Fanout
}

func New(cfg *config.Config, node *raft.Node, kv *statemachine.KV, chaos *rpc.Chaos, bus *observability.Bus, tracer *observability.Tracer, ring *observability.Ring) *Server {
	s := &Server{cfg: cfg, node: node, kv: kv, chaos: chaos, bus: bus, tracer: tracer, ring: ring, fan: observability.NewFanout()}
	s.bindBus()
	return s
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", s.handleHealth)
	mux.HandleFunc("GET /api/v1/cluster", s.handleCluster)
	mux.HandleFunc("GET /api/v1/kv/{key}", s.handleGetKV)
	mux.HandleFunc("PUT /api/v1/kv/{key}", s.handlePutKV)
	mux.HandleFunc("DELETE /api/v1/kv/{key}", s.handleDeleteKV)
	mux.HandleFunc("GET /api/v1/kv", s.handleListKV)
	mux.HandleFunc("GET /api/v1/observe/state", s.handleObserveState)
	mux.HandleFunc("GET /api/v1/observe/logs", s.handleObserveLogs)
	mux.HandleFunc("GET /api/v1/observe/traces", s.handleObserveTraces)
	mux.HandleFunc("POST /api/v1/chaos/kill", s.handleChaosKill)
	mux.HandleFunc("POST /api/v1/chaos/revive", s.handleChaosRevive)
	mux.HandleFunc("POST /api/v1/chaos/partition", s.handleChaosPartition)
	mux.HandleFunc("POST /api/v1/chaos/heal", s.handleChaosHeal)
	mux.HandleFunc("POST /api/v1/chaos/delay", s.handleChaosDelay)
	mux.HandleFunc("POST /api/v1/chaos/drop", s.handleChaosDrop)
	mux.HandleFunc("POST /api/v1/members", s.handleAddMember)
	mux.HandleFunc("DELETE /api/v1/members/{id}", s.handleRemoveMember)
	mux.HandleFunc("GET /api/v1/ws", s.handleWS)
	rpc.Mount(mux, s.node, s.chaos)
	return withCORS(withLogging(mux))
}

func (s *Server) HTTPServer() *http.Server {
	return &http.Server{
		Addr:              s.cfg.HTTPAddr,
		Handler:           s.Handler(),
		ReadHeaderTimeout: 4 * time.Second,
	}
}
