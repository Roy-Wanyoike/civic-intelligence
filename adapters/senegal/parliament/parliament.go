// Package parliament adapts the Assemblée Nationale du Sénégal source
// (assemblee-nationale.sn). It implements discovery + fetch + parse for Bills,
// Hansard, Order Papers, and committee documents.
//
// Senegal's National Assembly is UNICAMERAL — there is no Senate. All Bills
// pass through a single chamber ("Assemblée Nationale du Sénégal"). The
// canonical Bills listing page is https://www.assemblee-nationale.sn/travaux/lois,
// which renders one <div class="bill-card"> per Bill, with optional metadata
// spans for bill-number / sponsor / stage / date.
//
// The adapter implements discovery + fetch + parse for Bills. HTML parsing
// uses golang.org/x/net/html (same as the Ghana adapter). Stage mapping
// targets the SenegalBillStages defined in
// adapters/senegal/internal/senegal_data.go.
//
// IMPORTANT: fetchURL is a STANDALONE helper that does NOT delegate to Fetch.
// This avoids the infinite-recursion bug observed in the Kenya adapter (where
// Discover delegated to Fetch and Fetch delegated back through a code path
// that could re-enter Discover — see worklog ENG-A1 / issue #201).
//
// NOTE: The stage CODES are in English (for consistency with the rest of the
// platform); the stage DISPLAY NAMES are in French, as published by the
// Assemblée Nationale on assemblee-nationale.sn. Stage mapping accepts both
// French and English substrings — e.g., "dépôt" and "deposit" both map to
// the DEPOT stage code.
package parliament

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Roy-Wanyoike/civic-intelligence/adapters/senegal/internal"
	"github.com/Roy-Wanyoike/civic-intelligence/packages/contracts"
)

// HTTPClient is the interface the Senegal parliament adapter uses to fetch
// remote content.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// defaultUserAgent is the polite identifier used when the caller passes an
// empty string. It MUST include the project URL so source administrators can
// reach the platform operators if the crawler misbehaves.
const defaultUserAgent = "CivicIntelligence/0.1 (+https://github.com/Roy-Wanyoike/civic-intelligence)"

// parliamentHouse is the canonical House name for all Senegal Bills.
const parliamentHouse = "Assemblée Nationale du Sénégal"

// defaultBillsURL is the canonical Assemblée Nationale Bills listing page.
const defaultBillsURL = "https://www.assemblee-nationale.sn/travaux/lois"

// Adapter fetches from the Assemblée Nationale du Sénégal's website.
type Adapter struct {
	client    HTTPClient
	userAgent string
	billsURL  string
}

// NewAdapter creates a Senegal Parliament adapter.
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

// CountryCode returns "SN".
func (a *Adapter) CountryCode() string { return "SN" }

// Supports reports whether this adapter can handle the given URL. The
// Parliament adapter handles URLs from assemblee-nationale.sn.
func (a *Adapter) Supports(u string) bool {
	return strings.Contains(u, "assemblee-nationale.sn")
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
			ID:             "sn-parliament-bills",
			Country:        "SN",
			Institution:    "Assemblée Nationale du Sénégal — Lois",
			Authority:      "primary",
			URL:            a.billsURL,
			Adapter:        "senegal.parliament",
			DocumentTypes:  []string{"bill"},
			CrawlFrequency: 6 * time.Hour,
		},
	}
}

// GetLegislativeStructure returns Senegal's unicameral legislative structure.
func (a *Adapter) GetLegislativeStructure(ctx context.Context) (*contracts.LegislativeStructure, error) {
	_ = ctx
	s := internal.SenegalLegislativeStructure()
	return &s, nil
}

// GetStages returns Senegal's Bill stages (unicameral lifecycle).
func (a *Adapter) GetStages(ctx context.Context) ([]contracts.StageDefinition, error) {
	_ = ctx
	return internal.SenegalBillStages, nil
}

// GetTerminology returns Senegal's parliamentary terminology registry.
func (a *Adapter) GetTerminology(ctx context.Context) ([]contracts.TermDefinition, error) {
	_ = ctx
	return internal.SenegalTerminology, nil
}

