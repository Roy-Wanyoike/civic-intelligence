package middleware

import (
        "context"
        "encoding/json"
        "net/http"
        "net/http/httptest"
        "testing"
)

// TestCountry_HeaderSetsContext verifies that when a caller sends an
// X-Civic-Country header, the middleware stores that code on the request
// context (so handlers can read it via CountryFromContext) and echoes it
// back on the response.
func TestCountry_HeaderSetsContext(t *testing.T) {
        var sawCountry string
        handler := Country(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                sawCountry = CountryFromContext(r.Context())
                w.WriteHeader(http.StatusOK)
        }))

        req := httptest.NewRequest("GET", "/api/v1/acts", nil)
        req.Header.Set(CountryHeader, "UG")
        rr := httptest.NewRecorder()
        handler.ServeHTTP(rr, req)

        if sawCountry != "UG" {
                t.Errorf("expected context country=UG, got %q", sawCountry)
        }
        if got := rr.Header().Get(CountryHeader); got != "UG" {
                t.Errorf("expected response header X-Civic-Country=UG, got %q", got)
        }
        if rr.Code != http.StatusOK {
                t.Errorf("expected 200, got %d", rr.Code)
        }
}

// TestCountry_QueryParamFallback verifies that when the header is absent,
// the middleware falls back to the ?country= query parameter. This is the
// path used by curl quick-tests and server-to-server integrations that
// cannot easily set headers.
func TestCountry_QueryParamFallback(t *testing.T) {
        var sawCountry string
        handler := Country(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                sawCountry = CountryFromContext(r.Context())
                w.WriteHeader(http.StatusOK)
        }))

        req := httptest.NewRequest("GET", "/api/v1/acts?country=NG", nil)
        rr := httptest.NewRecorder()
        handler.ServeHTTP(rr, req)

        if sawCountry != "NG" {
                t.Errorf("expected context country=NG from query, got %q", sawCountry)
        }
}

// TestCountry_HeaderBeatsQueryParam verifies the documented precedence:
// when BOTH the header and the query param are present, the header wins.
func TestCountry_HeaderBeatsQueryParam(t *testing.T) {
        var sawCountry string
        handler := Country(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                sawCountry = CountryFromContext(r.Context())
                w.WriteHeader(http.StatusOK)
        }))

        req := httptest.NewRequest("GET", "/api/v1/acts?country=UG", nil)
        req.Header.Set(CountryHeader, "ZA")
        rr := httptest.NewRecorder()
        handler.ServeHTTP(rr, req)

        if sawCountry != "ZA" {
                t.Errorf("expected header (ZA) to beat query (UG), got %q", sawCountry)
        }
}

// TestCountry_DefaultsToKenya verifies that when neither the header nor
// the query param is present, the middleware defaults to "KE" (Kenya).
// This preserves the historical API contract for callers that have not
// been updated to send the header.
func TestCountry_DefaultsToKenya(t *testing.T) {
        var sawCountry string
        handler := Country(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                sawCountry = CountryFromContext(r.Context())
                w.WriteHeader(http.StatusOK)
        }))

        req := httptest.NewRequest("GET", "/api/v1/acts", nil)
        rr := httptest.NewRecorder()
        handler.ServeHTTP(rr, req)

        if sawCountry != "KE" {
                t.Errorf("expected default country=KE, got %q", sawCountry)
        }
        if got := rr.Header().Get(CountryHeader); got != "KE" {
                t.Errorf("expected response header X-Civic-Country=KE, got %q", got)
        }
}

// TestCountry_GlobalMode verifies that "ALL" is accepted as the special
// "global dashboard" country. This is what the /compare, /indicators,
// /dashboard, and /graph endpoints use to return data across all 6
// countries at once.
func TestCountry_GlobalMode(t *testing.T) {
        var sawCountry string
        handler := Country(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                sawCountry = CountryFromContext(r.Context())
                w.WriteHeader(http.StatusOK)
        }))

        req := httptest.NewRequest("GET", "/api/v1/compare/countries", nil)
        req.Header.Set(CountryHeader, "ALL")
        rr := httptest.NewRecorder()
        handler.ServeHTTP(rr, req)

        if sawCountry != GlobalCountry {
                t.Errorf("expected country=ALL, got %q", sawCountry)
        }
        if got := rr.Header().Get(CountryHeader); got != "ALL" {
                t.Errorf("expected response header X-Civic-Country=ALL, got %q", got)
        }
}

