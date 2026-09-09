// Package observability provides shared logging + tracing + metrics helpers
// for all Go services of the Civic Intelligence Platform.
package observability

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strings"
)

// Field is a structured-logging key/value pair.
type Field struct {
	Key   string
	Value any
}

// String creates a string Field.
func String(key, val string) Field {
	return Field{Key: key, Value: val}
}

// Int creates an int Field.
func Int(key string, val int) Field {
	return Field{Key: key, Value: val}
}

// Logger is the structured logger interface used by all services.
type Logger interface {
	Debug(msg string, fields ...Field)
	Info(msg string, fields ...Field)
	Warn(msg string, fields ...Field)
	Error(msg string, err error, fields ...Field)
}

// NewLogger creates a logger writing to w at the given level. The service
// name is included in every log line for filtering in multi-service logs.
func NewLogger(w io.Writer, level, serviceName string) Logger {
	var lvl slog.Level
	switch strings.ToLower(level) {
	case "debug":
		lvl = slog.LevelDebug
	case "info", "":
		lvl = slog.LevelInfo
	case "warn", "warning":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}
	handler := slog.NewJSONHandler(w, &slog.HandlerOptions{Level: lvl})
	return &slogLogger{logger: slog.New(handler).With("service", serviceName)}
}

type slogLogger struct {
	logger *slog.Logger
}

func (l *slogLogger) Debug(msg string, fields ...Field) {
	l.logger.Debug(msg, toArgs(fields)...)
}

func (l *slogLogger) Info(msg string, fields ...Field) {
	l.logger.Info(msg, toArgs(fields)...)
}

func (l *slogLogger) Warn(msg string, fields ...Field) {
	l.logger.Warn(msg, toArgs(fields)...)
}

func (l *slogLogger) Error(msg string, err error, fields ...Field) {
	all := append([]Field{String("error", err.Error())}, fields...)
	l.logger.Error(msg, toArgs(all)...)
}

func toArgs(fields []Field) []any {
	args := make([]any, 0, len(fields)*2)
	for _, f := range fields {
		args = append(args, f.Key, f.Value)
	}
	return args
}

// DefaultLogger is the package-level logger used when no service-specific
// logger is configured. Writes JSON to stdout at INFO level.
var DefaultLogger = NewLogger(os.Stdout, "info", "civic")

// WithContext returns a logger enriched with the correlation ID from ctx.
func WithContext(ctx context.Context) Logger {
	// TODO: extract correlation ID from context and add as a field
	return DefaultLogger
}
