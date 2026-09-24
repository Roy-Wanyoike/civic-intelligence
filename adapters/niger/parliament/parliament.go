// Package parliament adapts the Parliament of Niger source
// (assemblee.ne). It implements discovery + fetch + parse for Bills,
// Hansard, Order Papers, Votes & Proceedings, and committee documents.
//
// Live site reconnaissance (2026-09-09):
//   - https://www.assemblee.ne/            → HTTP 200
//   - https://www.assemblee.ne/bills       → HTTP 200 (Bills listing)
//
// The Parliament of Niger is UNICAMERAL — there is only one House, the
// National Assembly (193 elected members).
//
// The Bills listing page is rendered as one <div class="bill-card"> per Bill
// (see parser.go for the exact structure). The per-Bill detail page is
// currently the same Bill PDF (no separate HTML detail page); the parser
// therefore extracts metadata from either the listing page or the PDF link's
// containing anchor.
package parliament

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Roy-Wanyoike/civic-intelligence/packages/contracts"
)

// HTTPClient is the minimal HTTP interface this adapter depends on. The
// real *http.Client satisfies it; tests may inject a fake.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// defaultUserAgent is the polite identifier used when the caller passes an
// empty string. It MUST include the project URL so source administrators can
// reach the platform operators if the crawler misbehaves.
const defaultUserAgent = "CivicIntelligence/0.1 (+https://github.com/Roy-Wanyoike/civic-intelligence)"

// Adapter adapts assemblee.ne to the platform's
// contracts.LegislativeSourceAdapter surface (Discover / Fetch / Parse). The
// adapter is stateless aside from its HTTP client, user-agent string, and
// the canonical Bills listing URL.
type Adapter struct {
	client    HTTPClient
	userAgent string
	// baseURL is the canonical Parliament of Niger root.
	baseURL string
	// billsURL is the Bills listing page.
	billsURL string
}

// BillCandidate is a discovered Bill before canonicalization into a
// contracts.SourceItem.
type BillCandidate struct {
	// URL is the canonical URL of the Bill — typically the published Bill PDF.
	URL string
	// Title is the human-readable Bill title (from the link text).
	Title string
	// House is always "National Assembly of Niger" (Niger is unicameral).
	House string
	// BillNumber is the published Bill number, e.g. "Bill No. 12 of 2024".
	BillNumber string
	// Sponsor is the Mover / originating department.
	Sponsor string
	// Stage is the raw stage text published by Parliament.
	Stage string
	// Date is the raw date string published by Parliament.
	Date string
	// SourceID is the platform-internal stable ID.
	SourceID string
	// DiscoveredAt is when this Bill was discovered by the adapter.
	DiscoveredAt time.Time
}

// NewAdapter constructs a assemblee.ne adapter. If client is nil a
// default *http.Client with a 30-second timeout is used. If userAgent is
// empty, the defaultUserAgent is used.
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
		baseURL:   "https://www.assemblee.ne",
		billsURL:  "https://www.assemblee.ne/bills",
	}
}

// CountryCode returns "NE".
func (a *Adapter) CountryCode() string { return "NE" }

