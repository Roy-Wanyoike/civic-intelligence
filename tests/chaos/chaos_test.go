// Package chaos implements the chaos-engineering contract tests for
// Production Gate #11 (docs/PRODUCTION_GATE.md).
//
// The tests in this package are STANDALONE: they do not import the API
// server, the Postgres driver, or any external dependency. Instead, they
// define minimal handler shapes that mirror the production contracts the
// chaos runbooks (tests/chaos/*.md) verify manually.
//
// The contract each test enforces is the SLO documented in
// tests/chaos/README.md §"SLOs (the universal pass bar)":
//
//   - Availability during partial outage ≥ 99% (degraded is OK; 500s are not)
//   - No `panic:` stack traces in logs (process must not crash)
//   - /healthz flips state within 1 s of the outage
//   - /healthz recovers within 5 s of the dependency returning
//   - No data loss / no duplicate side effects
//
// Each test below exercises ONE failure mode against the corresponding
// minimal handler shape. Running `go test ./...` from this directory
// should pass in < 1 s.
//
// The tests are intentionally cheap to run so they can be wired into CI
// as a smoke check between the heavier k6 + chaos-mesh game-day drills.
package chaos

import (
        "context"
        "encoding/json"
        "errors"
        "net/http"
        "net/http/httptest"
        "strings"
        "sync"
        "testing"
        "time"
)

// errDatabaseUnavailable is the sentinel error the failing DB layer returns
// when Postgres is unreachable. In production this would be a *pgconn.PgError
// or a *errors.errorString wrapping a dial failure — the handler's job is to
// translate it to HTTP 503, NOT to leak the driver-specific error type to the
// client.
var errDatabaseUnavailable = errors.New("database unavailable")

// errCacheUnavailable is the sentinel error the failing cache layer returns
// when Redis is unreachable. In production this would be *redis.Nil or a
// network error wrapped by packages/cache. The handler must fall back to the
// canonical source (DB) and return 200, not 500.
var errCacheUnavailable = errors.New("cache unavailable")

// errAITimeout is the sentinel error the AI gateway returns when the upstream
// provider times out. In production this would be a context.DeadlineExceeded
// after the 60 s StartToCloseTimeout configured in
// services/ingestion/internal/temporal/workflow.go:ActivityOptionsFor("extract").
// The handler must return 504 with a JSON body (not hang, not 500).
var errAITimeout = errors.New("ai provider timeout")

// ---------------------------------------------------------------------------
// Dependency interfaces (mirror the production contracts)
// ---------------------------------------------------------------------------

// Database is the minimal interface the chaos tests need from the canonical
// persistence layer. Production uses legislation.ActRepository /
// DebtRepository — both are read via a List-like call that returns a slice +
// error. The chaos tests only care about the error path.
type Database interface {
        List(ctx context.Context) ([]string, error)
}

// Cache is the minimal interface the chaos tests need from the cache layer.
// Production uses packages/cache.Client with Get/Set; the chaos tests only
// care about the Get-with-fallback path.
type Cache interface {
        Get(ctx context.Context, key string) (string, error)
}

// AIProvider is the minimal interface the chaos tests need from the AI
// gateway. Production uses services/ai/app/gateway.py — the contract is
// "returns a summary + model id, or an error".
type AIProvider interface {
        Summarize(ctx context.Context, text string) (string, error)
}

// DocumentParser is the minimal interface for the ingestion parse step.
// Production uses services/documents; the contract is "parses raw bytes into
// structured sections, or returns a malformed-input error".
type DocumentParser interface {
        Parse(ctx context.Context, raw []byte) ([]string, error)
}

// ---------------------------------------------------------------------------
// Fault-injecting implementations (for the chaos tests)
// ---------------------------------------------------------------------------

// failingDatabase returns errDatabaseUnavailable for every call. Simulates
// Postgres being killed mid-request (chaos runbook: db-failure.md).
type failingDatabase struct{ once sync.Once }

func (f *failingDatabase) List(_ context.Context) ([]string, error) {
        return nil, errDatabaseUnavailable
}

// failingCache returns errCacheUnavailable for every call. Simulates Redis
// being killed mid-request (chaos runbook: redis-down.md).
type failingCache struct{}

func (failingCache) Get(_ context.Context, _ string) (string, error) {
        return "", errCacheUnavailable
}

// stubDatabase is the canonical fallback when the cache fails. Returns a
// single hardcoded row so the handler can complete the request.
type stubDatabase struct{ rows []string }

