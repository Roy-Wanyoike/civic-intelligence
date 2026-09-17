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
// This is the adapter SKELETON. The Discover/Fetch/Parse methods are stubbed
// to return empty results so the global ingestion service can be wired up
// without waiting for the full HTML/PDF parser. Per-Bill HTML parsing will be
// implemented in issue #CI-AD-NG-001; per-Bill PDF (Gazette) parsing in
// issue #CI-AD-NG-002.
package parliament

import (
	"context"
	"fmt"
	"net/http"
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
		client:        client,
		userAgent:     userAgent,
		baseURL:       "https://nass.gov.ng",
		houseBillsURL: "https://nass.gov.ng/house/bills",
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

// Discover implements contracts.LegislativeSourceAdapter. It is a SKELETON —
// per the task description, this is the nass.gov.ng adapter skeleton.
//
// TODO(issue #CI-AD-NG-001): crawl nass.gov.ng/house/bills and
// nass.gov.ng/senate/bills and parse Bill candidates from the listing pages.
// The Kenya adapter's DiscoverBills is the reference implementation.
func (a *Adapter) Discover(ctx context.Context) ([]contracts.SourceItem, error) {
	return []contracts.SourceItem{}, nil
}

// Fetch implements contracts.LegislativeSourceAdapter.
func (a *Adapter) Fetch(ctx context.Context, item contracts.SourceItem) (*contracts.RawDocument, error) {
	if item.URL == "" {
		return nil, fmt.Errorf("nigeria.parliament.Fetch: empty URL")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, item.URL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", a.userAgent)
	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("nigeria.parliament.Fetch: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("nigeria.parliament.Fetch: HTTP %d", resp.StatusCode)
	}
	body, err := io_ReadAll(resp.Body)
	if err != nil {
		return nil, err
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

// Parse implements contracts.LegislativeSourceAdapter. SKELETON — the HTML
// Bill page parser is not yet implemented.
//
// TODO(issue #CI-AD-NG-001): parse nass.gov.ng Bill detail pages for stage +
// sponsor + publication date, following the Kenya parliament.ParseBillDetail
// pattern. PDFs are returned as a single low-confidence ExtractedRecord
// carrying only the source URL + RetrievedAt — full PDF text extraction is
// delegated to the documents service.
func (a *Adapter) Parse(ctx context.Context, doc contracts.RawDocument) ([]contracts.ExtractedRecord, error) {
	if doc.MimeType == "application/pdf" || endsWith(doc.URL, ".pdf") {
		return []contracts.ExtractedRecord{{
			Kind:          "bill",
			SourceURL:     doc.URL,
			RetrievedAt:   doc.RetrievedAt,
			ExtractorName: "nigeria.parliament.PDFPlaceholder",
			Confidence:    0.1,
		}}, nil
	}
	return []contracts.ExtractedRecord{}, nil
}

// NormalizeSourceItem implements contracts.LegislativeSourceAdapter. It
// accepts raw metadata maps (typically produced by external schedulers that
// have discovered URLs out of band) and projects them into the canonical
// SourceItem shape using the central Nigeria normalizer.
func (a *Adapter) NormalizeSourceItem(raw map[string]any) (contracts.SourceItem, error) {
	return internal.NormalizeSourceItem(raw)
}

// GetLegislativeStructure implements contracts.LegislativeSourceAdapter. It
// returns the static Nigeria legislative structure (bicameral House of
// Representatives + Senate) from the central Nigeria internal package.
func (a *Adapter) GetLegislativeStructure(ctx context.Context) (*contracts.LegislativeStructure, error) {
	s := internal.NigeriaLegislativeStructure()
	return &s, nil
}

// GetStages implements contracts.LegislativeSourceAdapter. It returns the
// Nigeria Bill stages (projected to contracts.StageDefinition via ToContract).
func (a *Adapter) GetStages(ctx context.Context) ([]contracts.StageDefinition, error) {
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
	terms := internal.NigeriaTerminology
	out := make([]contracts.TermDefinition, len(terms))
	for i, t := range terms {
		out[i] = t.ToContract()
	}
	return out, nil
}
