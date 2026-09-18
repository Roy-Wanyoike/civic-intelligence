package middleware

import (
        "net/http"
        "net/http/httptest"
        "testing"
        "time"
)

// === PerPathRateLimiter tests ===

func TestPerPathRateLimitPolicy_Defaults(t *testing.T) {
        p := NewPerPathRateLimitPolicy()
        cases := []struct {
                path     string
                expected int
        }{
                {"/api/v1/questions", AIRateLimitPerMinute},
                {"/api/v1/questions/stream", AIRateLimitPerMinute},
                {"/api/v1/questions/abc-123", AIRateLimitPerMinute},
                {"/api/v1/bills", DefaultRateLimitPerMinute}, // exact match for /api/v1/bills (no trailing slash)
                {"/api/v1/bills/abc", AIRateLimitPerMinute},  // /api/v1/bills/ matches
                {"/api/v1/bills/abc/summary", AIRateLimitPerMinute},
                {"/api/v1/bills/abc/impact", AIRateLimitPerMinute},
                {"/api/v1/search", SearchRateLimitPerMinute},
                {"/api/v1/scenarios", SearchRateLimitPerMinute},
                {"/api/v1/scenarios/abc/run", SearchRateLimitPerMinute},
                {"/api/v1/acts", DefaultRateLimitPerMinute},
                {"/api/v1/healthz", DefaultRateLimitPerMinute},
                {"/api/v1/anything-else", DefaultRateLimitPerMinute},
                {"/unknown", DefaultRateLimitPerMinute},
        }
        for _, tc := range cases {
                got := p.limitFor(tc.path)
                if got != tc.expected {
                        t.Errorf("limitFor(%q) = %d, want %d", tc.path, got, tc.expected)
                }
        }
}

func TestPerPathRateLimitPolicy_CustomDefault(t *testing.T) {
        p := &RateLimitPolicy{
                Rules:            nil,
                DefaultPerMinute: 42,
        }
        if got := p.limitFor("/anything"); got != 42 {
                t.Errorf("expected custom default 42, got %d", got)
        }
}

func TestPerPathRateLimiter_AllowsUnderLimit(t *testing.T) {
        limiter := NewPerPathRateLimiter(&RateLimitPolicy{
                Rules: []rateLimitRule{
                        {prefix: "/api/v1/ai", perMinute: 3},
                },
                DefaultPerMinute: 100,
        })
        handler := limiter.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                w.WriteHeader(http.StatusOK)
        }))

        for i := 0; i < 3; i++ {
                req := httptest.NewRequest("GET", "/api/v1/ai", nil)
                req.RemoteAddr = "1.2.3.4:1234"
                rr := httptest.NewRecorder()
                handler.ServeHTTP(rr, req)
                if rr.Code != http.StatusOK {
                        t.Errorf("request %d: expected 200, got %d", i, rr.Code)
                }
        }
}

func TestPerPathRateLimiter_BlocksOverLimit(t *testing.T) {
        limiter := NewPerPathRateLimiter(&RateLimitPolicy{
                Rules: []rateLimitRule{
                        {prefix: "/api/v1/ai", perMinute: 2},
                },
                DefaultPerMinute: 100,
        })
        handler := limiter.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                w.WriteHeader(http.StatusOK)
        }))

        // First 2 should pass.
        for i := 0; i < 2; i++ {
                req := httptest.NewRequest("GET", "/api/v1/ai", nil)
                req.RemoteAddr = "1.2.3.4:1234"
                rr := httptest.NewRecorder()
                handler.ServeHTTP(rr, req)
                if rr.Code != http.StatusOK {
                        t.Fatalf("request %d: expected 200, got %d", i, rr.Code)
                }
        }

        // 3rd should be blocked.
        req := httptest.NewRequest("GET", "/api/v1/ai", nil)
        req.RemoteAddr = "1.2.3.4:1234"
        rr := httptest.NewRecorder()
        handler.ServeHTTP(rr, req)
        if rr.Code != http.StatusTooManyRequests {
                t.Errorf("expected 429 on 3rd request, got %d", rr.Code)
        }
        if rr.Header().Get("Retry-After") == "" {
                t.Error("expected non-empty Retry-After header on 429")
        }
}

