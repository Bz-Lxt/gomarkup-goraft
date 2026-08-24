package snapshot

import (
	"bytes"
	"os"
	"testing"
)

func TestSaveLatestAndCRC(t *testing.T) {
	dir := t.TempDir()
	st, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	data := []byte(`{"a":1}`)
	if err := st.Save(Meta{LastIncludedIndex: 12, LastIncludedTerm: 3, Voters: []string{"n1"}}, data); err != nil {
		t.Fatal(err)
	}
	m, got, err := st.Latest()
	if err != nil {
		t.Fatal(err)
	}
	if m.LastIncludedIndex != 12 || !bytes.Equal(got, data) {
		t.Fatalf("meta=%+v data=%s", m, got)
	}
	if err := st.Save(Meta{LastIncludedIndex: 20, LastIncludedTerm: 4}, []byte("x")); err != nil {
		t.Fatal(err)
	}
	m2, _, err := st.Latest()
	if err != nil {
		t.Fatal(err)
	}
	if m2.LastIncludedIndex != 20 {
		t.Fatalf("latest=%d", m2.LastIncludedIndex)
	}
}

func TestChunkRoundtrip(t *testing.T) {
	raw := bytes.Repeat([]byte("z"), 300)
	chunks := Split(raw, 128)
	asm := NewAssembler()
	for _, c := range chunks {
		if err := asm.Add(c); err != nil {
			t.Fatal(err)
		}
	}
	if !bytes.Equal(asm.Bytes(), raw) {
		t.Fatal("mismatch")
	}
	if err := asm.Add(Chunk{Offset: 1, Data: []byte("a")}); err == nil {
		t.Fatal("expected offset error")
	}
}

func TestRejectCorruptFile(t *testing.T) {
	dir := t.TempDir()
	st, _ := Open(dir)
	_ = st.Save(Meta{LastIncludedIndex: 1, LastIncludedTerm: 1}, []byte("ok"))
	files, _ := listSnapFiles(dir)
	if len(files) == 0 {
		t.Fatal("no file")
	}
	b, _ := os.ReadFile(files[0])
	b[len(b)-1] ^= 0xff
	_ = os.WriteFile(files[0], b, 0o644)
	if _, _, err := st.Latest(); err == nil {
		t.Fatal("expected crc error")
	}
}
