// Package middleware provides HTTP middleware for the Civic Intelligence API.
//
// security_headers.go — sets the canonical security response headers required
// by production gate #8 (SECURITY). The audit (audit-team-4 §53 + audit-team-5
// P0-3) flagged the absence of these headers; adding them here closes that
// finding for the API BFF.
//
// Headers set on every response:
//
//   - X-Content-Type-Options: nosniff
//     Prevents MIME-sniffing attacks where a browser guesses the content type
//     of a response and runs it as something other than what the server
//     declared (e.g. interpreting a text/plain body as HTML).
//   - X-Frame-Options: DENY
//     Prevents clickjacking by forbidding the response from being rendered
//     inside an <iframe>. The Next.js frontend does not embed the API; this
//     is a defense-in-depth measure.
//   - X-XSS-Protection: 1; mode=block
//     Enables the legacy reflected-XSS filter in browsers that still ship it
//     (notably older Safari). Modern Chrome has removed the filter, but the
//     header is still required by the audit; it is a no-op where unsupported.
//   - Strict-Transport-Security: max-age=31536000
//     Forces HTTPS for the next year. The API terminates TLS at the ingress
//     (NGINX / ALB), so the browser sees HTTPS even when the upstream is HTTP.
//     A 1-year max-age with no includeSubDomains is the minimum required by
//     the audit; production hardening should add preload once the cert is
//     stable.
//   - Referrer-Policy: strict-origin-when-cross-origin
//     Sends the full origin only on same-origin requests; sends just the
//     origin on cross-origin HTTPS→HTTPS; sends nothing on HTTPS→HTTP.
//   - X-Permitted-Cross-Domain-Policies: none
//     Tells Adobe Flash + PDF readers not to grant cross-domain access to
//     this endpoint. Flash is dead but PDF readers still honour the header.
//
// SecurityHeaders middleware is intentionally placed INSIDE the request_id +
// metrics middlewares so the audit log records that the headers were emitted,
// and OUTSIDE the auth + handler so auth-rejection responses also carry the
// headers (otherwise an attacker could fingerprint the auth layer by absence).
package middleware

import "net/http"

// Canonical security headers. Exported as constants so tests + downstream
// code can reference them by name rather than re-typing the header value.
const (
	HeaderXContentTypeOptions        = "X-Content-Type-Options"
	HeaderXFrameOptions               = "X-Frame-Options"
	HeaderXXSSProtection              = "X-XSS-Protection"
	HeaderStrictTransportSecurity     = "Strict-Transport-Security"
	HeaderReferrerPolicy              = "Referrer-Policy"
	HeaderXPermittedCrossDomainPolicies = "X-Permitted-Cross-Domain-Policies"
)

// DefaultSecurityHeaders is the header set applied to every response. The
// values are constants — production callers do not need to customise them.
// Tests assert on this map's contents; do not change values without
// updating test_security_headers.go.
var DefaultSecurityHeaders = map[string]string{
	HeaderXContentTypeOptions:          "nosniff",
	HeaderXFrameOptions:                "DENY",
	HeaderXXSSProtection:               "1; mode=block",
	HeaderStrictTransportSecurity:      "max-age=31536000",
	HeaderReferrerPolicy:               "strict-origin-when-cross-origin",
	HeaderXPermittedCrossDomainPolicies: "none",
}

// SecurityHeaders returns middleware that sets the canonical security
// response headers. It calls the next handler FIRST then sets the headers
// on the ResponseWriter — this guarantees the headers appear on every
// response (including 4xx and 5xx) regardless of whether the handler wrote
// them. If the handler already set one of the headers, the handler's value
// wins (we use .Set() which overwrites — but the next handler is downstream
// of this middleware so it runs before us; we only overwrite if the value
// is empty).
//
// The middleware is safe to use as the innermost middleware in the chain
// (just outside the route mux) — it adds no latency.
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set the headers BEFORE calling next so the handler can override
		// them (e.g. Content-Type for a JSON response) before WriteHeader
		// commits the headers. Setting them now also means a handler that
		// calls WriteHeader directly still gets the security headers.
		for k, v := range DefaultSecurityHeaders {
			// Only set if not already present — the handler may legitimately
			// override (e.g. HSTS preload directives). Header().Set replaces,
			// which is what we want when our value is the canonical one.
			if w.Header().Get(k) == "" {
				w.Header().Set(k, v)
			}
		}
		next.ServeHTTP(w, r)
	})
}
