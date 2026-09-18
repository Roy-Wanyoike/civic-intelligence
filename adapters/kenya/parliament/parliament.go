// Package parliament adapts the Parliament of Kenya source (parliament.go.ke,
// nationalassembly.go.ke, senate.go.ke). It implements discovery + fetch + parse
// for Bills, Hansard, Order Papers, Votes & Proceedings, and committee documents,
// and stage tracking via the Parliament Bill Tracker.
//
// Live site reconnaissance (2026-09-09):
//   - https://www.parliament.go.ke/  → HTTP 200 (Drupal 8 site)
//   - https://www.parliament.go.ke/the-national-assembly/house-business/bills   → HTTP 200
//   - https://www.parliament.go.ke/the-senate/senate-bills                       → HTTP 200
//   - https://www.parliament.go.ke/the-national-assembly/house-business/bill-tracker → HTTP 200
//   - https://www.parliament.go.ke/bills                                          → HTTP 404
//
// The Bills listing pages use a "post-block" structure (see ParseBillsListing):
//
//	<div class="post-block">
//	  <div class="post-content">
//	    <div class="post-title">
//	      <a href="…/sites/default/files/YYYY-MM/<BILL>.pdf" title="…">Bill Title</a>
//	    </div>
//	    <div class="post-meta">
//	      <span class="post-digest">Bill Digest: <a href="…">…</a></span>
//	      <span class="post-billtracker">Bill Tracker: <a href="…">…</a></span>
//	      <span class="post-petition"><a href="…/contact/national_assembly_petition?bill=…">Submit Comments</a></span>
//	    </div>
//	  </div>
//	</div>
//
// The Bill Tracker page (`/the-national-assembly/house-business/bill-tracker`)
// is an index of weekly "Bills Tracker as at <DATE>" PDF documents — it is not
// a per-bill stage view. Per-bill stage information is currently published only
// inside those weekly PDF trackers, so the HTML adapter can return at best an
// empty stage with low confidence when a Bill's per-bill tracker slot is empty.
package parliament

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Roy-Wanyoike/civic-intelligence/adapters/kenya/internal"
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

// Adapter fetches from the Parliament of Kenya's websites.
type Adapter struct {
	client    HTTPClient
	userAgent string
	// baseURL is the canonical Parliament of Kenya root.
	baseURL string
	// naBillsURL is the National Assembly Bills listing page.
	naBillsURL string
	// senateBillsURL is the Senate Bills listing page.
	senateBillsURL string
	// billTrackerURL is the National Assembly Bill Tracker index.
	billTrackerURL string
}

// NewAdapter creates a Parliament adapter.
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
		baseURL:        "https://www.parliament.go.ke",
		naBillsURL:     "https://www.parliament.go.ke/the-national-assembly/house-business/bills",
		senateBillsURL: "https://www.parliament.go.ke/the-senate/senate-bills",
		billTrackerURL: "https://www.parliament.go.ke/the-national-assembly/house-business/bill-tracker",
	}
}

// CountryCode returns "KE".
func (a *Adapter) CountryCode() string { return "KE" }

// Supports reports whether this adapter can handle the given URL. The
// Parliament adapter handles URLs from parliament.go.ke,
// nationalassembly.go.ke, and senate.go.ke.
func (a *Adapter) Supports(u string) bool {
	for _, host := range []string{
		"parliament.go.ke",
		"nationalassembly.go.ke",
		"senate.go.ke",
	} {
		if strings.Contains(u, host) {
			return true
		}
	}
	return false
}

// BillCandidate is a discovered Bill before canonicalization into a
// contracts.SourceItem. Field names mirror the Kenya Law adapter's
// BillCandidate so the top-level Kenya adapter can treat both uniformly.
type BillCandidate struct {
	// URL is the canonical URL of the Bill — typically the published Bill PDF
	// on parliament.go.ke/sites/default/files/YYYY-MM/<slug>.pdf.
	URL string
	// Title is the human-readable Bill title (from the link text on the
	// listing page). May contain typos as published by Parliament.
	Title string
	// House is "National Assembly" or "Senate".
	House string
	// PetitionURL is the URL for submitting public comments on the Bill.
	// For National Assembly Bills it ends with /contact/national_assembly_petition?bill=…
	// For Senate Bills it ends with /contact/senate_petition?bill=…
	PetitionURL string
	// BillDigestURL is the optional URL of the Bill Digest PDF.
	BillDigestURL string
	// BillTrackerURL is the optional URL of the per-bill tracker page on
	// parliament.go.ke. Frequently empty for newly published Bills.
	BillTrackerURL string
	// SourceID is the platform-internal stable ID (e.g.,
	// "ke-parliament-na-bill-<slug>").
	SourceID string
	// DiscoveredAt is when this Bill was discovered by the adapter.
	DiscoveredAt time.Time
}