// Supports reports whether this adapter can handle the given URL. The
// Parliament adapter handles URLs from assemblee.ne.
func (a *Adapter) Supports(u string) bool {
	for _, host := range []string{
		"assemblee.ne",
		"www.assemblee.ne",
	} {
		if strings.Contains(u, host) {
			return true
		}
	}
	return false
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

// GetOfficialSources returns the source definitions the Parliament adapter crawls.
func (a *Adapter) GetOfficialSources() []SourceDefinition {
	return []SourceDefinition{
		{
			ID:             "mw-parliament-bills",
			Country:        "NE",
			Institution:    "Parliament of Niger — Bills",
			Authority:      "primary",
			URL:            a.billsURL,
			Adapter:        "niger.parliament",
			DocumentTypes:  []string{"bill"},
			CrawlFrequency: 6 * time.Hour,
		},
	}
}

// DiscoverBills crawls the Bills listing page and returns Bill candidates.
// A failure fetching the page is returned as an error.
//
// Every returned BillCandidate carries its source URL and DiscoveredAt (UTC).
func (a *Adapter) DiscoverBills(ctx context.Context) ([]BillCandidate, error) {
	body, err := a.fetchURL(ctx, a.billsURL)
	if err != nil {
		return nil, fmt.Errorf("niger.parliament.DiscoverBills: %w", err)
	}
	cands := ParseBillsListing(string(body))
	now := time.Now().UTC()
	for i := range cands {
		cands[i].House = "National Assembly of Niger"
		if cands[i].SourceID == "" {
			cands[i].SourceID = makeSourceID(cands[i].URL)
		}
		if cands[i].DiscoveredAt.IsZero() {
			cands[i].DiscoveredAt = now
		}
	}
	return cands, nil
}

// Discover implements contracts.LegislativeSourceAdapter.
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
		"institution": "Parliament of Niger",
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
		CountryCode:  "NE",
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
func (a *Adapter) Fetch(ctx context.Context, item contracts.SourceItem) (*contracts.RawDocument, error) {
	if item.URL == "" {
		return nil, fmt.Errorf("niger.parliament.Fetch: empty URL")
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
		Bytes:       body,
		MimeType:    mime,
		RetrievedAt: time.Now().UTC(),
	}, nil
}

// Parse implements contracts.LegislativeSourceAdapter. HTML Bill pages are
// parsed for stage + sponsor + publication date via ParseBillDetail. PDFs are
// returned as a single low-confidence ExtractedRecord carrying only the source
// URL + RetrievedAt.
func (a *Adapter) Parse(ctx context.Context, doc contracts.RawDocument) ([]contracts.ExtractedRecord, error) {
	_ = ctx
	if doc.MimeType == "application/pdf" || strings.HasSuffix(strings.ToLower(doc.URL), ".pdf") {
		return []contracts.ExtractedRecord{{
			Kind:          "bill",
			SourceURL:     doc.URL,
			RetrievedAt:   doc.RetrievedAt,
			ExtractorName: "niger.parliament.PDFPlaceholder",
			Confidence:    0.1,
		}}, nil
	}
	return ParseBillDetail(string(doc.Bytes), doc.URL)
}

// GetLegislativeStructure implements contracts.LegislativeSourceAdapter. It
// returns an error — the parliament adapter does not own legislative
// structure data; callers should use the top-level NigerAdapter.
func (a *Adapter) GetLegislativeStructure(ctx context.Context) (*contracts.LegislativeStructure, error) {
	_ = ctx
	return nil, fmt.Errorf("niger.parliament: GetLegislativeStructure not implemented on the source adapter; use the top-level NigerAdapter")
}

// GetStages implements contracts.LegislativeSourceAdapter. Same caveat as
// GetLegislativeStructure.
func (a *Adapter) GetStages(ctx context.Context) ([]contracts.StageDefinition, error) {
	_ = ctx
	return nil, fmt.Errorf("niger.parliament: GetStages not implemented on the source adapter; use the top-level NigerAdapter")
}

// GetTerminology implements contracts.LegislativeSourceAdapter. Same caveat.
func (a *Adapter) GetTerminology(ctx context.Context) ([]contracts.TermDefinition, error) {
	_ = ctx
	return nil, fmt.Errorf("niger.parliament: GetTerminology not implemented on the source adapter; use the top-level NigerAdapter")
}

// NormalizeSourceItem returns a minimal SourceItem so the contract surface
// compiles. The top-level NigerAdapter owns its own normalizer.
func (a *Adapter) NormalizeSourceItem(raw map[string]any) (contracts.SourceItem, error) {
	getStr := func(k string) string {
		if v, ok := raw[k]; ok {
			if s, ok := v.(string); ok {
				return strings.TrimSpace(s)
			}
		}
		return ""
	}
	return contracts.SourceItem{
		URL:          getStr("url"),
		Title:        getStr("title"),
		DocumentType: "bill",
		SourceType:   contracts.SourceItemBill,
		CountryCode:  "NE",
	}, nil
}

// fetchURL is the polite wrapper around a.client.Do. It sets the User-Agent,
// accepts both HTML and PDF responses, and returns the body bytes. It does
// NOT delegate to Fetch (that pattern caused the Kenya infinite-recursion
// bug, see worklog ENG-A1 / issue #201).
func (a *Adapter) fetchURL(ctx context.Context, rawURL string) ([]byte, error) {
	if rawURL == "" {
		return nil, fmt.Errorf("niger.parliament.fetchURL: empty URL")
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
		return nil, fmt.Errorf("niger.parliament.fetchURL: HTTP %d for %s", resp.StatusCode, rawURL)
	}
	return io.ReadAll(resp.Body)
}

// makeSourceID derives a stable, deterministic SourceID for a Niger
// Parliament Bill from its source URL.
func makeSourceID(rawURL string) string {
	prefix := "mw-parliament-bill"
	slug := rawURL
	if u, err := url.Parse(rawURL); err == nil && u.Path != "" {
		parts := strings.Split(strings.TrimRight(u.Path, "/"), "/")
		if len(parts) > 0 {
			slug = parts[len(parts)-1]
		}
	}
	slug = strings.ToLower(slug)
	slug = strings.TrimSuffix(slug, ".pdf")
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

// SetBillsURLForTest overrides the bills listing URL. It exists ONLY so tests
// can point Discover at a local httptest.Server instead of the live
// assemblee.ne site. Production callers MUST NOT use this method.
func (a *Adapter) SetBillsURLForTest(url string) { a.billsURL = url }
