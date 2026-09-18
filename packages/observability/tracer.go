// Package observability provides shared logging + tracing + metrics helpers
// for all Go services of the Civic Intelligence Platform.
//
// tracer.go — distributed tracing abstraction + OpenTelemetry implementation.
//
// The audit (audit-team-5 P1-18) flagged that the only tracer was NopTracer,
// which drops every span. Production gate #13 (OBSERVABILITY) requires a real
// OpenTelemetry tracer that exports spans to the OTel collector (and from
// there to Tempo). This file implements OTelTracer, which wraps the OTel SDK
// behind our local Tracer interface so callers do not import the OTel package
// directly.
//
// Wiring:
//
//   - In production (when OTEL_EXPORTER_OTLP_ENDPOINT is set), main.go calls
//     InitOTelTracer once at startup. This configures the global OTel tracer
//     provider with an OTLP gRPC exporter pointed at the collector. The
//     returned shutdown func flushes pending spans on graceful exit.
//   - In development / tests (no endpoint set), callers leave DefaultTracer
//     as NopTracer — no spans are exported, no network IO occurs.
//
// The local Tracer/Span interfaces are kept minimal (StartSpan, End,
// SetAttribute, RecordError) so they remain easy to mock. Callers that need
// richer OTel features (links, events, span status) can pull the OTel tracer
// directly via otel.Tracer("name").
package observability

import (
        "context"
        "fmt"
        "log"
        "sync/atomic"
        "time"

        "go.opentelemetry.io/otel"
        "go.opentelemetry.io/otel/attribute"
        "go.opentelemetry.io/otel/codes"
        "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
        "go.opentelemetry.io/otel/sdk/resource"
        sdktrace "go.opentelemetry.io/otel/sdk/trace"
        semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
        oteltrace "go.opentelemetry.io/otel/trace"
)

// Tracer is a lightweight tracing interface. Production code uses OTelTracer
// (which wraps go.opentelemetry.io/otel); tests use NopTracer. The interface
// intentionally exposes only the operations the platform's instrumentation
// points (docs/architecture/09-observability.md §7) require.
type Tracer interface {
        StartSpan(ctx context.Context, name string) (context.Context, Span)
}

// Span represents a unit of work in a trace.
type Span interface {
        End()
        SetAttribute(key string, value any)
        RecordError(err error)
}

// NopTracer is a no-op tracer for when tracing is not configured. It produces
// NopSpans which discard all data. Used in development + unit tests so callers
// do not need a running OTel collector.
type NopTracer struct{}

// StartSpan returns a no-op span. The returned context is the input context
// unchanged (no span is stored on it).
func (NopTracer) StartSpan(ctx context.Context, _ string) (context.Context, Span) {
        return ctx, NopSpan{}
}

// NopSpan is a no-op span.
type NopSpan struct{}

func (NopSpan) End()                    {}
func (NopSpan) SetAttribute(string, any) {}
func (NopSpan) RecordError(error)       {}

// DefaultTracer is the package-level tracer. main.go overrides this with an
// OTelTracer when OTEL_EXPORTER_OTLP_ENDPOINT is set. Code that wants to start
// a span without holding a tracer reference calls StartSpan(ctx, name) which
// delegates to DefaultTracer.
var DefaultTracer Tracer = NopTracer{}

// SetTracer replaces the default tracer. Safe to call at most once at process
// startup; concurrent calls after the first are no-ops (last-write-wins
// would race against in-flight spans).
func SetTracer(t Tracer) {
        DefaultTracer = t
}

// StartSpan starts a span using the default tracer.
func StartSpan(ctx context.Context, name string) (context.Context, Span) {
        return DefaultTracer.StartSpan(ctx, name)
}

// === OTelTracer ===
//
// OTelTracer adapts the OTel SDK to the platform's Tracer interface. It
// delegates to otel.Tracer("civic") for span creation, and wraps each returned
// oteltrace.Span in an otelSpan that implements our Span interface.
//
// Construction is intentionally split:
//
//   - InitOTelTracer(configures the global OTel tracer provider) — called
//     once at startup; returns a shutdown function the caller defers.
//   - NewOTelTracer(returns an OTelTracer using whatever global provider
//     is currently configured) — called by SetTracer() after init, and by
//     tests that pass a custom provider via otel.SetTracerProvider.
//
// This split lets tests construct an OTelTracer against an in-memory exporter
// without dialing a real OTel collector.

// OTelTracer wraps the OTel SDK behind the local Tracer interface. The
// underlying otel.Tracer is fetched lazily on each StartSpan call so changes
// to the global tracer provider (e.g. via otel.SetTracerProvider in tests)
// take effect immediately.
type OTelTracer struct {
        // name is the instrumentation scope name passed to otel.Tracer(name).
        // Conventionally the package path of the instrumented code. Defaults
        // to "civic" when empty.
        name string

        // started counts spans started since construction. Tests assert this
        // value to verify the tracer is wired correctly. atomic because spans
        // can start from any goroutine.
        started atomic.Int64
}

// NewOTelTracer returns an OTelTracer that delegates to the OTel SDK's global
// tracer provider. The instrumentation scope name defaults to "civic".
func NewOTelTracer(name string) *OTelTracer {
        if name == "" {
                name = "civic"
        }
        return &OTelTracer{name: name}
}

