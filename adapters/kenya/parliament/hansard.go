package parliament

import (
	"context"
	"fmt"
	"strings"
)

// DiscoverHansard fetches both houses' Hansard listings and aggregates the
// results into a single slice.
//
// Each Hansard entry corresponds to a single sitting transcript. The full
// transcript itself is not fetched here — that's deferred to the documents
// service, which performs OCR / PDF parsing on the linked URL.
func (p *ParliamentAdapter) DiscoverHansard(ctx context.Context) ([]HansardEntry, error) {
	na, err := p.discoverHansardFor(ctx, SourceNationalAssemblyHansard)
	if err != nil {
		return nil, err
	}
	sen, err := p.discoverHansardFor(ctx, SourceSenateHansard)
	if err != nil {
		return nil, err
	}
	return append(na, sen...), nil
}

func (p *ParliamentAdapter) discoverHansardFor(ctx context.Context, src SourceURL) ([]HansardEntry, error) {
	body, err := p.fetchSource(ctx, src, "hansard")
	if err != nil {
		return nil, err
	}
	if body == nil {
		return nil, nil
	}
	entries, err := ParseHansardHTML(strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrParse, err)
	}
	// Fill in House where the URL parser couldn't determine it.
	for i := range entries {
		if entries[i].House == "" {
			if src == SourceSenateHansard {
				entries[i].House = "Senate"
			} else {
				entries[i].House = "National Assembly"
			}
		}
	}
	return entries, nil
}

// HansardHouseCode maps a House string to the adapter's canonical house code
// ("NA" or "SEN"). Returns "" if the string is not recognised.
func HansardHouseCode(house string) string {
	switch strings.ToLower(strings.TrimSpace(house)) {
	case "senate":
		return "SEN"
	case "national assembly", "nationalassembly":
		return "NA"
	}
	return ""
}
