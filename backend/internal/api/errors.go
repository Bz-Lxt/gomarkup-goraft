package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"goraft/internal/raft"
)

type ErrorBody struct {
	Error      string `json:"error"`
	Code       string `json:"code"`
	Leader     string `json:"leader,omitempty"`
	LeaderAddr string `json:"leader_addr,omitempty"`
}

func writeErr(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(ErrorBody{Error: msg, Code: code})
}

func writeRedirect(w http.ResponseWriter, err error, leader, addr string) {
	status := http.StatusConflict
	code := "internal"
	msg := err.Error()
	if errors.Is(err, raft.ErrNotLeader) || errors.Is(err, raft.ErrLeaderNotReady) {
		status = http.StatusConflict
		code = "not_leader"
	} else if errors.Is(err, raft.ErrDead) {
		status = http.StatusServiceUnavailable
		code = "dead"
	} else if errors.Is(err, raft.ErrTimeout) {
		status = http.StatusGatewayTimeout
		code = "timeout"
	} else if errors.Is(err, raft.ErrInvalidCommand) {
		status = http.StatusBadRequest
		code = "bad_request"
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if addr != "" {
		w.Header().Set("Location", "http://"+addr)
	}
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(ErrorBody{Error: msg, Code: code, Leader: leader, LeaderAddr: addr})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