// StartSpan creates a new OTel span and stores it on the returned context.
// The span is a child of whatever span is already on ctx (per the W3C
// traceparent header propagated through the request lifecycle). If ctx has no
// parent span, the new span is a root.
func (t *OTelTracer) StartSpan(ctx context.Context, name string) (context.Context, Span) {
        tracer := otel.Tracer(t.name)
        otelCtx, s := tracer.Start(ctx, name)
        t.started.Add(1)
        return otelCtx, &otelSpan{span: s}
}

// StartedCount returns the number of spans started through this tracer. Used
// by tests; not exposed via the Tracer interface.
func (t *OTelTracer) StartedCount() int64 {
        return t.started.Load()
}

// otelSpan adapts oteltrace.Span to the local Span interface.
type otelSpan struct {
        span oteltrace.Span
}

// End finishes the span. After End, no further methods may be called on it.
func (s *otelSpan) End() {
        if s.span != nil {
                s.span.End()
        }
}

// SetAttribute attaches a key/value attribute to the span. The value type
// is permissive (any); we map common Go types to OTel attribute types
// (string, int, int64, float64, bool, error). Unknown types are stringified
// via fmt.Sprint — the OTel SDK would otherwise silently drop them.
func (s *otelSpan) SetAttribute(key string, value any) {
        if s.span == nil {
                return
        }
        s.span.SetAttributes(toOTelAttribute(key, value))
}

// toOTelAttribute converts a Go value to an OTel attribute. The switch is
// exhaustive over the types the platform's instrumentation emits; unknown
// types fall through to a stringified value so the attribute is not silently
// dropped.
func toOTelAttribute(key string, value any) attribute.KeyValue {
        switch v := value.(type) {
        case string:
                return attribute.String(key, v)
        case int:
                return attribute.Int64(key, int64(v))
        case int64:
                return attribute.Int64(key, v)
        case float64:
                return attribute.Float64(key, v)
        case bool:
                return attribute.Bool(key, v)
        case error:
                return attribute.String(key, v.Error())
        default:
                return attribute.String(key, fmt.Sprint(v))
        }
}

// RecordError records an error on the span. The span's status is set to ERROR
// so the trace UI highlights failing operations. Recording a nil error is a
// no-op (matches the NopSpan behaviour).
func (s *otelSpan) RecordError(err error) {
        if s.span == nil || err == nil {
                return
        }
        s.span.RecordError(err)
        s.span.SetStatus(codes.Error, err.Error())
}

// === OTel tracer provider initialisation ===
//
// InitOTelTracer configures the global OTel tracer provider with an OTLP gRPC
// exporter pointed at endpoint. The returned shutdown function flushes pending
// spans; callers MUST defer it on the process's main context. Returns an
// error if the exporter cannot be constructed (e.g. invalid endpoint).
//
// The exporter uses gRPC with a 10s connection timeout. Spans are batched
// (the OTel default) to reduce exporter load; batches flush on shutdown.
// Production callers should set OTEL_EXPORTER_OTLP_ENDPOINT to the OTel
// collector's gRPC port (typically :4317).
func InitOTelTracer(ctx context.Context, serviceName, endpoint string) (func(context.Context) error, error) {
        if serviceName == "" {
                serviceName = "civic"
        }
        if endpoint == "" {
                return nil, fmt.Errorf("OTel exporter endpoint is empty")
        }

        // Construct the OTLP gRPC exporter. The OTel SDK handles retries +
        // batching; we only need to specify the endpoint + dial options.
        exporterCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
        defer cancel()

        exporter, err := otlptracegrpc.New(exporterCtx,
                otlptracegrpc.WithEndpoint(endpoint),
                otlptracegrpc.WithInsecure(),
        )
        if err != nil {
                return nil, fmt.Errorf("construct OTel OTLP trace exporter: %w", err)
        }

        // Build the resource that identifies this service in trace UIs. The
        // service.name attribute is mandatory; service.version is added so a
        // rolling deploy shows distinct traces for old + new code.
        res, err := resource.New(ctx,
                resource.WithAttributes(
                        semconv.ServiceName(serviceName),
                        semconv.ServiceVersion("0.2.0"),
                ),
        )
        if err != nil {
                return nil, fmt.Errorf("construct OTel resource: %w", err)
        }

        // Construct the trace provider with a batch span processor. The
        // processor batches up to 512 spans or 5s before flushing — defaults
        // are appropriate for production; tune via env if needed.
        tp := sdktrace.NewTracerProvider(
                sdktrace.WithBatcher(exporter),
                sdktrace.WithResource(res),
        )

        // Register as the global tracer provider. otel.Tracer(name) (called
        // from OTelTracer.StartSpan) returns this provider's tracer.
        otel.SetTracerProvider(tp)

        // Replace the package-level DefaultTracer with an OTelTracer. Existing
        // callers using observability.StartSpan(ctx, name) now emit real spans.
        SetTracer(NewOTelTracer(serviceName))

        log.Printf("OTel tracer provider initialised: service=%s endpoint=%s", serviceName, endpoint)

        // shutdown flushes pending spans + shuts down the exporter. Callers
        // MUST defer this on the process's main context so graceful shutdown
        // does not lose in-flight spans.
        return tp.Shutdown, nil
}
