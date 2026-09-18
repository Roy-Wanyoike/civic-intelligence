// Package parliament adapts the Egyptian Parliament source
// (parliament.eg). It implements discovery + fetch + parse for Bills,
// Hansard, Order Papers, and committee documents.
//
// The Egyptian Parliament is BICAMERAL — Bills are listed separately for
// the House of Representatives (lower house) and the Senate (upper house).
// DiscoverBills therefore fetches BOTH listing pages and merges the results,
// tagging each Bill with the chamber it originated in.
//
// The Bills listing pages use a "bill-card" structure (see parser.go's
// ParseBillsListing): one <div class="bill-card"> per Bill, with optional
// metadata spans for bill-number / sponsor / stage / date.
//
// IMPORTANT: fetchURL is a STANDALONE helper that does NOT delegate to Fetch.
// This avoids the infinite-recursion bug observed in the Kenya adapter (where
// Discover delegated to Fetch and Fetch delegated back through a code path
// that could re-enter Discover — see worklog ENG-A1 / issue #201).
package parliament

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Roy-Wanyoike/civic-intelligence/adapters/egypt/internal"
	"github.com/Roy-Wanyoike/civic-intelligence/packages/contracts"
)

// HTTPClient is the interface parliament adapters use to fetch remote content.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// defaultUserAgent is the polite identifier used when the caller passes an
// empty string. It MUST include the project URL so source administrators can
// reach the platform operators if the crawler misbehaves.
const defaultUserAgent = "CivicIntelligence/0.1 (+https://github.com/Roy-Wanyoike/civic-intelligence)"

// Adapter fetches from the Egyptian Parliament's website.
type Adapter struct {
	client    HTTPClient
	userAgent string
	// baseURL is the canonical Egyptian Parliament root.
	baseURL string
	// houseBillsURL is the House of Representatives Bills listing page.
	houseBillsURL string
	// senateBillsURL is the Senate Bills listing page.
	senateBillsURL string
}

// BillCandidate is a discovered Bill before canonicalization into a
// contracts.SourceItem.
type BillCandidate struct {
	// URL is the canonical URL of the Bill — typically the published Bill PDF.
	URL string
	// Title is the human-readable Bill title (from the link text).
	Title string
	// House is "House of Representatives" or "Senate".
	House string
	// BillNumber is the published Bill number, e.g., "Bill No. 18/2024".
	BillNumber string
	// Sponsor is the Mover (member or Minister).
	Sponsor string
	// Stage is the raw stage text published by the Egyptian Parliament.
	Stage string
	// Date is the raw date string published by the Egyptian Parliament.
	Date string
	// SourceID is the platform-internal stable ID.
	SourceID string
	// DiscoveredAt is when this Bill was discovered by the adapter.
	DiscoveredAt time.Time
}

// NewAdapter creates an Egyptian Parliament adapter.
//
// If client is nil, a standard *http.Client with a 30-second timeout is used.
// If userAgent is empty, defaultUserAgent is used.
func NewAdapter(client HTTPClient, userAgent string) *Adapter {
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	if userAgent == "" {
		userAgent = defaultUserAgent
	}
	return &Adapter{
		client:        client,
		userAgent:     userAgent,
		baseURL:       "https://www.parliament.eg",
		houseBillsURL: "https://www.parliament.eg/house/bills",
		senateBillsURL: "https://www.parliament.eg/senate/bills",
	}
}

// CountryCode returns "EG".
func (a *Adapter) CountryCode() string { return "EG" }

// Supports reports whether this adapter can handle the given URL. The
// Parliament adapter handles URLs from parliament.eg.
func (a *Adapter) Supports(u string) bool {
	return contains(u, "parliament.eg")
}

// UserAgent returns the User-Agent header the adapter sends on every request.
func (a *Adapter) UserAgent() string { return a.userAgent }

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

// GetOfficialSources returns the source definitions the Parliament adapter crawls.
func (a *Adapter) GetOfficialSources() []SourceDefinition {
	return []SourceDefinition{
		{
			ID:             "eg-parliament-house-bills",
			Country:        "EG",
			Institution:    "Egyptian Parliament — House of Representatives",
			Authority:      "primary",
			URL:            a.houseBillsURL,
			Adapter:        "egypt.parliament",
			DocumentTypes:  []string{"bill"},
			CrawlFrequency: 6 * time.Hour,
		},
		{
			ID:             "eg-parliament-senate-bills",
			Country:        "EG",
			Institution:    "Egyptian Parliament — Senate",
			Authority:      "primary",
			URL:            a.senateBillsURL,
			Adapter:        "egypt.parliament",
			DocumentTypes:  []string{"bill"},
			CrawlFrequency: 6 * time.Hour,
		},
	}
}

