// Package president adapts the president.go.ke source — the official website
// of the President of the Republic of Kenya. It implements discovery +
// fetch + parse for Presidential Assent announcements (Article 115 of the
// Constitution of Kenya, 2010).
//
// When the President signs a Bill into law, the Office of the President
// publishes a press release on president.go.ke. The adapter detects these
// announcements and produces one AssentCandidate per BILL mentioned in the
// announcement — a single assent ceremony may cover several bills (e.g.,
// the Sep 8 2026 State House ceremony covered 4).
//
// This package is the authoritative source for the PRESIDENTIAL_ASSENT stage
// in the Kenya adapter set. The Parliament adapter owns the earlier stages
// (First Reading → ... → Third Reading / Mediation); this adapter takes
// over at the assent step and feeds the next stage (Commencement), which is
// in turn owned by the Kenya Gazette adapter.
package president

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Roy-Wanyoike/civic-intelligence/packages/contracts"
)

// StagePresidentialAssent is the canonical Kenya stage code that every record
// produced by this adapter carries in ExtractedRecord.Stage.
//
// It matches adapters/kenya/internal.StagePresidentialAssent. Duplicated here
// (rather than imported) so this package is a leaf source adapter with no
// dependency on the Kenya internal package — president is a sibling of
// parliament, kenya_law and gazette, not a child of the composite.
const StagePresidentialAssent = "PRESIDENTIAL_ASSENT"

// HTTPClient is the interface the president adapter uses to fetch remote
// content. Production uses a polite, rate-limited client (1 req/sec/host)
// to be a good citizen of the source website; tests use an in-memory
// implementation. This indirection is critical for SSRF protection and for
// testing offline.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// Adapter fetches Presidential Assent announcements from president.go.ke.
type Adapter struct {
	client    HTTPClient
	userAgent string
	baseURL   string
	// feedURL is the RSS endpoint used by DiscoverAssents. president.go.ke is
	// a WordPress instance, so /search/assent/feed/rss2/ returns a structured
	// RSS feed of every post whose content matches "assent".
	feedURL string
}

// NewAdapter creates a President.go.ke adapter.
//
// If client is nil, a default *http.Client with a 30-second timeout is used.
// In production, callers should pass a polite client (see parliament.PoliteClient
// for a reference implementation) to rate-limit requests.
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
		baseURL:   "https://www.president.go.ke",
		feedURL:   "https://www.president.go.ke/search/assent/feed/rss2/",
	}
}

// CountryCode returns "KE" — the ISO 3166-1 alpha-2 code for Kenya.
func (a *Adapter) CountryCode() string { return "KE" }

// Supports reports whether this adapter can handle the given URL. The
// President adapter handles URLs from president.go.ke (the official website
// of the President of the Republic of Kenya).
func (a *Adapter) Supports(url string) bool {
	return strings.Contains(url, "president.go.ke")
}

// DiscoverAssents crawls president.go.ke for assent announcements and returns
// a list of announcement-level candidates. Each candidate carries the
// announcement URL, the publication date, and the announcement's headline
// as a BillName hint.
//
// Callers should follow up with FetchAssent + ParseAssentDetail on each
// candidate to extract the precise per-bill names from the full HTML page.
//
// # Discovery strategy
//
// The president.go.ke WordPress instance exposes the search-as-RSS endpoint
// at /search/assent/feed/rss2/. This returns a well-formed RSS 2.0 feed of
// every post whose content matches "assent". The adapter parses this feed
// rather than scraping HTML — RSS is more stable across WordPress upgrades.
//
// If the RSS endpoint is unavailable, callers can fall back to fetching the
// HTML search page at /?s=assent and passing it to ParseAssentList, which
// also accepts HTML and extracts assent URLs from anchor tags.
func (a *Adapter) DiscoverAssents(ctx context.Context) ([]AssentCandidate, error) {
	body, err := a.fetchURL(ctx, a.feedURL, "application/rss+xml,application/xml,text/xml")
	if err != nil {
		return nil, fmt.Errorf("president.DiscoverAssents: %w", err)
	}
	return ParseAssentList(body), nil
}

// FetchAssent downloads the raw HTML of a single assent announcement page.
// The returned string is suitable for passing to ParseAssent.
func (a *Adapter) FetchAssent(ctx context.Context, url string) (string, error) {
	return a.fetchURL(ctx, url, "text/html,application/xhtml+xml")
}

// ParseAssent parses the HTML of an assent announcement and returns one
// AssentCandidate per BILL mentioned in the announcement. See
// ParseAssentDetail for the parsing contract.
func (a *Adapter) ParseAssent(html, url string) ([]AssentCandidate, error) {
	return ParseAssentDetail(html, url), nil
}

// SourceDefinition describes an official source the president adapter crawls.
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

// GetOfficialSources returns the source definitions for the President adapter.
func (a *Adapter) GetOfficialSources() []SourceDefinition {
	return []SourceDefinition{
		{
			ID:             "ke-president-assents",
			Country:        "KE",
			Institution:    "Office of the President of the Republic of Kenya",
			Authority:      "primary",
			URL:            a.feedURL,
			Adapter:        "kenya.president",
			DocumentTypes:  []string{"assent_announcement"},
			CrawlFrequency: 6 * time.Hour,
		},
	}
}

