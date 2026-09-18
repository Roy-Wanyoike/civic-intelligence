// Order Paper adapter — discovers and fetches daily House agendas.
package parliament

import (
	"context"
	"time"
)

// OrderPaperCandidate is a discovered Order Paper.
type OrderPaperCandidate struct {
	URL         string
	House       string
	SittingDate time.Time
	SourceID    string
}

// DiscoverOrderPapers discovers Order Papers from parliament.go.ke.
func (a *Adapter) DiscoverOrderPapers(ctx context.Context) ([]OrderPaperCandidate, error) {
	// TODO: crawl parliament.go.ke/order-papers
	return []OrderPaperCandidate{}, nil
}

// FetchOrderPaper downloads an Order Paper.
func (a *Adapter) FetchOrderPaper(ctx context.Context, url string) (string, error) {
	return a.fetchURL(ctx, url, "order_paper")
}

// OrderPaperItem is a single agenda item in an Order Paper.
type OrderPaperItem struct {
	Title       string
	Description string
	BillRef     string // optional Bill identifier
}

// ParseOrderPaper extracts agenda items from an Order Paper.
func (a *Adapter) ParseOrderPaper(html, sourceURL string) ([]OrderPaperItem, error) {
	return []OrderPaperItem{}, nil
}
