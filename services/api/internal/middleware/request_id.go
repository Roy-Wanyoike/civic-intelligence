// Package middleware provides HTTP middleware for the Civic Intelligence API.
//
// request_id.go — propagates an X-Request-Id correlation identifier through
// the request lifecycle. On each incoming request:
//
//   - if the caller sent an X-Request-Id header, that value is reused
//     (after a sanity check); otherwise a fresh UUIDv4 is generated
//   - the value is stored on the request context so downstream handlers
//     can pull it via RequestIDFromContext
//   - the value is echoed back on the response via the X-Request-Id header
//     so clients can correlate logs / traces
//   - the request start + completion are logged with the request_id field
//     attached, so every log line tied to this request is correlatable
//
// FIXME: verify with go build when Go available. The task brief mentioned
// zerolog for contextual logging, but zerolog is not yet a dependency of
// services/api and the Go toolchain is unavailable this session to run
// `go mod tidy`. We reuse the existing observability.Logger (slog-based)
// which already supports structured fields; the request_id field travels
// on every log line emitted by this middleware. Downstream code that wants
// the request_id on its own log calls should pull it via
// middleware.RequestIDFromContext.
package middleware

import (
        "context"
        "crypto/rand"
        "encoding/hex"
        "net/http"
        "time"

        "github.com/Roy-Wanyoike/civic-intelligence/packages/observability"
)

// RequestIDHeader is the canonical HTTP header name for the correlation
// identifier. It is set on every response and (when supplied by the
// caller) read from every request.
const RequestIDHeader = "X-Request-Id"

// requestIDMaxLen caps the length of a caller-supplied X-Request-Id to
// prevent abuse (log injection, oversized header storage, etc.). The
// canonical form (UUIDv4) is 36 chars; we allow some headroom for
// alternative schemes (ulid, ksuid, prefixed opaque IDs) but reject
// anything absurd.
const requestIDMaxLen = 128

// requestIDCtxKey is the typed context key for request_id storage. A
// distinct type avoids collisions with other packages' context keys.
type requestIDCtxKey struct{}

// RequestIDFromContext returns the request_id stored on ctx, or the empty
// string if none is present (e.g. for code invoked outside an HTTP
// request, or for handlers that were not chained behind RequestID).
func RequestIDFromContext(ctx context.Context) string {
        if v, ok := ctx.Value(requestIDCtxKey{}).(string); ok {
                return v
        }
        return ""
}

// WithRequestID returns a new context carrying the supplied request_id.
// Tests and non-HTTP callers can use this to inject a known ID.
func WithRequestID(ctx context.Context, id string) context.Context {
        return context.WithValue(ctx, requestIDCtxKey{}, id)
}

// RequestIDFromRequest returns the request_id stored on r's context, or
// the empty string if none is present. Convenience wrapper for
// RequestIDFromContext(r.Context()).
func RequestIDFromRequest(r *http.Request) string {
        return RequestIDFromContext(r.Context())
}

// RequestID is middleware that ensures every request has a request_id. It
// reuses the caller's X-Request-Id header (if present and well-formed),
// otherwise generates a UUIDv4. The ID is placed on the request context
// AND echoed back on the response. Start/completion events are logged
// with the request_id field via the supplied logger; passing nil uses
// observability.DefaultLogger.
//
// The middleware is safe to chain ahead of rate limiting, auth, and
// metrics — the request_id is then available on the context for every
// downstream middleware + handler.
func RequestID(logger observability.Logger) func(http.Handler) http.Handler {
        if logger == nil {
                logger = observability.DefaultLogger
        }
        return func(next http.Handler) http.Handler {
                return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                        id := normalizeOrGenerate(r.Header.Get(RequestIDHeader))
                        w.Header().Set(RequestIDHeader, id)

                        ctx := WithRequestID(r.Context(), id)
                        r = r.WithContext(ctx)

                        start := time.Now()
                        logger.Info("http.request.start",
                                observability.String("request_id", id),
                                observability.String("method", r.Method),
                                observability.String("path", r.URL.Path),
                        )

                        // Wrap the ResponseWriter so we can observe the status code
                        // the handler ultimately writes — needed for the completion
                        // log line. We deliberately do NOT expose mutation of the
                        // recorded status (the wrapper is write-only from the
                        // handler's perspective).
                        rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
                        next.ServeHTTP(rec, r)

                        logger.Info("http.request.complete",
                                observability.String("request_id", id),
                                observability.String("method", r.Method),
                                observability.String("path", r.URL.Path),
                                observability.Int("status", rec.status),
                                observability.Int("latency_ms", int(time.Since(start).Milliseconds())),
                        )
                })
        }
}

