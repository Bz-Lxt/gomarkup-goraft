package raft

import (
	"context"
	"testing"
	"time"

	"goraft/internal/raftpb"
	"goraft/internal/statemachine"
)

func TestProposeReplicatesAndRead(t *testing.T) {
	h := startCluster(t, []string{"n1", "n2", "n3"})
	lead := h.waitLeader(t)
	data, err := statemachine.Encode(statemachine.Command{Op: statemachine.OpPut, Key: "k", Value: "v", ClientID: "c", Seq: 1})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, _, _, err = lead.Propose(ctx, data, "tr_test")
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		ok := 0
		for _, kv := range h.kvs {
			if v, found := kv.Get("k"); found && v == "v" {
				ok++
			}
		}
		if ok >= 2 {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("value not replicated")
}

func TestMinorityPartitionCannotCommit(t *testing.T) {
	h := startCluster(t, []string{"n1", "n2", "n3"})
	lead := h.waitLeader(t)
	var others []raftpb.NodeID
	for id := range h.nodes {
		if id != string(lead.ID()) {
			others = append(others, raftpb.NodeID(id))
		}
	}
	h.net.Partition(lead.ID(), others[0], true)
	h.net.Partition(lead.ID(), others[1], true)

	data, _ := statemachine.Encode(statemachine.Command{Op: statemachine.OpPut, Key: "x", Value: "1", ClientID: "c", Seq: 2})
	ctx, cancel := context.WithTimeout(context.Background(), 400*time.Millisecond)
	defer cancel()
	if _, _, _, err := lead.Propose(ctx, data, "tr_part"); err == nil {
		t.Fatal("expected propose to time out without majority")
	}
}
