// Package parliament adapts the National Assembly of Zambia source
// (parliament.gov.zm). It implements discovery + fetch + parse for Bills,
// Hansard, Order Papers, Votes & Proceedings, and committee documents.
//
// Zambia's National Assembly is UNICAMERAL — there is no Senate. All Bills
// pass through a single chamber ("National Assembly of Zambia"). The canonical
// Bills listing page is https://www.parliament.gov.zm/business/bills, which
// renders one <div class="bill-card"> per Bill, with optional metadata spans
// for bill-number / sponsor / stage / date.
//
// The adapter implements discovery + fetch + parse for Bills. HTML parsing uses
// golang.org/x/net/html (same as the Ghana adapter). Stage mapping targets the
// ZambiaBillStages defined in adapters/zambia/internal/zambia_data.go.
//
// IMPORTANT: fetchURL is a STANDALONE helper that does NOT delegate to Fetch.
// This avoids the infinite-recursion bug observed in the Kenya adapter (where
// Discover delegated to Fetch and Fetch delegated back through a code path
// that could re-enter Discover — see worklog ENG-A1 / issue #201).
package parliament

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Roy-Wanyoike/civic-intelligence/adapters/zambia/internal"
	"github.com/Roy-Wanyoike/civic-intelligence/packages/contracts"
)

// HTTPClient is the interface the Zambia parliament adapter uses to fetch
// remote content. Production uses a standard *http.Client with a 30-second
// timeout; tests inject an in-memory implementation backed by httptest.Server.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// defaultUserAgent is the polite identifier used when the caller passes an
// empty string. It MUST include the project URL so source administrators can
// reach the platform operators if the crawler misbehaves.
const defaultUserAgent = "CivicIntelligence/0.1 (+https://github.com/Roy-Wanyoike/civic-intelligence)"

// parliamentHouse is the canonical House name for all Zambia Bills.
const parliamentHouse = "National Assembly of Zambia"

// defaultBillsURL is the canonical National Assembly of Zambia Bills listing
// page.
const defaultBillsURL = "https://www.parliament.gov.zm/business/bills"

// Adapter fetches from the National Assembly of Zambia's website.
type Adapter struct {
	client    HTTPClient
	userAgent string
	billsURL  string
}

// NewAdapter creates a Zambia Parliament adapter.
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
		client:    client,
		userAgent: userAgent,
		billsURL:  defaultBillsURL,
	}
}

// CountryCode returns "ZM".
func (a *Adapter) CountryCode() string { return "ZM" }

// Supports reports whether this adapter can handle the given URL. The
// Parliament adapter handles URLs from parliament.gov.zm.
func (a *Adapter) Supports(u string) bool {
	return strings.Contains(u, "parliament.gov.zm")
}

// UserAgent returns the User-Agent header the adapter sends on every request.
func (a *Adapter) UserAgent() string { return a.userAgent }

// BillsURL returns the canonical Bills listing URL the adapter crawls.
func (a *Adapter) BillsURL() string { return a.billsURL }

// SourceDefinition describes an official source the adapter crawls.
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
			ID:             "zm-parliament-bills",
			Country:        "ZM",
			Institution:    "National Assembly of Zambia — Bills",
			Authority:      "primary",
			URL:            a.billsURL,
			Adapter:        "zambia.parliament",
			DocumentTypes:  []string{"bill"},
			CrawlFrequency: 6 * time.Hour,
		},
	}
}

// GetLegislativeStructure returns Zambia's unicameral legislative structure.
func (a *Adapter) GetLegislativeStructure(ctx context.Context) (*contracts.LegislativeStructure, error) {
	_ = ctx
	s := internal.ZambiaLegislativeStructure()
	return &s, nil
}

// GetStages returns Zambia's Bill stages (unicameral lifecycle).
func (a *Adapter) GetStages(ctx context.Context) ([]contracts.StageDefinition, error) {
	_ = ctx
	return internal.ZambiaBillStages, nil
}

// GetTerminology returns Zambia's parliamentary terminology registry.
func (a *Adapter) GetTerminology(ctx context.Context) ([]contracts.TermDefinition, error) {
	_ = ctx
	return internal.ZambiaTerminology, nil
}

