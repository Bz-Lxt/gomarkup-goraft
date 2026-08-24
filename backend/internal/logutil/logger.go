package logutil

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"

	"goraft/internal/clock"
)

var (
	mu     sync.RWMutex
	global *slog.Logger
	level  slog.Level
)

func init() {
	Init("info", os.Stdout)
}

func Init(levelName string, w io.Writer) {
	mu.Lock()
	defer mu.Unlock()
	level = parseLevel(levelName)
	if w == nil {
		w = os.Stdout
	}
	h := slog.NewJSONHandler(w, &slog.HandlerOptions{
		Level: level,
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				if t, ok := a.Value.Any().(interface{ Time() interface{} }); ok {
					_ = t
				}
				return slog.String(slog.TimeKey, clock.FormatNowPrecise())
			}
			return a
		},
	})
	global = slog.New(&beijingHandler{inner: h})
	slog.SetDefault(global)
}

func parseLevel(name string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func L() *slog.Logger {
	mu.RLock()
	defer mu.RUnlock()
	if global == nil {
		return slog.Default()
	}
	return global
}

func With(args ...any) *slog.Logger {
	return L().With(args...)
}

func Debug(msg string, args ...any) { L().Debug(msg, args...) }
func Info(msg string, args ...any)  { L().Info(msg, args...) }
func Warn(msg string, args ...any)  { L().Warn(msg, args...) }
func Error(msg string, args ...any) { L().Error(msg, args...) }

type beijingHandler struct{ inner slog.Handler }

func (h *beijingHandler) Enabled(ctx context.Context, l slog.Level) bool {
	return h.inner.Enabled(ctx, l)
}

func (h *beijingHandler) Handle(ctx context.Context, r slog.Record) error {
	r.Time = clock.Now()
	return h.inner.Handle(ctx, r)
}

func (h *beijingHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &beijingHandler{inner: h.inner.WithAttrs(attrs)}
}

func (h *beijingHandler) WithGroup(name string) slog.Handler {
	return &beijingHandler{inner: h.inner.WithGroup(name)}
}