// BillTracker is the result of a stage-tracking query for a single Bill. It
// captures the Bill's current stage (as a Kenya stage code), the raw status
// text from Parliament, the date the stage was reached, and the source URL
// the information was derived from (always set, with RetrievedAt).
type BillTracker struct {
	// Stage is the canonical Kenya stage code (FIRST_READING, SECOND_READING,
	// …) or "" if the stage could not be determined from the source.
	Stage string
	// Status is the raw status text published by Parliament (e.g.,
	// "Second Reading — 12 March 2026"). May be empty.
	Status string
	// Date is the date the current stage was reached (UTC, zero if unknown).
	Date time.Time
	// SourceURL is the URL the tracker info was derived from.
	SourceURL string
	// RetrievedAt is when the adapter fetched the tracker info (always UTC, always set).
	RetrievedAt time.Time
	// Confidence is 0.0-1.0 — how confident the adapter is in the stage
	// mapping. Lower confidence is returned when the stage was inferred
	// indirectly (e.g., from a tracker PDF that hasn't been parsed yet).
	Confidence float64
}

// SourceDefinition describes an official source (mirrors kenya_law.SourceDefinition).
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
			ID:             "ke-parliament-na-bills",
			Country:        "KE",
			Institution:    "Parliament of Kenya — National Assembly",
			Authority:      "primary",
			URL:            a.naBillsURL,
			Adapter:        "kenya.parliament",
			DocumentTypes:  []string{"bill"},
			CrawlFrequency: 6 * time.Hour,
		},
		{
			ID:             "ke-parliament-senate-bills",
			Country:        "KE",
			Institution:    "Parliament of Kenya — Senate",
			Authority:      "primary",
			URL:            a.senateBillsURL,
			Adapter:        "kenya.parliament",
			DocumentTypes:  []string{"bill"},
			CrawlFrequency: 6 * time.Hour,
		},
		{
			ID:             "ke-parliament-bill-tracker",
			Country:        "KE",
			Institution:    "Parliament of Kenya — Bill Tracker",
			Authority:      "primary",
			URL:            a.billTrackerURL,
			Adapter:        "kenya.parliament",
			DocumentTypes:  []string{"bill_tracker"},
			CrawlFrequency: 24 * time.Hour,
		},
	}
}

