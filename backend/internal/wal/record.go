package wal

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

const (
	magic      = "GORAFT01"
	headerSize = 16
	recHdrSize = 9 // crc32 + len + type

	RecHardState byte = 1
	RecEntry     byte = 2
	RecCommit    byte = 3
)

var (
	ErrCorrupt    = errors.New("wal: corrupt record")
	ErrIncomplete = errors.New("wal: incomplete tail record")
	ErrClosed     = errors.New("wal: closed")
	ErrBadMagic   = errors.New("wal: bad segment magic")
)

type Record struct {
	Type    byte
	Payload []byte
}

func encodeRecord(buf []byte, rec Record) []byte {
	payload := rec.Payload
	n := recHdrSize + len(payload)
	if cap(buf) < n {
		buf = make([]byte, n)
	} else {
		buf = buf[:n]
	}
	binary.LittleEndian.PutUint32(buf[4:8], uint32(1+len(payload)))
	buf[8] = rec.Type
	copy(buf[9:], payload)
	crc := checksum(buf[4:])
	binary.LittleEndian.PutUint32(buf[0:4], crc)
	return buf
}

func decodeRecord(r io.Reader) (Record, error) {
	var hdr [recHdrSize]byte
	if _, err := io.ReadFull(r, hdr[:]); err != nil {
		if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
			return Record{}, ErrIncomplete
		}
		return Record{}, err
	}
	length := binary.LittleEndian.Uint32(hdr[4:8])
	if length < 1 || length > 64<<20 {
		return Record{}, fmt.Errorf("%w: invalid length %d", ErrCorrupt, length)
	}
	payload := make([]byte, length-1)
	if length > 1 {
		if _, err := io.ReadFull(r, payload); err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
				return Record{}, ErrIncomplete
			}
			return Record{}, err
		}
	}
	body := make([]byte, 4+1+len(payload))
	copy(body[:4], hdr[4:8])
	body[4] = hdr[8]
	copy(body[5:], payload)
	want := binary.LittleEndian.Uint32(hdr[0:4])
	if checksum(body) != want {
		return Record{}, ErrCorrupt
	}
	return Record{Type: hdr[8], Payload: payload}, nil
}
