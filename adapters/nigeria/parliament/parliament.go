// Package parliament adapts the National Assembly of Nigeria source
// (nass.gov.ng). It implements discovery + fetch + parse for Bills, Hansard,
// Order Papers, Votes & Proceedings, and committee documents.
//
// Live site reconnaissance (2026-09-09):
//   - https://nass.gov.ng/  → HTTP 200 (official National Assembly portal)
//   - https://nass.gov.ng/house/bills                    → HTTP 200
//   - https://nass.gov.ng/senate/bills                    → HTTP 200
//   - https://nass.gov.ng/house/committees                → HTTP 200
//   - https://nass.gov.ng/senate/committees               → HTTP 200
//
// The National Assembly of Nigeria is bicameral — Bills are listed
// separately for the House of Representatives and the Senate. DiscoverBills
// therefore fetches BOTH listing pages and merges the results, tagging each
// Bill with the house it originated in.
//
// The Bills listing pages use a "bill-card" structure (see parser.go's
// ParseBillsListing): one <div class="bill-card"> per Bill, with optional
// metadata spans for bill-number / sponsor / stage / date.
//
// FIXME: verify with go build when Go available
package parliament

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Roy-Wanyoike/civic-intelligence/adapters/nigeria/internal"
	"github.com/Roy-Wanyoike/civic-intelligence/packages/contracts"
)

// HTTPClient is the interface parliament adapters use to fetch remote content.
// Production uses a polite, rate-limited client (see PoliteClient). Tests use
// an in-memory implementation. This indirection is critical for SSRF
// protection and for testing offline.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// defaultUserAgent is the polite identifier used when the caller passes an
// empty string. It MUST include the project URL so source administrators can
// reach the platform operators if the crawler misbehaves.
const defaultUserAgent = "CivicIntelligence/0.1 (+https://github.com/Roy-Wanyoike/civic-intelligence)"

// Adapter fetches from the National Assembly of Nigeria's website.
type Adapter struct {
	client    HTTPClient
	userAgent string
	// baseURL is the canonical National Assembly of Nigeria root.
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
	// BillNumber is the published Bill number, e.g. "HB. 1234" / "SB. 567".
	BillNumber string
	// Sponsor is the Mover (member or Minister).
	Sponsor string
	// Stage is the raw stage text published by the National Assembly.
	Stage string
	// Date is the raw date string published by the National Assembly.
	Date string
	// SourceID is the platform-internal stable ID.
	SourceID string
	// DiscoveredAt is when this Bill was discovered by the adapter.
	DiscoveredAt time.Time
}

// NewAdapter creates a National Assembly of Nigeria adapter.
//
// If client is nil, a PoliteClient wrapping the default http.Client is used
// (1 request/sec/host). If userAgent is empty, defaultUserAgent is used.
func NewAdapter(client HTTPClient, userAgent string) *Adapter {
	if client == nil {
		client = &PoliteClient{Client: &http.Client{Timeout: 30 * time.Second}, ratePerSec: 1.0}
	}
	if userAgent == "" {
		userAgent = defaultUserAgent
	}
	return &Adapter{
		client:         client,
		userAgent:      userAgent,
		baseURL:        "https://nass.gov.ng",
		houseBillsURL:  "https://nass.gov.ng/house/bills",
		senateBillsURL: "https://nass.gov.ng/senate/bills",
	}
}

// CountryCode returns "NG".
func (a *Adapter) CountryCode() string { return "NG" }

// Supports reports whether this adapter can handle the given URL. The
// Parliament adapter handles URLs from nass.gov.ng.
func (a *Adapter) Supports(u string) bool {
	return contains(u, "nass.gov.ng")
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
			ID:             "ng-nass-house-bills",
			Country:        "NG",
			Institution:    "National Assembly of Nigeria — House of Representatives",
			Authority:      "primary",
			URL:            a.houseBillsURL,
			Adapter:        "nigeria.parliament",
			DocumentTypes:  []string{"bill"},
			CrawlFrequency: 6 * time.Hour,
		},
		{
			ID:             "ng-nass-senate-bills",
			Country:        "NG",
			Institution:    "National Assembly of Nigeria — Senate",
			Authority:      "primary",
			URL:            a.senateBillsURL,
			Adapter:        "nigeria.parliament",
			DocumentTypes:  []string{"bill"},
			CrawlFrequency: 6 * time.Hour,
		},
	}
}

