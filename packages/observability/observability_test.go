package observability

import (
        "net/http"
        "testing"
        "net/http/httptest"
        "strings"
)

func TestMetricRegistry_Counter(t *testing.T) {
        r := NewMetricRegistry()
        c := r.RegisterCounter("test_counter", "test help")
        c.Inc()
        c.Inc()
        c.Add(5)
        if c.Value() != 7 {
                t.Errorf("expected 7, got %d", c.Value())
        }
}

func TestMetricRegistry_Histogram(t *testing.T) {
        r := NewMetricRegistry()
        h := r.RegisterHistogram("test_hist", "test help")
        h.Observe(0.1)
        h.Observe(0.2)
        h.Observe(0.3)
        if h.Count() != 3 {
                t.Errorf("expected count 3, got %d", h.Count())
        }
        if h.Sum() < 0.59 || h.Sum() > 0.61 {
                t.Errorf("expected sum ~0.6, got %f", h.Sum())
        }
}

func TestMetricRegistry_PrometheusFormat(t *testing.T) {
        r := NewMetricRegistry()
        c := r.RegisterCounter("requests", "total requests")
        c.Inc()
        h := r.RegisterHistogram("latency", "request latency")
        h.Observe(0.5)

        output := r.PrometheusFormat()
        if !strings.Contains(output, "requests") {
                t.Error("expected 'requests' in output")
        }
        if !strings.Contains(output, "latency") {
                t.Error("expected 'latency' in output")
        }
        if !strings.Contains(output, "counter") {
                t.Error("expected 'counter' type in output")
        }
        if !strings.Contains(output, "histogram") {
                t.Error("expected 'histogram' type in output")
        }
}

func TestMetricsMiddleware(t *testing.T) {
        r := NewMetricRegistry()
        handler := MetricsMiddleware(r, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                w.WriteHeader(http.StatusOK)
        }))

        req := httptest.NewRequest("GET", "/test", nil)
        rr := httptest.NewRecorder()
        handler.ServeHTTP(rr, req)

        c := r.counters[MetricAPIRequests]
        if c.Value() != 1 {
                t.Errorf("expected 1 request, got %d", c.Value())
        }
}

func TestMetricsHandler(t *testing.T) {
        r := NewMetricRegistry()
        r.RegisterCounter("test", "test help").Inc()

        handler := MetricsHandler(r)
        req := httptest.NewRequest("GET", "/metrics", nil)
        rr := httptest.NewRecorder()
        handler.ServeHTTP(rr, req)

        if rr.Code != http.StatusOK {
                t.Errorf("expected 200, got %d", rr.Code)
        }
        if !strings.Contains(rr.Body.String(), "test") {
                t.Error("expected 'test' in metrics output")
        }
}

func TestTiming(t *testing.T) {
        tm := StartTiming()
        if tm.Elapsed() < 0 {
                t.Error("elapsed should be non-negative")
        }
}

func TestNopTracer(t *testing.T) {
        tracer := NopTracer{}
        ctx, span := tracer.StartSpan(nil, "test")
        if span == nil {
                t.Error("expected non-nil span")
        }
        span.End()
        span.SetAttribute("key", "value")
        span.RecordError(nil)
        _ = ctx
}
