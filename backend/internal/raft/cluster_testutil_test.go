package raft

import (
	"io"
	"path/filepath"
	"testing"
	"time"

	"goraft/internal/config"
	"goraft/internal/logutil"
	"goraft/internal/observability"
	"goraft/internal/rpc"
	"goraft/internal/statemachine"
)

func testCfg(t *testing.T, id string, peers map[string]string) *config.Config {
	t.Helper()
	c := &config.Config{
		ID:                id,
		HTTPAddr:          ":0",
		AdvertiseAddr:     id,
		DataDir:           t.TempDir(),
		Peers:             peers,
		Mode:              config.ModeDemo,
		LogLevel:          "error",
		TickInterval:      5 * time.Millisecond,
		HeartbeatTicks:    8,
		ElectionTicksMin:  40,
		ElectionTicksMax:  80,
		WALSegmentBytes:   1 << 20,
		SnapshotLogN:      80,
		SnapshotBytes:     1 << 20,
		SnapshotChunkSize: 1024,
		RPCTimeout:        300 * time.Millisecond,
		GroupCommitWin:    time.Millisecond,
	}
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
	return c
}

type harness struct {
	nodes map[string]*Node
	kvs   map[string]*statemachine.KV
	net   *MemoryTransport
}

func startCluster(t *testing.T, ids []string) *harness {
	t.Helper()
	logutil.Init("error", io.Discard)
	peers := map[string]string{}
	for _, id := range ids {
		peers[id] = id
	}
	h := &harness{nodes: map[string]*Node{}, kvs: map[string]*statemachine.KV{}, net: NewMemoryTransport()}
	for _, id := range ids {
		cfg := testCfg(t, id, peers)
		cfg.DataDir = filepath.Join(t.TempDir(), id)
		kv := statemachine.NewKV()
		n, err := New(cfg, kv, observability.NewBus(), observability.NewTracer(), observability.NewRing(128), rpc.NewChaos())
		if err != nil {
			t.Fatal(err)
		}
		n.SetTransport(h.net)
		h.net.Register(n)
		n.Start()
		h.nodes[id] = n
		h.kvs[id] = kv
		t.Cleanup(n.Stop)
	}
	return h
}

func (h *harness) waitLeader(t *testing.T) *Node {
	t.Helper()
	deadline := time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) {
		for _, n := range h.nodes {
			st := n.Status()
			if st.State == "leader" && !st.Dead {
				return n
			}
		}
		time.Sleep(15 * time.Millisecond)
	}
	t.Fatal("no leader")
	return nil
}
