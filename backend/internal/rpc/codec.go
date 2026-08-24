package rpc

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	_ = enc.Encode(v)
}

func readJSON(r io.Reader, max int64, dest any) error {
	if dest == nil {
		return fmt.Errorf("rpc: nil dest")
	}
	lr := io.LimitReader(r, max)
	dec := json.NewDecoder(lr)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dest); err != nil {
		return fmt.Errorf("rpc: decode: %w", err)
	}
	return nil
}

const maxRPCBody = 8 << 20
