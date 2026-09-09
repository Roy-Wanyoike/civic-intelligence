// Gazette adapter — extends the existing gazette package with discovery.
package gazette

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Roy-Wanyoike/civic-intelligence/packages/contracts"
)

// DiscoverNotices discovers Kenya Gazette notices.
func (a *Adapter) DiscoverNotices(ctx context.Context) ([]GazetteEntry, error) {
	// TODO: crawl the Kenya Gazette archive
	// The Gazette is published weekly — crawl the latest issue
	return []GazetteEntry{}, nil
}

// FetchNotice downloads a single gazette notice.
func (a *Adapter) FetchNotice(ctx context.Context, url string) (*contracts.RawDocument, error) {
	return a.Fetch(ctx, contracts.SourceItem{URL: url, DocumentType: "gazette_notice"})
}

// GazetteNotice is a parsed gazette notice with full metadata.
type GazetteNotice struct {
	Volume          string
	NoticeNo        string
	Title           string
	Issuer          string
	PublicationDate time.Time
	URL             string
}

// ParseNotice parses a gazette notice into structured data.
func (a *Adapter) ParseNotice(html, sourceURL string) (*GazetteNotice, error) {
	// Reuse the existing ParseNoticesHTML function
	entries, err := ParseNoticesHTML(strings.NewReader(html))
	if err != nil || len(entries) == 0 {
		return nil, fmt.Errorf("parse notice: %w", err)
	}
	e := entries[0]
	return &GazetteNotice{
		Volume:          e.Volume,
		NoticeNo:        e.NoticeNo,
		Title:           e.Title,
		Issuer:          e.Issuer,
		PublicationDate: e.PublishedAt,
		URL:             sourceURL,
	}, nil
}
