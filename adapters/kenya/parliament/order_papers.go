package parliament

import (
	"context"
	"fmt"
	"strings"
)

// DiscoverOrderPapers fetches both houses' Order Paper listings and aggregates
// the results.
//
// Each Order Paper entry corresponds to a single sitting's agenda document.
// The full Order Paper is usually a PDF; the documents service performs
// page-level parsing later.
func (p *ParliamentAdapter) DiscoverOrderPapers(ctx context.Context) ([]OrderPaperEntry, error) {
	na, err := p.discoverOrderPapersFor(ctx, SourceNationalAssemblyOrderPaper)
	if err != nil {
		return nil, err
	}
	sen, err := p.discoverOrderPapersFor(ctx, SourceSenateOrderPaper)
	if err != nil {
		return nil, err
	}
	return append(na, sen...), nil
}

func (p *ParliamentAdapter) discoverOrderPapersFor(ctx context.Context, src SourceURL) ([]OrderPaperEntry, error) {
	body, err := p.fetchSource(ctx, src, "order_paper")
	if err != nil {
		return nil, err
	}
	if body == nil {
		return nil, nil
	}
	entries, err := ParseOrderPaperHTML(strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrParse, err)
	}
	for i := range entries {
		if entries[i].House == "" {
			if src == SourceSenateOrderPaper {
				entries[i].House = "Senate"
			} else {
				entries[i].House = "National Assembly"
			}
		}
	}
	return entries, nil
}
