package snapshot

import (
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"os"
	"path/filepath"
	"sort"
	"sync"
)

const (
	fileMagic   = "GOSNAP01"
	metaMaxSize = 1 << 20
)

type Store struct {
	dir string
	mu  sync.Mutex
}

func Open(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return &Store{dir: dir}, nil
}

func (s *Store) Save(meta Meta, data []byte) error {
	if data == nil {
		return fmt.Errorf("snapshot: nil data")
	}
	meta.Size = int64(len(data))
	meta.CRC32 = crc32.ChecksumIEEE(data)
	mb, err := MarshalMeta(meta)
	if err != nil {
		return err
	}
	name := fmt.Sprintf("snap-%016x-%016x.bin", meta.LastIncludedIndex, meta.LastIncludedTerm)
	path := filepath.Join(s.dir, name)
	tmp := path + ".tmp"

	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	var hdr [16]byte
	copy(hdr[:8], fileMagic)
	binary.LittleEndian.PutUint32(hdr[8:12], uint32(len(mb)))
	if _, err := f.Write(hdr[:]); err != nil {
		_ = f.Close()
		return err
	}
	if _, err := f.Write(mb); err != nil {
		_ = f.Close()
		return err
	}
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.gcLocked(meta.LastIncludedIndex)
}

func (s *Store) Latest() (Meta, []byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	files, err := listSnapFiles(s.dir)
	if err != nil {
		return Meta{}, nil, err
	}
	if len(files) == 0 {
		return Meta{}, nil, os.ErrNotExist
	}
	return readSnap(files[len(files)-1])
}

func (s *Store) gcLocked(keepFrom uint64) error {
	files, err := listSnapFiles(s.dir)
	if err != nil {
		return err
	}
	for i := 0; i < len(files)-1; i++ {
		m, _, err := readSnap(files[i])
		if err != nil || m.LastIncludedIndex < keepFrom {
			_ = os.Remove(files[i])
		}
	}
	return nil
}

func listSnapFiles(dir string) ([]string, error) {
	ents, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var files []string
	for _, e := range ents {
		if e.IsDir() || filepath.Ext(e.Name()) != ".bin" {
			continue
		}
		files = append(files, filepath.Join(dir, e.Name()))
	}
	sort.Strings(files)
	return files, nil
}

func readSnap(path string) (Meta, []byte, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Meta{}, nil, err
	}
	if len(b) < 16 || string(b[:8]) != fileMagic {
		return Meta{}, nil, fmt.Errorf("snapshot: bad magic")
	}
	metaLen := binary.LittleEndian.Uint32(b[8:12])
	if metaLen == 0 || metaLen > metaMaxSize || int(16+metaLen) > len(b) {
		return Meta{}, nil, fmt.Errorf("snapshot: bad meta length")
	}
	meta, err := UnmarshalMeta(b[16 : 16+metaLen])
	if err != nil {
		return Meta{}, nil, err
	}
	data := b[16+metaLen:]
	if uint32(len(data)) != 0 && crc32.ChecksumIEEE(data) != meta.CRC32 {
		return Meta{}, nil, fmt.Errorf("snapshot: crc mismatch")
	}
	return meta, data, nil
}