func TestPerPathRateLimiter_PerIPIsolation(t *testing.T) {
        // Two different IPs should have independent budgets.
        limiter := NewPerPathRateLimiter(&RateLimitPolicy{
                Rules: []rateLimitRule{
                        {prefix: "/api/v1/ai", perMinute: 1},
                },
                DefaultPerMinute: 100,
        })
        handler := limiter.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                w.WriteHeader(http.StatusOK)
        }))

        // IP 1 — first request allowed, second blocked.
        req1 := httptest.NewRequest("GET", "/api/v1/ai", nil)
        req1.RemoteAddr = "1.1.1.1:1234"
        rr1 := httptest.NewRecorder()
        handler.ServeHTTP(rr1, req1)
        if rr1.Code != http.StatusOK {
                t.Errorf("IP 1 first request: expected 200, got %d", rr1.Code)
        }

        req1b := httptest.NewRequest("GET", "/api/v1/ai", nil)
        req1b.RemoteAddr = "1.1.1.1:1234"
        rr1b := httptest.NewRecorder()
        handler.ServeHTTP(rr1b, req1b)
        if rr1b.Code != http.StatusTooManyRequests {
                t.Errorf("IP 1 second request: expected 429, got %d", rr1b.Code)
        }

        // IP 2 — first request should succeed because buckets are per-IP.
        req2 := httptest.NewRequest("GET", "/api/v1/ai", nil)
        req2.RemoteAddr = "2.2.2.2:5678"
        rr2 := httptest.NewRecorder()
        handler.ServeHTTP(rr2, req2)
        if rr2.Code != http.StatusOK {
                t.Errorf("IP 2 first request: expected 200, got %d (per-IP isolation broken)", rr2.Code)
        }
}

func TestPerPathRateLimiter_WindowReset(t *testing.T) {
        // Use a fake clock so the test is deterministic.
        frozen := time.Now()
        limiter := &PerPathRateLimiter{
                policy: &RateLimitPolicy{
                        Rules: []rateLimitRule{
                                {prefix: "/api/v1/ai", perMinute: 1},
                        },
                        DefaultPerMinute: 100,
                },
                now:     func() time.Time { return frozen },
                buckets: make(map[string]*rateBucket),
        }
        handler := limiter.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                w.WriteHeader(http.StatusOK)
        }))

        // First request fills the bucket.
        req := httptest.NewRequest("GET", "/api/v1/ai", nil)
        req.RemoteAddr = "1.1.1.1:1234"
        rr := httptest.NewRecorder()
        handler.ServeHTTP(rr, req)
        if rr.Code != http.StatusOK {
                t.Fatalf("first request: expected 200, got %d", rr.Code)
        }

        // Second request (same window) is blocked.
        rr2 := httptest.NewRecorder()
        handler.ServeHTTP(rr2, req)
        if rr2.Code != http.StatusTooManyRequests {
                t.Fatalf("second request (same window): expected 429, got %d", rr2.Code)
        }

        // Advance past the window — bucket should reset.
        frozen = frozen.Add(time.Minute + time.Second)
        rr3 := httptest.NewRecorder()
        handler.ServeHTTP(rr3, req)
        if rr3.Code != http.StatusOK {
                t.Errorf("third request (new window): expected 200, got %d", rr3.Code)
        }
}

func TestPerPathRateLimiter_DefaultPolicyIsPlatformStandard(t *testing.T) {
        // The zero-value policy (passed via NewPerPathRateLimiter(nil)) MUST
        // pre-populate the platform's standard tiered limits — AI at 10, search
        // at 60, default at 300. This is the contract that main.go relies on
        // when it constructs the limiter without an explicit policy.
        limiter := NewPerPathRateLimiter(nil)
        if limiter.policy == nil {
                t.Fatal("expected non-nil policy")
        }
        if got := limiter.policy.limitFor("/api/v1/questions"); got != AIRateLimitPerMinute {
                t.Errorf("questions limit = %d, want %d", got, AIRateLimitPerMinute)
        }
        if got := limiter.policy.limitFor("/api/v1/search"); got != SearchRateLimitPerMinute {
                t.Errorf("search limit = %d, want %d", got, SearchRateLimitPerMinute)
        }
        if got := limiter.policy.limitFor("/api/v1/acts"); got != DefaultRateLimitPerMinute {
                t.Errorf("default limit = %d, want %d", got, DefaultRateLimitPerMinute)
        }
}

