package observability

import (
	"context"
)

// Tracer is a lightweight tracing interface. In production, replace with
// go.opentelemetry.io/otel/trace.Tracer. For now, this provides context
// propagation without external dependencies.
type Tracer interface {
	StartSpan(ctx context.Context, name string) (context.Context, Span)
}

// Span represents a unit of work in a trace.
type Span interface {
	End()
	SetAttribute(key string, value any)
	RecordError(err error)
}

// NopTracer is a no-op tracer for when tracing is not configured.
type NopTracer struct{}

// StartSpan returns a no-op span.
func (NopTracer) StartSpan(ctx context.Context, _ string) (context.Context, Span) {
	return ctx, NopSpan{}
}

// NopSpan is a no-op span.
type NopSpan struct{}

func (NopSpan) End()                  {}
func (NopSpan) SetAttribute(string, any) {}
func (NopSpan) RecordError(error)     {}

// DefaultTracer is the package-level tracer.
var DefaultTracer Tracer = NopTracer{}

// SetTracer replaces the default tracer.
func SetTracer(t Tracer) {
	DefaultTracer = t
}

// StartSpan starts a span using the default tracer.
func StartSpan(ctx context.Context, name string) (context.Context, Span) {
	return DefaultTracer.StartSpan(ctx, name)
}