// NormalizeSourceItem projects a raw metadata map into the canonical SourceItem.
func (a *Adapter) NormalizeSourceItem(raw map[string]any) (contracts.SourceItem, error) {
	return contracts.SourceItem{
		URL:          getString(raw, "url"),
		Title:        getString(raw, "title"),
		DocumentType: "bill",
		SourceType:   contracts.SourceItemBill,
		CountryCode:  "SN",
		House:        parliamentHouse,
	}, nil
}

// BillCandidate is a discovered Bill before canonicalization into a
// contracts.SourceItem.
type BillCandidate struct {
	// URL is the canonical URL of the Bill.
	URL string
	// Title is the human-readable Bill title (from the link text).
	Title string
	// Number is the official Bill number, e.g., "Projet de loi n° 12/2024".
	Number string
	// Sponsor is the Bill's sponsor (usually a Minister), if available.
	Sponsor string
	// Stage is the canonical Senegal stage code (see SenegalBillStages).
	Stage string
	// StageRaw is the raw stage text as published.
	StageRaw string
	// PublicationDate is the date the Bill was published, if available.
	PublicationDate time.Time
	// DiscoveredAt is when this Bill was discovered by the adapter.
	DiscoveredAt time.Time
}

// fetchURL downloads a URL and returns the body as a string. It is a STANDALONE
// helper that does NOT delegate to Fetch — this avoids the infinite-recursion
// bug observed in the Kenya adapter.
func (a *Adapter) fetchURL(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", a.userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml")

	resp, err := a.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("senegal.fetchURL %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("senegal.fetchURL %s: HTTP %d", url, resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("senegal.fetchURL %s: read body: %w", url, err)
	}
	return string(body), nil
}

// Discover implements contracts.LegislativeSourceAdapter. It scrapes the
// Assemblée Nationale Bills listing page (assemblee-nationale.sn/travaux/lois)
// and returns one SourceItem per Bill found.
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
// canonical SourceItem shape.
func (a *Adapter) billCandidateToSourceItem(b BillCandidate, now time.Time) contracts.SourceItem {
	meta := map[string]string{
		"house":       parliamentHouse,
		"institution": "Assemblée Nationale du Sénégal",
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
		CountryCode:  "SN",
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
// bytes of a single Bill page.
func (a *Adapter) Fetch(ctx context.Context, item contracts.SourceItem) (*contracts.RawDocument, error) {
	if item.URL == "" {
		return nil, fmt.Errorf("senegal.Fetch: empty URL")
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
// metadata from an Assemblée Nationale HTML page.
func (a *Adapter) Parse(ctx context.Context, doc contracts.RawDocument) ([]contracts.ExtractedRecord, error) {
	_ = ctx
	if doc.MimeType == "application/pdf" || strings.HasSuffix(strings.ToLower(doc.URL), ".pdf") {
		return []contracts.ExtractedRecord{{
			Kind:          "bill",
			SourceURL:     doc.URL,
			RetrievedAt:   doc.RetrievedAt,
			ExtractorName: "senegal.parliament.PDFPlaceholder",
			Confidence:    0.1,
		}}, nil
	}
	records, err := ParseBillDetail(string(doc.Bytes), doc.URL)
	if err != nil {
		return nil, err
	}
	if !doc.RetrievedAt.IsZero() {
		for i := range records {
			records[i].RetrievedAt = doc.RetrievedAt
		}
	}
	return records, nil
}

// makeSourceID derives a stable, deterministic SourceID for a Senegal Bill
// from its source URL. Example:
//
//	https://www.assemblee-nationale.sn/travaux/lois/loi-de-finances-2024
//	→ "sn-parliament-bill-loi-de-finances-2024"
func makeSourceID(rawURL string) string {
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
	// Strip diacritics and collapse non-alphanumeric runs to a single hyphen.
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
	return "sn-parliament-bill-" + slug
}

func getString(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

// SetBillsURLForTest overrides the bills listing URL. It exists ONLY so tests
// can point Discover at a local httptest.Server instead of the live
// assemblee-nationale.sn site. Production callers MUST NOT use this method.
func (a *Adapter) SetBillsURLForTest(u string) { a.billsURL = u }

// Compile-time assertion: Adapter satisfies contracts.LegislativeSourceAdapter.
var _ contracts.LegislativeSourceAdapter = (*Adapter)(nil)
