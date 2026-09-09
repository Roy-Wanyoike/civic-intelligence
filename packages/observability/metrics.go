package observability

import (
	"context"
	"sync"
	"time"
)

// Metrics is the minimal interface domain code depends on for instrumenting
// business operations. Concrete implementations live in infrastructure and
// are typically backed by Prometheus. The interface is intentionally small
// so domain code stays decoupled from any particular metrics library.
type Metrics interface {
	// IncCounter increments a named counter by one, optionally with labels.
	IncCounter(name string, labels ...string)
	// ObserveHistogram records a duration observation on a named histogram.
	ObserveHistogram(name string, dur time.Duration, labels ...string)
	// SetGauge sets a named gauge to the given value.
	SetGauge(name string, value float64, labels ...string)
}

// NopMetrics is a no-op Metrics implementation.
type NopMetrics struct{}

// IncCounter implements Metrics.
func (NopMetrics) IncCounter(string, ...string) {}
// ObserveHistogram implements Metrics.
func (NopMetrics) ObserveHistogram(string, time.Duration, ...string) {}
// SetGauge implements Metrics.
func (NopMetrics) SetGauge(string, float64, ...string) {}

// NewNopMetrics returns a no-op metrics instance.
func NewNopMetrics() Metrics { return NopMetrics{} }

// InMemoryMetrics is a simple in-memory implementation useful for tests and
// for services that don't yet have a Prometheus endpoint. It is safe for
// concurrent use.
type InMemoryMetrics struct {
	mu        sync.Mutex
	counters  map[string]int64
	histograms map[string][]time.Duration
	gauges    map[string]float64
}

// NewInMemoryMetrics returns a fresh InMemoryMetrics.
func NewInMemoryMetrics() *InMemoryMetrics {
	return &InMemoryMetrics{
		counters:   make(map[string]int64),
		histograms: make(map[string][]time.Duration),
		gauges:     make(map[string]float64),
	}
}

// IncCounter implements Metrics.
func (m *InMemoryMetrics) IncCounter(name string, _ ...string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.counters[name]++
}

// ObserveHistogram implements Metrics.
func (m *InMemoryMetrics) ObserveHistogram(name string, dur time.Duration, _ ...string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.histograms[name] = append(m.histograms[name], dur)
}

// SetGauge implements Metrics.
func (m *InMemoryMetrics) SetGauge(name string, v float64, _ ...string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.gauges[name] = v
}

// Counter returns the current value of a counter (for tests).
func (m *InMemoryMetrics) Counter(name string) int64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.counters[name]
}

// Histogram returns the recorded observations for a histogram (for tests).
func (m *InMemoryMetrics) Histogram(name string) []time.Duration {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]time.Duration, len(m.histograms[name]))
	copy(out, m.histograms[name])
	return out
}

// Gauge returns the current gauge value.
func (m *InMemoryMetrics) Gauge(name string) float64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.gauges[name]
}

// Metric name constants used across services. Centralising these keeps the
// dashboards consistent.
const (
	MetricBillsCreated        = "civic_bills_created_total"
	MetricBillStageTransition = "civic_bill_stage_transitions_total"
	MetricSourceCrawled       = "civic_sources_crawled_total"
	MetricDocumentsParsed     = "civic_documents_parsed_total"
	MetricEvidenceAttached    = "civic_evidence_attached_total"
	MetricEvidenceMissing     = "civic_evidence_missing_total"
	MetricCandidateFactAccepted = "civic_candidate_facts_accepted_total"
	MetricCandidateFactRejected = "civic_candidate_facts_rejected_total"
	MetricNotificationsSent    = "civic_notifications_sent_total"
	MetricSearchLatency        = "civic_search_latency_seconds"
	MetricAIRequestLatency     = "civic_ai_request_latency_seconds"
)

// WithLabels is a tiny helper that lets callers pre-bind labels to a counter
// for cleaner call sites.
func WithLabels(name string, labels ...string) (string, []string) {
	return name, labels
}

// Use the context package so the import is not dropped; metrics calls in
// production pass context-derived values, but the interface stays simple.
var _ = context.Background
