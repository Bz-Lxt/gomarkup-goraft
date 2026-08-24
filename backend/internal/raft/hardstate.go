package raft

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"goraft/internal/raftpb"
	"goraft/internal/wal"
)

func encodeHardState(hs raftpb.HardState) ([]byte, error) {
	if hs.CurrentTerm == 0 && hs.VotedFor == "" && hs.Commit == 0 {
		return json.Marshal(hs)
	}
	return json.Marshal(hs)
}

func decodeHardState(b []byte) (raftpb.HardState, error) {
	var hs raftpb.HardState
	if len(b) == 0 {
		return hs, nil
	}
	if err := json.Unmarshal(b, &hs); err != nil {
		return hs, fmt.Errorf("hardstate: %w", err)
	}
	return hs, nil
}

func persistHardStateFile(dir string, hs raftpb.HardState) error {
	b, err := encodeHardState(hs)
	if err != nil {
		return err
	}
	path := filepath.Join(dir, "hardstate.json")
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	f, err := os.Open(tmp)
	if err != nil {
		return err
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return err
	}
	_ = f.Close()
	return os.Rename(tmp, path)
}

func loadHardStateFile(dir string) (raftpb.HardState, error) {
	b, err := os.ReadFile(filepath.Join(dir, "hardstate.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return raftpb.HardState{}, nil
		}
		return raftpb.HardState{}, err
	}
	return decodeHardState(b)
}

func (n *Node) persistHardState() error {
	n.hs.Commit = n.commitIndex
	b, err := encodeHardState(n.hs)
	if err != nil {
		return err
	}
	if err := n.wal.AppendHardState(b); err != nil {
		return err
	}
	return persistHardStateFile(n.cfg.DataDir, n.hs)
}

func (n *Node) persistEntries(ents []raftpb.Entry, sync bool) error {
	for i, e := range ents {
		b, err := encodeEntry(e)
		if err != nil {
			return err
		}
		wait := sync && i == len(ents)-1
		if err := n.wal.AppendEntry(b, wait); err != nil {
			return err
		}
	}
	return nil
}

func restoreLog(wrec wal.Recovered) (*Log, raftpb.HardState, error) {
	lg := NewLog()
	hs, err := decodeHardState(wrec.HardState)
	if err != nil {
		return nil, hs, err
	}
	for _, raw := range wrec.Entries {
		e, err := decodeEntry(raw)
		if err != nil {
			return nil, hs, err
		}
		if e.Index <= lg.LastIndex() {
			lg.TruncateFrom(e.Index)
		}
		lg.Append(e)
	}
	if raftpb.Index(wrec.Commit) > hs.Commit {
		hs.Commit = raftpb.Index(wrec.Commit)
	}
	return lg, hs, nil
}
