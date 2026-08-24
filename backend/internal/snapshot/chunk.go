package snapshot

import "fmt"

type Chunk struct {
	Offset uint64 `json:"offset"`
	Data   []byte `json:"data"`
	Done   bool   `json:"done"`
}

func Split(data []byte, size int) []Chunk {
	if size <= 0 {
		size = 64 << 10
	}
	if len(data) == 0 {
		return []Chunk{{Offset: 0, Data: []byte{}, Done: true}}
	}
	var out []Chunk
	for off := 0; off < len(data); off += size {
		end := off + size
		if end > len(data) {
			end = len(data)
		}
		cp := append([]byte(nil), data[off:end]...)
		out = append(out, Chunk{Offset: uint64(off), Data: cp, Done: end == len(data)})
	}
	return out
}

type Assembler struct {
	expect uint64
	buf    []byte
}

func NewAssembler() *Assembler { return &Assembler{} }

func (a *Assembler) Add(ch Chunk) error {
	if ch.Offset != a.expect {
		return fmt.Errorf("snapshot: unexpected chunk offset %d want %d", ch.Offset, a.expect)
	}
	if ch.Data == nil {
		return fmt.Errorf("snapshot: nil chunk")
	}
	a.buf = append(a.buf, ch.Data...)
	a.expect += uint64(len(ch.Data))
	return nil
}

func (a *Assembler) Bytes() []byte { return a.buf }

func (a *Assembler) Reset() {
	a.expect = 0
	a.buf = nil
}
