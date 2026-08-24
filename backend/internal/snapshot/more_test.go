package snapshot

import "testing"

func TestMetaValidateAndInstall(t *testing.T) {
	if _, err := MarshalMeta(Meta{}); err == nil {
		t.Fatal("empty meta")
	}
	if _, err := UnmarshalMeta(nil); err == nil {
		t.Fatal("empty bytes")
	}
	st, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	in, err := st.BeginInstall(Meta{LastIncludedIndex: 4, LastIncludedTerm: 2, Size: 1})
	if err != nil {
		t.Fatal(err)
	}
	if err := in.Write(0, []byte("abcd"), true); err != nil {
		t.Fatal(err)
	}
	m, data, err := st.Latest()
	if err != nil || m.LastIncludedIndex != 4 || string(data) != "abcd" {
		t.Fatalf("%+v %s %v", m, data, err)
	}
	in.Abort()
}

func TestSplitEmptyAndBadAssembler(t *testing.T) {
	ch := Split(nil, 0)
	if len(ch) != 1 || !ch[0].Done {
		t.Fatalf("%+v", ch)
	}
	asm := NewAssembler()
	if err := asm.Add(Chunk{Offset: 0, Data: nil}); err == nil {
		t.Fatal("nil data")
	}
}
