package snapshot

import (
	"fmt"
	"os"
	"path/filepath"
)

type Incoming struct {
	Meta  Meta
	asm   *Assembler
	dir   string
	tmp   string
}

func (s *Store) BeginInstall(meta Meta) (*Incoming, error) {
	if err := meta.Validate(); err != nil {
		return nil, err
	}
	tmp := filepath.Join(s.dir, fmt.Sprintf("incoming-%016x.tmp", meta.LastIncludedIndex))
	return &Incoming{Meta: meta, asm: NewAssembler(), dir: s.dir, tmp: tmp}, nil
}

func (in *Incoming) Write(offset uint64, data []byte, done bool) error {
	if data == nil {
		return fmt.Errorf("snapshot: nil install chunk")
	}
	if err := in.asm.Add(Chunk{Offset: offset, Data: data, Done: done}); err != nil {
		return err
	}
	if !done {
		return nil
	}
	st, err := Open(in.dir)
	if err != nil {
		return err
	}
	return st.Save(in.Meta, in.asm.Bytes())
}

func (in *Incoming) Abort() {
	_ = os.Remove(in.tmp)
	in.asm.Reset()
}
