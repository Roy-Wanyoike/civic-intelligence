// Package parliament adapts the Parliament of DR Congo source
// (assemblee-nationale.cd). It implements discovery + fetch + parse for Bills
// and committee documents.
//
// Live site reconnaissance (2026-09-09):
//   - https://www.assemblee-nationale.cd/   → HTTP 200
//   - https://www.assemblee-nationale.cd/projets-lois → HTTP 200 (Bills listing)
//
// The Parliament of DR Congo is BICAMERAL:
//   - National Assembly (Assemblée Nationale) — 500 members
//   - Senate (Sénat)                          — 109 members
//
// Bills originate in the National Assembly by default; the Senate reviews
// them. The President promulgates Acts.
//
// The Bills listing page is rendered as one <div class="bill-card"> per Bill
// (see parser.go for the exact structure).
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

// House code constants used to identify DR Congo's bicameral chambers.
const (
	HouseCodeNationalAssembly = "NA"  // lower house — Assemblée Nationale
	HouseCodeSenate           = "SEN" // upper house — Sénat
)

// Adapter adapts assemblee-nationale.cd to the platform's
// contracts.LegislativeSourceAdapter surface (Discover / Fetch / Parse).
type Adapter struct {
	client    HTTPClient
	userAgent string
	// baseURL is the canonical National Assembly of DR Congo root.
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
	// House is "National Assembly" or "Senate".
	House string
	// BillNumber is the published Bill number, e.g. "Projet de Loi N° 24/012".
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

// NewAdapter constructs an assemblee-nationale.cd adapter. If client is nil a
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
		baseURL:   "https://www.assemblee-nationale.cd",
		billsURL:  "https://www.assemblee-nationale.cd/projets-lois",
	}
}

// CountryCode returns "CD".
func (a *Adapter) CountryCode() string { return "CD" }

// Supports reports whether this adapter can handle the given URL. The
// Parliament adapter handles URLs from assemblee-nationale.cd and senat.cd.
func (a *Adapter) Supports(u string) bool {
	for _, host := range []string{
		"assemblee-nationale.cd",
		"senat.cd",
		"www.assemblee-nationale.cd",
		"www.senat.cd",
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
			ID:             "cd-parliament-bills",
			Country:        "CD",
			Institution:    "Parliament of DR Congo — Bills",
			Authority:      "primary",
			URL:            a.billsURL,
			Adapter:        "dr_congo.parliament",
			DocumentTypes:  []string{"bill"},
			CrawlFrequency: 6 * time.Hour,
		},
	}
}

// DiscoverBills crawls the Bills listing page and returns Bill candidates.
// A failure fetching the page is returned as an error.
//
// Every returned BillCandidate carries its source URL and DiscoveredAt (UTC).
// Bills default to the National Assembly (Bills originate there by default);
// a Bill may override this via a <span class="bill-house"> in its bill-card
// markup (used for Senate-originated Bills, which are rare).
func (a *Adapter) DiscoverBills(ctx context.Context) ([]BillCandidate, error) {
	body, err := a.fetchURL(ctx, a.billsURL)
	if err != nil {
		return nil, fmt.Errorf("dr_congo.parliament.DiscoverBills: %w", err)
	}
	cands := ParseBillsListing(string(body))
	now := time.Now().UTC()
	for i := range cands {
		if cands[i].House == "" {
			cands[i].House = "National Assembly"
		}
		if cands[i].SourceID == "" {
			cands[i].SourceID = makeSourceID(cands[i].House, cands[i].URL)
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
	house := b.House
	if house == "" {
		house = "National Assembly"
	}
	meta := map[string]string{
		"house":       house,
		"institution": "Parliament of DR Congo",
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
		CountryCode:  "CD",
		SourceID:     b.SourceID,
		House:        house,
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
		return nil, fmt.Errorf("dr_congo.parliament.Fetch: empty URL")
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
			ExtractorName: "dr_congo.parliament.PDFPlaceholder",
			Confidence:    0.1,
		}}, nil
	}
	return ParseBillDetail(string(doc.Bytes), doc.URL)
}

// GetLegislativeStructure implements contracts.LegislativeSourceAdapter. It
// returns an error — the parliament adapter does not own legislative
// structure data; callers should use the top-level DRCongoAdapter.
func (a *Adapter) GetLegislativeStructure(ctx context.Context) (*contracts.LegislativeStructure, error) {
	_ = ctx
	return nil, fmt.Errorf("dr_congo.parliament: GetLegislativeStructure not implemented on the source adapter; use the top-level DRCongoAdapter")
}

// GetStages implements contracts.LegislativeSourceAdapter. Same caveat as
// GetLegislativeStructure.
func (a *Adapter) GetStages(ctx context.Context) ([]contracts.StageDefinition, error) {
	_ = ctx
	return nil, fmt.Errorf("dr_congo.parliament: GetStages not implemented on the source adapter; use the top-level DRCongoAdapter")
}

// GetTerminology implements contracts.LegislativeSourceAdapter. Same caveat.
func (a *Adapter) GetTerminology(ctx context.Context) ([]contracts.TermDefinition, error) {
	_ = ctx
	return nil, fmt.Errorf("dr_congo.parliament: GetTerminology not implemented on the source adapter; use the top-level DRCongoAdapter")
}

// NormalizeSourceItem returns a minimal SourceItem so the contract surface
// compiles. The top-level DRCongoAdapter owns its own normalizer.
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
		CountryCode:  "CD",
	}, nil
}

// fetchURL is the polite wrapper around a.client.Do. It sets the User-Agent,
// accepts both HTML and PDF responses, and returns the body bytes. It does
// NOT delegate to Fetch (that pattern caused the Kenya infinite-recursion
// bug, see worklog ENG-A1 / issue #201).
func (a *Adapter) fetchURL(ctx context.Context, rawURL string) ([]byte, error) {
	if rawURL == "" {
		return nil, fmt.Errorf("dr_congo.parliament.fetchURL: empty URL")
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
		return nil, fmt.Errorf("dr_congo.parliament.fetchURL: HTTP %d for %s", resp.StatusCode, rawURL)
	}
	return io.ReadAll(resp.Body)
}

// makeSourceID derives a stable, deterministic SourceID for a DR Congo
// Parliament Bill from its house and source URL.
//
//	house="Senate", url="…/projets-lois/loi-donnees-personnelles.pdf"
//	→ "cd-parliament-sen-bill-loi-donnees-personnelles"
func makeSourceID(house, rawURL string) string {
	prefix := "cd-parliament-na-bill"
	if strings.EqualFold(house, "Senate") ||
		strings.Contains(strings.ToLower(house), "senate") ||
		strings.Contains(strings.ToLower(house), "sénat") {
		prefix = "cd-parliament-sen-bill"
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

// SetBillsURLForTest overrides the bills listing URL. It exists ONLY so tests
// can point Discover at a local httptest.Server instead of the live
// assemblee-nationale.cd site. Production callers MUST NOT use this method.
func (a *Adapter) SetBillsURLForTest(url string) { a.billsURL = url }
