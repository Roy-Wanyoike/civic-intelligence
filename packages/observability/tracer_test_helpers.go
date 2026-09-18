package observability

import (
        "context"
        "sync"

        "go.opentelemetry.io/otel/sdk/resource"
        sdktrace "go.opentelemetry.io/otel/sdk/trace"
        semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// inMemorySpanExporter is a minimal in-memory span exporter for tests. It
// implements sdktrace.SpanExporter and records every SpanSnapshot it receives
// in a slice guarded by a mutex. The OTel SDK ships tracetest.InMemoryExporter
// which is more featureful; this lightweight version keeps the test surface
// small + avoids coupling tests to tracetest's API surface.
type inMemorySpanExporter struct {
        mu     sync.Mutex
        spans  []sdktrace.ReadOnlySpan
        stopped bool
}

// newInMemorySpanExporter returns a fresh inMemorySpanExporter.
func newInMemorySpanExporter() *inMemorySpanExporter {
        return &inMemorySpanExporter{}
}

// ExportSpans stores the batch in memory. Implements sdktrace.SpanExporter.
func (e *inMemorySpanExporter) ExportSpans(_ context.Context, spans []sdktrace.ReadOnlySpan) error {
        e.mu.Lock()
        defer e.mu.Unlock()
        if e.stopped {
                return nil
        }
        e.spans = append(e.spans, spans...)
        return nil
}

// Shutdown marks the exporter as stopped + drops any stored spans. Implements
// sdktrace.SpanExporter.
func (e *inMemorySpanExporter) Shutdown(_ context.Context) error {
        e.mu.Lock()
        defer e.mu.Unlock()
        e.stopped = true
        e.spans = nil
        return nil
}

// spanCount returns the number of spans currently stored. Tests use this to
// assert the exporter received the expected number of spans.
func (e *inMemorySpanExporter) spanCount() int {
        e.mu.Lock()
        defer e.mu.Unlock()
        return len(e.spans)
}

// newTracerProviderWithExporter returns a TracerProvider that uses exporter
// for span export and tags every span with service.name=serviceName. The
// provider uses a SimpleSpanProcessor (synchronous) so spans are flushed
// immediately on End — no ForceFlush needed in tests.
func newTracerProviderWithExporter(exporter sdktrace.SpanExporter, serviceName string) *sdktrace.TracerProvider {
        res, err := resource.Merge(
                resource.Default(),
                resource.NewWithAttributes(semconv.SchemaURL,
                        semconv.ServiceName(serviceName),
                ),
        )
        if err != nil {
                // resource.Merge only fails on schema-url mismatch — fall back to
                // a minimal resource so the test does not crash.
                res = resource.NewWithAttributes(semconv.SchemaURL,
                        semconv.ServiceName(serviceName),
                )
        }
        return sdktrace.NewTracerProvider(
                sdktrace.WithSyncer(exporter),
                sdktrace.WithResource(res),
        )
}

// ForceFlush is a method on TracerProvider that the OTel SDK exposes; tests
// call it to ensure pending spans are exported before assertions. We use
// WithSyncer above so flushes are immediate, but ForceFlush is still called
// defensively in TestOTelTracer_StartSpanWithInMemoryExporter.
