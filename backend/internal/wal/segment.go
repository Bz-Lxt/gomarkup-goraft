package wal

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

type segment struct {
	id   uint64
	path string
	f    *os.File
	size int64
}

func segmentName(id uint64) string {
	return fmt.Sprintf("%016x.log", id)
}

func openSegment(dir string, id uint64, create bool) (*segment, error) {
	path := filepath.Join(dir, segmentName(id))
	flag := os.O_RDWR
	if create {
		flag |= os.O_CREATE
	}
	f, err := os.OpenFile(path, flag, 0o644)
	if err != nil {
		return nil, err
	}
	info, err := f.Stat()
	if err != nil {
		_ = f.Close()
		return nil, err
	}
	sg := &segment{id: id, path: path, f: f, size: info.Size()}
	if info.Size() == 0 && create {
		if err := sg.writeHeader(); err != nil {
			_ = f.Close()
			return nil, err
		}
	} else if info.Size() > 0 {
		if err := sg.verifyHeader(); err != nil {
			_ = f.Close()
			return nil, err
		}
	}
	if _, err := f.Seek(0, io.SeekEnd); err != nil {
		_ = f.Close()
		return nil, err
	}
	return sg, nil
}

func (s *segment) writeHeader() error {
	var hdr [headerSize]byte
	copy(hdr[:8], magic)
	binary.LittleEndian.PutUint64(hdr[8:], uint64(time.Now().Unix()))
	if _, err := s.f.Write(hdr[:]); err != nil {
		return err
	}
	if err := s.f.Sync(); err != nil {
		return err
	}
	s.size = headerSize
	return nil
}

func (s *segment) verifyHeader() error {
	var hdr [headerSize]byte
	if _, err := s.f.ReadAt(hdr[:], 0); err != nil {
		return err
	}
	if string(hdr[:8]) != magic {
		return ErrBadMagic
	}
	return nil
}

func (s *segment) append(rec Record) error {
	buf := encodeRecord(nil, rec)
	n, err := s.f.Write(buf)
	if err != nil {
		return err
	}
	s.size += int64(n)
	return nil
}

func (s *segment) sync() error {
	return s.f.Sync()
}

func (s *segment) close() error {
	if s.f == nil {
		return nil
	}
	err := s.f.Close()
	s.f = nil
	return err
}

func (s *segment) truncate(off int64) error {
	if err := s.f.Truncate(off); err != nil {
		return err
	}
	s.size = off
	_, err := s.f.Seek(off, io.SeekStart)
	return err
}
