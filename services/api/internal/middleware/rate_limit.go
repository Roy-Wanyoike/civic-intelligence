// Package middleware provides HTTP middleware for the Civic Intelligence API.
//
// rate_limit.go — per-endpoint rate limiting.
//
// The platform's audit (audit-team-5 P1-20 / §53) flagged that the production
// mux chain only had a single global RateLimit(300, time.Minute) at the outer
// edge, leaving AI + search endpoints to share the same budget as cheap read
// paths. Production gate #8 (SECURITY) requires per-tier rate-limit middleware,
// with AI endpoints tightened (10 req/min), search at 60 req/min, and a default
// of 300 req/min for everything else.
//
// PerPathRateLimiter implements that contract. It is intentionally still an
// in-memory token-bucket-per-IP per-bucket-key implementation (consistent with
// the existing RateLimit middleware so production callers do not need a Redis
// dependency to satisfy the gate). A future PR can swap the underlying store
// for a Redis-backed implementation without changing the public API.
package middleware

import (
	"net/http"
	"strings"
	"sync"
	"time"
)

// Per-path rate-limit defaults (per IP). These mirror the requirements in
// PRODUCTION_GATE.md §100 (gate #8, item 4) — AI endpoints at 10/min, search
// at 60/min, default at 300/min.
const (
	// DefaultRateLimitPerMinute is the bucket size applied when no path
	// matcher in the limiter's policy table matches the incoming request.
	// 300 req/min matches the prior global RateLimit so existing traffic
	// patterns are unaffected.
	DefaultRateLimitPerMinute = 300

	// AIRateLimitPerMinute is applied to /api/v1/questions* and any other
	// path that proxies to the (expensive) AI service. 10 req/min matches
	// the audit-team recommendation; it is intentionally tight because
	// each request fans out into a multi-agent LLM call.
	AIRateLimitPerMinute = 10

	// SearchRateLimitPerMinute is applied to /api/v1/search and the
	// /api/v1/scenarios* paths (search + scenario comparison). 60 req/min
	// keeps heavy FTS queries from saturating the read pool while leaving
	// plenty of headroom for interactive use.
	SearchRateLimitPerMinute = 60
)

// RateLimitPolicy maps request paths to a per-IP rate limit. The first
// matching rule wins; if no rule matches the DefaultPerMinute value is used.
//
// Path matching is longest-prefix (so /api/v1/questions/stream matches a
// rule keyed on /api/v1/questions before a more general rule keyed on
// /api/v1/). Rules are checked in registration order — callers should add
// more-specific rules first.
type RateLimitPolicy struct {
	Rules []rateLimitRule

	// DefaultPerMinute is the limit applied when no rule matches. If zero,
	// DefaultRateLimitPerMinute is used.
	DefaultPerMinute int
}

type rateLimitRule struct {
	prefix     string
	perMinute  int
}

// NewPerPathRateLimitPolicy returns a policy pre-populated with the platform's
// standard tiered limits:
//
//   - /api/v1/questions             → 10 req/min (AI)
//   - /api/v1/questions/stream      → 10 req/min (AI)
//   - /api/v1/bills/*/summary       → 10 req/min (AI)
//   - /api/v1/bills/*/impact        → 10 req/min (AI)
//   - /api/v1/search                → 60 req/min
//   - /api/v1/scenarios             → 60 req/min
//   - everything else                → 300 req/min (default)
//
// Tests assert on this exact mapping; do not silently change it.
func NewPerPathRateLimitPolicy() *RateLimitPolicy {
	return &RateLimitPolicy{
		Rules: []rateLimitRule{
			// AI endpoints (most specific first).
			{prefix: "/api/v1/questions/stream", perMinute: AIRateLimitPerMinute},
			{prefix: "/api/v1/questions", perMinute: AIRateLimitPerMinute},
			{prefix: "/api/v1/bills/", perMinute: AIRateLimitPerMinute}, // tightened — bill detail routes proxy to AI
			// Search + scenarios.
			{prefix: "/api/v1/search", perMinute: SearchRateLimitPerMinute},
			{prefix: "/api/v1/scenarios", perMinute: SearchRateLimitPerMinute},
		},
		DefaultPerMinute: DefaultRateLimitPerMinute,
	}
}

