// Package south_africa implements the South Africa country adapter for the
// Civic Intelligence Platform. It implements contracts.LegislativeSourceAdapter.
//
// ALL South Africa-specific knowledge lives here:
//   - The 1996 Constitution's bicameral Parliament
//     (National Assembly + National Council of Provinces)
//   - South African Bill stages
//     (Introduction → Committee → Public Participation → NA Vote →
//      NCOP Concurrence → Presidential Assent → Commencement)
//   - South African parliamentary terminology
//     (Hansard, Order Paper, Portfolio Committee, Section 76 Bill, etc.)
//   - Source URLs for parliament.gov.za, gov.za (Government Gazette)
//
// The global domain model in services/legislation/ contains ZERO of these
// strings. Adding South Africa means writing adapters/south_africa/ — the
// legislation service code is unchanged.
package south_africa

import (
        "context"
        "errors"
        "fmt"
        "strings"
        "time"

        "github.com/Roy-Wanyoike/civic-intelligence/adapters/south_africa/internal"
        "github.com/Roy-Wanyoike/civic-intelligence/adapters/south_africa/parliament"
        "github.com/Roy-Wanyoike/civic-intelligence/packages/contracts"
)

// SouthAfricaAdapter implements contracts.LegislativeSourceAdapter.
// Compile-time assertion at the bottom guarantees interface compliance.
type SouthAfricaAdapter struct {
        parliament *parliament.Adapter
}

// Dependencies holds the collaborators the adapter needs.
type Dependencies struct {
        HTTPClient parliament.HTTPClient
        UserAgent  string
}

// NewSouthAfricaAdapter constructs a South Africa adapter.
func NewSouthAfricaAdapter(deps Dependencies) *SouthAfricaAdapter {
        if deps.UserAgent == "" {
                deps.UserAgent = "CivicIntelligence/0.1 (+https://github.com/Roy-Wanyoike/civic-intelligence)"
        }
        return &SouthAfricaAdapter{
                parliament: parliament.NewAdapter(deps.HTTPClient, deps.UserAgent),
        }
}

// CountryCode returns "ZA" — the ISO 3166-1 alpha-2 code for South Africa.
func (a *SouthAfricaAdapter) CountryCode() string { return "ZA" }

// Supports reports whether this adapter can handle the given URL. The South
// Africa adapter handles URLs from parliament.gov.za (with or without the
// "www." subdomain) and the Government Gazette on gov.za.
func (a *SouthAfricaAdapter) Supports(url string) bool {
        for _, host := range []string{
                "parliament.gov.za",
                "www.parliament.gov.za",
                "gov.za",
                "www.gov.za",
                "gazettes.africa",
        } {
                if strings.Contains(url, host) {
                        return true
                }
        }
        return false
}

