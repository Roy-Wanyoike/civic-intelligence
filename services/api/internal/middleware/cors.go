// Package middleware provides HTTP middleware for the Civic Intelligence API.
//
// cors.go — Cross-Origin Resource Sharing (CORS) middleware.
//
// Production gate #8 (SECURITY) requires a configurable CORS policy. The
// platform's Next.js frontend is served from a different origin than the API
// BFF in production (e.g. app.civicintelligence.com → api.civicintelligence.com),
// so a CORS policy is required for the browser to read API responses.
//
// The middleware implements the W3C CORS specification (REC-cors-20140116)
// with the following behaviour:
//
//   - For simple requests (GET, HEAD, POST with simple content types), the
//     middleware sets Access-Control-Allow-Origin to the request's Origin
//     header IF that origin is in the allowlist, then forwards to next.
//   - For preflight (OPTIONS) requests, the middleware short-circuits with
//     a 204 No Content response carrying the full CORS preflight headers.
//   - When the request's Origin is not in the allowlist, the middleware
//     forwards to next WITHOUT setting Access-Control-Allow-Origin. The
//     browser will then block the cross-origin read, which is the correct
//     fail-closed behaviour.
//   - When the allowlist is empty, CORS is disabled entirely (the middleware
//     becomes a no-op). This is the safe default for development.
//
// The allowlist is configurable via the CORSConfig struct. Production callers
// should populate AllowedOrigins from the CORS_ALLOWED_ORIGINS env var (a
// comma-separated list).
package middleware

import (
	"net/http"
	"strconv"
	"strings"
)

// CORSConfig configures the CORS middleware. The zero value disables CORS
// (the middleware becomes a no-op), which is appropriate for development
// where the API and frontend share an origin.
type CORSConfig struct {
	// AllowedOrigins is the list of origins permitted to make cross-origin
	// requests. Each value must include the scheme + host + port (e.g.
	// "https://app.civicintelligence.com"). An empty list disables CORS.
	AllowedOrigins []string

	// AllowedMethods is the list of HTTP methods allowed for cross-origin
	// requests. Defaults to the standard set (GET, POST, PUT, DELETE,
	// OPTIONS, PATCH, HEAD) when empty.
	AllowedMethods []string

	// AllowedHeaders is the list of request headers the browser is permitted
	// to send in a cross-origin request. Defaults to the standard set
	// (Content-Type, Authorization, X-Request-Id, X-Civic-Country) when
	// empty.
	AllowedHeaders []string

	// ExposedHeaders is the list of response headers the browser is permitted
	// to read in a cross-origin response. Defaults to the standard set
	// (X-Request-Id, X-Total-Count) when empty.
	ExposedHeaders []string

	// AllowCredentials controls whether the browser is permitted to send
	// cookies + Authorization headers in cross-origin requests. When true,
	// Access-Control-Allow-Credentials: true is set, and the
	// Access-Control-Allow-Origin header is set to the specific request
	// origin (never "*") per the CORS spec.
	AllowCredentials bool

	// MaxAgeSeconds is the number of seconds the browser is permitted to
	// cache the preflight response. Defaults to 600 (10 minutes) when zero.
	// A higher value reduces OPTIONS traffic but means preflight policy
	// changes take longer to propagate to existing clients.
	MaxAgeSeconds int
}

// DefaultCORSConfig returns a CORSConfig pre-populated with the platform's
// standard method/header allowlists. Origins + credentials are left blank;
// callers must populate AllowedOrigins (and set AllowCredentials when
// cookies are needed) before passing to CORSMiddleware.
func DefaultCORSConfig() CORSConfig {
	return CORSConfig{
		AllowedMethods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodDelete,
			http.MethodOptions,
			http.MethodPatch,
			http.MethodHead,
		},
		AllowedHeaders: []string{
			"Content-Type",
			"Authorization",
			"X-Request-Id",
			"X-Civic-Country",
		},
		ExposedHeaders: []string{
			"X-Request-Id",
			"X-Total-Count",
		},
		AllowCredentials: false,
		MaxAgeSeconds:    600,
	}
}

// CORSMiddleware returns middleware that enforces the CORS policy in cfg.
// When cfg.AllowedOrigins is empty, the middleware is a no-op (CORS disabled).
func CORSMiddleware(cfg CORSConfig) func(http.Handler) http.Handler {
	// Normalize the allowlist into a set for O(1) lookups. We rebuild it
	// here rather than on every request.
	allowed := make(map[string]struct{}, len(cfg.AllowedOrigins))
	for _, o := range cfg.AllowedOrigins {
		allowed[strings.ToLower(strings.TrimSpace(o))] = struct{}{}
	}

	// Fall back to defaults for the method/header lists.
	methods := cfg.AllowedMethods
	if len(methods) == 0 {
		methods = DefaultCORSConfig().AllowedMethods
	}
	headers := cfg.AllowedHeaders
	if len(headers) == 0 {
		headers = DefaultCORSConfig().AllowedHeaders
	}
	exposed := cfg.ExposedHeaders
	if len(exposed) == 0 {
		exposed = DefaultCORSConfig().ExposedHeaders
	}
	maxAge := cfg.MaxAgeSeconds
	if maxAge == 0 {
		maxAge = 600
	}

	methodsHeader := strings.Join(methods, ", ")
	headersHeader := strings.Join(headers, ", ")
	exposedHeader := strings.Join(exposed, ", ")
	maxAgeHeader := strconv.Itoa(maxAge)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// CORS is disabled when no origins are configured.
			if len(allowed) == 0 {
				next.ServeHTTP(w, r)
				return
			}

			origin := r.Header.Get("Origin")
			if origin == "" {
				// Not a CORS request — pass through.
				next.ServeHTTP(w, r)
				return
			}

			// Look up the origin in the allowlist. If not present, fail
			// closed: forward to next without setting
			// Access-Control-Allow-Origin. The browser will block the read.
			if _, ok := allowed[strings.ToLower(origin)]; !ok {
				next.ServeHTTP(w, r)
				return
			}

			// Allowed origin — set the CORS response headers.
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			if cfg.AllowCredentials {
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}
			if exposedHeader != "" {
				w.Header().Set("Access-Control-Expose-Headers", exposedHeader)
			}

			// Handle preflight (OPTIONS) short-circuit.
			if r.Method == http.MethodOptions {
				w.Header().Set("Access-Control-Allow-Methods", methodsHeader)
				w.Header().Set("Access-Control-Allow-Headers", headersHeader)
				w.Header().Set("Access-Control-Max-Age", maxAgeHeader)
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
