// Package middleware provides HTTP middleware for the Civic Intelligence API.
//
// country.go — country-scoping middleware (task ENG-J1).
//
// The platform serves 6 countries (Kenya, Uganda, Tanzania, Ghana, Nigeria,
// South Africa) from a single codebase. A user from Uganda should see
// Ugandan Bills, Acts, Institutions, People, and Debt — not Kenyan ones.
// The frontend Government Selector (apps/web/src/components/government-selector.tsx)
// sets the X-Civic-Country header on every API call; this middleware reads
// that header, validates it against the supported-country list, stores it on
// the request context, and echoes it back on the response so the frontend
// knows which country was ultimately used.
//
// Resolution order (first non-empty wins):
//
//  1. X-Civic-Country header (set by the frontend Government Selector)
//  2. ?country=UG query parameter (fallback for clients that cannot set
//     headers — e.g. curl quick-tests, server-to-server integrations)
//  3. "KE" (Kenya) — the historical default and the first country adapter
//     shipped. Falling back to a real country (rather than 400-ing) preserves
//     the existing API contract for callers that have not been updated.
//
// The special value "ALL" is the "global / cross-country dashboard" mode.
// It is intended for the /compare, /indicators, /dashboard, and /graph
// endpoints that return data across all 6 countries at once. For list
// endpoints (/acts, /people, …) "ALL" returns data from every country —
// the handler decides whether that is meaningful.
//
// If the supplied country code is not in the supported list, the middleware
// returns 400 Bad Request with a JSON body listing the supported codes —
// so a misconfigured client sees an actionable error instead of silently
// falling back to Kenya.
package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
)

// CountryHeader is the canonical HTTP header name for the country scope.
// The frontend Government Selector sets it on every API call so the BFF
// knows which country adapter to dispatch to.
const CountryHeader = "X-Civic-Country"

// CountryQueryParam is the fallback query-string parameter for clients that
// cannot set headers (e.g. curl quick-tests, server-to-server integrations).
const CountryQueryParam = "country"

// DefaultCountry is the country used when no country is supplied. Kenya is
// the first country adapter shipped and is the historical default; falling
// back to a real country (rather than 400-ing) preserves the existing API
// contract for callers that have not been updated to send the header.
const DefaultCountry = "KE"

// GlobalCountry is the special country code that means "all countries".
// It is used by the /compare, /indicators, /dashboard, and /graph endpoints
// to return data across all 6 countries at once (the "dashboard" view).
// For list endpoints (/acts, /people, …) it returns data from every
// country — the handler decides whether that is meaningful.
const GlobalCountry = "ALL"

// SupportedCountries is the canonical list of country codes the platform
// recognises. Order is significant: the 400-error body lists them in this
// order, and the test suite asserts on the exact membership + ordering.
//
// Add a new country here when its adapter lands in adapters/{country}/.
// The middleware deliberately does NOT auto-discover adapters — a country
// is "supported" only when it is in this list, which forces an explicit
// enablement step (see ENG-J1 / CONTRIBUTING.md).
var SupportedCountries = []string{"KE", "UG", "TZ", "GH", "NG", "ZA"}

// countryCtxKey is the typed context key for country storage. A distinct
// type avoids collisions with other packages' context keys (and with the
// request_id + principal keys defined elsewhere in this package).
type countryCtxKey struct{}

// CountryFromContext returns the country code stored on ctx, or DefaultCountry
// ("KE") if none is present. Handlers call this to scope their responses:
//
//	country := middleware.CountryFromContext(r.Context())
//	if country == middleware.GlobalCountry {
//	    // return data across all countries (dashboard view)
//	} else {
//	    // filter the response by `country`
//	}
//
// The helper never returns the empty string — even for code invoked
// outside an HTTP request (background jobs, tests that did not chain
// the middleware), it returns DefaultCountry. This means a handler that
// forgets to chain the middleware still produces Kenya-scoped data
// instead of crashing on a nil/empty country — which is the historical
// behaviour the rest of the codebase expects.
func CountryFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(countryCtxKey{}).(string); ok && v != "" {
		return v
	}
	return DefaultCountry
}

// WithCountry returns a new context carrying the supplied country code.
// Tests and non-HTTP callers can use this to inject a known country
// (mirroring the WithRequestID pattern in request_id.go).
//
// The supplied code is NOT validated here — callers are expected to
// pass a known-good code (KE, UG, TZ, GH, NG, ZA, or ALL). The
// middleware itself performs validation before calling WithCountry.
func WithCountry(ctx context.Context, code string) context.Context {
	return context.WithValue(ctx, countryCtxKey{}, code)
}

// IsSupportedCountry reports whether code is one of the supported country
// codes (KE, UG, TZ, GH, NG, ZA) or the special GlobalCountry ("ALL").
// The check is case-insensitive on the input — callers routinely pass
// lowercase values from URL query strings — but the canonicalised
// uppercase form is what is stored on the context.
func IsSupportedCountry(code string) bool {
	if code == GlobalCountry {
		return true
	}
	for _, c := range SupportedCountries {
		if c == code {
			return true
		}
	}
	return false
}

// resolveCountry picks the country code for the request, applying the
// documented precedence: header → query → default. The result is
// upper-cased so downstream code can compare against the canonical
// constants without worrying about case.
func resolveCountry(r *http.Request) string {
	if h := strings.TrimSpace(r.Header.Get(CountryHeader)); h != "" {
		return strings.ToUpper(h)
	}
	if q := strings.TrimSpace(r.URL.Query().Get(CountryQueryParam)); q != "" {
		return strings.ToUpper(q)
	}
	return DefaultCountry
}

// Country is middleware that scopes every request to a single country (or
// to GlobalCountry for the cross-country dashboard endpoints). It:
//
//   - resolves the country code per the precedence above
//   - validates it against the supported list (returning 400 with the
//     supported list on invalid input)
//   - stores the code on the request context (handlers pull it via
//     CountryFromContext)
//   - echoes the resolved code back on the response as X-Civic-Country so
//     the frontend knows which country was ultimately used (this is
//     especially useful when the request omitted the header — the frontend
//     can detect "the API fell back to KE" and sync its selector)
//
// The middleware is safe to chain anywhere in the stack. The recommended
// position is just inside RequestID (so the request_id is on the context
// for any 400 logs) and outside RateLimit (so a 429 still carries the
// country header — the frontend can attribute the throttle to the right
// country in its telemetry).
func Country(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		code := resolveCountry(r)
		if !IsSupportedCountry(code) {
			writeCountryError(w, code)
			return
		}
		// Echo the resolved code on the response BEFORE calling next so
		// even a handler that panics or short-circuits (e.g. auth 401)
		// still carries the header — the frontend can always trust the
		// response header to know which country scope was active.
		w.Header().Set(CountryHeader, code)

		ctx := WithCountry(r.Context(), code)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// writeCountryError writes the canonical 400 response for an unsupported
// country code. The body is JSON (matching the rest of the API's error
// shape) and lists the supported codes so a misconfigured client sees an
// actionable error rather than a vague "bad request".
func writeCountryError(w http.ResponseWriter, supplied string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error":               "invalid_country",
		"message":             "X-Civic-Country must be one of: KE, UG, TZ, GH, NG, ZA (or ALL for the global dashboard view)",
		"supplied":            supplied,
		"supported_countries": SupportedCountries,
		"global_country":     GlobalCountry,
	})
}