// DiscoverBills crawls the National Assembly and Senate Bills listing pages
// and returns Bill candidates. Both pages are fetched politely and parsed
// with ParseBillsListing. A failure on one page does not abort the other.
//
// Every returned BillCandidate carries its source URL and DiscoveredAt (UTC).
func (a *Adapter) DiscoverBills(ctx context.Context) ([]BillCandidate, error) {
	var out []BillCandidate
	for _, page := range []struct {
		url   string
		house string
	}{
		{a.naBillsURL, "National Assembly"},
		{a.senateBillsURL, "Senate"},
	} {
		body, err := a.fetchURL(ctx, page.url, "bill")
		if err != nil {
			// A single failing page does not abort discovery of the other.
			continue
		}
		cands := ParseBillsListing(body)
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

// FetchBillTracker returns stage tracking information for a single Bill. The
// billURL is typically the BillCandidate.BillTrackerURL (when populated by the
// listing page) or, as a fallback, the BillCandidate.URL itself.
//
// Implementation note: the live Parliament Bills listing page currently has
// an empty "Bill Tracker" link slot for most recent Bills. The canonical
// per-Bill stage information lives inside the weekly "Bills Tracker" PDF
// published at /the-national-assembly/house-business/bill-tracker. PDF
// parsing is out of scope for this adapter (see TODO below); when only a PDF
// is available, FetchBillTracker returns an empty Stage with low Confidence
// and the weekly-tracker PDF URL as SourceURL so the caller can schedule a
// follow-up PDF-parsing job.
//
// TODO(issue #CI-AD-005): integrate an external PDF parser (pdfcpu or unidoc)
// to extract per-Bill stage rows from the weekly tracker PDF.
func (a *Adapter) FetchBillTracker(ctx context.Context, billURL string) (*BillTracker, error) {
	if billURL == "" {
		return nil, fmt.Errorf("parliament.FetchBillTracker: empty billURL")
	}
	body, err := a.fetchURL(ctx, billURL, "bill_tracker")
	if err != nil {
		return nil, fmt.Errorf("parliament.FetchBillTracker: %w", err)
	}
	bt := ParseBillTrackerHTML(body, billURL)
	return bt, nil
}

// ParseStage maps Parliament's raw stage text (e.g., "Second Reading",
// "committee of the whole house", "presidential assent") to the platform's
// canonical stage codes defined in adapters/kenya/internal/stages.go
// (FIRST_READING, SECOND_READING, …).
//
// The mapping is case-insensitive and tolerant of common Parliament
// typography variants (e.g., "Committee of the Whole House" vs "Committee
// of Whole House"). Unknown inputs return an empty string — the caller
// SHOULD treat an empty stage as "could not be determined" and downgrade
// the confidence on any downstream ExtractedRecord.
func (a *Adapter) ParseStage(ctx context.Context, input string) string {
	return MapStageText(input)
}



// Discover implements contracts.LegislativeSourceAdapter. It discovers Bills
// (and, in future, Hansard / Order Papers / Votes & Proceedings) by
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
	meta := map[string]string{
		"house":       b.House,
		"institution": "Parliament of Kenya",
	}
	if b.PetitionURL != "" {
		meta["petition_url"] = b.PetitionURL
	}
	if b.BillDigestURL != "" {
		meta["bill_digest_url"] = b.BillDigestURL
	}
	if b.BillTrackerURL != "" {
		meta["bill_tracker_url"] = b.BillTrackerURL
	}
	return contracts.SourceItem{
		URL:          b.URL,
		Title:        b.Title,
		DocumentType: "bill",
		SourceType:   contracts.SourceItemBill,
		CountryCode:  "KE",
		SourceID:     b.SourceID,
		House:        b.House,
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
	body, err := a.fetchURL(ctx, item.URL, "bill")
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

// Parse implements contracts.LegislativeSourceAdapter. HTML Bill pages are
// parsed for stage + sponsor + publication date. PDFs are returned as a
// single low-confidence ExtractedRecord carrying only the source URL +
// RetrievedAt — full PDF text extraction is delegated to the documents
// service.
func (a *Adapter) Parse(ctx context.Context, doc contracts.RawDocument) ([]contracts.ExtractedRecord, error) {
	if doc.MimeType == "application/pdf" || strings.HasSuffix(strings.ToLower(doc.URL), ".pdf") {
		// We cannot parse the PDF here; the documents service handles it.
		return []contracts.ExtractedRecord{{
			Kind:          "bill",
			SourceURL:     doc.URL,
			RetrievedAt:   doc.RetrievedAt,
			ExtractorName: "kenya.parliament.PDFPlaceholder",
			Confidence:    0.1,
		}}, nil
	}
	return ParseBillDetail(string(doc.Bytes), doc.URL)
}

// NormalizeSourceItem implements contracts.LegislativeSourceAdapter. It
// accepts raw metadata maps (typically produced by external schedulers that
// have discovered URLs out of band) and projects them into the canonical
// SourceItem shape using the central Kenya normalizer.
func (a *Adapter) NormalizeSourceItem(raw map[string]any) (contracts.SourceItem, error) {
	return internal.NormalizeSourceItem(raw)
}

// GetLegislativeStructure implements contracts.LegislativeSourceAdapter. It
// returns the static Kenya legislative structure (bicameral National Assembly
// + Senate) from the central Kenya internal package.
func (a *Adapter) GetLegislativeStructure(ctx context.Context) (*contracts.LegislativeStructure, error) {
	s := internal.KenyaLegislativeStructure()
	return &s, nil
}

// GetStages implements contracts.LegislativeSourceAdapter. It returns the
// Kenya Bill stages (projected to contracts.StageDefinition via ToContract).
func (a *Adapter) GetStages(ctx context.Context) ([]contracts.StageDefinition, error) {
	stages := internal.KenyaBillStages
	out := make([]contracts.StageDefinition, len(stages))
	for i, s := range stages {
		out[i] = s.ToContract()
	}
	return out, nil
}

// GetTerminology implements contracts.LegislativeSourceAdapter. It returns
// the Kenya parliamentary terminology registry.
func (a *Adapter) GetTerminology(ctx context.Context) ([]contracts.TermDefinition, error) {
	terms := internal.KenyaTerminology
	out := make([]contracts.TermDefinition, len(terms))
	for i, t := range terms {
		out[i] = t.ToContract()
	}
	return out, nil
}

// makeSourceID derives a stable, deterministic SourceID for a Parliament Bill
// from its house and source URL. Example:
//
//	house="National Assembly", url="…/sites/default/files/2026-08/THE%20HERALDRY%20BILL%2C%202026..pdf"
//	→ "ke-parliament-na-bill-the-heraldry-bill-2026"
func makeSourceID(house, rawURL string) string {
	prefix := "ke-parliament-na-bill"
	if strings.EqualFold(house, "Senate") {
		prefix = "ke-parliament-senate-bill"
	}
	// Try to extract the last path segment of the URL.
	slug := rawURL
	if u, err := url.Parse(rawURL); err == nil && u.Path != "" {
		parts := strings.Split(strings.TrimRight(u.Path, "/"), "/")
		if len(parts) > 0 {
			slug = parts[len(parts)-1]
		}
	}
	// URL-decode and lowercase.
	slug = strings.ToLower(slug)
	// Strip a trailing ".pdf" extension if present.
	slug = strings.TrimSuffix(slug, ".pdf")
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
	return prefix + "-" + slug
}
