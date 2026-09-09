package observability

import "context"

// Tracer is a minimal tracing interface that domain code can depend on
// without pulling in OpenTelemetry. The concrete implementation (which may
// be a no-op when tracing is disabled) lives in infrastructure. Real OTel
// spans are created in HTTP / messaging middleware.
type Tracer interface {
	// StartSpan begins a new span named after op. The returned context
	// carries the span; the returned function must be called when the
	// operation completes (defer it).
	StartSpan(ctx context.Context, op string, attrs ...Field) (context.Context, func(error))
}

// NopTracer is a no-op Tracer.
type NopTracer struct{}

// StartSpan implements Tracer.
func (NopTracer) StartSpan(ctx context.Context, _ string, _ ...Field) (context.Context, func(error)) {
	return ctx, func(error) {}
}

// NewNopTracer returns a no-op tracer.
func NewNopTracer() Tracer { return NopTracer{} }
