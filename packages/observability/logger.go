// Package observability provides shared logging + tracing + metrics helpers
// for all Go services of the Civic Intelligence Platform.
package observability

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"
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

// NewLogger creates a logger writing to w at the given level.
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

// DefaultLogger is the package-level logger.
var DefaultLogger = NewLogger(os.Stdout, "info", "civic")

// WithContext returns a logger enriched with the correlation ID from ctx.
func WithContext(ctx context.Context) Logger {
	return DefaultLogger
}

// --- Metrics ---

// MetricRegistry is a simple in-memory metric registry that also exposes
// a Prometheus-compatible /metrics endpoint. In production, replace with
// the OpenTelemetry SDK (go.opentelemetry.io/otel).
type MetricRegistry struct {
	mu         sync.RWMutex
	counters   map[string]*Counter
	histograms map[string]*Histogram
}

// NewMetricRegistry creates a new metric registry.
func NewMetricRegistry() *MetricRegistry {
	return &MetricRegistry{
		counters:   make(map[string]*Counter),
		histograms: make(map[string]*Histogram),
	}
}

// Counter is a monotonically increasing counter.
type Counter struct {
	name  string
	help  string
	value int64
}

// Inc increments the counter by 1.
func (c *Counter) Inc() {
	c.value++
}

// Add adds n to the counter.
func (c *Counter) Add(n int64) {
	c.value += n
}

// Value returns the current counter value.
func (c *Counter) Value() int64 {
	return c.value
}

// Histogram tracks a distribution of values (e.g., request latency).
type Histogram struct {
	name   string
	help   string
	count  int64
	sum    float64
	values []float64
}

// Observe records a value in the histogram.
func (h *Histogram) Observe(v float64) {
	h.count++
	h.sum += v
	h.values = append(h.values, v)
}

// Count returns the number of observations.
func (h *Histogram) Count() int64 {
	return h.count
}

// Sum returns the sum of all observations.
func (h *Histogram) Sum() float64 {
	return h.sum
}

// Mean returns the average of all observations.
func (h *Histogram) Mean() float64 {
	if h.count == 0 {
		return 0
	}
	return h.sum / float64(h.count)
}

// RegisterCounter registers a counter (or returns the existing one).
func (r *MetricRegistry) RegisterCounter(name, help string) *Counter {
	r.mu.RLock()
	if c, ok := r.counters[name]; ok {
		r.mu.RUnlock()
		return c
	}
	r.mu.RUnlock()
	r.mu.Lock()
	defer r.mu.Unlock()
	if c, ok := r.counters[name]; ok {
		return c
	}
	c := &Counter{name: name, help: help}
	r.counters[name] = c
	return c
}

// RegisterHistogram registers a histogram (or returns the existing one).
func (r *MetricRegistry) RegisterHistogram(name, help string) *Histogram {
	r.mu.RLock()
	if h, ok := r.histograms[name]; ok {
		r.mu.RUnlock()
		return h
	}
	r.mu.RUnlock()
	r.mu.Lock()
	defer r.mu.Unlock()
	if h, ok := r.histograms[name]; ok {
		return h
	}
	h := &Histogram{name: name, help: help}
	r.histograms[name] = h
	return h
}

// PrometheusFormat returns all metrics in Prometheus text format.
func (r *MetricRegistry) PrometheusFormat() string {
	var sb strings.Builder
	r.mu.RLock()
	defer r.mu.RUnlock()

	for name, c := range r.counters {
		sb.WriteString("# HELP " + name + " " + c.help + "\n")
		sb.WriteString("# TYPE " + name + " counter\n")
		sb.WriteString(fmt.Sprintf("%s %d\n", name, c.value))
	}
	for name, h := range r.histograms {
		sb.WriteString("# HELP " + name + " " + h.help + "\n")
		sb.WriteString("# TYPE " + name + " histogram\n")
		sb.WriteString(fmt.Sprintf("%s_count %d\n", name, h.count))
		sb.WriteString(fmt.Sprintf("%s_sum %f\n", name, h.sum))
	}
	return sb.String()
}

// --- Standard metric names ---

const (
	MetricAPIRequests         = "civic_api_requests_total"
	MetricAPIRequestLatency   = "civic_api_request_latency_seconds"
	MetricBillsDiscovered     = "civic_bills_discovered_total"
	MetricAdapterErrors       = "civic_adapter_errors_total"
	MetricAIQuestions         = "civic_ai_questions_total"
	MetricAICitationFailures  = "civic_ai_citation_failures_total"
)

// Timing is a helper for recording latency.
type Timing struct {
	start time.Time
}

// StartTiming begins a latency measurement.
func StartTiming() *Timing {
	return &Timing{start: time.Now()}
}

// Elapsed returns the elapsed time since Start.
func (t *Timing) Elapsed() time.Duration {
	return time.Since(t.start)
}

// ElapsedSeconds returns the elapsed time in seconds (for histograms).
func (t *Timing) ElapsedSeconds() float64 {
	return time.Since(t.start).Seconds()
}