// DiscoverBills crawls BOTH the House of Representatives and the Senate Bills
// listing pages and returns Bill candidates. Both pages are fetched and parsed
// with ParseBillsListing. A failure on one page does not abort the other — a
// single failing source does not abort the whole discovery.
//
// Every returned BillCandidate carries its source URL and DiscoveredAt (UTC).
func (a *Adapter) DiscoverBills(ctx context.Context) ([]BillCandidate, error) {
	var out []BillCandidate
	for _, page := range []struct {
		url   string
		house string
	}{
		{a.houseBillsURL, "House of Representatives"},
		{a.senateBillsURL, "Senate"},
	} {
		body, err := a.fetchURL(ctx, page.url)
		if err != nil {
			// A single failing page does not abort discovery of the other.
			continue
		}
		cands := ParseBillsListing(string(body))
		now := time.Now().UTC()
		for i := range cands {
			cands[i].House = page.house
			if cands[i].SourceID == "" {
				cands[i].SourceID = makeSourceID(page.house, cands[i].URL)
			}
			cands[i].DiscoveredAt = now
		}
		out = append(out, cands...)
	}
	return out, nil
}

// Discover implements contracts.LegislativeSourceAdapter. It discovers Bills
// by delegating to DiscoverBills (which fetches both chambers' listing pages).
func (a *Adapter) Discover(ctx context.Context) ([]contracts.SourceItem, error) {
	bills, err := a.DiscoverBills(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]contracts.SourceItem, 0, len(bills))
	for _, b := range bills {
		items = append(items, a.billCandidateToSourceItem(b))
	}
	return items, nil
}

// billCandidateToSourceItem projects a BillCandidate into the platform's
// canonical SourceItem shape.
func (a *Adapter) billCandidateToSourceItem(b BillCandidate) contracts.SourceItem {
	meta := map[string]string{
		"house":       b.House,
		"institution": "Egyptian Parliament",
	}
	if b.BillNumber != "" {
		meta["bill_number"] = b.BillNumber
	}
	if b.Sponsor != "" {
		meta["sponsor"] = b.Sponsor
	}
	if b.Stage != "" {
		meta["stage"] = b.Stage
	}
	if b.Date != "" {
		meta["date"] = b.Date
	}
	return contracts.SourceItem{
		URL:          b.URL,
		Title:        b.Title,
		DocumentType: "bill",
		SourceType:   contracts.SourceItemBill,
		CountryCode:  "EG",
		SourceID:     b.SourceID,
		House:        b.House,
		DiscoveredAt: b.DiscoveredAt,
		Metadata:     meta,
		RawMetadata:  meta,
	}
}

// Fetch implements contracts.LegislativeSourceAdapter. It delegates the raw
// HTTP work to fetchURL — Fetch is responsible for shaping the bytes into a
// contracts.RawDocument (with the correct MIME type) so the caller can route
// it to the right downstream parser.
//
// IMPORTANT: fetchURL is a STANDALONE helper that does NOT delegate to Fetch.
func (a *Adapter) Fetch(ctx context.Context, item contracts.SourceItem) (*contracts.RawDocument, error) {
	if item.URL == "" {
		return nil, fmt.Errorf("egypt.parliament.Fetch: empty URL")
	}
	body, err := a.fetchURL(ctx, item.URL)
	if err != nil {
		return nil, fmt.Errorf("egypt.parliament.Fetch: %w", err)
	}
	mime := "text/html"
	if endsWith(item.URL, ".pdf") {
		mime = "application/pdf"
	}
	return &contracts.RawDocument{
		URL:         item.URL,
		Bytes:       body,
		MimeType:    mime,
		RetrievedAt: time.Now().UTC(),
	}, nil
}

// Parse implements contracts.LegislativeSourceAdapter. HTML Bill pages are
// parsed for stage + sponsor + publication date via ParseBillDetail. PDFs are
// returned as a single low-confidence ExtractedRecord carrying only the source
// URL + RetrievedAt — full PDF text extraction is delegated to the documents
// service.
func (a *Adapter) Parse(ctx context.Context, doc contracts.RawDocument) ([]contracts.ExtractedRecord, error) {
	_ = ctx
	if doc.MimeType == "application/pdf" || endsWith(doc.URL, ".pdf") {
		return []contracts.ExtractedRecord{{
			Kind:          "bill",
			SourceURL:     doc.URL,
			RetrievedAt:   doc.RetrievedAt,
			ExtractorName: "egypt.parliament.PDFPlaceholder",
			Confidence:    0.1,
		}}, nil
	}
	return ParseBillDetail(string(doc.Bytes), doc.URL)
}

