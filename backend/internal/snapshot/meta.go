package snapshot

import (
	"encoding/json"
	"fmt"
)

type Meta struct {
	LastIncludedIndex uint64   `json:"last_included_index"`
	LastIncludedTerm  uint64   `json:"last_included_term"`
	CRC32             uint32   `json:"crc32"`
	Size              int64    `json:"size"`
	Voters            []string `json:"voters"`
	Learners          []string `json:"learners"`
}

func (m Meta) Validate() error {
	if m.LastIncludedIndex == 0 && m.LastIncludedTerm == 0 && m.Size == 0 {
		return fmt.Errorf("snapshot: empty meta")
	}
	if m.Size < 0 {
		return fmt.Errorf("snapshot: negative size")
	}
	return nil
}

func MarshalMeta(m Meta) ([]byte, error) {
	if err := m.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(m)
}

func UnmarshalMeta(b []byte) (Meta, error) {
	var m Meta
	if len(b) == 0 {
		return m, fmt.Errorf("snapshot: empty meta bytes")
	}
	if err := json.Unmarshal(b, &m); err != nil {
		return m, fmt.Errorf("snapshot: meta json: %w", err)
	}
	return m, m.Validate()
}
