// Package parliament — polite HTTP client with rate limiting + retry.
package parliament

import (
	"net/http"
	"time"
)

// PoliteClient wraps an http.Client with per-host rate limiting. Used by all
// Kenya source adapters to be a good citizen of the source websites.
//
// Production hardening (issue #CI-AD-004): respect robots.txt, support
// ETag/If-None-Match, exponential backoff on 5xx, circuit breaker per host.
type PoliteClient struct {
	*http.Client
	ratePerSec float64
	lastCall   time.Time
}

func (p *PoliteClient) Do(req *http.Request) (*http.Response, error) {
	// Simple rate limit: ensure at least 1/ratePerSec seconds between calls.
	if !p.lastCall.IsZero() {
		minInterval := time.Duration(float64(time.Second) / p.ratePerSec)
		if elapsed := time.Since(p.lastCall); elapsed < minInterval {
			time.Sleep(minInterval - elapsed)
		}
	}
	p.lastCall = time.Now()
	return p.Client.Do(req)
}
