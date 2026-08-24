package raft

import (
	"context"
	"testing"
	"time"
)

func TestProposeSurfacesStateMachineApplyError(t *testing.T) {
	h := startCluster(t, []string{"n1"})
	lead := h.waitLeader(t)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	idx, term, _, err := lead.Propose(ctx, []byte(`{"op":"replace","key":"rejected","client_id":"c","seq":1}`), "")
	if err == nil {
		t.Fatalf("proposal at index %d term %d succeeded after the state machine rejected its command", idx, term)
	}
	if _, found := h.kvs["n1"].Get("rejected"); found {
		t.Fatal("rejected command changed the state machine")
	}
}
