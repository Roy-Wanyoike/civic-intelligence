// Votes & Proceedings adapter — discovers and fetches official vote records.
package parliament

import (
	"context"
	"time"
)

// VotesCandidate is a discovered Votes & Proceedings document.
type VotesCandidate struct {
	URL         string
	House       string
	SittingDate time.Time
	SourceID    string
}

// DiscoverVotesProceedings discovers V&P documents from parliament.go.ke.
func (a *Adapter) DiscoverVotesProceedings(ctx context.Context) ([]VotesCandidate, error) {
	// TODO: crawl parliament.go.ke/votes-and-proceedings
	return []VotesCandidate{}, nil
}

// FetchVotesProceedings downloads a V&P document.
func (a *Adapter) FetchVotesProceedings(ctx context.Context, url string) (string, error) {
	return a.fetchURL(ctx, url, "votes_proceedings")
}

// VoteRecord is a single vote in a V&P document.
type VoteRecord struct {
	PersonID  string
	PersonName string
	Vote      string // aye, nay, abstain, absent
	BillRef   string
}

// ParseVotesProceedings extracts vote records from a V&P document.
func (a *Adapter) ParseVotesProceedings(html, sourceURL string) ([]VoteRecord, error) {
	return []VoteRecord{}, nil
}