func (s *stubDatabase) List(_ context.Context) ([]string, error) { return s.rows, nil }

// timingOutAIProvider blocks until the supplied context is cancelled, then
// returns the context error. Simulates the AI service hanging past its
// StartToCloseTimeout (chaos runbook: ai-timeout.md).
type timingOutAIProvider struct{}

func (timingOutAIProvider) Summarize(ctx context.Context, _ string) (string, error) {
        <-ctx.Done()
        return "", ctx.Err()
}

// fallbackAIProvider is the production-grade fallback: returns a stub
// summary tagged validated=false when the primary AI provider is
// unavailable. The handler should serve this fallback rather than 500.
type fallbackAIProvider struct{}

func (fallbackAIProvider) Summarize(_ context.Context, _ string) (string, error) {
        return "AI summary unavailable; serving cached or fallback content.", nil
}

// strictDocumentParser rejects malformed input. Simulates the documents
// service's parser when handed truncated / corrupt bytes (chaos runbook:
// malformed-document — implicit in db-failure.md's "no panic" guarantee).
type strictDocumentParser struct{}

func (strictDocumentParser) Parse(_ context.Context, raw []byte) ([]string, error) {
        if len(raw) == 0 {
                return nil, errors.New("malformed document: empty payload")
        }
        s := string(raw)
        // A document must be a complete JSON object or array — both opening
        // AND closing brace must be present. This rejects truncated payloads
        // like `{"title":"` which start with `{` but never close.
        isObj := strings.HasPrefix(s, "{") && strings.HasSuffix(s, "}")
        isArr := strings.HasPrefix(s, "[") && strings.HasSuffix(s, "]")
        if !isObj && !isArr {
                return nil, errors.New("malformed document: expected complete JSON object or array")
        }
        return []string{"section-1"}, nil
}

// ---------------------------------------------------------------------------
// Handlers — minimal shapes that mirror the production contracts
// ---------------------------------------------------------------------------

// makeDBBackedHandler mirrors the production pattern: handler calls DB,
// returns 200 on success, 503 (with JSON envelope) on DB failure, never 500.
//
// This is the contract chaos runbook db-failure.md verifies manually; the
// test below verifies it programmatically.
func makeDBBackedHandler(db Database) http.HandlerFunc {
        return func(w http.ResponseWriter, r *http.Request) {
                rows, err := db.List(r.Context())
                if err != nil {
                        // Graceful degradation: 503 + JSON envelope, NEVER 500 + stack trace.
                        w.Header().Set("Content-Type", "application/json")
                        w.Header().Set("Retry-After", "30")
                        w.WriteHeader(http.StatusServiceUnavailable)
                        _ = json.NewEncoder(w).Encode(map[string]any{
                                "error":       "database_unavailable",
                                "message":     "the canonical data source is temporarily unavailable",
                                "retry_after": 30,
                        })
                        return
                }
                w.Header().Set("Content-Type", "application/json")
                w.WriteHeader(http.StatusOK)
                _ = json.NewEncoder(w).Encode(map[string]any{"items": rows})
        }
}

// makeCacheBackedHandler mirrors the production read-through cache pattern:
// try cache → on cache error, fall back to DB → return 200 with X-Cache: MISS.
//
// This is the contract chaos runbook redis-down.md verifies manually.
func makeCacheBackedHandler(cache Cache, db Database) http.HandlerFunc {
        return func(w http.ResponseWriter, r *http.Request) {
                const cacheKey = "list:v1"
                cached, err := cache.Get(r.Context(), cacheKey)
                if err == nil {
                        w.Header().Set("Content-Type", "application/json")
                        w.Header().Set("X-Cache", "HIT")
                        w.WriteHeader(http.StatusOK)
                        _ = json.NewEncoder(w).Encode(map[string]any{"items": []string{cached}})
                        return
                }
                // Cache miss (or cache failure) → fall back to DB. The handler
                // MUST NOT propagate the cache error as a 500; the canonical
                // source is still authoritative.
                rows, dbErr := db.List(r.Context())
                if dbErr != nil {
                        w.Header().Set("Content-Type", "application/json")
                        w.WriteHeader(http.StatusServiceUnavailable)
                        _ = json.NewEncoder(w).Encode(map[string]any{
                                "error":   "data_unavailable",
                                "message": "cache and canonical source are both unavailable",
                        })
                        return
                }
                w.Header().Set("Content-Type", "application/json")
                w.Header().Set("X-Cache", "MISS")
                w.WriteHeader(http.StatusOK)
                _ = json.NewEncoder(w).Encode(map[string]any{"items": rows})
        }
}

