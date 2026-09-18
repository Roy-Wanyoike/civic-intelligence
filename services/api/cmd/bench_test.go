package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// BenchmarkHandleActsList measures the cost of a single GET /api/v1/acts
// request against the in-memory actRepo (seeded with the Kenyan dataset).
// The handler is the hot path for the Acts Browse page; the SLO
// documented in docs/PERFORMANCE_BASELINES.md is p50 ≤ 200ms / p95 ≤
// 500ms. The in-memory store keeps handler cost in the sub-microsecond
// range so the SLO headroom is consumed by serialization, middleware,
// and network — not by the handler itself.
func BenchmarkHandleActsList(b *testing.B) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/acts", nil)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		handleActsList(rr, req)
		if rr.Code != http.StatusOK {
			b.Fatalf("expected 200, got %d", rr.Code)
		}
	}
}

// BenchmarkHandleSearch measures the cost of a single GET /api/v1/search
// request. The handler exercises the in-memory substring search across
// bills, acts, and constitution articles in the Kenya seed dataset.
// This is the upper bound on the cost the API BFF pays before the
// Postgres FTS migration (022_search_fts) is wired end-to-end.
func BenchmarkHandleSearch(b *testing.B) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/search?q=data", nil)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		handleSearch(rr, req)
		if rr.Code != http.StatusOK {
			b.Fatalf("expected 200, got %d", rr.Code)
		}
	}
}

// BenchmarkHandleDebtDashboard measures the cost of a single GET
// /api/v1/debt request via makeDebtRouter. The handler pulls the
// latest CBK snapshot from the seeded DebtRepository and synthesises
// the dashboard summary. The SLO documented in
// docs/PERFORMANCE_BASELINES.md is p95 ≤ 200ms for the debt dashboard;
// this benchmark establishes the in-process baseline cost.
func BenchmarkHandleDebtDashboard(b *testing.B) {
	repo := newTestDebtRepo(&testing.T{})
	router := makeDebtRouter(repo)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/debt", nil)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		router(rr, req)
		if rr.Code != http.StatusOK {
			b.Fatalf("expected 200, got %d", rr.Code)
		}
	}
}
