// Hansard adapter — discovers and fetches parliamentary debate transcripts.
package parliament

import (
	"context"
	"fmt"
	"time"

	"github.com/Roy-Wanyoike/civic-intelligence/packages/contracts"
)

// HansardCandidate is a discovered Hansard document.
type HansardCandidate struct {
	URL          string
	Title        string
	House        string
	SittingDate  time.Time
	SourceID     string
}

// DiscoverHansard discovers Hansard documents from parliament.go.ke/hansard.
func (a *Adapter) DiscoverHansard(ctx context.Context) ([]HansardCandidate, error) {
	// TODO: crawl parliament.go.ke/hansard
	// The Hansard page lists debate transcripts by sitting date.
	return []HansardCandidate{}, nil
}

// FetchHansard downloads a Hansard document.
func (a *Adapter) FetchHansard(ctx context.Context, url string) (string, error) {
	return a.fetchURL(ctx, url, "hansard")
}

// ParseHansard extracts speeches from a Hansard document.
func (a *Adapter) ParseHansard(html, sourceURL string) ([]HansardSpeech, error) {
	// TODO: implement HTML parsing for Hansard transcripts
	// Extract: speaker, role, topic, speech text, sitting date
	return []HansardSpeech{}, nil
}

// HansardSpeech is a single speech from a Hansard document.
type HansardSpeech struct {
	Speaker    string
	Role       string
	Topic      string
	Text       string
	SittingDate time.Time
}

func (a *Adapter) fetchURL(ctx context.Context, url, docType string) (string, error) {
	// Reuse the existing Fetch method's HTTP logic
	doc, err := a.Fetch(ctx, contracts.SourceItem{URL: url, DocumentType: docType})
	if err != nil {
		return "", fmt.Errorf("fetch %s: %w", url, err)
	}
	return string(doc.Bytes), nil
}
