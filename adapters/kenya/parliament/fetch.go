// Package parliament — fetchURL helper.
//
// This is the single low-level network call used by Fetch, DiscoverBills,
// FetchBillTracker, FetchHansard, FetchOrderPaper, FetchVotesProceedings,
// and FetchCommitteeReport. Concentrating it here means:
//
//   - the polite PoliteClient rate-limiter is applied uniformly
//   - SSRF defence (User-Agent, Accept header, context-bound timeouts) lives
//     in one place
//   - the recursion-prevention comment for issue #201 is co-located with the
//     implementation so future maintainers don't reintroduce it
package parliament

import (
	"context"
	"fmt"
	"io"
	"net/http"
)

// fetchURL performs the actual HTTP GET against url and returns the response
// body as a string. It is the single low-level network call used by Fetch,
// DiscoverBills, FetchBillTracker and FetchHansard. The docType parameter is
// retained for future per-type request tuning (e.g. Accept headers) but does
// not alter the request today.
//
// NOTE: this must NOT delegate back to Fetch — doing so causes infinite
// recursion because Fetch itself calls fetchURL (see issue #201).
func (a *Adapter) fetchURL(ctx context.Context, url, docType string) (string, error) {
	_ = docType // reserved for future per-type request tuning
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", a.userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/pdf")

	resp, err := a.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("fetch %s: HTTP %d", url, resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", url, err)
	}
	return string(body), nil
}