// DiscoverBills crawls BOTH the House of Representatives and the Senate Bills
// listing pages and returns Bill candidates. Both pages are fetched politely
// and parsed with ParseBillsListing. A failure on one page does not abort
// the other — a single failing source does not abort the whole discovery.
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
		"institution": "National Assembly of Nigeria",
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
		CountryCode:  "NG",
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
// NOTE: Fetch used to be the implementation site of the HTTP call. Per the
// Kenya infinite-recursion bug (worklog ENG-A1 / issue #201), fetchURL is
// now the single HTTP entrypoint; Fetch wraps it with RawDocument shaping.
func (a *Adapter) Fetch(ctx context.Context, item contracts.SourceItem) (*contracts.RawDocument, error) {
	if item.URL == "" {
		return nil, fmt.Errorf("nigeria.parliament.Fetch: empty URL")
	}
	body, err := a.fetchURL(ctx, item.URL)
	if err != nil {
		return nil, fmt.Errorf("nigeria.parliament.Fetch: %w", err)
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
			ExtractorName: "nigeria.parliament.PDFPlaceholder",
			Confidence:    0.1,
		}}, nil
	}
	return ParseBillDetail(string(doc.Bytes), doc.URL)
}

// NormalizeSourceItem implements contracts.LegislativeSourceAdapter. It
// accepts raw metadata maps and projects them into the canonical SourceItem
// shape using the central Nigeria normalizer.
func (a *Adapter) NormalizeSourceItem(raw map[string]any) (contracts.SourceItem, error) {
	return internal.NormalizeSourceItem(raw)
}

// GetLegislativeStructure implements contracts.LegislativeSourceAdapter. It
// returns the static Nigeria legislative structure (bicameral House of
// Representatives + Senate) from the central Nigeria internal package.
func (a *Adapter) GetLegislativeStructure(ctx context.Context) (*contracts.LegislativeStructure, error) {
	_ = ctx
	s := internal.NigeriaLegislativeStructure()
	return &s, nil
}

// GetStages implements contracts.LegislativeSourceAdapter. It returns the
// Nigeria Bill stages (projected to contracts.StageDefinition via ToContract).
func (a *Adapter) GetStages(ctx context.Context) ([]contracts.StageDefinition, error) {
	_ = ctx
	stages := internal.NigeriaBillStages
	out := make([]contracts.StageDefinition, len(stages))
	for i, s := range stages {
		out[i] = s.ToContract()
	}
	return out, nil
}

// GetTerminology implements contracts.LegislativeSourceAdapter. It returns
// the Nigeria parliamentary terminology registry.
func (a *Adapter) GetTerminology(ctx context.Context) ([]contracts.TermDefinition, error) {
	_ = ctx
	terms := internal.NigeriaTerminology
	out := make([]contracts.TermDefinition, len(terms))
	for i, t := range terms {
		out[i] = t.ToContract()
	}
	return out, nil
}

// fetchURL is the polite wrapper around a.client.Do. It sets the User-Agent,
// accepts both HTML and PDF responses, and returns the body bytes. It does
// NOT delegate to Fetch (that pattern caused the Kenya infinite-recursion
// bug, see worklog ENG-A1 / issue #201).
func (a *Adapter) fetchURL(ctx context.Context, rawURL string) ([]byte, error) {
	if rawURL == "" {
		return nil, fmt.Errorf("nigeria.parliament.fetchURL: empty URL")
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
		return nil, fmt.Errorf("nigeria.parliament.fetchURL: HTTP %d for %s", resp.StatusCode, rawURL)
	}
	return io_ReadAll(resp.Body)
}

// makeSourceID derives a stable, deterministic SourceID for a Nigeria
// Parliament Bill from its house and source URL.
//
//	house="Senate", url="…/senate/bills/sb-2024-567.pdf"
//	→ "ng-parliament-senate-bill-sb-2024-567"
func makeSourceID(house, rawURL string) string {
	prefix := "ng-parliament-house-bill"
	if strings.EqualFold(house, "Senate") {
		prefix = "ng-parliament-senate-bill"
	}
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

// SetHouseBillsURLForTest overrides the House of Representatives Bills URL.
// It exists ONLY so tests can point Discover at a local httptest.Server
// instead of the live nass.gov.ng site. Production callers MUST NOT use it.
func (a *Adapter) SetHouseBillsURLForTest(u string) { a.houseBillsURL = u }

// SetSenateBillsURLForTest overrides the Senate Bills URL. Same caveat as
// SetHouseBillsURLForTest.
func (a *Adapter) SetSenateBillsURLForTest(u string) { a.senateBillsURL = u }
