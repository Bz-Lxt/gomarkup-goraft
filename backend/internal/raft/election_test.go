package raft

import (
	"testing"
	"time"
)

func TestSingleNodeBecomesLeader(t *testing.T) {
	h := startCluster(t, []string{"n1"})
	n := h.waitLeader(t)
	if n.ID() != "n1" {
		t.Fatalf("leader=%s", n.ID())
	}
}

func TestThreeNodeElectsOneLeader(t *testing.T) {
	h := startCluster(t, []string{"n1", "n2", "n3"})
	lead := h.waitLeader(t)
	time.Sleep(80 * time.Millisecond)
	leaders := 0
	for _, n := range h.nodes {
		if n.Status().State == "leader" {
			leaders++
		}
	}
	if leaders != 1 {
		t.Fatalf("leaders=%d first=%s", leaders, lead.ID())
	}
}
