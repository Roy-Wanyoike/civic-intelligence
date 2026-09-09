// Package parliament adapts the Parliament of Kenya source (parliament.go.ke,
// nationalassembly.go.ke, senate.go.ke). It implements discovery + fetch + parse
// for Bills, Hansard, Order Papers, Votes & Proceedings, and committee documents.
package parliament

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/Roy-Wanyoike/civic-intelligence/packages/contracts"
)

// HTTPClient is the interface parliament adapters use to fetch remote content.
// Production uses a polite, rate-limited client (see PoliteClient). Tests use
// an in-memory implementation. This indirection is critical for SSRF
// protection and for testing offline.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// Adapter fetches from the Parliament of Kenya's websites.
type Adapter struct {
	client    HTTPClient
	userAgent string
}

func NewAdapter(client HTTPClient, userAgent string) *Adapter {
	if client == nil {
		client = &PoliteClient{Client: &http.Client{Timeout: 30 * time.Second}, ratePerSec: 1.0}
	}
	return &Adapter{client: client, userAgent: userAgent}
}

// Discover finds current Bills, Hansard, Order Papers, and committee documents
// listed on the Parliament of Kenya websites.
func (a *Adapter) Discover(ctx context.Context) ([]contracts.SourceItem, error) {
	// TODO(issue #CI-AD-002): crawl parliament.go.ke/bills, /hansard,
	// /order-papers, /votes-and-proceedings, and committee pages.
	// Respect robots.txt; rate-limit to 1 req/sec/host.
	return []contracts.SourceItem{}, nil
}

// Fetch downloads the raw bytes of a single SourceItem.
func (a *Adapter) Fetch(ctx context.Context, item contracts.SourceItem) (*contracts.RawDocument, error) {
	if item.URL == "" {
		return nil, fmt.Errorf("parliament.Fetch: empty URL")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, item.URL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", a.userAgent)
	req.Header.Set("Accept", "text/html,application/pdf,application/vnd.openxmlformats-officedocument.wordprocessingml.document")

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("parliament.Fetch: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("parliament.Fetch: HTTP %d for %s", resp.StatusCode, item.URL)
	}
	body, err := io_ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return &contracts.RawDocument{
		URL:         item.URL,
		Bytes:       body,
		MimeType:    resp.Header.Get("Content-Type"),
		RetrievedAt: time.Now().UTC(),
		SourceID:    "", // populated by the ingestion service
	}, nil
}

// Parse converts a raw document into normalized ExtractedRecords. The actual
// parsing logic depends on the document type and MIME type. HTML uses
// golang.org/x/net/html; PDF uses an external parser (e.g., unidoc or pdfcpu).
func (a *Adapter) Parse(ctx context.Context, doc contracts.RawDocument) ([]contracts.ExtractedRecord, error) {
	// TODO(issue #CI-AD-003): implement HTML + PDF parsing.
	return []contracts.ExtractedRecord{}, nil
}
