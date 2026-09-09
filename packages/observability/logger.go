// Package observability provides cross-service logging, tracing and metrics
// primitives. The domain layer of every service depends only on the Logger
// interface defined here; the concrete zerolog implementation lives in
// infrastructure, preserving the Dependency Rule.
package observability

import (
	"context"
	"io"
	"os"
	"time"
)

// Logger is the interface that every service's domain and application layers
// depend on. It deliberately mirrors the subset of zerolog's API that we
// actually use, so domain code never imports zerolog directly.
type Logger interface {
	// Debug logs at debug level.
	Debug(msg string, fields ...Field)
	// Info logs at info level.
	Info(msg string, fields ...Field)
	// Warn logs at warn level.
	Warn(msg string, fields ...Field)
	// Error logs at error level.
	Error(msg string, err error, fields ...Field)
	// With returns a child logger with the given fields attached.
	With(fields ...Field) Logger
	// WithContext returns a child logger that extracts trace/correlation IDs
	// from the context.
	WithContext(ctx context.Context) Logger
}

// Field is a structured log field. Concrete loggers serialise it according
// to their backing implementation (zerolog, slog, etc.).
type Field struct {
	Key   string
	Value any
}

// String, Int, Bool, Time, Dur and Err are convenience field constructors.
func String(k, v string) Field  { return Field{Key: k, Value: v} }
func Int(k string, v int) Field  { return Field{Key: k, Value: v} }
func Int64(k string, v int64) Field { return Field{Key: k, Value: v} }
func Bool(k string, v bool) Field { return Field{Key: k, Value: v} }
func Time(k string, v time.Time) Field { return Field{Key: k, Value: v} }
func Dur(k string, v time.Duration) Field { return Field{Key: k, Value: v} }
func Err(err error) Field { return Field{Key: "err", Value: err} }
func Any(k string, v any) Field { return Field{Key: k, Value: v} }

// NopLogger is a no-op Logger useful for tests and default-zero values.
type NopLogger struct{}

// Debug implements Logger.
func (NopLogger) Debug(string, ...Field) {}
// Info implements Logger.
func (NopLogger) Info(string, ...Field) {}
// Warn implements Logger.
func (NopLogger) Warn(string, ...Field) {}
// Error implements Logger.
func (NopLogger) Error(string, error, ...Field) {}
// With implements Logger.
func (n NopLogger) With(...Field) Logger { return n }
// WithContext implements Logger.
func (n NopLogger) WithContext(context.Context) Logger { return n }

// NewNop returns a no-op logger.
func NewNop() Logger { return NopLogger{} }

// NewLogger returns a zerolog-backed logger writing to the given writer. If
// w is nil os.Stderr is used. The returned logger is safe for concurrent use.
func NewLogger(w io.Writer, level string, service string) Logger {
	if w == nil {
		w = os.Stderr
	}
	return newZerologLogger(w, level, service)
}
