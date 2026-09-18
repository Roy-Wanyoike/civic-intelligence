// Package parliament adapts the Parliament of South Africa source
// (parliament.gov.za). It implements discovery + fetch + parse for Bills,
// Hansard, Order Papers, Questions & Replies, and committee documents.
//
// Live site reconnaissance (2026-09-09):
//   - https://www.parliament.gov.za/                                  → HTTP 200 (Drupal)
//   - https://www.parliament.gov.za/bills-and-laws                    → HTTP 200
//   - https://www.parliament.gov.za/parliamentary-committees          → HTTP 200
//   - https://www.parliament.gov.za/hansard                           → HTTP 200
//   - https://www.parliament.gov.za/order-paper                       → HTTP 200
//   - https://www.parliament.gov.za/questions-and-replies             → HTTP 200
//
// The Bills listing page (https://www.parliament.gov.za/bills-and-laws)
// exposes a search interface over current and historical Bills. The per-Bill
// detail page carries the title, sponsor (member or Minister), portfolio
// committee, the latest stage ("Introduced" / "Passed by NA" / "Passed by
// NCOP" / "Assented to" / "In force"), and links to the published Bill text
// PDFs.
//
package parliament

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Roy-Wanyoike/civic-intelligence/adapters/south_africa/internal"
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

// Adapter fetches from the Parliament of South Africa's website.
type Adapter struct {
	client    HTTPClient
	userAgent string
	// baseURL is the canonical Parliament of South Africa root.
	baseURL string
	// billsURL is the Bills & Laws landing page.
	billsURL string
	// committeesURL is the parliamentary committees landing page.
	committeesURL string
	// hansardURL is the Hansard landing page.
	hansardURL string
}

// NewAdapter creates a Parliament of South Africa adapter.
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
		baseURL:       "https://www.parliament.gov.za",
		billsURL:      "https://www.parliament.gov.za/bills-and-laws",
		committeesURL: "https://www.parliament.gov.za/parliamentary-committees",
		hansardURL:    "https://www.parliament.gov.za/hansard",
	}
}

// CountryCode returns "ZA".
func (a *Adapter) CountryCode() string { return "ZA" }

// Supports reports whether this adapter can handle the given URL. The
// Parliament adapter handles URLs from parliament.gov.za (with or without the
// "www." subdomain).
func (a *Adapter) Supports(u string) bool {
	for _, host := range []string{
		"parliament.gov.za",
		"www.parliament.gov.za",
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
			ID:             "za-parliament-bills",
			Country:        "ZA",
			Institution:    "Parliament of South Africa — Bills & Laws",
			Authority:      "primary",
			URL:            a.billsURL,
			Adapter:        "south_africa.parliament",
			DocumentTypes:  []string{"bill"},
			CrawlFrequency: 6 * time.Hour,
		},
		{
			ID:             "za-parliament-committees",
			Country:        "ZA",
			Institution:    "Parliament of South Africa — Committees",
			Authority:      "primary",
			URL:            a.committeesURL,
			Adapter:        "south_africa.parliament",
			DocumentTypes:  []string{"committee_report"},
			CrawlFrequency: 6 * time.Hour,
		},
		{
			ID:             "za-parliament-hansard",
			Country:        "ZA",
			Institution:    "Parliament of South Africa — Hansard",
			Authority:      "primary",
			URL:            a.hansardURL,
			Adapter:        "south_africa.parliament",
			DocumentTypes:  []string{"hansard"},
			CrawlFrequency: 24 * time.Hour,
		},
	}
}

// BillCandidate is a discovered Bill before canonicalization into a
// contracts.SourceItem.
type BillCandidate struct {
	// URL is the canonical URL of the Bill on parliament.gov.za.
	URL string
	// Title is the human-readable Bill title.
	Title string
	// House is "National Assembly" or "NCOP" (rare; Bills usually originate
	// in the NA).
	House string
	// BillNumber is the published Bill number, e.g. "B 12—2026".
	BillNumber string
	// Sponsor is the member or Minister who introduced the Bill.
	Sponsor string
	// PortfolioCommittee is the committee the Bill was referred to.
	PortfolioCommittee string
	// SourceID is the platform-internal stable ID.
	SourceID string
	// DiscoveredAt is when this Bill was discovered by the adapter.
	DiscoveredAt time.Time
}