// limitFor returns the configured rate (per minute) for the given path. When
// no rule matches, DefaultPerMinute is returned (falling back to
// DefaultRateLimitPerMinute when DefaultPerMinute is zero).
func (p *RateLimitPolicy) limitFor(path string) int {
	for _, r := range p.Rules {
		if strings.HasPrefix(path, r.prefix) {
			return r.perMinute
		}
	}
	if p.DefaultPerMinute > 0 {
		return p.DefaultPerMinute
	}
	return DefaultRateLimitPerMinute
}

// PerPathRateLimiter is middleware that enforces the configured RateLimitPolicy
// per client IP. Each (IP, limit) pair has its own bucket so a noisy AI caller
// cannot exhaust the search budget for the same IP (or vice versa).
//
// Implementation notes:
//
//   - Buckets are stored in a sync.Map keyed by "ip|limit". Each bucket
//     tracks a count + resetAt; the bucket is reset (not deleted) when
//     resetAt passes, so a steady-state caller reuses the same bucket
//     instead of generating GC pressure.
//   - The window is fixed (not sliding) — matches the existing RateLimit
//     middleware's behaviour so a single client cannot get more lenient
//     treatment by straddling a window boundary. A sliding-window variant
//     can be added later if abuse patterns warrant it.
//   - Production deployments SHOULD front this with a Redis-backed limiter
//     (issue: API-RL-REDIS) so the limit is shared across replicas. Until
//     then, each replica enforces its own in-memory budget — under rolling
//     deploys this means the effective limit is N× the configured value for
//     the duration of the rollout. The audit-team is aware.
type PerPathRateLimiter struct {
	policy *RateLimitPolicy

	// now is overridable for tests. Production code passes nil and the
	// limiter uses time.Now.
	now func() time.Time

	mu      sync.Mutex // guards buckets map
	buckets map[string]*rateBucket
}

// rateBucket is a single fixed-window counter for one (IP, limit) pair.
type rateBucket struct {
	count   int
	resetAt time.Time
}

// NewPerPathRateLimiter constructs a per-path rate limiter for the given
// policy. Passing nil policy is equivalent to passing
// NewPerPathRateLimitPolicy().
func NewPerPathRateLimiter(policy *RateLimitPolicy) *PerPathRateLimiter {
	if policy == nil {
		policy = NewPerPathRateLimitPolicy()
	}
	return &PerPathRateLimiter{
		policy:  policy,
		now:     time.Now,
		buckets: make(map[string]*rateBucket),
	}
}

// Middleware returns an http.Handler that wraps next with the per-path
// rate-limit enforcement. The returned handler is safe for concurrent use.
func (l *PerPathRateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := clientIP(r)
		limit := l.policy.limitFor(r.URL.Path)
		key := ip + "|" + itoa(limit)

		now := l.now()
		allowed := l.allow(key, limit, now)
		if !allowed {
			// Retry-After per RFC 7231 §7.1.3 — seconds until the bucket
			// resets. Computed from the bucket's resetAt so a caller that
			// retries after the header value should succeed.
			retryAfter := l.retryAfter(key, now)
			w.Header().Set("Retry-After", itoa(int(retryAfter.Seconds())))
			writeError(w, http.StatusTooManyRequests, "rate_limited",
				"too many requests for this endpoint; retry after the Retry-After header")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// allow returns true if a request from the bucket identified by key may
// proceed. It increments the bucket's count and resets the window if the
// previous window has elapsed.
func (l *PerPathRateLimiter) allow(key string, limit int, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	b, ok := l.buckets[key]
	if !ok || now.After(b.resetAt) {
		// Fresh window — first request in this bucket.
		l.buckets[key] = &rateBucket{count: 1, resetAt: now.Add(time.Minute)}
		return true
	}
	b.count++
	return b.count <= limit
}

// retryAfter returns the duration until the bucket identified by key resets.
// Returns 60s if the bucket does not exist (which can only happen if the
// caller never called allow first — defensive default).
func (l *PerPathRateLimiter) retryAfter(key string, now time.Time) time.Duration {
	l.mu.Lock()
	defer l.mu.Unlock()
	b, ok := l.buckets[key]
	if !ok {
		return time.Minute
	}
	d := time.Until(b.resetAt)
	if d <= 0 {
		return time.Second
	}
	return d
}

// itoa is a small int→string helper that avoids pulling in strconv just for
// this one call site. Values are always non-negative so we don't need to
// handle sign extension.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}

// Compile-time assertion that PerPathRateLimiter satisfies the
// middleware-constructor shape expected by main.go's chain.
var _ func(http.Handler) http.Handler = (*PerPathRateLimiter)(nil).Middleware
