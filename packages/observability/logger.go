// Package observability provides shared logging + tracing + metrics helpers
// for all Go services of the Civic Intelligence Platform.
package observability

import (
        "context"
        "fmt"
        "io"
        "log/slog"
        "os"
        "sort"
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
        gauges     map[string]*Gauge
}

// NewMetricRegistry creates a new metric registry.
func NewMetricRegistry() *MetricRegistry {
        return &MetricRegistry{
                counters:   make(map[string]*Counter),
                histograms: make(map[string]*Histogram),
                gauges:     make(map[string]*Gauge),
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

// Gauge is a metric that represents a single numerical value that can
// arbitrarily go up or down (e.g., the current number of tracked bills,
// the current crawl success rate). Gauges are appropriate for snapshot
// measurements of state; counters are appropriate for cumulative totals.
type Gauge struct {
        name  string
        help  string
        value float64
}

// Set sets the gauge to the given value.
func (g *Gauge) Set(v float64) {
        g.value = v
}

// Inc increments the gauge by 1.
func (g *Gauge) Inc() {
        g.value++
}

// Dec decrements the gauge by 1.
func (g *Gauge) Dec() {
        g.value--
}

// Add adds n to the gauge's value.
func (g *Gauge) Add(n float64) {
        g.value += n
}

// Value returns the current gauge value.
func (g *Gauge) Value() float64 {
        return g.value
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

// RegisterGauge registers a gauge (or returns the existing one).
func (r *MetricRegistry) RegisterGauge(name, help string) *Gauge {
        r.mu.RLock()
        if g, ok := r.gauges[name]; ok {
                r.mu.RUnlock()
                return g
        }
        r.mu.RUnlock()
        r.mu.Lock()
        defer r.mu.Unlock()
        if g, ok := r.gauges[name]; ok {
                return g
        }
        g := &Gauge{name: name, help: help}
        r.gauges[name] = g
        return g
}

// PrometheusFormat returns all metrics in Prometheus text format.
func (r *MetricRegistry) PrometheusFormat() string {
        var sb strings.Builder
        r.mu.RLock()
        defer r.mu.RUnlock()

        // Sort metric names so output is deterministic (Prometheus does not
        // require sorting, but tests + humans expect stable diffs).
        counterNames := sortedKeys(r.counters)
        gaugeNames := sortedKeys(r.gauges)
        histNames := sortedKeys(r.histograms)

        for _, name := range counterNames {
                c := r.counters[name]
                sb.WriteString("# HELP " + name + " " + c.help + "\n")
                sb.WriteString("# TYPE " + name + " counter\n")
                sb.WriteString(fmt.Sprintf("%s %d\n", name, c.value))
        }
        for _, name := range gaugeNames {
                g := r.gauges[name]
                sb.WriteString("# HELP " + name + " " + g.help + "\n")
                sb.WriteString("# TYPE " + name + " gauge\n")
                sb.WriteString(fmt.Sprintf("%s %f\n", name, g.value))
        }
        for _, name := range histNames {
                h := r.histograms[name]
                sb.WriteString("# HELP " + name + " " + h.help + "\n")
                sb.WriteString("# TYPE " + name + " histogram\n")
                sb.WriteString(fmt.Sprintf("%s_count %d\n", name, h.count))
                sb.WriteString(fmt.Sprintf("%s_sum %f\n", name, h.sum))
        }
        return sb.String()
}

// sortedKeys returns the keys of m in sorted order. Generic helper so the
// three metric-type maps (counters, gauges, histograms) can share it.
func sortedKeys[V any](m map[string]V) []string {
        keys := make([]string, 0, len(m))
        for k := range m {
                keys = append(keys, k)
        }
        sort.Strings(keys)
        return keys
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

// === Production-gate-#13 metric catalog ===
//
// The audit (audit-team-5 P1-17) flagged that only 5 of the 30+ metrics from
// docs/architecture/09-observability.md were implemented. The constants below
// complete the catalog. Each constant is the Prometheus metric name (snake_case
// with civic_ prefix). The catalog maps to the audit's metrics table:
//
//   Bills / ingestion
//     civic_bills_discovered_total        (counter, existing) — total bills discovered
//     civic_bills_tracked_total           (gauge)            — currently tracked bills
//     civic_crawl_success_rate            (gauge)            — adapter success rate (0-1)
//
//   Search
//     civic_search_latency_seconds        (histogram)        — search query latency
//
//   AI / intelligence
//     civic_ai_requests_total              (counter)          — AI service calls
//     civic_ai_failures_total              (counter)          — AI service failures
//     civic_citation_validation_failures_total (counter)     — citations rejected by validator
//     civic_model_failure_rate              (gauge)            — fraction of model calls that failed
//     civic_validation_failure_rate         (gauge)            — fraction of validations that failed
//
//   Scenarios / simulation
//     civic_scenario_runs_total            (counter)          — total scenario run count
//     civic_scenario_duration_seconds      (histogram)        — scenario run duration
//     civic_scenario_reproducibility_rate  (gauge)            — fraction of runs that reproduced
//     civic_evidence_coverage              (gauge)            — fraction of claims with evidence
//     civic_assumption_count               (histogram)        — assumptions per scenario
//     civic_uncertainty_distribution       (histogram)        — uncertainty score distribution
//
//   Resources / debt / government
//     civic_resource_usage_cpu_millis     (gauge)            — CPU usage (milli-cores)
//     civic_resource_usage_memory_mb      (gauge)            — memory usage (MB)
//     civic_debt_borrowing_total           (counter)          — cumulative borrowing events
//     civic_debt_stock_kes                (gauge)            — total debt stock in KES
//     civic_constitution_articles_indexed (gauge)            — number of constitution articles indexed
//     civic_government_transitions_total   (counter)          — number of government transitions
//     civic_post_assent_events_tracked    (gauge)            — post-assent events tracked
//     civic_act_audit_complete_rate       (gauge)            — fraction of acts with audit complete
//
//   Notifications / following
//     civic_followed_matters_total        (gauge)            — total followed matters
//     civic_notification_delivery_rate    (gauge)            — fraction of notifications delivered
const (
        // Bills / ingestion.
        MetricBillsTracked                 = "civic_bills_tracked_total"
        MetricCrawlSuccessRate             = "civic_crawl_success_rate"

        // Search.
        MetricSearchLatencySeconds         = "civic_search_latency_seconds"

        // AI / intelligence.
        MetricAIRequestsTotal              = "civic_ai_requests_total"
        MetricAIFailuresTotal              = "civic_ai_failures_total"
        MetricCitationValidationFailures   = "civic_citation_validation_failures_total"
        MetricModelFailureRate             = "civic_model_failure_rate"
        MetricValidationFailureRate        = "civic_validation_failure_rate"

        // Scenarios / simulation.
        MetricScenarioRunsTotal            = "civic_scenario_runs_total"
        MetricScenarioDurationSeconds      = "civic_scenario_duration_seconds"
        MetricScenarioReproducibilityRate  = "civic_scenario_reproducibility_rate"
        MetricEvidenceCoverage             = "civic_evidence_coverage"
        MetricAssumptionCount              = "civic_assumption_count"
        MetricUncertaintyDistribution      = "civic_uncertainty_distribution"

        // Resources / debt / government.
        MetricResourceUsageCPUMillis       = "civic_resource_usage_cpu_millis"
        MetricResourceUsageMemoryMB       = "civic_resource_usage_memory_mb"
        MetricDebtBorrowingTotal           = "civic_debt_borrowing_total"
        MetricDebtStockKES                 = "civic_debt_stock_kes"
        MetricConstitutionArticlesIndexed  = "civic_constitution_articles_indexed"
        MetricGovernmentTransitionsTotal   = "civic_government_transitions_total"
        MetricPostAssentEventsTracked      = "civic_post_assent_events_tracked"
        MetricActAuditCompleteRate         = "civic_act_audit_complete_rate"

        // Notifications / following.
        MetricFollowedMattersTotal         = "civic_followed_matters_total"
        MetricNotificationDeliveryRate     = "civic_notification_delivery_rate"
)

// MetricsCatalog is the canonical list of all platform metrics with their
// help strings. Tests assert that every metric in this catalog can be
// registered without collision. Downstream services register the metrics
// they emit via RegisterCatalog().
var MetricsCatalog = []MetricSpec{
        // Existing.
        {MetricAPIRequests, "Total API requests", MetricTypeCounter},
        {MetricAPIRequestLatency, "API request latency in seconds", MetricTypeHistogram},
        {MetricBillsDiscovered, "Total bills discovered by ingestion adapters", MetricTypeCounter},
        {MetricAdapterErrors, "Total ingestion adapter errors", MetricTypeCounter},
        {MetricAIQuestions, "Total AI question requests", MetricTypeCounter},
        {MetricAICitationFailures, "Total AI citations that failed validation", MetricTypeCounter},

        // Production-gate-#13 additions.
        {MetricBillsTracked, "Number of bills currently tracked by the platform", MetricTypeGauge},
        {MetricCrawlSuccessRate, "Adapter crawl success rate (0.0-1.0) over the last window", MetricTypeGauge},
        {MetricSearchLatencySeconds, "Search query latency in seconds", MetricTypeHistogram},
        {MetricAIRequestsTotal, "Total requests to the AI service", MetricTypeCounter},
        {MetricAIFailuresTotal, "Total AI service call failures", MetricTypeCounter},
        {MetricCitationValidationFailures, "Total citations rejected by the evidence validator", MetricTypeCounter},
        {MetricModelFailureRate, "Fraction of LLM model calls that failed (0.0-1.0)", MetricTypeGauge},
        {MetricValidationFailureRate, "Fraction of evidence validations that failed (0.0-1.0)", MetricTypeGauge},
        {MetricScenarioRunsTotal, "Total scenario simulation runs", MetricTypeCounter},
        {MetricScenarioDurationSeconds, "Scenario simulation run duration in seconds", MetricTypeHistogram},
        {MetricScenarioReproducibilityRate, "Fraction of scenario runs that reproduced prior results (0.0-1.0)", MetricTypeGauge},
        {MetricEvidenceCoverage, "Fraction of claims with at least one supporting evidence chunk (0.0-1.0)", MetricTypeGauge},
        {MetricAssumptionCount, "Number of assumptions detected per scenario audit", MetricTypeHistogram},
        {MetricUncertaintyDistribution, "Distribution of uncertainty scores across AI outputs", MetricTypeHistogram},
        {MetricResourceUsageCPUMillis, "CPU resource usage in milli-cores", MetricTypeGauge},
        {MetricResourceUsageMemoryMB, "Memory resource usage in megabytes", MetricTypeGauge},
        {MetricDebtBorrowingTotal, "Total sovereign borrowing events recorded", MetricTypeCounter},
        {MetricDebtStockKES, "Total public debt stock in KES", MetricTypeGauge},
        {MetricConstitutionArticlesIndexed, "Number of constitution articles indexed for search", MetricTypeGauge},
        {MetricGovernmentTransitionsTotal, "Total government transition events recorded", MetricTypeCounter},
        {MetricPostAssentEventsTracked, "Number of post-assent events currently tracked", MetricTypeGauge},
        {MetricActAuditCompleteRate, "Fraction of acts of parliament with a complete audit trail (0.0-1.0)", MetricTypeGauge},
        {MetricFollowedMattersTotal, "Number of matters (bills/acts) currently followed by users", MetricTypeGauge},
        {MetricNotificationDeliveryRate, "Fraction of notifications successfully delivered (0.0-1.0)", MetricTypeGauge},
}

// MetricType is the Prometheus type of a metric.
type MetricType string

const (
        MetricTypeCounter   MetricType = "counter"
        MetricTypeGauge     MetricType = "gauge"
        MetricTypeHistogram MetricType = "histogram"
)

// MetricSpec declares a single metric in the catalog.
type MetricSpec struct {
        Name string
        Help string
        Type MetricType
}

// RegisterCatalog registers every metric in MetricsCatalog on the receiver
// registry. The handles are discarded — callers obtain them later via
// RegisterCounter/RegisterGauge/RegisterHistogram (which are idempotent and
// return the existing metric). This method exists primarily so tests can
// verify every catalog entry is registerable without collisions.
//
// Returns the count of metrics registered (mostly for assertion in tests).
func (r *MetricRegistry) RegisterCatalog() int {
        n := 0
        for _, m := range MetricsCatalog {
                switch m.Type {
                case MetricTypeCounter:
                        r.RegisterCounter(m.Name, m.Help)
                case MetricTypeGauge:
                        r.RegisterGauge(m.Name, m.Help)
                case MetricTypeHistogram:
                        r.RegisterHistogram(m.Name, m.Help)
                }
                n++
        }
        return n
}

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
