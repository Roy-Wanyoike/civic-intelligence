// Package gazette adapts the Kenya Gazette source — official gazette notices.
package gazette

import (
        "context"
        "fmt"
        "net/http"
        "time"

        "github.com/Roy-Wanyoike/civic-intelligence/packages/contracts"
)

// GazetteEntry is the parsed representation of a single Kenya Gazette notice.
// The parser (parser.go) produces these from the Gazette index page or from
// the plain text of a single notice.
type GazetteEntry struct {
        Volume      string
        NoticeNo    string
        Title       string
        Issuer      string
        PublishedAt time.Time
        URL         string
}

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
        // TODO(issue #CI-AD-007): crawl the Kenya Gazette archive.
        return []contracts.SourceItem{}, nil
}

func (a *Adapter) Fetch(ctx context.Context, item contracts.SourceItem) (*contracts.RawDocument, error) {
        if item.URL == "" {
                return nil, fmt.Errorf("gazette.Fetch: empty URL")
        }
        req, err := http.NewRequestWithContext(ctx, http.MethodGet, item.URL, nil)
        if err != nil {
                return nil, err
        }
        req.Header.Set("User-Agent", a.userAgent)
        resp, err := a.client.Do(req)
        if err != nil {
                return nil, fmt.Errorf("gazette.Fetch: %w", err)
        }
        defer resp.Body.Close()
        if resp.StatusCode != http.StatusOK {
                return nil, fmt.Errorf("gazette.Fetch: HTTP %d for %s", resp.StatusCode, item.URL)
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
        // TODO(issue #CI-AD-008): parse gazette notices (calls the documents
        // service OCR for scanned PDFs).
        return []contracts.ExtractedRecord{}, nil
}
