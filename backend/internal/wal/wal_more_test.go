package wal

import (
	"os"
	"testing"
	"time"
)

func TestWALClosedAndRotate(t *testing.T) {
	dir := t.TempDir()
	w, _, err := Open(dir, 200, time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 30; i++ {
		if err := w.AppendEntry([]byte("0123456789abcdef"), false); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Sync(); err != nil {
		t.Fatal(err)
	}
	if err := w.TruncatePrefix(1); err != nil {
		t.Fatal(err)
	}
	if w.Dir() != dir {
		t.Fatal(w.Dir())
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if err := w.AppendEntry([]byte("x"), true); err != ErrClosed {
		t.Fatalf("want closed got %v", err)
	}
	if err := w.Sync(); err != ErrClosed {
		t.Fatalf("sync closed %v", err)
	}
}

func TestOpenRejectsBadMagic(t *testing.T) {
	dir := t.TempDir()
	w, _, err := Open(dir, 1<<20, time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	_ = w.Close()
	ents, _ := os.ReadDir(dir)
	if len(ents) == 0 {
		t.Fatal("no segment")
	}
	p := dir + "/" + ents[0].Name()
	if err := os.WriteFile(p, []byte("XXXXXXXXBADMAGIC!!"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := Open(dir, 1<<20, time.Millisecond); err == nil {
		t.Fatal("expected bad magic")
	}
}

func TestAppendRejectsZeroType(t *testing.T) {
	w, _, err := Open(t.TempDir(), 1<<20, time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	if err := w.Append(Record{}); err == nil {
		t.Fatal("zero type")
	}
	if err := w.AppendCommit(7); err != nil {
		t.Fatal(err)
	}
}
