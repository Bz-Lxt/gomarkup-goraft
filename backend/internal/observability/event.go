package observability

import "goraft/internal/clock"

const (
	EvtStateChange      = "state_change"
	EvtTermChange       = "term_change"
	EvtVoteRequest      = "vote_request"
	EvtVoteGranted      = "vote_granted"
	EvtVoteDenied       = "vote_denied"
	EvtHeartbeat        = "heartbeat"
	EvtAppendSend       = "append_send"
	EvtAppendAck        = "append_ack"
	EvtAppendReject     = "append_reject"
	EvtWALAppend        = "wal_append"
	EvtWALSync          = "wal_sync"
	EvtCommit           = "commit"
	EvtApply            = "apply"
	EvtSnapshotStart    = "snapshot_start"
	EvtSnapshotDone     = "snapshot_done"
	EvtSnapshotInstall  = "snapshot_install"
	EvtChaos            = "chaos"
	EvtTrace            = "trace_span"
	EvtMembership       = "membership"
	EvtReadIndex        = "read_index"
	EvtLog              = "log"
)

type Event struct {
	TS         string         `json:"ts"`
	TSUnixUS   int64          `json:"ts_unix_us"`
	NodeID     string         `json:"node_id"`
	Type       string         `json:"type"`
	Term       uint64         `json:"term"`
	TraceID    string         `json:"trace_id,omitempty"`
	Level      string         `json:"level,omitempty"`
	Message    string         `json:"message,omitempty"`
	Payload    map[string]any `json:"payload"`
}

func NewEvent(nodeID, typ string, term uint64) Event {
	now := clock.Now()
	return Event{
		TS:       clock.FormatPrecise(now),
		TSUnixUS: now.UnixMicro(),
		NodeID:   nodeID,
		Type:     typ,
		Term:     term,
		Level:    "info",
		Payload:  map[string]any{},
	}
}

func (e Event) WithTrace(id string) Event {
	e.TraceID = id
	return e
}

func (e Event) WithMsg(level, msg string) Event {
	e.Level = level
	e.Message = msg
	return e
}

func (e Event) With(k string, v any) Event {
	if e.Payload == nil {
		e.Payload = map[string]any{}
	}
	e.Payload[k] = v
	return e
}
