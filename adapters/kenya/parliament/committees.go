package parliament

import (
	"context"
	"fmt"
	"strings"
)

// DiscoverCommittees fetches both houses' committee listings and aggregates
// the results.
//
// Each CommitteeEntry corresponds to a single committee page. The committee
// page itself links to reports and inquiries; the documents service is
// responsible for fetching and parsing those reports.
func (p *ParliamentAdapter) DiscoverCommittees(ctx context.Context) ([]CommitteeEntry, error) {
	na, err := p.discoverCommitteesFor(ctx, SourceNationalAssemblyCommittees)
	if err != nil {
		return nil, err
	}
	sen, err := p.discoverCommitteesFor(ctx, SourceSenateCommittees)
	if err != nil {
		return nil, err
	}
	return append(na, sen...), nil
}

func (p *ParliamentAdapter) discoverCommitteesFor(ctx context.Context, src SourceURL) ([]CommitteeEntry, error) {
	body, err := p.fetchSource(ctx, src, "committees")
	if err != nil {
		return nil, err
	}
	if body == nil {
		return nil, nil
	}
	entries, err := ParseCommitteesHTML(strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrParse, err)
	}
	for i := range entries {
		if entries[i].House == "" {
			if src == SourceSenateCommittees {
				entries[i].House = "Senate"
			} else {
				entries[i].House = "National Assembly"
			}
		}
	}
	return entries, nil
}

// CommitteeHouseCode maps a House string to the adapter's canonical house
// code ("NA" or "SEN"). Returns "" if the string is not recognised.
func CommitteeHouseCode(house string) string {
	switch strings.ToLower(strings.TrimSpace(house)) {
	case "senate":
		return "SEN"
	case "national assembly", "nationalassembly":
		return "NA"
	}
	return ""
}
