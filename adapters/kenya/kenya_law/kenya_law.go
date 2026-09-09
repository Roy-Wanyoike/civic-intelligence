// Package kenya_law adapts the Kenya Law Reports source (new.kenyalaw.org/bills/).
// It implements discovery + fetch + parse for Bills using the Akoma Ntoso
// legal document standard.
package kenya_law

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/Roy-Wanyoike/civic-intelligence/packages/contracts"
)

// HTTPClient is the interface for fetching remote content.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// Adapter fetches Bills from new.kenyalaw.org/bills/.
type Adapter struct {
	client    HTTPClient
	userAgent string
	baseURL   string
}

// NewAdapter creates a Kenya Law adapter.
func NewAdapter(client HTTPClient, userAgent string) *Adapter {
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	if userAgent == "" {
		userAgent = "CivicIntelligence/0.1 (+https://github.com/Roy-Wanyoike/civic-intelligence)"
	}
	return &Adapter{
		client:    client,
		userAgent: userAgent,
		baseURL:   "https://new.kenyalaw.org",
	}
}

// CountryCode returns "KE".
func (a *Adapter) CountryCode() string { return "KE" }

// Supports reports whether this adapter handles the given URL.
func (a *Adapter) Supports(url string) bool {
	return strings.Contains(url, "kenyalaw.org")
}

// DiscoverBills crawls the Kenya Law bills listing page and returns Bill
// candidates. Each candidate carries the URL, title, house, and publication
// date extracted from the Akoma Ntoso URL structure.
//
// The Kenya Law bills page lists Bills with links in the format:
//   /akn/ke/bill/{house}/{date}/{slug}/eng@{date}
//
// where {house} is "na" (National Assembly) or "senate", and {date} is
// the publication date in YYYY-MM-DD format.
func (a *Adapter) DiscoverBills(ctx context.Context) ([]BillCandidate, error) {
	url := a.baseURL + "/bills/"
	body, err := a.fetchURL(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("kenya_law.DiscoverBills: %w", err)
	}
	return ParseBillsListing(body), nil
}

// BillCandidate is a discovered Bill before canonicalization.
type BillCandidate struct {
	URL           string    // full URL on kenyalaw.org
	Slug          string    // URL slug (e.g., "the-housing-bill-2024")
	Title         string    // human-readable title
	House         string    // "National Assembly" or "Senate"
	PublicationDate time.Time // parsed from the URL date component
	SourceID      string    // platform-internal ID (e.g., "ke-bill-2026-09-07-local-authorities...")
}

// FetchBill downloads the bill detail page HTML.
func (a *Adapter) FetchBill(ctx context.Context, url string) (string, error) {
	return a.fetchURL(ctx, url)
}

// GetOfficialSources returns the source definitions for the Kenya Law adapter.
func (a *Adapter) GetOfficialSources() []SourceDefinition {
	return []SourceDefinition{
		{
			ID:           "ke-kenyalaw-bills",
			Country:      "KE",
			Institution:  "Kenya Law Reports",
			Authority:    "primary",
			URL:          a.baseURL + "/bills/",
			Adapter:      "kenya.kenya_law",
			DocumentTypes: []string{"bill"},
			CrawlFrequency: 6 * time.Hour,
		},
	}
}

// SourceDefinition describes an official source.
type SourceDefinition struct {
	ID             string
	Country        string
	Institution    string
	Authority      string
	URL            string
	Adapter        string
	DocumentTypes  []string
	CrawlFrequency time.Duration
}

// fetchURL downloads a URL and returns the body as a string.
func (a *Adapter) fetchURL(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", a.userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml")

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

// Discover implements the contracts.LegislativeSourceAdapter interface.
// It discovers all document types (currently just Bills).
func (a *Adapter) Discover(ctx context.Context) ([]contracts.SourceItem, error) {
	bills, err := a.DiscoverBills(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]contracts.SourceItem, len(bills))
	for i, b := range bills {
		items[i] = contracts.SourceItem{
			URL:            b.URL,
			Title:          b.Title,
			DocumentType:   "bill",
			SourceType:     contracts.SourceItemBill,
			CountryCode:    "KE",
			PublishedAt:    b.PublicationDate,
			DiscoveredAt:   time.Now().UTC(),
			SourceID:       b.SourceID,
			Metadata: map[string]string{
				"house": b.House,
				"slug":  b.Slug,
			},
		}
	}
	return items, nil
}

