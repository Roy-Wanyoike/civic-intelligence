// Package kenya_law adapts the Kenya Law Reports source (kenyalaw.org) — Acts
// of Parliament, subsidiary legislation (regulations), and legal notices.
package kenya_law

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/Roy-Wanyoike/civic-intelligence/packages/contracts"
)

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type Adapter struct {
	client    HTTPClient
	userAgent string
}

func NewAdapter(client HTTPClient, userAgent string) *Adapter {
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	return &Adapter{client: client, userAgent: userAgent}
}

// Discover finds current Acts of Parliament, regulations, and legal notices
// from kenyalaw.org / the Kenya Gazette.
func (a *Adapter) Discover(ctx context.Context) ([]contracts.SourceItem, error) {
	// TODO(issue #CI-AD-005): crawl kenyalaw.org/kl/index.php?id=398 (Acts)
	// and /kl/index.php?id=589 (Kenya Gazette).
	return []contracts.SourceItem{}, nil
}

func (a *Adapter) Fetch(ctx context.Context, item contracts.SourceItem) (*contracts.RawDocument, error) {
	if item.URL == "" {
		return nil, fmt.Errorf("kenya_law.Fetch: empty URL")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, item.URL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", a.userAgent)
	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("kenya_law.Fetch: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("kenya_law.Fetch: HTTP %d for %s", resp.StatusCode, item.URL)
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
	}, nil
}

func (a *Adapter) Parse(ctx context.Context, doc contracts.RawDocument) ([]contracts.ExtractedRecord, error) {
	// TODO(issue #CI-AD-006): parse Acts + regulations.
	return []contracts.ExtractedRecord{}, nil
}
