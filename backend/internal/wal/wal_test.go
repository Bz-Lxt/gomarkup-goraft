package wal

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWALAppendRecoverAndCorruptTail(t *testing.T) {
	dir := t.TempDir()
	w, rec, err := Open(dir, 1<<20, time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	if len(rec.Entries) != 0 {
		t.Fatalf("fresh wal entries=%d", len(rec.Entries))
	}
	if err := w.AppendEntry([]byte("e1"), true); err != nil {
		t.Fatal(err)
	}
	if err := w.AppendEntry([]byte("e2"), true); err != nil {
		t.Fatal(err)
	}
	if err := w.AppendHardState([]byte("hs")); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("double close: %v", err)
	}

	w2, rec2, err := Open(dir, 1<<20, time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(rec2.HardState, []byte("hs")) {
		t.Fatalf("hardstate=%q", rec2.HardState)
	}
	if len(rec2.Entries) != 2 || string(rec2.Entries[1]) != "e2" {
		t.Fatalf("entries=%v", rec2.Entries)
	}
	if err := w2.Close(); err != nil {
		t.Fatal(err)
	}

	logs, err := filepath.Glob(filepath.Join(dir, "*.log"))
	if err != nil || len(logs) == 0 {
		t.Fatalf("no logs: %v", err)
	}
	f, err := os.OpenFile(logs[0], os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.Write([]byte{0x01, 0x02, 0x03}); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()

	w3, rec3, err := Open(dir, 1<<20, time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	if len(rec3.Entries) < 2 {
		t.Fatalf("truncated recover lost good records: %d", len(rec3.Entries))
	}
	_ = w3.Close()
}

func TestRecordCRCRejectsTamper(t *testing.T) {
	rec := encodeRecord(nil, Record{Type: RecEntry, Payload: []byte("abc")})
	rec[10] ^= 0xff
	_, err := decodeRecord(bytes.NewReader(rec))
	if err == nil {
		t.Fatal("expected corrupt")
	}
}