// TestCountry_CaseInsensitive verifies the middleware upper-cases the
// supplied code before storing it. URL query strings routinely carry
// lowercase values; downstream code should not have to do its own
// canonicalisation.
func TestCountry_CaseInsensitive(t *testing.T) {
        cases := []struct {
                in   string
                want string
        }{
                {"ke", "KE"},
                {"ug", "UG"},
                {"tz", "TZ"},
                {"gh", "GH"},
                {"ng", "NG"},
                {"za", "ZA"},
                {"all", GlobalCountry},
        }
        for _, tc := range cases {
                t.Run(tc.in, func(t *testing.T) {
                        var saw string
                        handler := Country(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                                saw = CountryFromContext(r.Context())
                                w.WriteHeader(http.StatusOK)
                        }))
                        req := httptest.NewRequest("GET", "/api/v1/acts", nil)
                        req.Header.Set(CountryHeader, tc.in)
                        rr := httptest.NewRecorder()
                        handler.ServeHTTP(rr, req)
                        if saw != tc.want {
                                t.Errorf("input %q: expected context country=%q, got %q", tc.in, tc.want, saw)
                        }
                        if got := rr.Header().Get(CountryHeader); got != tc.want {
                                t.Errorf("input %q: expected response header=%q, got %q", tc.in, tc.want, got)
                        }
                })
        }
}

// TestCountry_InvalidReturns400 verifies that an unsupported country code
// returns 400 with a JSON body listing the supported countries. The body
// must be actionable — a misconfigured client should be able to read it
// and fix the request without consulting docs.
func TestCountry_InvalidReturns400(t *testing.T) {
        var sawCountry string
        called := false
        handler := Country(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                called = true
                sawCountry = CountryFromContext(r.Context())
                w.WriteHeader(http.StatusOK)
        }))

        req := httptest.NewRequest("GET", "/api/v1/acts", nil)
        req.Header.Set(CountryHeader, "XX")
        rr := httptest.NewRecorder()
        handler.ServeHTTP(rr, req)

        if called {
                t.Fatal("next handler should NOT be called for an invalid country")
        }
        if sawCountry != "" {
                t.Errorf("handler context should be untouched, got %q", sawCountry)
        }
        if rr.Code != http.StatusBadRequest {
                t.Fatalf("expected 400 for invalid country, got %d", rr.Code)
        }

        var body struct {
                Error              string   `json:"error"`
                Message            string   `json:"message"`
                Supplied           string   `json:"supplied"`
                SupportedCountries []string `json:"supported_countries"`
                GlobalCountry      string   `json:"global_country"`
        }
        if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
                t.Fatalf("decode: %v", err)
        }
        if body.Error != "invalid_country" {
                t.Errorf("expected error=invalid_country, got %q", body.Error)
        }
        if body.Supplied != "XX" {
                t.Errorf("expected supplied=XX echoed back, got %q", body.Supplied)
        }
        if len(body.SupportedCountries) != 6 {
                t.Errorf("expected 6 supported countries, got %d", len(body.SupportedCountries))
        }
        if body.GlobalCountry != GlobalCountry {
                t.Errorf("expected global_country=ALL, got %q", body.GlobalCountry)
        }
        // Verify the supported list contains the expected codes in order.
        expected := []string{"KE", "UG", "TZ", "GH", "NG", "ZA"}
        for i, c := range expected {
                if i >= len(body.SupportedCountries) || body.SupportedCountries[i] != c {
                        t.Errorf("supported_countries[%d] = %q, want %q", i, body.SupportedCountries[i], c)
                }
        }
}

// TestCountry_EmptyHeaderFallsThrough verifies that an empty header value
// is treated as "not supplied" — the middleware should fall through to the
// query param (and ultimately to the default), not 400 on an empty string.
func TestCountry_EmptyHeaderFallsThrough(t *testing.T) {
        var sawCountry string
        handler := Country(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                sawCountry = CountryFromContext(r.Context())
                w.WriteHeader(http.StatusOK)
        }))

        // Empty header → falls through to default (no query param either).
        req := httptest.NewRequest("GET", "/api/v1/acts", nil)
        req.Header.Set(CountryHeader, "")
        rr := httptest.NewRecorder()
        handler.ServeHTTP(rr, req)

        if sawCountry != "KE" {
                t.Errorf("expected default KE when header is empty, got %q", sawCountry)
        }
}