// DiscoverBills crawls the Bills & Laws page and returns Bill candidates.
// A failure fetching the page is returned as an error so the caller can
// surface it.
//
// Every returned BillCandidate carries its source URL and DiscoveredAt (UTC).
// Bills default to the National Assembly house (Bills originate in the NA by
// default); a Bill may override this via a <span class="bill-house"> in its
// bill-card markup (used for NCOP-only Bills, which are rare).
func (a *Adapter) DiscoverBills(ctx context.Context) ([]BillCandidate, error) {
	body, err := a.fetchURL(ctx, a.billsURL)
	if err != nil {
		return nil, fmt.Errorf("south_africa.parliament.DiscoverBills: %w", err)
	}
	cands := ParseBillsListing(string(body))
	now := time.Now().UTC()
	for i := range cands {
		if cands[i].House == "" {
			cands[i].House = "National Assembly" // Bills originate in the NA by default
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

// ParseStage maps Parliament of South Africa's raw stage text (e.g.,
// "Introduced", "Passed by the National Assembly", "Passed by the NCOP",
// "Assented to", "In force") to the platform's canonical stage codes defined
// in adapters/south_africa/internal/south_africa_data.go
// (INTRODUCTION, COMMITTEE, …, NA_VOTE, NCOP_CONCURRENCE, …).
//
// The mapping is case-insensitive. Unknown inputs return an empty string —
// the caller SHOULD treat an empty stage as "could not be determined".
func (a *Adapter) ParseStage(ctx context.Context, input string) string {
	return MapStageText(input)
}

// MapStageText maps raw stage text from parliament.gov.za to the canonical
// South Africa stage code. Unknown inputs return "".
//
// Mapping is intentionally permissive: it accepts the common variations
// Parliament publishes ("Introduced" vs "Introduction", "Passed by NA" vs
// "NA Vote", etc.).
func MapStageText(input string) string {
	s := strings.ToLower(strings.TrimSpace(input))
	switch {
	case strings.Contains(s, "introduc"):
		return string(internal.StageIntroduction)
	case strings.Contains(s, "mediat"):
		return string(internal.StageMediation)
	case strings.Contains(s, "public participation"), strings.Contains(s, "public comment"):
		return string(internal.StagePublicParticipation)
	case strings.Contains(s, "committee"):
		return string(internal.StageCommittee)
	case strings.Contains(s, "na vote"), strings.Contains(s, "passed by the national assembly"), strings.Contains(s, "passed by na"):
		return string(internal.StageNAVote)
	case strings.Contains(s, "ncop"), strings.Contains(s, "national council of provinces"), strings.Contains(s, "passed by the ncop"):
		return string(internal.StageNCOPConcurrence)
	case strings.Contains(s, "assent"), strings.Contains(s, "signed by the president"), strings.Contains(s, "presidential assent"):
		return string(internal.StagePresidentialAssent)
	case strings.Contains(s, "commenc"), strings.Contains(s, "in force"), strings.Contains(s, "effective"):
		return string(internal.StageCommencement)
	case strings.Contains(s, "reject"), strings.Contains(s, "defeated"), strings.Contains(s, "negative"):
		return string(internal.StageRejected)
	case strings.Contains(s, "withdraw"):
		return string(internal.StageWithdrawn)
	case strings.Contains(s, "laps"):
		return string(internal.StageLapsed)
	default:
		return ""
	}
}

// Discover implements contracts.LegislativeSourceAdapter. It discovers Bills
// (and, in future, Hansard / Order Papers / Questions & Replies) by
// delegating to DiscoverBills.
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
// canonical SourceItem shape. SourceURL + DiscoveredAt are always set.
func (a *Adapter) billCandidateToSourceItem(b BillCandidate) contracts.SourceItem {
	house := b.House
	if house == "" {
		house = "National Assembly" // Bills originate in the NA by default
	}
	meta := map[string]string{
		"house":       house,
		"institution": "Parliament of South Africa",
	}
	if b.BillNumber != "" {
		meta["bill_number"] = b.BillNumber
	}
	if b.Sponsor != "" {
		meta["sponsor"] = b.Sponsor
	}
	if b.PortfolioCommittee != "" {
		meta["portfolio_committee"] = b.PortfolioCommittee
	}
	return contracts.SourceItem{
		URL:          b.URL,
		Title:        b.Title,
		DocumentType: "bill",
		SourceType:   contracts.SourceItemBill,
		CountryCode:  "ZA",
		SourceID:     b.SourceID,
		House:        house,
		DiscoveredAt: b.DiscoveredAt,
		Metadata:     meta,
		RawMetadata:  meta,
	}
}

// Fetch implements contracts.LegislativeSourceAdapter.
func (a *Adapter) Fetch(ctx context.Context, item contracts.SourceItem) (*contracts.RawDocument, error) {
	if item.URL == "" {
		return nil, fmt.Errorf("parliament.Fetch: empty URL")
	}
	body, err := a.fetchURL(ctx, item.URL)
	if err != nil {
		return nil, fmt.Errorf("parliament.Fetch: %w", err)
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
// URL + RetrievedAt — full PDF text extraction is delegated to the documents
// service.
func (a *Adapter) Parse(ctx context.Context, doc contracts.RawDocument) ([]contracts.ExtractedRecord, error) {
	_ = ctx
	if doc.MimeType == "application/pdf" || strings.HasSuffix(strings.ToLower(doc.URL), ".pdf") {
		// We cannot parse the PDF here; the documents service handles it.
		return []contracts.ExtractedRecord{{
			Kind:          "bill",
			SourceURL:     doc.URL,
			RetrievedAt:   doc.RetrievedAt,
			ExtractorName: "south_africa.parliament.PDFPlaceholder",
			Confidence:    0.1,
		}}, nil
	}
	return ParseBillDetail(string(doc.Bytes), doc.URL)
}

// GetLegislativeStructure implements contracts.LegislativeSourceAdapter. It
// returns the static South Africa legislative structure (bicameral National
// Assembly + NCOP) from the central South Africa internal package.
func (a *Adapter) GetLegislativeStructure(ctx context.Context) (*contracts.LegislativeStructure, error) {
	s := internal.SouthAfricaLegislativeStructure()
	return &s, nil
}

// GetStages implements contracts.LegislativeSourceAdapter. It returns the
// South Africa Bill stages (projected to contracts.StageDefinition via ToContract).
func (a *Adapter) GetStages(ctx context.Context) ([]contracts.StageDefinition, error) {
	stages := internal.SouthAfricaBillStages
	out := make([]contracts.StageDefinition, len(stages))
	for i, s := range stages {
		out[i] = s.ToContract()
	}
	return out, nil
}

// GetTerminology implements contracts.LegislativeSourceAdapter. It returns the
// South Africa parliamentary terminology registry.
func (a *Adapter) GetTerminology(ctx context.Context) ([]contracts.TermDefinition, error) {
	terms := internal.SouthAfricaTerminology
	out := make([]contracts.TermDefinition, len(terms))
	for i, t := range terms {
		out[i] = t.ToContract()
	}
	return out, nil
}

// fetchURL is the polite wrapper around a.client.Do. It sets the User-Agent,
// respects the per-host rate limit, and returns the body bytes.
func (a *Adapter) fetchURL(ctx context.Context, rawURL string) ([]byte, error) {
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
		return nil, fmt.Errorf("HTTP %d for %s", resp.StatusCode, rawURL)
	}
	return io.ReadAll(resp.Body)
}

// PoliteClient wraps an http.Client with per-host rate limiting. Used by all
// South Africa source adapters to be a good citizen of the source websites.
//
// Production hardening (issue #CI-AD-ZA-002): respect robots.txt, support
// ETag/If-None-Match, exponential backoff on 5xx, circuit breaker per host.
type PoliteClient struct {
	*http.Client
	ratePerSec float64
	lastCall   time.Time
}

// Do enforces the per-host rate limit before delegating to the wrapped
// http.Client.
func (p *PoliteClient) Do(req *http.Request) (*http.Response, error) {
	if !p.lastCall.IsZero() {
		minInterval := time.Duration(float64(time.Second) / p.ratePerSec)
		if elapsed := time.Since(p.lastCall); elapsed < minInterval {
			time.Sleep(minInterval - elapsed)
		}
	}
	p.lastCall = time.Now()
	return p.Client.Do(req)
}

// makeSourceID derives a stable, deterministic SourceID for a South Africa
// Parliament Bill from its house and source URL.
//
//      house="NCOP", url="…/bills-and-laws/b31-2026.pdf"
//      → "za-parliament-ncop-bill-b31-2026"
func makeSourceID(house, rawURL string) string {
	prefix := "za-parliament-na-bill"
	if strings.EqualFold(house, "NCOP") ||
		strings.Contains(strings.ToLower(house), "national council of provinces") {
		prefix = "za-parliament-ncop-bill"
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

// SetBillsURLForTest overrides the Bills listing URL. It exists ONLY so tests
// can point Discover at a local httptest.Server instead of the live
// parliament.gov.za site. Production callers MUST NOT use this method.
func (a *Adapter) SetBillsURLForTest(u string) { a.billsURL = u }
