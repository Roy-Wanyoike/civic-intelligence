package parliament

import (
	"context"
	"fmt"
	"strings"
)

// DiscoverVotesProceedings fetches both houses' Votes and Proceedings listings
// and aggregates the results.
//
// Votes and Proceedings are the authoritative record of decisions taken in a
// sitting, including divisions and votes. Each entry corresponds to a single
// sitting's record; the documents service downloads the linked PDF for
// page-level parsing.
func (p *ParliamentAdapter) DiscoverVotesProceedings(ctx context.Context) ([]VotesProceedingsEntry, error) {
	na, err := p.discoverVotesProceedingsFor(ctx, SourceNationalAssemblyVotesProceedings)
	if err != nil {
		return nil, err
	}
	sen, err := p.discoverVotesProceedingsFor(ctx, SourceSenateVotesProceedings)
	if err != nil {
		return nil, err
	}
	return append(na, sen...), nil
}

func (p *ParliamentAdapter) discoverVotesProceedingsFor(ctx context.Context, src SourceURL) ([]VotesProceedingsEntry, error) {
	body, err := p.fetchSource(ctx, src, "votes_proceedings")
	if err != nil {
		return nil, err
	}
	if body == nil {
		return nil, nil
	}
	entries, err := ParseVotesProceedingsHTML(strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrParse, err)
	}
	for i := range entries {
		if entries[i].House == "" {
			if src == SourceSenateVotesProceedings {
				entries[i].House = "Senate"
			} else {
				entries[i].House = "National Assembly"
			}
		}
	}
	return entries, nil
}
