// Committee adapter — discovers and fetches committee documents.
package parliament

import (
	"context"
	"time"
)

// CommitteeCandidate is a discovered committee document.
type CommitteeCandidate struct {
	URL           string
	CommitteeName string
	House         string
	ReportType    string
	PublishedDate time.Time
	SourceID      string
}

// DiscoverCommittees discovers committee reports from parliament.go.ke.
func (a *Adapter) DiscoverCommittees(ctx context.Context) ([]CommitteeCandidate, error) {
	// TODO: crawl parliament.go.ke/committees
	return []CommitteeCandidate{}, nil
}

// FetchCommitteeReport downloads a committee report.
func (a *Adapter) FetchCommitteeReport(ctx context.Context, url string) (string, error) {
	return a.fetchURL(ctx, url, "committee_report")
}

// CommitteeReport is a parsed committee report.
type CommitteeReport struct {
	Title       string
	Committee   string
	House       string
	ReportType  string
	PublishedAt time.Time
	BillRefs    []string
}

// ParseCommitteeReport extracts metadata from a committee report.
func (a *Adapter) ParseCommitteeReport(html, sourceURL string) (*CommitteeReport, error) {
	return nil, nil
}
