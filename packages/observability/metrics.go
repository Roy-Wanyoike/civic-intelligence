package observability

import (
	"net/http"
	"time"
)

// MetricsMiddleware wraps an http.Handler and records request count + latency.
// This is the OpenTelemetry-compatible middleware — in production, replace
// the in-memory MetricRegistry with the OTel SDK.
func MetricsMiddleware(registry *MetricRegistry, next http.Handler) http.Handler {
	requests := registry.RegisterCounter(MetricAPIRequests, "Total API requests")
	latency := registry.RegisterHistogram(MetricAPIRequestLatency, "API request latency in seconds")

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap the ResponseWriter to capture the status code.
		ww := &statusWriter{ResponseWriter: w, status: 200}

		next.ServeHTTP(ww, r)

		requests.Inc()
		latency.Observe(time.Since(start).Seconds())

		// Log the request.
		DefaultLogger.Info("request",
			String("method", r.Method),
			String("path", r.URL.Path),
			Int("status", ww.status),
			Int("duration_ms", int(time.Since(start).Milliseconds())),
		)
	})
}

// MetricsHandler returns an http.HandlerFunc that serves Prometheus-format metrics.
func MetricsHandler(registry *MetricRegistry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		w.Write([]byte(registry.PrometheusFormat()))
	}
}

// statusWriter wraps http.ResponseWriter to capture the status code.
type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}