// NormalizeSourceItem projects a raw metadata map into the canonical SourceItem.
func (a *Adapter) NormalizeSourceItem(raw map[string]any) (contracts.SourceItem, error) {
	return contracts.SourceItem{
		URL:          getString(raw, "url"),
		Title:        getString(raw, "title"),
		DocumentType: "bill",
		SourceType:   contracts.SourceItemBill,
		CountryCode:  "ZM",
		House:        parliamentHouse,
	}, nil
}

// BillCandidate is a discovered Bill before canonicalization into a
// contracts.SourceItem.
type BillCandidate struct {
	// URL is the canonical URL of the Bill — typically the Bill detail page
	// or the published Bill PDF on parliament.gov.zm.
	URL string
	// Title is the human-readable Bill title (from the link text on the
	// listing page).
	Title string
	// Number is the official Bill number as published, e.g.,
	// "Bill No. 12 of 2024".
	Number string
	// Sponsor is the Bill's sponsor (usually a Minister), if available.
	Sponsor string
	// Stage is the canonical Zambia stage code (FIRST_READING, …) or "" if
	// the stage could not be determined from the listing.
	Stage string
	// StageRaw is the raw stage text as published (e.g., "Second Reading").
	StageRaw string
	// PublicationDate is the date the Bill was published, if available.
	PublicationDate time.Time
	// DiscoveredAt is when this Bill was discovered by the adapter.
	DiscoveredAt time.Time
}

// fetchURL downloads a URL and returns the body as a string. It is a STANDALONE
// helper that does NOT delegate to Fetch — this avoids the infinite-recursion
// bug observed in the Kenya adapter (where Discover delegated to Fetch and
// Fetch delegated back through a code path that could re-enter Discover).
//
// All HTTP requests carry the adapter's User-Agent header and a 30-second
// timeout (set on the client by NewAdapter).
func (a *Adapter) fetchURL(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", a.userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml")

	resp, err := a.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("zambia.fetchURL %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("zambia.fetchURL %s: HTTP %d", url, resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("zambia.fetchURL %s: read body: %w", url, err)
	}
	return string(body), nil
}

// Discover implements contracts.LegislativeSourceAdapter. It scrapes the
// National Assembly of Zambia Bills listing page (parliament.gov.zm/business/bills)
// and returns one SourceItem per Bill found.
//
// On a fetch failure, Discover returns the error (no partial results). On a
// parse failure (e.g., malformed HTML), Discover returns whatever Bills could
// be extracted; the parser is tolerant and never returns a parse error.
func (a *Adapter) Discover(ctx context.Context) ([]contracts.SourceItem, error) {
	body, err := a.fetchURL(ctx, a.billsURL)
	if err != nil {
		return nil, err
	}
	cands := ParseBillsListing(body)
	now := time.Now().UTC()
	items := make([]contracts.SourceItem, 0, len(cands))
	for _, c := range cands {
		c.DiscoveredAt = now
		items = append(items, a.billCandidateToSourceItem(c, now))
	}
	return items, nil
}

// billCandidateToSourceItem projects a BillCandidate into the platform's
// canonical SourceItem shape. URL, Title, House, CountryCode, DiscoveredAt
// and Metadata are always set; the rest are populated when available.
func (a *Adapter) billCandidateToSourceItem(b BillCandidate, now time.Time) contracts.SourceItem {
	meta := map[string]string{
		"house":       parliamentHouse,
		"institution": "National Assembly of Zambia",
	}
	if b.Number != "" {
		meta["bill_number"] = b.Number
	}
	if b.Sponsor != "" {
		meta["sponsor"] = b.Sponsor
	}
	if b.Stage != "" {
		meta["stage"] = b.Stage
	}
	if b.StageRaw != "" {
		meta["stage_raw"] = b.StageRaw
	}
	if !b.PublicationDate.IsZero() {
		meta["publication_date"] = b.PublicationDate.Format("2006-01-02")
	}
	return contracts.SourceItem{
		URL:          b.URL,
		Title:        b.Title,
		DocumentType: "bill",
		SourceType:   contracts.SourceItemBill,
		CountryCode:  "ZM",
		House:        parliamentHouse,
		SourceID:     makeSourceID(b.URL),
		ExternalID:   b.Number,
		PublishedAt:  b.PublicationDate,
		DiscoveredAt: now,
		Metadata:     meta,
		RawMetadata:  meta,
	}
}

