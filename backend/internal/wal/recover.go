package wal

import (
	"encoding/binary"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

type Recovered struct {
	HardState []byte
	Entries   [][]byte
	Commit    uint64
}

func listSegmentIDs(dir string) ([]uint64, error) {
	ents, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var ids []uint64
	for _, e := range ents {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".log") {
			continue
		}
		hex := strings.TrimSuffix(name, ".log")
		id, err := strconv.ParseUint(hex, 16, 64)
		if err != nil {
			continue
		}
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids, nil
}

func recoverDir(dir string) (Recovered, uint64, error) {
	var out Recovered
	ids, err := listSegmentIDs(dir)
	if err != nil {
		return out, 0, err
	}
	var lastID uint64
	for _, id := range ids {
		lastID = id
		path := filepath.Join(dir, segmentName(id))
		if err := replaySegment(path, &out); err != nil && !errors.Is(err, ErrIncomplete) && !errors.Is(err, ErrCorrupt) {
			return out, lastID, err
		}
	}
	return out, lastID, nil
}

func replaySegment(path string, out *Recovered) error {
	f, err := os.OpenFile(path, os.O_RDWR, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return err
	}
	if info.Size() < headerSize {
		return f.Truncate(0)
	}
	var hdr [headerSize]byte
	if _, err := io.ReadFull(f, hdr[:]); err != nil {
		return err
	}
	if string(hdr[:8]) != magic {
		return ErrBadMagic
	}
	valid := int64(headerSize)
	for {
		pos, _ := f.Seek(0, io.SeekCurrent)
		rec, err := decodeRecord(f)
		if err != nil {
			if errors.Is(err, ErrIncomplete) || errors.Is(err, ErrCorrupt) || errors.Is(err, io.EOF) {
				if pos < info.Size() {
					if terr := f.Truncate(pos); terr != nil {
						return terr
					}
				}
				if errors.Is(err, ErrCorrupt) {
					return err
				}
				return nil
			}
			return err
		}
		switch rec.Type {
		case RecHardState:
			out.HardState = rec.Payload
		case RecEntry:
			cp := append([]byte(nil), rec.Payload...)
			out.Entries = append(out.Entries, cp)
		case RecCommit:
			if len(rec.Payload) >= 8 {
				out.Commit = binary.LittleEndian.Uint64(rec.Payload[:8])
			}
		}
		valid, _ = f.Seek(0, io.SeekCurrent)
		_ = valid
	}
}
