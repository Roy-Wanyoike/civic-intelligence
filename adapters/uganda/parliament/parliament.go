// Package parliament adapts the Parliament of Uganda source (parliament.go.ug).
package parliament

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

func (a *Adapter) Discover(ctx context.Context) ([]contracts.SourceItem, error) {
	// TODO: crawl parliament.go.ug/bills
	return []contracts.SourceItem{}, nil
}

func (a *Adapter) Fetch(ctx context.Context, item contracts.SourceItem) (*contracts.RawDocument, error) {
	if item.URL == "" {
		return nil, fmt.Errorf("uganda.Fetch: empty URL")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, item.URL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", a.userAgent)
	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("uganda.Fetch: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("uganda.Fetch: HTTP %d", resp.StatusCode)
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
	return []contracts.ExtractedRecord{}, nil
}

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
