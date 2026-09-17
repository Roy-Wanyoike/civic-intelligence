// Package parliament adapts the Parliament of Ghana source (parliament.gh).
package parliament

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/Roy-Wanyoike/civic-intelligence/packages/contracts"
)

// HTTPClient is the minimal HTTP client interface used by the adapter.
// Real callers inject *http.Client; tests inject a stub.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// Adapter adapts the Parliament of Ghana source.
type Adapter struct {
	client    HTTPClient
	userAgent string
}

// NewAdapter constructs a Parliament of Ghana adapter.
func NewAdapter(client HTTPClient, userAgent string) *Adapter {
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	return &Adapter{client: client, userAgent: userAgent}
}

// Discover returns the items currently available from the Parliament of Ghana source.
// TODO: crawl parliament.gh/bills once the upstream bills listing is stable.
func (a *Adapter) Discover(ctx context.Context) ([]contracts.SourceItem, error) {
	return []contracts.SourceItem{}, nil
}

// Fetch downloads a single item's raw bytes.
func (a *Adapter) Fetch(ctx context.Context, item contracts.SourceItem) (*contracts.RawDocument, error) {
	if item.URL == "" {
		return nil, fmt.Errorf("ghana.Fetch: empty URL")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, item.URL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", a.userAgent)
	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ghana.Fetch: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ghana.Fetch: HTTP %d", resp.StatusCode)
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

// Parse converts a raw document into normalized ExtractedRecords.
// TODO: implement country-specific HTML/PDF parsing for parliament.gh.
func (a *Adapter) Parse(ctx context.Context, doc contracts.RawDocument) ([]contracts.ExtractedRecord, error) {
	return []contracts.ExtractedRecord{}, nil
}

// io_ReadAll reads all bytes from r without importing io/io.ReadAll to keep
// the package dependency-free for tests that inject a stub reader.
func io_ReadAll(r interface{ Read([]byte) (int, error) }) ([]byte, error) {
	buf := make([]byte, 0, 4096)
	tmp := make([]byte, 4096)
	for {
		n, err := r.Read(tmp)
		if n > 0 {
			buf = append(buf, tmp[:n]...)
		}
		if err != nil {
			if err.Error() == "EOF" {
				return buf, nil
			}
			return buf, err
		}
	}
}