func TestItoa(t *testing.T) {
        cases := []struct {
                in   int
                want string
        }{
                {0, "0"},
                {1, "1"},
                {9, "9"},
                {10, "10"},
                {99, "99"},
                {300, "300"},
                {31536000, "31536000"},
        }
        for _, tc := range cases {
                if got := itoa(tc.in); got != tc.want {
                        t.Errorf("itoa(%d) = %q, want %q", tc.in, got, tc.want)
                }
        }
}

// === SecurityHeaders tests ===

func TestSecurityHeaders_AllPresent(t *testing.T) {
        handler := SecurityHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                w.WriteHeader(http.StatusOK)
        }))

        req := httptest.NewRequest("GET", "/api/v1/acts", nil)
        rr := httptest.NewRecorder()
        handler.ServeHTTP(rr, req)

        for k, v := range DefaultSecurityHeaders {
                if got := rr.Header().Get(k); got != v {
                        t.Errorf("header %q = %q, want %q", k, got, v)
                }
        }
}

func TestSecurityHeaders_DoesNotOverrideHandlerSet(t *testing.T) {
        // A handler that explicitly sets X-Frame-Options to SAMEORIGIN should
        // have its value preserved — SecurityHeaders only fills in missing
        // headers.
        handler := SecurityHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                w.Header().Set(HeaderXFrameOptions, "SAMEORIGIN")
                w.WriteHeader(http.StatusOK)
        }))

        req := httptest.NewRequest("GET", "/api/v1/acts", nil)
        rr := httptest.NewRecorder()
        handler.ServeHTTP(rr, req)

        if got := rr.Header().Get(HeaderXFrameOptions); got != "SAMEORIGIN" {
                t.Errorf("expected handler's X-Frame-Options=SAMEORIGIN to win, got %q", got)
        }
        // Other headers should still be set.
        if got := rr.Header().Get(HeaderXContentTypeOptions); got != "nosniff" {
                t.Errorf("expected X-Content-Type-Options=nosniff, got %q", got)
        }
}

func TestSecurityHeaders_AppliedOnErrorResponses(t *testing.T) {
        // Auth-rejection (401) must still carry the security headers — otherwise
        // an attacker could fingerprint the auth layer by header absence.
        handler := SecurityHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                w.WriteHeader(http.StatusUnauthorized)
        }))

        req := httptest.NewRequest("GET", "/api/v1/questions", nil)
        rr := httptest.NewRecorder()
        handler.ServeHTTP(rr, req)

        if rr.Code != http.StatusUnauthorized {
                t.Fatalf("expected 401, got %d", rr.Code)
        }
        if got := rr.Header().Get(HeaderXContentTypeOptions); got != "nosniff" {
                t.Errorf("expected nosniff on 401 response, got %q", got)
        }
        if got := rr.Header().Get(HeaderXFrameOptions); got != "DENY" {
                t.Errorf("expected DENY on 401 response, got %q", got)
        }
}

// === CORS tests ===

func TestCORS_DisabledWhenNoOrigins(t *testing.T) {
        // No origins configured — CORS middleware should be a no-op.
        called := false
        handler := CORSMiddleware(CORSConfig{})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                called = true
                w.WriteHeader(http.StatusOK)
        }))

        req := httptest.NewRequest("GET", "/api/v1/acts", nil)
        req.Header.Set("Origin", "https://evil.example.com")
        rr := httptest.NewRecorder()
        handler.ServeHTTP(rr, req)

        if !called {
                t.Error("expected next handler to be called when CORS is disabled")
        }
        if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "" {
                t.Errorf("expected no Access-Control-Allow-Origin header, got %q", got)
        }
}

func TestCORS_AllowedOriginSimpleRequest(t *testing.T) {
        cfg := DefaultCORSConfig()
        cfg.AllowedOrigins = []string{"https://app.civicintelligence.com"}
        handler := CORSMiddleware(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                w.WriteHeader(http.StatusOK)
        }))

        req := httptest.NewRequest("GET", "/api/v1/acts", nil)
        req.Header.Set("Origin", "https://app.civicintelligence.com")
        rr := httptest.NewRecorder()
        handler.ServeHTTP(rr, req)

        if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "https://app.civicintelligence.com" {
                t.Errorf("expected allowed origin, got %q", got)
        }
        if got := rr.Header().Get("Vary"); got != "Origin" {
                t.Errorf("expected Vary=Origin, got %q", got)
        }
}