// makeAIProxyHandler mirrors the production AI-proxy pattern: call the AI
// provider with a per-request timeout; on timeout, serve the fallback (NOT
// a 500). The handler returns 200 with validated=false if the fallback was
// used, or 504 if both primary + fallback fail.
//
// This is the contract chaos runbook ai-timeout.md verifies manually.
func makeAIProxyHandler(primary, fallback AIProvider, timeout time.Duration) http.HandlerFunc {
        return func(w http.ResponseWriter, r *http.Request) {
                ctx, cancel := context.WithTimeout(r.Context(), timeout)
                defer cancel()

                // Read the request body directly. We deliberately do NOT use
                // r.GetBody() — that returns nil for requests constructed
                // via httptest.NewRequest, and we want the handler to behave
                // identically in tests and in production (where net/http sets
                // GetBody only for retry-safe bodies).
                buf := make([]byte, 4096)
                n, _ := r.Body.Read(buf)

                summary, err := primary.Summarize(ctx, string(buf[:n]))
                if err == nil {
                        writeAISummary(w, summary, true)
                        return
                }
                // Primary failed (likely timeout). Fall back to the cached / stub
                // provider rather than returning a 5xx — the chaos contract is
                // "cached AI outputs still serve" (tests/chaos/ai-timeout.md).
                // Note: we deliberately use context.Background() (not ctx) for
                // the fallback call so the timed-out primary does not poison
                // the fallback's cancellation.
                fbSummary, fbErr := fallback.Summarize(context.Background(), string(buf[:n]))
                if fbErr != nil {
                        writeAIError(w, http.StatusGatewayTimeout)
                        return
                }
                writeAISummary(w, fbSummary, false)
        }
}

func writeAISummary(w http.ResponseWriter, summary string, validated bool) {
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusOK)
        _ = json.NewEncoder(w).Encode(map[string]any{
                "summary":   summary,
                "validated": validated,
                "source":    "ai",
        })
}

func writeAIError(w http.ResponseWriter, status int) {
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(status)
        _ = json.NewEncoder(w).Encode(map[string]any{
                "error": "ai_unavailable",
                "message": "the AI provider is unavailable; please retry",
        })
}

// makeDocumentIngestHandler mirrors the production ingestion parse step:
// accept raw bytes, parse, return 200 with structured sections on success,
// 400 on malformed input (NOT 500).
//
// The chaos contract is "no crash" — the handler must reject malformed input
// cleanly, not propagate the parse error as a panic.
func makeDocumentIngestHandler(parser DocumentParser) http.HandlerFunc {
        return func(w http.ResponseWriter, r *http.Request) {
                // Defensive: a request with no Content-Type or with an absurd
                // Content-Length must NOT crash the handler.
                raw := make([]byte, 0, 4096)
                buf := make([]byte, 4096)
                for {
                        n, err := r.Body.Read(buf)
                        if n > 0 {
                                raw = append(raw, buf[:n]...)
                                if len(raw) > 1<<20 { // 1 MiB cap — defends against memory exhaustion
                                        w.Header().Set("Content-Type", "application/json")
                                        w.WriteHeader(http.StatusRequestEntityTooLarge)
                                        _ = json.NewEncoder(w).Encode(map[string]any{
                                                "error":   "payload_too_large",
                                                "message": "document exceeds 1 MiB ingestion cap",
                                        })
                                        return
                                }
                        }
                        if err != nil {
                                break
                        }
                }
                sections, err := parser.Parse(r.Context(), raw)
                if err != nil {
                        w.Header().Set("Content-Type", "application/json")
                        w.WriteHeader(http.StatusBadRequest)
                        _ = json.NewEncoder(w).Encode(map[string]any{
                                "error":   "malformed_document",
                                "message": err.Error(),
                        })
                        return
                }
                w.Header().Set("Content-Type", "application/json")
                w.WriteHeader(http.StatusOK)
                _ = json.NewEncoder(w).Encode(map[string]any{
                        "sections": sections,
                        "count":    len(sections),
                })
        }
}

// ---------------------------------------------------------------------------
// Tests — one per chaos runbook
// ---------------------------------------------------------------------------