// NormalizeSourceItem converts raw South Africa-source metadata into the
// platform's canonical SourceItem shape. It accepts the field names used on
// parliament.gov.za ("bill_number", "house", "portfolio_committee", etc.) and
// the Government Gazette.
//
// Required fields: external_id, url, title. Anything missing returns a
// descriptive error so that the ingestion service can quarantine the record
// rather than silently dropping it.
func (a *SouthAfricaAdapter) NormalizeSourceItem(raw map[string]any) (contracts.SourceItem, error) {
        if raw == nil {
                return contracts.SourceItem{}, errors.New("south_africa normalizer: nil raw map")
        }

        var (
                missing []string
                getStr  = func(k string) string {
                        if v, ok := raw[k]; ok {
                                if s, ok := v.(string); ok {
                                        return strings.TrimSpace(s)
                                }
                                return strings.TrimSpace(fmt.Sprint(v))
                        }
                        return ""
                }
        )

        externalID := getStr("external_id")
        if externalID == "" {
                externalID = getStr("bill_number")
        }
        if externalID == "" {
                missing = append(missing, "external_id")
        }

        url := getStr("url")
        if url == "" {
                missing = append(missing, "url")
        }

        title := getStr("title")
        if title == "" {
                // parliament.gov.za sometimes uses "bill_title" instead.
                title = getStr("bill_title")
        }
        if title == "" {
                missing = append(missing, "title")
        }

        if len(missing) > 0 {
                return contracts.SourceItem{}, fmt.Errorf("south_africa normalizer: missing required field(s): %s", strings.Join(missing, ", "))
        }

        item := contracts.SourceItem{
                SourceID:     getStr("source_id"),
                CountryCode:  "ZA",
                ExternalID:   externalID,
                Title:        title,
                Description:  getStr("description"),
                URL:          url,
                Type:         mapSourceType(getStr("type")),
                House:        mapHouse(firstNonEmpty(getStr("house"), getStr("chamber"))),
                PublishedAt:  parseTime(getStr("published_at"), getStr("date")),
                DiscoveredAt: time.Now().UTC(),
                ContentHash:  getStr("content_hash"),
                RawMetadata:  flattenMetadata(raw),
        }
        // If the source provided an explicit discovered_at, prefer it (it allows
        // idempotent re-crawls).
        if d := getStr("discovered_at"); d != "" {
                if t, err := timeParseFlexible(d); err == nil {
                        item.DiscoveredAt = t
                }
        }
        // DocumentType + SourceType default to bill for parliament.gov.za URLs;
        // allow callers to override via the "type" field.
        item.DocumentType = "bill"
        item.SourceType = item.Type
        // parliament.gov.za is primarily a Bills source. When the upstream record
        // doesn't carry an explicit "type", default to Bill so the SourceID prefix
        // is "za-bill:..." rather than the generic "za:...".
        if item.SourceType == "" || item.SourceType == contracts.SourceItemUnknown {
                item.SourceType = contracts.SourceItemBill
        }

        if item.SourceID == "" {
                item.SourceID = defaultSourceID(item.SourceType, externalID)
        }

        // Mirror Metadata so consumers reading either field see the same data.
        item.Metadata = item.RawMetadata

        return item, nil
}

// Discover delegates to the parliament source adapter and returns discovered
// SourceItems.
func (a *SouthAfricaAdapter) Discover(ctx context.Context) ([]contracts.SourceItem, error) {
        return a.parliament.Discover(ctx)
}

// Fetch downloads a single item's raw bytes from the upstream source.
func (a *SouthAfricaAdapter) Fetch(ctx context.Context, item contracts.SourceItem) (*contracts.RawDocument, error) {
        return a.parliament.Fetch(ctx, item)
}

// Parse converts a raw document into normalized ExtractedRecords.
func (a *SouthAfricaAdapter) Parse(ctx context.Context, doc contracts.RawDocument) ([]contracts.ExtractedRecord, error) {
        return a.parliament.Parse(ctx, doc)
}

// GetLegislativeStructure returns South Africa's institutions, legislature,
// houses, and committees. This is ADAPTER DATA — these strings are values, not
// constants in the global domain model.
func (a *SouthAfricaAdapter) GetLegislativeStructure(ctx context.Context) (*contracts.LegislativeStructure, error) {
        s := internal.SouthAfricaLegislativeStructure()
        return &s, nil
}

// GetStages returns the South African Bill stages per the 1996 Constitution
// and the Rules of the National Assembly and NCOP. The internal package
// stores these as a richer []SouthAfricaStage type; we project them down to
// []contracts.StageDefinition via each stage's ToContract() method.
func (a *SouthAfricaAdapter) GetStages(ctx context.Context) ([]contracts.StageDefinition, error) {
        stages := internal.SouthAfricaBillStages
        out := make([]contracts.StageDefinition, len(stages))
        for i, s := range stages {
                out[i] = s.ToContract()
        }
        return out, nil
}

// GetTerminology returns the South African parliamentary terminology registry.
// Same projection pattern as GetStages.
func (a *SouthAfricaAdapter) GetTerminology(ctx context.Context) ([]contracts.TermDefinition, error) {
        terms := internal.SouthAfricaTerminology
        out := make([]contracts.TermDefinition, len(terms))
        for i, t := range terms {
                out[i] = t.ToContract()
        }
        return out, nil
}

// ---------------------------------------------------------------------------
// normalizer helpers (kept here because internal/south_africa_data.go is
// restricted to stages + terminology + legislative structure per issue #157).
// ---------------------------------------------------------------------------

// firstNonEmpty returns the first non-empty (after trimming) string in s.
// Returns "" if all inputs are empty.
func firstNonEmpty(s ...string) string {
        for _, v := range s {
                if t := strings.TrimSpace(v); t != "" {
                        return t
                }
        }
        return ""
}