// Fetch implements the contracts.LegislativeSourceAdapter interface.
func (a *Adapter) Fetch(ctx context.Context, item contracts.SourceItem) (*contracts.RawDocument, error) {
	if item.URL == "" {
		return nil, fmt.Errorf("kenya_law.Fetch: empty URL")
	}
	body, err := a.fetchURL(ctx, item.URL)
	if err != nil {
		return nil, err
	}
	return &contracts.RawDocument{
		URL:         item.URL,
		Bytes:       []byte(body),
		MimeType:    "text/html",
		RetrievedAt: time.Now().UTC(),
	}, nil
}

// Parse implements the contracts.LegislativeSourceAdapter interface.
func (a *Adapter) Parse(ctx context.Context, doc contracts.RawDocument) ([]contracts.ExtractedRecord, error) {
	return ParseBillDetail(string(doc.Bytes), doc.URL)
}

// NormalizeSourceItem implements the contracts.LegislativeSourceAdapter interface.
func (a *Adapter) NormalizeSourceItem(raw map[string]any) (contracts.SourceItem, error) {
	return contracts.SourceItem{
		URL:          getString(raw, "url"),
		Title:        getString(raw, "title"),
		DocumentType: "bill",
		SourceType:   contracts.SourceItemBill,
		CountryCode:  "KE",
	}, nil
}

// GetLegislativeStructure implements the contracts.LegislativeSourceAdapter interface.
func (a *Adapter) GetLegislativeStructure(ctx context.Context) (*contracts.LegislativeStructure, error) {
	s := contracts.LegislativeStructure{Country: "KE", CountryCode: "KE", CountryName: "Kenya"}
	return &s, nil
}

// GetStages implements the contracts.LegislativeSourceAdapter interface.
func (a *Adapter) GetStages(ctx context.Context) ([]contracts.StageDefinition, error) {
	return nil, nil // Stages come from the parliament adapter, not Kenya Law
}

// GetTerminology implements the contracts.LegislativeSourceAdapter interface.
func (a *Adapter) GetTerminology(ctx context.Context) ([]contracts.TermDefinition, error) {
	return nil, nil
}

func getString(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

// billLinkRe matches Akoma Ntoso bill URLs on kenyalaw.org.
// Example: /akn/ke/bill/na/2026-09-07/the-local-authorities-provident-fund-amendment-bill-2026/eng@2026-09-07
var billLinkRe = regexp.MustCompile(`/akn/ke/bill/(na|senate)/(\d{4}-\d{2}-\d{2})/([^/]+)/eng@\d{4}-\d{2}-\d{2}`)

// titleRe extracts the bill title from the link text (slug).
var titleRe = regexp.MustCompile(`-([a-z])`)

// ParseBillsListing extracts Bill candidates from the Kenya Law bills HTML.
func ParseBillsListing(html string) []BillCandidate {
	matches := billLinkRe.FindAllStringSubmatch(html, -1)
	if matches == nil {
		return nil
	}

	seen := make(map[string]bool, len(matches))
	out := make([]BillCandidate, 0, len(matches))

	for _, m := range matches {
		houseCode := m[1]      // "na" or "senate"
		dateStr := m[2]        // "2026-09-07"
		slug := m[3]           // "the-local-authorities-provident-fund-amendment-bill-2026"
		fullURL := m[0]        // the full match

		// Deduplicate (the listing page may have the same bill linked multiple times).
		if seen[fullURL] {
			continue
		}
		seen[fullURL] = true

		pubDate, _ := time.Parse("2006-01-02", dateStr)

		house := "National Assembly"
		if houseCode == "senate" {
			house = "Senate"
		}

		// Convert slug to title: "the-local-authorities..." → "The Local Authorities..."
		title := slugToTitle(slug)

		out = append(out, BillCandidate{
			URL:             "https://new.kenyalaw.org" + fullURL,
			Slug:            slug,
			Title:           title,
			House:           house,
			PublicationDate: pubDate,
			SourceID:        fmt.Sprintf("ke-bill-%s-%s", dateStr, slug[:min(len(slug), 40)]),
		})
	}

	return out
}

// slugToTitle converts a URL slug to a human-readable title.
// e.g., "the-local-authorities-provident-fund-amendment-bill-2026" → "The Local Authorities Provident Fund Amendment Bill 2026"
func slugToTitle(slug string) string {
	// Replace hyphens with spaces
	title := strings.ReplaceAll(slug, "-", " ")
	// Capitalize each word
	words := strings.Fields(title)
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, " ")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