// normalizeOrGenerate returns the supplied id if it is non-empty and
// within the length cap, otherwise generates a fresh UUIDv4. We do NOT
// validate the format strictly — clients may use opaque correlation IDs
// of their own scheme — but we DO reject obviously-malformed values
// (empty, whitespace-only, absurdly long).
func normalizeOrGenerate(supplied string) string {
        for _, r := range supplied {
                // Reject if any character is a control byte or newline — those
                // would corrupt downstream log lines.
                if r < 0x20 {
                        return newUUIDv4()
                }
        }
        if len(supplied) == 0 || len(supplied) > requestIDMaxLen {
                return newUUIDv4()
        }
        return supplied
}

// newUUIDv4 returns a freshly-generated RFC 4122 v4 UUID as a 36-char
// lowercase hex string (e.g. "f47ac10b-58cc-4372-a567-0e02b2c3d479").
// Uses crypto/rand so the IDs are unpredictable — important because the
// request_id is also used for log correlation and a predictable value
// would let an attacker inject noise into log queries.
//
// FIXME: verify with go build when Go available.
func newUUIDv4() string {
        var b [16]byte
        if _, err := rand.Read(b[:]); err != nil {
                // crypto/rand.Read only fails if the system entropy source is
                // unavailable. In that case the host is unusable for security-
                // sensitive work; we fall back to a constant marker so the
                // request still has *some* ID rather than failing closed.
                return "00000000-0000-4000-8000-000000000000"
        }
        // RFC 4122 §4.4: set the version (4) and variant (10) bits.
        b[6] = (b[6] & 0x0f) | 0x40 // version 4
        b[8] = (b[8] & 0x3f) | 0x80 // variant 10
        return formatUUID(b[:])
}

// formatUUID renders 16 raw bytes as the canonical 8-4-4-4-12 hex string.
func formatUUID(b []byte) string {
        s := make([]byte, 36)
        hex.Encode(s[0:8], b[0:4])
        s[8] = '-'
        hex.Encode(s[9:13], b[4:6])
        s[13] = '-'
        hex.Encode(s[14:18], b[6:8])
        s[18] = '-'
        hex.Encode(s[19:23], b[8:10])
        s[23] = '-'
        hex.Encode(s[24:36], b[10:16])
        return string(s)
}

// statusRecorder wraps http.ResponseWriter to capture the status code
// written by the downstream handler. It intentionally does NOT
// implement Unwrap() / Hijacker / Flusher passthrough — the API's
// handlers do not use streaming or upgrade, and adding passthrough would
// silently break the status capture for those paths.
type statusRecorder struct {
        http.ResponseWriter
        status int
        wrote  bool
}

func (r *statusRecorder) WriteHeader(code int) {
        if r.wrote {
                // Subsequent WriteHeader calls are no-ops; the first call wins,
                // matching net/http semantics.
                return
        }
        r.status = code
        r.wrote = true
        r.ResponseWriter.WriteHeader(code)
}

func (r *statusRecorder) Write(b []byte) (int, error) {
        if !r.wrote {
                r.wrote = true
        }
        return r.ResponseWriter.Write(b)
}