// GetLegislativeStructure implements contracts.LegislativeSourceAdapter. It
// returns the static Egypt legislative structure (bicameral Senate + House of
// Representatives) from the central Egypt internal package.
func (a *Adapter) GetLegislativeStructure(ctx context.Context) (*contracts.LegislativeStructure, error) {
	_ = ctx
	s := internal.EgyptLegislativeStructure()
	return &s, nil
}

// GetStages implements contracts.LegislativeSourceAdapter. It returns the
// Egypt Bill stages.
func (a *Adapter) GetStages(ctx context.Context) ([]contracts.StageDefinition, error) {
	_ = ctx
	return internal.EgyptBillStages, nil
}

// GetTerminology implements contracts.LegislativeSourceAdapter. It returns
// the Egypt parliamentary terminology registry.
func (a *Adapter) GetTerminology(ctx context.Context) ([]contracts.TermDefinition, error) {
	_ = ctx
	return internal.EgyptTerminology, nil
}

// NormalizeSourceItem implements contracts.LegislativeSourceAdapter. It
// accepts raw metadata maps and projects them into the canonical SourceItem
// shape.
func (a *Adapter) NormalizeSourceItem(raw map[string]any) (contracts.SourceItem, error) {
	return contracts.SourceItem{
		URL:          getString(raw, "url"),
		Title:        getString(raw, "title"),
		DocumentType: "bill",
		SourceType:   contracts.SourceItemBill,
		CountryCode:  "EG",
	}, nil
}

// fetchURL is the standalone helper that performs the HTTP GET. It does NOT
// delegate to Fetch (per the Kenya infinite-recursion bug — see worklog
// ENG-A1 / issue #201).
func (a *Adapter) fetchURL(ctx context.Context, rawURL string) ([]byte, error) {
	if rawURL == "" {
		return nil, fmt.Errorf("egypt.parliament.fetchURL: empty URL")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", a.userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/pdf;q=0.9,*/*;q=0.8")
	resp, err := a.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("egypt.parliament.fetchURL: HTTP %d for %s", resp.StatusCode, rawURL)
	}
	return io_ReadAll(resp.Body)
}

// makeSourceID derives a stable, deterministic SourceID for an Egypt
// Parliament Bill from its house and source URL.
//
//	house="Senate", url="…/senate/bills/new-investment-law-2024.pdf"
//	→ "eg-parliament-senate-bill-new-investment-law-2024"
func makeSourceID(house, rawURL string) string {
	prefix := "eg-parliament-house-bill"
	if strings.EqualFold(house, "Senate") {
		prefix = "eg-parliament-senate-bill"
	}
	slug := rawURL
	if u, err := url.Parse(rawURL); err == nil && u.Path != "" {
		parts := strings.Split(strings.TrimRight(u.Path, "/"), "/")
		if len(parts) > 0 {
			slug = parts[len(parts)-1]
		}
	}
	slug = strings.ToLower(slug)
	for _, suffix := range []string{".html", ".htm", ".pdf"} {
		slug = strings.TrimSuffix(slug, suffix)
	}
	var sb strings.Builder
	prevDash := false
	for _, r := range slug {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			sb.WriteRune(r)
			prevDash = false
		default:
			if !prevDash && sb.Len() > 0 {
				sb.WriteRune('-')
				prevDash = true
			}
		}
	}
	slug = strings.Trim(sb.String(), "-")
	if slug == "" {
		slug = "unknown"
	}
	return prefix + "-" + slug
}

// contains is a small helper to avoid importing strings just for this.
func contains(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// endsWith is a small case-insensitive suffix helper.
func endsWith(s, suffix string) bool {
	if len(suffix) > len(s) {
		return false
	}
	for i := 0; i < len(suffix); i++ {
		a := s[len(s)-len(suffix)+i]
		b := suffix[i]
		if a >= 'A' && a <= 'Z' {
			a += 'a' - 'A'
		}
		if b >= 'A' && b <= 'Z' {
			b += 'a' - 'A'
		}
		if a != b {
			return false
		}
	}
	return true
}

func getString(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

// SetHouseBillsURLForTest overrides the House of Representatives Bills URL.
// It exists ONLY so tests can point Discover at a local httptest.Server
// instead of the live parliament.eg site. Production callers MUST NOT use it.
func (a *Adapter) SetHouseBillsURLForTest(u string) { a.houseBillsURL = u }

// SetSenateBillsURLForTest overrides the Senate Bills URL. Same caveat as
// SetHouseBillsURLForTest.
func (a *Adapter) SetSenateBillsURLForTest(u string) { a.senateBillsURL = u }

// Compile-time assertion: Adapter satisfies contracts.LegislativeSourceAdapter.
var _ contracts.LegislativeSourceAdapter = (*Adapter)(nil)
