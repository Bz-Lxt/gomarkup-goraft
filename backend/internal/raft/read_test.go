package raft

import (
	"context"
	"testing"
	"time"

	"goraft/internal/statemachine"
)

func TestLinearizableReadAfterWrite(t *testing.T) {
	h := startCluster(t, []string{"n1"})
	lead := h.waitLeader(t)
	data, err := statemachine.Encode(statemachine.Command{Op: statemachine.OpPut, Key: "r", Value: "9", ClientID: "c", Seq: 1})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if _, _, _, err := lead.Propose(ctx, data, "tr_r"); err != nil {
		t.Fatal(err)
	}
	val, err := lead.LinearizableRead(ctx, "tr_r2", func() any {
		v, _ := h.kvs["n1"].Get("r")
		return v
	})
	if err != nil {
		t.Fatal(err)
	}
	if val != "9" {
		t.Fatalf("got %v", val)
	}
	st := lead.Status()
	if st.State != "leader" || st.CommitIndex == 0 {
		t.Fatalf("%+v", st)
	}
}