// fetchURL downloads a URL and returns the body as a string. Sets a polite
// User-Agent and an Accept header appropriate for the resource type.
func (a *Adapter) fetchURL(ctx context.Context, url, accept string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", a.userAgent)
	if accept != "" {
		req.Header.Set("Accept", accept)
	}
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

// Discover implements contracts.LegislativeSourceAdapter. It calls
// DiscoverAssents and projects each announcement candidate to a SourceItem
// of type "assent_announcement". The platform's ingestion service then
// fetches + parses each item via Fetch / Parse to extract the per-bill
// candidates.
func (a *Adapter) Discover(ctx context.Context) ([]contracts.SourceItem, error) {
	candidates, err := a.DiscoverAssents(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]contracts.SourceItem, len(candidates))
	now := time.Now().UTC()
	for i, c := range candidates {
		items[i] = contracts.SourceItem{
			URL:          c.SourceURL,
			Title:        c.BillName, // announcement headline as a hint
			DocumentType: "assent_announcement",
			SourceType:   contracts.SourceItemBill,
			Type:         contracts.SourceItemBill,
			CountryCode:  "KE",
			PublishedAt:  c.AssentDate,
			DiscoveredAt: now,
			SourceID:     c.SourceID,
		}
	}
	return items, nil
}

// Fetch implements contracts.LegislativeSourceAdapter.
func (a *Adapter) Fetch(ctx context.Context, item contracts.SourceItem) (*contracts.RawDocument, error) {
	if item.URL == "" {
		return nil, fmt.Errorf("president.Fetch: empty URL")
	}
	body, err := a.FetchAssent(ctx, item.URL)
	if err != nil {
		return nil, err
	}
	return &contracts.RawDocument{
		URL:         item.URL,
		Bytes:       []byte(body),
		MimeType:    "text/html",
		RetrievedAt: time.Now().UTC(),
		SourceID:    contracts.ID(item.SourceID),
	}, nil
}

// Parse implements contracts.LegislativeSourceAdapter. It produces one
// ExtractedRecord per BILL mentioned in the announcement, each carrying the
// PRESIDENTIAL_ASSENT stage code.
func (a *Adapter) Parse(ctx context.Context, doc contracts.RawDocument) ([]contracts.ExtractedRecord, error) {
	candidates := ParseAssentDetail(string(doc.Bytes), doc.URL)
	out := make([]contracts.ExtractedRecord, len(candidates))
	for i, c := range candidates {
		out[i] = c.ToExtractedRecord()
	}
	return out, nil
}

// NormalizeSourceItem implements contracts.LegislativeSourceAdapter.
func (a *Adapter) NormalizeSourceItem(raw map[string]any) (contracts.SourceItem, error) {
	return contracts.SourceItem{
		URL:          getString(raw, "url"),
		Title:        getString(raw, "title"),
		DocumentType: "assent_announcement",
		SourceType:   contracts.SourceItemBill,
		Type:         contracts.SourceItemBill,
		CountryCode:  "KE",
	}, nil
}

// GetLegislativeStructure implements contracts.LegislativeSourceAdapter.
// The President is not part of the bicameral legislature structure, so this
// returns a minimal record identifying the country.
func (a *Adapter) GetLegislativeStructure(ctx context.Context) (*contracts.LegislativeStructure, error) {
	s := contracts.LegislativeStructure{
		Country:     contracts.Country("KE"),
		CountryCode: "KE",
		CountryName: "Kenya",
	}
	return &s, nil
}

// GetStages implements contracts.LegislativeSourceAdapter. The President
// adapter is the authoritative source for the PRESIDENTIAL_ASSENT stage;
// earlier stages (First Reading → Third Reading / Mediation) come from the
// Parliament adapter.
func (a *Adapter) GetStages(ctx context.Context) ([]contracts.StageDefinition, error) {
	return []contracts.StageDefinition{
		{
			Code:                StagePresidentialAssent,
			Name:                "Presidential Assent",
			Description:         "The President signs the Bill into law per Article 115 of the Constitution of Kenya, 2010. The President may refer a Bill back once; if returned and re-passed, assent is mandatory.",
			SimpleExplanation:   "The President signs the Bill into law. May be referred back once; otherwise it becomes law.",
			Country:             contracts.Country("KE"),
			Order:               8,
			RequiresEvidence:    true,
			RequiresVote:        false,
			TypicalDurationDays: 14,
			AllowedNext:         []string{"COMMENCEMENT", "LAPSED"},
			AllowedTransitions:  []string{"COMMENCEMENT", "LAPSED"},
		},
	}, nil
}

// GetTerminology implements contracts.LegislativeSourceAdapter. The President
// adapter does not own any Kenya-specific terminology — that's the Parliament
// adapter's responsibility.
func (a *Adapter) GetTerminology(ctx context.Context) ([]contracts.TermDefinition, error) {
	return nil, nil
}

// ToExtractedRecord projects an AssentCandidate to the platform's canonical
// ExtractedRecord shape, tagged with the PRESIDENTIAL_ASSENT stage code and
// the official presidential source URL for evidence.
//
// The Confidence is set high (0.95) because this is the OFFICIAL primary
// source — when president.go.ke publishes an assent announcement, that is
// the canonical truth about whether the President signed the Bill.
func (c AssentCandidate) ToExtractedRecord() contracts.ExtractedRecord {
	return contracts.ExtractedRecord{
		Kind:          "assent",
		Title:         c.BillName,
		Stage:         StagePresidentialAssent,
		PublishedAt:   c.AssentDate,
		SourceURL:     c.SourceURL,
		RetrievedAt:   time.Now().UTC(),
		Confidence:    0.95, // high — official primary source
		ExtractorName: "kenya.president.AssentHTMLParser",
		House:         "", // the President signs bills from either house
		Extra: map[string]interface{}{
			"source_id": c.SourceID,
		},
	}
}

func getString(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

// Compile-time assertion: Adapter satisfies contracts.LegislativeSourceAdapter.
var _ contracts.LegislativeSourceAdapter = (*Adapter)(nil)
