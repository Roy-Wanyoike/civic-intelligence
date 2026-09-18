package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestRequestID_GeneratesWhenMissing verifies the middleware generates a
// fresh UUIDv4 when the caller did not send X-Request-Id, places it on
// the response header, and propagates it on the request context.
func TestRequestID_GeneratesWhenMissing(t *testing.T) {
	var sawID string
	handler := RequestID(nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawID = RequestIDFromRequest(r)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/v1/acts", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if sawID == "" {
		t.Fatal("expected non-empty request_id on context")
	}
	if got := rr.Header().Get(RequestIDHeader); got != sawID {
		t.Errorf("response header = %q, want %q", got, sawID)
	}
	// UUIDv4 format: 8-4-4-4-12 hex chars.
	if len(sawID) != 36 || !strings.Contains(sawID[14:18], "4") {
		t.Errorf("expected UUIDv4, got %q", sawID)
	}
}

// TestRequestID_ReusesCallerSupplied verifies the middleware trusts a
// well-formed X-Request-Id header supplied by the caller (so external
// services can correlate traces end-to-end).
func TestRequestID_ReusesCallerSupplied(t *testing.T) {
	const supplied = "req-abc-123"
	var sawID string
	handler := RequestID(nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawID = RequestIDFromRequest(r)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/v1/acts", nil)
	req.Header.Set(RequestIDHeader, supplied)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if sawID != supplied {
		t.Errorf("expected request_id %q, got %q", supplied, sawID)
	}
	if got := rr.Header().Get(RequestIDHeader); got != supplied {
		t.Errorf("response header = %q, want %q", got, supplied)
	}
}

// TestRequestID_RejectsOversizedHeader verifies an absurdly long
// X-Request-Id header is dropped (replaced with a fresh UUID) to prevent
// log-injection and oversized-header-storage abuse.
func TestRequestID_RejectsOversizedHeader(t *testing.T) {
	oversized := strings.Repeat("a", 1000)
	var sawID string
	handler := RequestID(nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawID = RequestIDFromRequest(r)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/v1/acts", nil)
	req.Header.Set(RequestIDHeader, oversized)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if sawID == oversized {
		t.Fatal("expected oversized header to be dropped, but it was reused")
	}
	if len(sawID) != 36 {
		t.Errorf("expected fresh UUIDv4 (36 chars), got %q (len=%d)", sawID, len(sawID))
	}
}

// TestRequestID_RejectsNewlineInjection verifies a request_id containing a
// newline (which would corrupt downstream log lines) is rejected and a
// fresh UUID is generated instead.
func TestRequestID_RejectsNewlineInjection(t *testing.T) {
	injected := "abc\nFAKE LOG LINE"
	var sawID string
	handler := RequestID(nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawID = RequestIDFromRequest(r)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/v1/acts", nil)
	req.Header.Set(RequestIDHeader, injected)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if sawID == injected {
		t.Fatalf("expected newline-injected header to be rejected, got %q", sawID)
	}
	if strings.Contains(sawID, "\n") {
		t.Errorf("generated request_id contains newline: %q", sawID)
	}
}

// TestRequestID_PropagatesThroughChain verifies the request_id survives a
// nested middleware chain (RequestID outermost, then a trivial passthrough,
// then the handler).
func TestRequestID_PropagatesThroughChain(t *testing.T) {
	var sawID string
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawID = RequestIDFromRequest(r)
		w.WriteHeader(http.StatusOK)
	})
	// Wrap inner in a passthrough so we can prove context propagation.
	passthrough := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)
		})
	}
	handler := RequestID(nil)(passthrough(inner))

	req := httptest.NewRequest("GET", "/api/v1/acts", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if sawID == "" {
		t.Error("expected request_id to be visible to inner handler")
	}
}

// TestRequestIDFromContext_OutsideRequest verifies the helper returns
// empty string for a context that has no request_id (e.g. background
// work invoked outside the HTTP handler chain).
func TestRequestIDFromContext_OutsideRequest(t *testing.T) {
	if got := RequestIDFromContext(context.Background()); got != "" {
		t.Errorf("expected empty request_id outside HTTP chain, got %q", got)
	}
}

// TestRequestIDFromContext_RoundTrip verifies WithRequestID +
// RequestIDFromContext are inverses.
func TestRequestIDFromContext_RoundTrip(t *testing.T) {
	const id = "req-xyz-456"
	ctx := WithRequestID(context.Background(), id)
	if got := RequestIDFromContext(ctx); got != id {
		t.Errorf("expected %q, got %q", id, got)
	}
}

// TestNewUUIDv4_Format verifies the generated ID matches the canonical
// 8-4-4-4-12 lowercase hex shape and carries the v4 marker bits.
func TestNewUUIDv4_Format(t *testing.T) {
	id := newUUIDv4()
	if len(id) != 36 {
		t.Fatalf("expected 36-char UUID, got %d (%q)", len(id), id)
	}
	if id[14] != '4' {
		t.Errorf("expected version nibble '4' at index 14, got %q (id=%q)", string(id[14]), id)
	}
	// Variant bits: first char of the 4th group must be 8, 9, a, or b.
	v := id[19]
	if v != '8' && v != '9' && v != 'a' && v != 'b' {
		t.Errorf("expected variant 8/9/a/b at index 19, got %q (id=%q)", string(v), id)
	}
}

// TestNewUUIDv4_Unique verifies two consecutive calls produce different
// IDs. (Not a cryptographic uniqueness test — just a smoke test that the
// entropy source is wired correctly.)
func TestNewUUIDv4_Unique(t *testing.T) {
	a, b := newUUIDv4(), newUUIDv4()
	if a == b {
		t.Errorf("expected distinct UUIDs, got %q twice", a)
	}
}