// Fetch implements contracts.LegislativeSourceAdapter. It downloads the raw
// bytes of a single Bill page. The MimeType is set to "text/html" by default
// and "application/pdf" if the URL ends in .pdf (Bill PDFs are linked from
// the detail page).
func (a *Adapter) Fetch(ctx context.Context, item contracts.SourceItem) (*contracts.RawDocument, error) {
	if item.URL == "" {
		return nil, fmt.Errorf("zambia.Fetch: empty URL")
	}
	body, err := a.fetchURL(ctx, item.URL)
	if err != nil {
		return nil, err
	}
	mime := "text/html"
	if strings.HasSuffix(strings.ToLower(item.URL), ".pdf") {
		mime = "application/pdf"
	}
	return &contracts.RawDocument{
		URL:         item.URL,
		Bytes:       []byte(body),
		MimeType:    mime,
		RetrievedAt: time.Now().UTC(),
	}, nil
}

// Parse implements contracts.LegislativeSourceAdapter. It extracts Bill
// metadata from a National Assembly of Zambia HTML page.
//
// Both listing pages and detail pages are tolerated: a listing page yields
// one ExtractedRecord per Bill row; a detail page yields a single record.
// PDFs are returned as a single low-confidence ExtractedRecord carrying only
// the source URL + RetrievedAt — full PDF text extraction is delegated to the
// documents service.
//
// Parse never returns an error for a partial extraction; it returns whatever
// records could be extracted with lowered Confidence. An error is returned
// only when no Bills can be extracted at all.
//
// Each record's RetrievedAt is set to the RawDocument's RetrievedAt (the time
// the source was fetched), NOT the parse time. This preserves the correct
// provenance timestamp across the Fetch → Parse boundary.
func (a *Adapter) Parse(ctx context.Context, doc contracts.RawDocument) ([]contracts.ExtractedRecord, error) {
	_ = ctx
	if doc.MimeType == "application/pdf" || strings.HasSuffix(strings.ToLower(doc.URL), ".pdf") {
		// We cannot parse the PDF here; the documents service handles it.
		return []contracts.ExtractedRecord{{
			Kind:          "bill",
			SourceURL:     doc.URL,
			RetrievedAt:   doc.RetrievedAt,
			ExtractorName: "zambia.parliament.PDFPlaceholder",
			Confidence:    0.1,
		}}, nil
	}
	records, err := ParseBillDetail(string(doc.Bytes), doc.URL)
	if err != nil {
		return nil, err
	}
	// Override RetrievedAt with the document's actual retrieval time (the
	// parser defaults to time.Now() at parse time, which is semantically wrong
	// when Parse runs long after Fetch — e.g., in a retry or backfill).
	if !doc.RetrievedAt.IsZero() {
		for i := range records {
			records[i].RetrievedAt = doc.RetrievedAt
		}
	}
	return records, nil
}

// makeSourceID derives a stable, deterministic SourceID for a Zambia
// Parliament Bill from its source URL. Example:
//
//	https://www.parliament.gov.zm/business/bills/public-health-amendment-bill-2024
//	→ "zm-parliament-bill-public-health-amendment-bill-2024"
func makeSourceID(rawURL string) string {
	slug := rawURL
	if u, err := url.Parse(rawURL); err == nil && u.Path != "" {
		parts := strings.Split(strings.TrimRight(u.Path, "/"), "/")
		if len(parts) > 0 {
			slug = parts[len(parts)-1]
		}
	}
	slug = strings.ToLower(slug)
	// Strip common suffixes.
	for _, suffix := range []string{".html", ".htm", ".pdf"} {
		slug = strings.TrimSuffix(slug, suffix)
	}
	// Collapse non-alphanumeric runs to a single hyphen.
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
	return "zm-parliament-bill-" + slug
}

func getString(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

// SetBillsURLForTest overrides the bills listing URL. It exists ONLY so tests
// can point Discover at a local httptest.Server instead of the live
// parliament.gov.zm site. Production callers MUST NOT use this method.
func (a *Adapter) SetBillsURLForTest(u string) { a.billsURL = u }

// Compile-time assertion: Adapter satisfies contracts.LegislativeSourceAdapter.
var _ contracts.LegislativeSourceAdapter = (*Adapter)(nil)
