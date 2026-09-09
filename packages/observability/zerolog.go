package observability

import (
	"context"
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
)

// zerologLogger is the concrete implementation of Logger backed by zerolog.
// It is defined here, in the shared observability package, rather than in any
// service's infrastructure layer because zerolog is shared infrastructure.
// Service domain layers never reference zerolog directly — they consume the
// Logger interface only.
type zerologLogger struct {
	zl     zerolog.Logger
	fields []Field
}

func newZerologLogger(w io.Writer, level string, service string) Logger {
	zerolog.TimeFieldFormat = time.RFC3339Nano
	lvl, err := zerolog.ParseLevel(level)
	if err != nil || lvl == zerolog.NoLevel {
		lvl = zerolog.InfoLevel
	}
	zl := zerolog.New(w).Level(lvl).With().Timestamp().Str("svc", service).Logger()
	return &zerologLogger{zl: zl}
}

// fieldToZerolog converts a Field into a zerolog field-appending function.
func fieldToZerolog(f Field) func(*zerolog.Event) {
	return func(e *zerolog.Event) {
		switch v := f.Value.(type) {
		case string:
			e.Str(f.Key, v)
		case int:
			e.Int(f.Key, v)
		case int64:
			e.Int64(f.Key, v)
		case bool:
			e.Bool(f.Key, v)
		case time.Time:
			e.Time(f.Key, v)
		case time.Duration:
			e.Dur(f.Key, v)
		case error:
			if v != nil {
				e.AnErr(f.Key, v)
			}
		default:
			e.Interface(f.Key, v)
		}
	}
}

// applyFields appends all fields to a zerolog event.
func applyFields(e *zerolog.Event, fields []Field) {
	for _, f := range fields {
		fieldToZerolog(f)(e)
	}
}

// Debug implements Logger.
func (l *zerologLogger) Debug(msg string, fields ...Field) {
	ev := l.zl.Debug()
	applyFields(ev, l.fields)
	applyFields(ev, fields)
	ev.Msg(msg)
}

// Info implements Logger.
func (l *zerologLogger) Info(msg string, fields ...Field) {
	ev := l.zl.Info()
	applyFields(ev, l.fields)
	applyFields(ev, fields)
	ev.Msg(msg)
}

// Warn implements Logger.
func (l *zerologLogger) Warn(msg string, fields ...Field) {
	ev := l.zl.Warn()
	applyFields(ev, l.fields)
	applyFields(ev, fields)
	ev.Msg(msg)
}

// Error implements Logger.
func (l *zerologLogger) Error(msg string, err error, fields ...Field) {
	ev := l.zl.Error()
	if err != nil {
		ev.Err(err)
	}
	applyFields(ev, l.fields)
	applyFields(ev, fields)
	ev.Msg(msg)
}

// With implements Logger.
func (l *zerologLogger) With(fields ...Field) Logger {
	combined := make([]Field, 0, len(l.fields)+len(fields))
	combined = append(combined, l.fields...)
	combined = append(combined, fields...)
	return &zerologLogger{zl: l.zl, fields: combined}
}

// WithContext implements Logger. It extracts the correlation ID and trace ID
// (if present) from the context and attaches them as fields.
func (l *zerologLogger) WithContext(ctx context.Context) Logger {
	out := l
	if corr, ok := ctx.Value(CorrelationIDKey).(string); ok && corr != "" {
		out = &zerologLogger{zl: l.zl, fields: append(out.fields, String("corr_id", corr))}
	}
	if trace, ok := ctx.Value(TraceIDKey).(string); ok && trace != "" {
		out = &zerologLogger{zl: out.zl, fields: append(out.fields, String("trace_id", trace))}
	}
	return out
}

// ctxKey is the type for context keys in this package.
type ctxKey int

const (
	CorrelationIDKey ctxKey = iota
	TraceIDKey
)

// WithCorrelationID returns a new context carrying the given correlation ID.
func WithCorrelationID(ctx context.Context, id string) context.Context {
	if id == "" {
		return ctx
	}
	return context.WithValue(ctx, CorrelationIDKey, id)
}

// CorrelationIDFromContext returns the correlation ID stored in ctx, or "".
func CorrelationIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(CorrelationIDKey).(string); ok {
		return v
	}
	return ""
}

// ensure os.Stderr is referenced so this package always compiles even when
// callers don't import os.
var _ = os.Stderr