// mapSourceType maps South African source-specific type strings to the
// platform's SourceItemType enum.
func mapSourceType(s string) contracts.SourceItemType {
        switch strings.ToLower(strings.TrimSpace(s)) {
        case "bill":
                return contracts.SourceItemBill
        case "hansard", "hansard_report", "official_report":
                return contracts.SourceItemHansard
        case "committee_report", "committee":
                return contracts.SourceItemCommittee
        case "gazette_notice", "gazette", "government_gazette", "legal_notice":
                return contracts.SourceItemGazette
        case "act", "act_of_parliament":
                return contracts.SourceItemAct
        case "regulation", "subsidiary_legislation":
                return contracts.SourceItemRegulation
        case "policy":
                return contracts.SourceItemPolicy
        case "":
                return contracts.SourceItemUnknown
        default:
                return contracts.SourceItemUnknown
        }
}

// mapHouse maps the South African house string ("National Assembly",
// "National Council of Provinces", or their codes "NA"/"NCOP") into the
// adapter's canonical house code.
func mapHouse(s string) string {
        switch strings.ToLower(strings.TrimSpace(s)) {
        case "national assembly", "nationalassembly", "na":
                return internal.HouseCodeNationalAssembly
        case "national council of provinces", "nationalcouncilofprovinces", "ncop":
                return internal.HouseCodeNationalCouncilOfProvinces
        default:
                return ""
        }
}

// defaultSourceID generates a fallback SourceID for a record when the
// upstream source does not provide one.
func defaultSourceID(t contracts.SourceItemType, externalID string) string {
        prefix := "za"
        switch t {
        case contracts.SourceItemBill:
                prefix = "za-bill"
        case contracts.SourceItemHansard:
                prefix = "za-hansard"
        case contracts.SourceItemCommittee:
                prefix = "za-committee"
        case contracts.SourceItemGazette:
                prefix = "za-gazette"
        case contracts.SourceItemAct:
                prefix = "za-act"
        case contracts.SourceItemRegulation:
                prefix = "za-reg"
        case contracts.SourceItemPolicy:
                prefix = "za-policy"
        case contracts.SourceItemUnknown:
                prefix = "za"
        }
        if externalID == "" {
                return prefix
        }
        return prefix + ":" + externalID
}

// flattenMetadata copies the raw map's known values into a stable string
// representation. Unknown keys are passed through verbatim so no information
// is lost (the documents service may surface them later).
func flattenMetadata(raw map[string]any) map[string]string {
        out := make(map[string]string, len(raw))
        for k, v := range raw {
                switch v := v.(type) {
                case string:
                        if strings.TrimSpace(v) != "" {
                                out[k] = v
                        }
                case nil:
                        // skip
                default:
                        out[k] = fmt.Sprint(v)
                }
        }
        return out
}

// parseTime accepts zero, one or two input strings and tries multiple date
// layouts. Returns the zero time if nothing parseable was provided.
func parseTime(candidates ...string) time.Time {
        for _, c := range candidates {
                c = strings.TrimSpace(c)
                if c == "" {
                        continue
                }
                if t, err := timeParseFlexible(c); err == nil {
                        return t
                }
        }
        return time.Time{}
}

// timeParseFlexible tries the most common date layouts used by South African
// parliamentary and government sites. Returns an error if none match.
func timeParseFlexible(s string) (time.Time, error) {
        layouts := []string{
                time.RFC3339,
                "2006-01-02T15:04:05Z",
                "2006-01-02 15:04:05",
                "2006-01-02",
                "02/01/2006",
                "02 January 2006",
                "02 Jan 2006",
                "January 02, 2006",
                "Jan 02, 2006",
        }
        for _, l := range layouts {
                if t, err := time.Parse(l, s); err == nil {
                        return t.UTC(), nil
                }
        }
        return time.Time{}, fmt.Errorf("south_africa normalizer: cannot parse time %q", s)
}

// Compile-time assertion: SouthAfricaAdapter satisfies
// contracts.LegislativeSourceAdapter.
var _ contracts.LegislativeSourceAdapter = (*SouthAfricaAdapter)(nil)