// TestChaos_DatabaseUnavailable verifies the contract documented in
// tests/chaos/db-failure.md: when the DB is killed, the API returns 503 with a
// JSON envelope, never a panic or a 500.
//
// Pass criteria (runbook):
//   - API returns 503 (NOT 500, NOT 502, NOT panic)
//   - Response body is JSON with an "error" field
//   - Retry-After header is set so clients can back off
func TestChaos_DatabaseUnavailable(t *testing.T) {
        t.Parallel()
        h := makeDBBackedHandler(&failingDatabase{})

        // Use a recover guard so a panic in the handler is reported as a test
        // failure rather than crashing the test process.
        defer func() {
                if r := recover(); r != nil {
                        t.Fatalf("handler panicked under DB failure: %v", r)
                }
        }()

        req := httptest.NewRequest(http.MethodGet, "/api/v1/items", nil)
        rr := httptest.NewRecorder()
        h(rr, req)

        if rr.Code != http.StatusServiceUnavailable {
                t.Fatalf("expected 503; got %d (body=%s)", rr.Code, rr.Body.String())
        }
        if got := rr.Header().Get("Retry-After"); got != "30" {
                t.Errorf("expected Retry-After=30; got %q", got)
        }
        var body map[string]any
        if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
                t.Fatalf("response body is not JSON: %v (body=%s)", err, rr.Body.String())
        }
        if body["error"] != "database_unavailable" {
                t.Errorf("expected error=database_unavailable; got %v", body["error"])
        }
}

// TestChaos_RedisUnavailable verifies the contract documented in
// tests/chaos/redis-down.md: when Redis is killed, the cache layer fails
// closed (returns an error), and the handler falls back to the canonical
// source. The response is 200 with X-Cache: MISS (degraded but functional),
// NOT a 500.
//
// Pass criteria (runbook):
//   - API returns 200 (NOT 500, NOT 503)
//   - X-Cache header reports MISS (not ERROR)
//   - Response body is correct data from the canonical source
func TestChaos_RedisUnavailable(t *testing.T) {
        t.Parallel()
        cache := failingCache{}
        db := &stubDatabase{rows: []string{"canonical-row-1", "canonical-row-2"}}
        h := makeCacheBackedHandler(cache, db)

        defer func() {
                if r := recover(); r != nil {
                        t.Fatalf("handler panicked under cache failure: %v", r)
                }
        }()

        req := httptest.NewRequest(http.MethodGet, "/api/v1/items", nil)
        rr := httptest.NewRecorder()
        h(rr, req)

        if rr.Code != http.StatusOK {
                t.Fatalf("expected 200 (graceful degradation); got %d (body=%s)", rr.Code, rr.Body.String())
        }
        if got := rr.Header().Get("X-Cache"); got != "MISS" {
                t.Errorf("expected X-Cache=MISS; got %q (expected MISS, not ERROR)", got)
        }
        var body map[string]any
        if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
                t.Fatalf("response body is not JSON: %v", err)
        }
        items, _ := body["items"].([]any)
        if len(items) != 2 {
                t.Errorf("expected 2 items from canonical source; got %d", len(items))
        }
}

// TestChaos_AIProviderTimeout verifies the contract documented in
// tests/chaos/ai-timeout.md: when the AI provider hangs past its timeout,
// the handler returns either:
//   - 200 with validated=false (fallback summary served), OR
//   - 504 with JSON body (if no fallback available).
//
// The handler must NEVER hang indefinitely or return a 500 with a stack
// trace. This test exercises the fallback path.
//
// Pass criteria (runbook):
//   - Handler returns within timeout + small grace (not indefinite hang)
//   - Response is JSON with "validated": false (degraded, not FAILED)
//   - No panic
func TestChaos_AIProviderTimeout(t *testing.T) {
        t.Parallel()
        primary := timingOutAIProvider{}
        fallback := fallbackAIProvider{}
        // 50 ms timeout — short enough that the test is fast, long enough to
        // prove the handler doesn't hang. Production uses 60 s
        // (ActivityOptionsFor("extract")).
        h := makeAIProxyHandler(primary, fallback, 50*time.Millisecond)

        defer func() {
                if r := recover(); r != nil {
                        t.Fatalf("handler panicked under AI timeout: %v", r)
                }
        }()

        req := httptest.NewRequest(http.MethodPost, "/api/v1/summarize", strings.NewReader("some bill text"))
        rr := httptest.NewRecorder()

        // Bound the test wall-clock so a regression that hangs the handler
        // fails the test rather than the CI job.
        done := make(chan struct{})
        go func() {
                defer close(done)
                h(rr, req)
        }()
        select {
        case <-done:
                // good — handler returned
        case <-time.After(2 * time.Second):
                t.Fatal("handler did not return within 2 s — likely hanging on the timed-out AI call")
        }

        if rr.Code != http.StatusOK {
                t.Fatalf("expected 200 (fallback served); got %d (body=%s)", rr.Code, rr.Body.String())
        }
        var body map[string]any
        if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
                t.Fatalf("response body is not JSON: %v", err)
        }
        if body["validated"] != false {
                t.Errorf("expected validated=false (fallback); got %v", body["validated"])
        }
        if body["source"] != "ai" {
                t.Errorf("expected source=ai; got %v", body["source"])
        }
}

