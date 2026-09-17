// Package parliament adapts the Parliament of Tanzania source (parliament.go.tz).
//
// This is a skeleton adapter: Discover/Parse return empty slices (TODO) while
// Fetch is fully implemented so the contract surface compiles and the adapter
// can be wired into the ingestion pipeline. The full HTML/PDF parser for
// parliament.go.tz Bills, Order Papers, and Hansard will be added in a
// subsequent change.
package parliament

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/Roy-Wanyoike/civic-intelligence/packages/contracts"
)

// HTTPClient is the minimal HTTP interface this adapter depends on. The
// real *http.Client satisfies it; tests may inject a fake.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// Adapter adapts parliament.go.tz to the platform's contracts.LegislativeSourceAdapter
// surface (Discover / Fetch / Parse). The adapter is stateless aside from its
// HTTP client and user-agent string.
type Adapter struct {
	client    HTTPClient
	userAgent string
}

// NewAdapter constructs a parliament.go.tz adapter. If client is nil a default
// *http.Client with a 30-second timeout is used.
func NewAdapter(client HTTPClient, userAgent string) *Adapter {
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	if userAgent == "" {
		userAgent = "CivicIntelligence/0.1 (+https://github.com/Roy-Wanyoike/civic-intelligence)"
	}
	return &Adapter{client: client, userAgent: userAgent}
}

// Discover returns the SourceItems currently available from parliament.go.tz.
// TODO(issue #154): crawl the Bills index on parliament.go.tz and return one
// SourceItem per Bill. For now returns an empty slice so the adapter can be
// registered with the ingestion service.
func (a *Adapter) Discover(ctx context.Context) ([]contracts.SourceItem, error) {
	_ = ctx
	return []contracts.SourceItem{}, nil
}

// Fetch downloads the raw bytes for a single SourceItem from parliament.go.tz.
// It enforces a non-empty URL, sets the configured User-Agent header, and
// returns an error on any non-200 response.
func (a *Adapter) Fetch(ctx context.Context, item contracts.SourceItem) (*contracts.RawDocument, error) {
	if item.URL == "" {
		return nil, fmt.Errorf("tanzania.parliament.Fetch: empty URL")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, item.URL, nil)
	if err != nil {
		return nil, fmt.Errorf("tanzania.parliament.Fetch: %w", err)
	}
	req.Header.Set("User-Agent", a.userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/pdf;q=0.9,*/*;q=0.8")

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("tanzania.parliament.Fetch: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("tanzania.parliament.Fetch: HTTP %d for %s", resp.StatusCode, item.URL)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("tanzania.parliament.Fetch: read body: %w", err)
	}
	return &contracts.RawDocument{
		URL:         item.URL,
		Bytes:       body,
		MimeType:    resp.Header.Get("Content-Type"),
		RetrievedAt: time.Now().UTC(),
	}, nil
}

// Parse converts a raw parliament.go.tz document into normalized
// ExtractedRecords. TODO(issue #154): implement the HTML/PDF parsers. For now
// returns an empty slice so the adapter satisfies the contract.
func (a *Adapter) Parse(ctx context.Context, doc contracts.RawDocument) ([]contracts.ExtractedRecord, error) {
	_ = ctx
	_ = doc
	return []contracts.ExtractedRecord{}, nil
}
