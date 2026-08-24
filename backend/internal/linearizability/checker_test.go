package linearizability

import (
	"testing"
	"time"
)

func TestLinearizableRegister(t *testing.T) {
	t0 := time.Now()
	h := History{
		{Kind: Invoke, ID: 1, Op: OpPut, Key: "a", Value: "1", Time: t0},
		{Kind: OK, ID: 1, Op: OpPut, Key: "a", Value: "1", Time: t0.Add(time.Millisecond)},
		{Kind: Invoke, ID: 2, Op: OpGet, Key: "a", Time: t0.Add(2 * time.Millisecond)},
		{Kind: OK, ID: 2, Op: OpGet, Key: "a", Value: "1", Time: t0.Add(3 * time.Millisecond)},
	}
	if err := Check(h); err != nil {
		t.Fatal(err)
	}
}

func TestRejectsStaleRead(t *testing.T) {
	t0 := time.Now()
	h := History{
		{Kind: Invoke, ID: 1, Op: OpPut, Key: "a", Value: "1", Time: t0},
		{Kind: OK, ID: 1, Op: OpPut, Key: "a", Value: "1", Time: t0.Add(time.Millisecond)},
		{Kind: Invoke, ID: 2, Op: OpGet, Key: "a", Time: t0.Add(2 * time.Millisecond)},
		{Kind: OK, ID: 2, Op: OpGet, Key: "a", Value: "", Time: t0.Add(3 * time.Millisecond)},
	}
	if err := Check(h); err == nil {
		t.Fatal("expected violation")
	}
}
