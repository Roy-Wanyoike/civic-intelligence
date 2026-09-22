// Package main provides the Topics API endpoints (issue #267).
//
// A "civic topic" is a subject-matter lens the platform exposes so
// citizens can browse Bills, Acts, and people associated with that
// subject (e.g. "Health", "Education", "Finance & National Planning").
// The list endpoint returns the 10 seed Kenyan topics with their
// associated counts; the detail endpoint returns the topic plus the
// concrete Bills / Acts / People slices.
//
// Endpoints (registered in main.go):
//
//	GET /api/v1/topics          -- list topics (filtered by country context)
//	GET /api/v1/topics/{id}     -- topic detail with associated Bills/Acts/People
//
// The country scope is read from middleware.CountryFromContext — the
// same path used by /api/v1/people, /api/v1/committees, etc. Only
// Kenya has seed topics today; other countries return an empty list
// (200, not 404) so the frontend Topics page degrades gracefully.
package main

import (
	"net/http"
	"strings"

	"github.com/Roy-Wanyoike/civic-intelligence/adapters/kenya/kenya_seed"
	"github.com/Roy-Wanyoike/civic-intelligence/services/api/internal/middleware"
)

// topicsDisclaimer is the canonical note returned with every topics
// response. Mirrors the existing scorecard / constituency disclaimer
// pattern: factual records only, no ranking implied.
const topicsDisclaimer = "Topic associations are based on the seed Bills + Acts shipped with the platform. Full ingestion pending (issue #19)."

// handleTopicsList handles GET /api/v1/topics. The handler is
// registered on BOTH /api/v1/topics (no trailing slash) and
// /api/v1/topics/ (per-issue #264 + #266 — the no-slash form MUST
// return 200, not a 301 redirect). It dispatches on the trailing
// path tail: empty tail → list, non-empty tail → detail handler.
//
// Country filtering:
//   - middleware.Country places the resolved country code on the
//     request context (X-Civic-Country header → ?country= query → "KE").
//   - "ALL" (GlobalCountry) returns topics from every country that
//     has seed data — the dashboard view.
//   - "KE" returns the 10 Kenyan topics.
//   - Any other country returns an empty list (200, not 404) because
//     no other country adapter ships topics yet.
func handleTopicsList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	// Dispatch: any path tail beyond the base route goes to the detail
	// handler. The base route is registered WITHOUT a trailing slash
	// (per issue #264 + #266), so the only path that reaches this
	// function with a tail is /api/v1/topics/{id}.
	tail := strings.TrimPrefix(r.URL.Path, "/api/v1/topics")
	tail = strings.TrimPrefix(tail, "/")
	if tail != "" {
		handleTopicDetail(w, r)
		return
	}

	country := middleware.CountryFromContext(r.Context())

	// The seed slice is Kenya-only today. Filter by country; ALL
	// (GlobalCountry) returns the full Kenya seed (no other country
	// has topics yet, so the dashboard view is the same as KE).
	var items []kenya_seed.Topic
	switch {
	case country == "" || country == middleware.GlobalCountry || country == "KE":
		items = kenya_seed.KenyaTopics
	default:
		// Other countries do not yet have seed topics — return an
		// empty list (200, not 404) so the frontend Topics page
		// degrades gracefully with an empty-state UI.
		items = []kenya_seed.Topic{}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"items":   items,
		"total":   len(items),
		"country": country,
		"source":  "kenya_seed",
		"note":    topicsDisclaimer,
	})
}

// handleTopicDetail handles GET /api/v1/topics/{id}. Returns the
// topic plus the associated Bills / Acts / People slices. Returns
// 404 when the supplied ID does not match any seed topic.
//
// The ID lookup is case-insensitive (Topic IDs are normalised to
// lower-case kebab-case, but /topics/Health is treated as equivalent
// to /topics/health) — matching the path-normalisation pattern used
// by the /people/{id} handler.
func handleTopicDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/topics/")
	// Strip any sub-resource tail so /topics/health/bills doesn't
	// accidentally treat "health/bills" as the topic ID. The
	// detail handler does not currently expose sub-resources — a
	// future /topics/{id}/bills / /acts / /people could re-use the
	// associated slices from kenya_seed directly.
	if i := strings.IndexByte(id, '/'); i >= 0 {
		id = id[:i]
	}
	if id == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "topic ID required")
		return
	}

	// Country visibility gate: a Uganda-scoped request asking for a
	// Kenyan topic ID returns 404 (mirrors the /people/{id} gate).
	// "ALL" (GlobalCountry) is allowed so the cross-country dashboard
	// can deep-link into the Kenya topic detail page.
	country := middleware.CountryFromContext(r.Context())
	if country != "" && country != middleware.GlobalCountry && country != "KE" {
		writeError(w, http.StatusNotFound, "not_found", "topic not visible in country "+country+": "+id)
		return
	}

	detail := kenya_seed.KenyaTopicDetail(id)
	if detail == nil {
		writeError(w, http.StatusNotFound, "not_found", "topic not found: "+id)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"topic":   detail.Topic,
		"bills":   detail.Bills,
		"acts":    detail.Acts,
		"people":  detail.People,
		"country": detail.Country,
		"source":  "kenya_seed",
		"note":    topicsDisclaimer,
	})
}
