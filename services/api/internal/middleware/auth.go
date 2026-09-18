// Package middleware provides HTTP middleware for the Civic Intelligence API.
package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/Roy-Wanyoike/civic-intelligence/packages/auth"
)

// contextKey is the type for principal storage in request context.
type contextKey int

const principalKey contextKey = 0

// PrincipalFromRequest returns the Principal stored in the request context,
// or an anonymous principal if none is present.
func PrincipalFromRequest(r *http.Request) auth.Principal {
	if v, ok := r.Context().Value(principalKey).(auth.Principal); ok {
		return v
	}
	return auth.Anonymous()
}

// RequireScope is middleware that checks the principal has at least one of
// the given scopes. If not, it returns 401 (if anonymous) or 403 (if
// authenticated but insufficient).
func RequireScope(scopes ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			p := PrincipalFromRequest(r)
			if p.IsAnonymous() {
				writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
				return
			}
			if !p.Can(scopes...) {
				writeError(w, http.StatusForbidden, "forbidden", "insufficient permissions")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequireAuth is middleware that checks the principal is authenticated
// (without checking specific scopes). Use this for endpoints where any
// authenticated user is allowed.
func RequireAuth() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			p := PrincipalFromRequest(r)
			if p.IsAnonymous() {
				writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// OptionalAuth is middleware that extracts the principal if a valid token is
// present, but does not require it. Use this for endpoints that behave
// differently for anonymous vs. authenticated users (e.g., search results
// may include more detail for authenticated researchers).
func OptionalAuth(verifier auth.OIDCTokenVerifier) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			p := extractPrincipal(r, verifier)
			ctx := context.WithValue(r.Context(), principalKey, p)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireToken is middleware that requires a valid bearer token. If the
// token is missing or invalid, it returns 401. Use this for all protected
// endpoints.
func RequireToken(verifier auth.OIDCTokenVerifier) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			p, err := verifyToken(r, verifier)
			if err != nil {
				writeError(w, http.StatusUnauthorized, "unauthorized", err.Error())
				return
			}
			ctx := context.WithValue(r.Context(), principalKey, p)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// extractPrincipal extracts the bearer token from the Authorization header
// (if present) and verifies it. If the token is missing or invalid, it
// returns an anonymous principal (no error — the caller decides whether to
// require auth).
func extractPrincipal(r *http.Request, verifier auth.OIDCTokenVerifier) auth.Principal {
	bearer := extractBearer(r)
	if bearer == "" {
		return auth.Anonymous()
	}
	p, err := verifier.Verify(r.Context(), bearer)
	if err != nil {
		return auth.Anonymous()
	}
	return p
}

// verifyToken extracts and verifies the bearer token, returning an error if
// the token is missing or invalid.
func verifyToken(r *http.Request, verifier auth.OIDCTokenVerifier) (auth.Principal, error) {
	bearer := extractBearer(r)
	if bearer == "" {
		return auth.Anonymous(), errMissingToken
	}
	return verifier.Verify(r.Context(), bearer)
}

// extractBearer pulls the bearer token from the Authorization header.
func extractBearer(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return ""
	}
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return ""
	}
	return strings.TrimPrefix(authHeader, "Bearer ")
}

// writeError writes a structured JSON error response.
func writeError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error":   code,
		"message": message,
	})
}

// RateLimit is a simple in-memory rate limiter. Production should use Redis.
// Limit: N requests per window per IP.
func RateLimit(requests int, window time.Duration) func(http.Handler) http.Handler {
	type entry struct {
		count   int
		resetAt time.Time
	}
	clients := make(map[string]*entry)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := clientIP(r)
			now := time.Now()
			e, ok := clients[ip]
			if !ok || now.After(e.resetAt) {
				clients[ip] = &entry{count: 1, resetAt: now.Add(window)}
			} else {
				e.count++
				if e.count > requests {
					w.Header().Set("Retry-After", "60")
					writeError(w, http.StatusTooManyRequests, "rate_limited", "too many requests")
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

// clientIP extracts the client IP from the request, respecting X-Forwarded-For.
func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	return r.RemoteAddr
}

var errMissingToken = &tokenError{"missing bearer token"}

type tokenError struct{ msg string }

func (e *tokenError) Error() string { return e.msg }