// TestChaos_MalformedDocument verifies the contract documented implicitly
// in tests/chaos/db-failure.md's "no panic" guarantee: when the ingestion
// parse step receives malformed input, the handler returns 400 with a JSON
// envelope, NEVER a 500 or a panic.
//
// Pass criteria (runbook):
//   - Handler returns 400 (NOT 500, NOT panic)
//   - Response body identifies the failure mode (malformed_document)
//   - Process is still alive (no goroutine leak, no panic)
func TestChaos_MalformedDocument(t *testing.T) {
        t.Parallel()
        parser := strictDocumentParser{}
        h := makeDocumentIngestHandler(parser)

        cases := []struct {
                name string
                body string
        }{
                {name: "empty payload", body: ""},
                {name: "non-JSON garbage", body: "this is not JSON"},
                {name: "truncated JSON", body: "{\"title\":\""},
                {name: "binary garbage", body: "\x00\x01\x02\x03\xff\xfe"},
        }
        for _, tc := range cases {
                t.Run(tc.name, func(t *testing.T) {
                        defer func() {
                                if r := recover(); r != nil {
                                        t.Fatalf("handler panicked on malformed input: %v", r)
                                }
                        }()

                        req := httptest.NewRequest(http.MethodPost, "/api/v1/documents", strings.NewReader(tc.body))
                        rr := httptest.NewRecorder()
                        h(rr, req)

                        if rr.Code == http.StatusInternalServerError {
                                t.Fatalf("handler returned 500 on malformed input — must be 400 (body=%s)", rr.Body.String())
                        }
                        if rr.Code != http.StatusBadRequest {
                                t.Fatalf("expected 400; got %d (body=%s)", rr.Code, rr.Body.String())
                        }
                        var body map[string]any
                        if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
                                t.Fatalf("response body is not JSON: %v (body=%s)", err, rr.Body.String())
                        }
                        if body["error"] != "malformed_document" {
                                t.Errorf("expected error=malformed_document; got %v", body["error"])
                        }
                })
        }
}

// TestChaos_NoPanicUnderDeadlineExceeded verifies that a context cancellation
// (e.g. client disconnect mid-request) does NOT propagate as a panic. This is
// the universal SLO documented in tests/chaos/README.md §"SLOs" — "No
// `panic:` stack traces in logs: 0".
//
// The test cancels the request context mid-flight and asserts the handler
// returns gracefully (with whatever status code is appropriate — the contract
// is "no panic", not a specific status).
func TestChaos_NoPanicUnderDeadlineExceeded(t *testing.T) {
        t.Parallel()

        // A handler that respects context cancellation.
        h := makeDBBackedHandler(&slowDatabase{delay: 100 * time.Millisecond})

        ctx, cancel := context.WithCancel(context.Background())
        cancel() // already cancelled when the handler runs

        req := httptest.NewRequest(http.MethodGet, "/api/v1/items", nil)
        req = req.WithContext(ctx)
        rr := httptest.NewRecorder()

        defer func() {
                if r := recover(); r != nil {
                        t.Fatalf("handler panicked under cancelled context: %v", r)
                }
        }()
        h(rr, req)
        // No status-code assertion: the contract is "no panic" only. A handler
        // that returns 503 or 499 is fine; a handler that panics is not.
}

// slowDatabase sleeps for the configured delay before returning. Used by the
// cancellation test to simulate a slow DB query that gets cancelled
// mid-flight.
type slowDatabase struct{ delay time.Duration }

func (s *slowDatabase) List(ctx context.Context) ([]string, error) {
        select {
        case <-time.After(s.delay):
                return []string{"row-1"}, nil
        case <-ctx.Done():
                return nil, ctx.Err()
        }
}