func TestCORS_DisallowedOrigin(t *testing.T) {
        cfg := DefaultCORSConfig()
        cfg.AllowedOrigins = []string{"https://app.civicintelligence.com"}
        handler := CORSMiddleware(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                w.WriteHeader(http.StatusOK)
        }))

        req := httptest.NewRequest("GET", "/api/v1/acts", nil)
        req.Header.Set("Origin", "https://evil.example.com")
        rr := httptest.NewRecorder()
        handler.ServeHTTP(rr, req)

        if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "" {
                t.Errorf("expected no Access-Control-Allow-Origin for disallowed origin, got %q", got)
        }
}

func TestCORS_PreflightShortCircuits(t *testing.T) {
        cfg := DefaultCORSConfig()
        cfg.AllowedOrigins = []string{"https://app.civicintelligence.com"}
        called := false
        handler := CORSMiddleware(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                called = true
                w.WriteHeader(http.StatusOK)
        }))

        req := httptest.NewRequest("OPTIONS", "/api/v1/acts", nil)
        req.Header.Set("Origin", "https://app.civicintelligence.com")
        req.Header.Set("Access-Control-Request-Method", "POST")
        req.Header.Set("Access-Control-Request-Headers", "Content-Type, Authorization")
        rr := httptest.NewRecorder()
        handler.ServeHTTP(rr, req)

        if called {
                t.Error("expected next handler NOT to be called on preflight")
        }
        if rr.Code != http.StatusNoContent {
                t.Errorf("expected 204 on preflight, got %d", rr.Code)
        }
        if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "https://app.civicintelligence.com" {
                t.Errorf("expected allowed origin on preflight, got %q", got)
        }
        if got := rr.Header().Get("Access-Control-Allow-Methods"); got == "" {
                t.Error("expected non-empty Access-Control-Allow-Methods on preflight")
        }
        if got := rr.Header().Get("Access-Control-Allow-Headers"); got == "" {
                t.Error("expected non-empty Access-Control-Allow-Headers on preflight")
        }
        if got := rr.Header().Get("Access-Control-Max-Age"); got == "" {
                t.Error("expected non-empty Access-Control-Max-Age on preflight")
        }
}

func TestCORS_AllowCredentials(t *testing.T) {
        cfg := DefaultCORSConfig()
        cfg.AllowedOrigins = []string{"https://app.civicintelligence.com"}
        cfg.AllowCredentials = true
        handler := CORSMiddleware(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                w.WriteHeader(http.StatusOK)
        }))

        req := httptest.NewRequest("GET", "/api/v1/acts", nil)
        req.Header.Set("Origin", "https://app.civicintelligence.com")
        rr := httptest.NewRecorder()
        handler.ServeHTTP(rr, req)

        if got := rr.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
                t.Errorf("expected Access-Control-Allow-Credentials=true, got %q", got)
        }
}

func TestCORS_NoOriginHeaderNotACORSRequest(t *testing.T) {
        // A request with no Origin header is a same-origin request — CORS
        // middleware should not set any CORS response headers.
        cfg := DefaultCORSConfig()
        cfg.AllowedOrigins = []string{"https://app.civicintelligence.com"}
        handler := CORSMiddleware(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                w.WriteHeader(http.StatusOK)
        }))

        req := httptest.NewRequest("GET", "/api/v1/acts", nil)
        rr := httptest.NewRecorder()
        handler.ServeHTTP(rr, req)

        if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "" {
                t.Errorf("expected no Access-Control-Allow-Origin for same-origin request, got %q", got)
        }
}

func TestCORS_OriginCaseInsensitive(t *testing.T) {
        // The allowlist lookup is case-insensitive (per RFC 6454 §3.2 — scheme
        // and host are case-insensitive). A request with "HTTPS://App." should
        // match an allowlist entry of "https://app.".
        cfg := DefaultCORSConfig()
        cfg.AllowedOrigins = []string{"https://app.civicintelligence.com"}
        handler := CORSMiddleware(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                w.WriteHeader(http.StatusOK)
        }))

        req := httptest.NewRequest("GET", "/api/v1/acts", nil)
        req.Header.Set("Origin", "HTTPS://APP.civicintelligence.com")
        rr := httptest.NewRecorder()
        handler.ServeHTTP(rr, req)

        if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "HTTPS://APP.civicintelligence.com" {
                t.Errorf("expected case-insensitive origin match, got %q", got)
        }
}
