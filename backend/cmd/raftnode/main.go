package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"goraft/internal/api"
	"goraft/internal/config"
	"goraft/internal/logutil"
	"goraft/internal/observability"
	"goraft/internal/raft"
	"goraft/internal/raftpb"
	"goraft/internal/rpc"
	"goraft/internal/statemachine"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}
	logutil.Init(cfg.LogLevel, os.Stdout)
	if err := os.MkdirAll(cfg.DataDir, 0o755); err != nil {
		panic(err)
	}

	kv := statemachine.NewKV()
	bus := observability.NewBus()
	tracer := observability.NewTracer()
	ring := observability.NewRing(8192)
	chaos := rpc.NewChaos()

	node, err := raft.New(cfg, kv, bus, tracer, ring, chaos)
	if err != nil {
		logutil.Error("raft init failed", logutil.Err(err))
		os.Exit(1)
	}
	pool := rpc.NewPool(cfg.RPCTimeout)
	client := rpc.NewClient(pool, chaos, cfg.ID)
	addrs := map[raftpb.NodeID]string{}
	for id, addr := range cfg.Peers {
		addrs[raftpb.NodeID(id)] = addr
	}
	node.SetTransport(raft.NewHTTPTransport(client, addrs))
	node.Start()

	srv := api.New(cfg, node, kv, chaos, bus, tracer, ring).HTTPServer()
	logutil.Info("raftnode listen", logutil.Node(cfg.ID), "addr", cfg.HTTPAddr, "mode", string(cfg.Mode))

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logutil.Error("http exit", logutil.Err(err))
			os.Exit(1)
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
	node.Stop()
	pool.Close()
}