// TestCountry_AllSupportedCodes verifies every code in SupportedCountries
// is accepted by the middleware (no accidental gap in the list).
func TestCountry_AllSupportedCodes(t *testing.T) {
        for _, code := range SupportedCountries {
                t.Run(code, func(t *testing.T) {
                        var saw string
                        handler := Country(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                                saw = CountryFromContext(r.Context())
                                w.WriteHeader(http.StatusOK)
                        }))
                        req := httptest.NewRequest("GET", "/api/v1/acts", nil)
                        req.Header.Set(CountryHeader, code)
                        rr := httptest.NewRecorder()
                        handler.ServeHTTP(rr, req)
                        if rr.Code != http.StatusOK {
                                t.Fatalf("expected 200 for supported country %s, got %d", code, rr.Code)
                        }
                        if saw != code {
                                t.Errorf("expected context country=%s, got %q", code, saw)
                        }
                })
        }
}

// TestCountry_ResponseHeaderAlwaysSet verifies the response carries the
// X-Civic-Country header even when the request did not supply one —
// this lets the frontend sync its selector to the actual scope the API
// used (e.g. on first load when the cookie has not been set yet).
func TestCountry_ResponseHeaderAlwaysSet(t *testing.T) {
        handler := Country(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                w.WriteHeader(http.StatusOK)
        }))

        req := httptest.NewRequest("GET", "/api/v1/acts", nil)
        rr := httptest.NewRecorder()
        handler.ServeHTTP(rr, req)

        if got := rr.Header().Get(CountryHeader); got != "KE" {
                t.Errorf("expected response header X-Civic-Country=KE (the default), got %q", got)
        }
}

// TestCountryFromContext_DefaultsToKenya verifies the helper returns "KE"
// for a context with no country (e.g. code invoked outside an HTTP request,
// or a handler that was not chained behind the Country middleware).
func TestCountryFromContext_DefaultsToKenya(t *testing.T) {
        if got := CountryFromContext(context.Background()); got != "KE" {
                t.Errorf("expected default KE outside HTTP chain, got %q", got)
        }
}

// TestCountryFromContext_DefaultsToKenyaForNilParent verifies the helper
// does not panic on a nil context (the stdlib's context.Value is nil-safe,
// but we want a regression guard for this contract).
func TestCountryFromContext_DefaultsToKenyaForNilParent(t *testing.T) {
        if got := CountryFromContext(context.Background()); got != DefaultCountry {
                t.Errorf("expected %q, got %q", DefaultCountry, got)
        }
}

// TestWithCountry_RoundTrip verifies WithCountry + CountryFromContext
// are inverses — every supported code stored by the helper is recoverable
// (including the special GlobalCountry).
func TestWithCountry_RoundTrip(t *testing.T) {
        cases := []string{"KE", "UG", "TZ", "GH", "NG", "ZA", GlobalCountry}
        for _, code := range cases {
                ctx := WithCountry(context.Background(), code)
                if got := CountryFromContext(ctx); got != code {
                        t.Errorf("WithCountry(%q) → CountryFromContext = %q, want %q", code, got, code)
                }
        }
}

// TestIsSupportedCountry verifies the helper accepts the 6 supported codes
// + the special ALL, and rejects everything else.
func TestIsSupportedCountry(t *testing.T) {
        for _, c := range SupportedCountries {
                if !IsSupportedCountry(c) {
                        t.Errorf("expected %q to be supported", c)
                }
        }
        if !IsSupportedCountry(GlobalCountry) {
                t.Errorf("expected ALL to be supported")
        }
        for _, c := range []string{"", "XX", "kenya", "uk", "us", "ca", "fr"} {
                if IsSupportedCountry(c) {
                        t.Errorf("expected %q to be unsupported", c)
                }
        }
}

// TestCountry_PropagatesThroughChain verifies the country survives a
// nested middleware chain (Country outermost, then a trivial passthrough,
// then the handler). This guards against accidental context-loss when
// other middlewares call r.WithContext().
func TestCountry_PropagatesThroughChain(t *testing.T) {
        var saw string
        inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                saw = CountryFromContext(r.Context())
                w.WriteHeader(http.StatusOK)
        })
        passthrough := func(next http.Handler) http.Handler {
                return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                        next.ServeHTTP(w, r)
                })
        }
        handler := Country(passthrough(inner))

        req := httptest.NewRequest("GET", "/api/v1/acts", nil)
        req.Header.Set(CountryHeader, "GH")
        rr := httptest.NewRecorder()
        handler.ServeHTTP(rr, req)

        if saw != "GH" {
                t.Errorf("expected country to propagate to inner handler, got %q", saw)
        }
}
