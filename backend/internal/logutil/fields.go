package logutil

import "log/slog"

func Node(id string) slog.Attr   { return slog.String("node", id) }
func Term(t uint64) slog.Attr    { return slog.Uint64("term", t) }
func Index(i uint64) slog.Attr   { return slog.Uint64("index", i) }
func Trace(id string) slog.Attr  { return slog.String("trace_id", id) }
func Peer(id string) slog.Attr   { return slog.String("peer", id) }
func Err(err error) slog.Attr    { return slog.Any("err", err) }
func State(s string) slog.Attr   { return slog.String("state", s) }
func Event(name string) slog.Attr { return slog.String("event", name) }
