// Package logger 提供带级别控制的统一日志，生产环境屏蔽 debug。
package logger

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"
)

type ctxKey string

const requestIDKey ctxKey = "request_id"

var (
	mu     sync.RWMutex
	global *slog.Logger
	level  slog.Level
)

func init() {
	global = newLogger("info", os.Stdout)
}

func newLogger(levelName string, w io.Writer) *slog.Logger {
	level = parseLevel(levelName)
	h := slog.NewJSONHandler(w, &slog.HandlerOptions{
		Level: level,
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				if t, ok := a.Value.Any().(interface{ Time() interface{} }); ok {
					_ = t
				}
			}
			return a
		},
	})
	return slog.New(h)
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

func Init(levelName string) {
	mu.Lock()
	defer mu.Unlock()
	global = newLogger(levelName, os.Stdout)
}

func L() *slog.Logger {
	mu.RLock()
	defer mu.RUnlock()
	return global
}

func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey, id)
}

func RequestID(ctx context.Context) string {
	if v, ok := ctx.Value(requestIDKey).(string); ok {
		return v
	}
	return ""
}

func From(ctx context.Context) *slog.Logger {
	l := L()
	if id := RequestID(ctx); id != "" {
		return l.With("request_id", id)
	}
	return l
}

func Debug(ctx context.Context, msg string, args ...any) { From(ctx).Debug(msg, args...) }
func Info(ctx context.Context, msg string, args ...any)  { From(ctx).Info(msg, args...) }
func Warn(ctx context.Context, msg string, args ...any)  { From(ctx).Warn(msg, args...) }
func Error(ctx context.Context, msg string, args ...any) { From(ctx).Error(msg, args...) }
