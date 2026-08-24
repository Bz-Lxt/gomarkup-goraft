package statemachine

import (
	"testing"

	"goraft/internal/raftpb"
)

func TestKVIdempotentAndSnapshot(t *testing.T) {
	kv := NewKV()
	b, err := Encode(Command{Op: OpPut, Key: "a", Value: "1", ClientID: "c1", Seq: 1})
	if err != nil {
		t.Fatal(err)
	}
	r1, err := kv.Apply(1, raftpb.EntryNormal, b)
	if err != nil {
		t.Fatal(err)
	}
	r2, err := kv.Apply(2, raftpb.EntryNormal, b)
	if err != nil {
		t.Fatal(err)
	}
	res := r2.(Result)
	if !res.Dup || res.Value != "1" {
		t.Fatalf("dup %+v first=%+v", res, r1)
	}
	snap, err := kv.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	kv2 := NewKV()
	if err := kv2.Restore(snap); err != nil {
		t.Fatal(err)
	}
	v, ok := kv2.Get("a")
	if !ok || v != "1" {
		t.Fatalf("restore %q %v", v, ok)
	}
}

func TestCommandValidate(t *testing.T) {
	if _, err := Encode(Command{Op: "put"}); err == nil {
		t.Fatal("empty key")
	}
	if _, err := Decode([]byte(`{"op":"nope","key":"x"}`)); err == nil {
		t.Fatal("bad op")
	}
}
