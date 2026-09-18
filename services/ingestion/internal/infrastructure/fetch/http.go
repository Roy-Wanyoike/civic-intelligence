// Package fetch provides the HTTPFetcher implementation used by the
// ingestion worker to download raw documents. It is in infrastructure
// because domain code never imports net/http.
package fetch

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// HTTPFetcher is the production HTTP fetcher.
type HTTPFetcher struct {
	client  *http.Client
	headers map[string]string
}

// New constructs an HTTPFetcher with sensible defaults: 30s timeout, a
// respectful User-Agent, and an opt-out compression handling.
func New(userAgent string) *HTTPFetcher {
	if userAgent == "" {
		userAgent = "CivicIntelligenceBot/1.0 (+https://civic-intelligence.example.org)"
	}
	return &HTTPFetcher{
		client: &http.Client{
			Timeout: 30 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 5 {
					return fmt.Errorf("too many redirects")
				}
				return nil
			},
		},
		headers: map[string]string{
			"User-Agent": userAgent,
			"Accept":     "*/*",
		},
	}
}

// Fetch downloads the URL and returns the body bytes and detected MIME type.
// The body is fully buffered in memory; for large gazettes a streaming
// version should be used.
func (f *HTTPFetcher) Fetch(ctx context.Context, url string) ([]byte, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, "", fmt.Errorf("build request: %w", err)
	}
	for k, v := range f.headers {
		req.Header.Set(k, v)
	}
	resp, err := f.client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("http get: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, "", fmt.Errorf("http status %d for %s", resp.StatusCode, url)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 50*1024*1024)) // 50MB cap
	if err != nil {
		return nil, "", fmt.Errorf("read body: %w", err)
	}
	mime := resp.Header.Get("Content-Type")
	if i := strings.Index(mime, ";"); i > 0 {
		mime = strings.TrimSpace(mime[:i])
	}
	if mime == "" {
		mime = "application/octet-stream"
	}
	return body, mime, nil
}
