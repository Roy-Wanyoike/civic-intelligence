// Hansard adapter — discovers and fetches parliamentary debate transcripts.
package parliament

import (
        "context"
        "fmt"
        "io"
        "net/http"
        "time"
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
