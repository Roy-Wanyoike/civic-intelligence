package internal

import (
        "errors"
        "fmt"
        "strings"
        "time"

        "github.com/Roy-Wanyoike/civic-intelligence/packages/contracts"
)

// NormalizeSourceItem maps a raw key/value record harvested from a Kenyan
// parliamentary source into the platform's contracts.SourceItem shape.
//
// The raw map's keys are the names used on parliament.go.ke,
// nationalassembly.go.ke, senate.go.ke, kenyalaw.org and the Kenya Gazette
// portal. By centralising the mapping here, the ingestion service never has
// to learn Kenyan field names.
//
// Required fields: external_id, url, title. Anything missing returns a
// descriptive error so that the ingestion service can quarantine the record
// rather than silently dropping it.
func NormalizeSourceItem(raw map[string]any) (contracts.SourceItem, error) {
        if raw == nil {
                return contracts.SourceItem{}, errors.New("kenya normalizer: nil raw map")
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
                externalID = getStr("bill_no")
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
                // Kenya's tracker sometimes uses "bill_title" instead.
                title = getStr("bill_title")
        }
        if title == "" {
                missing = append(missing, "title")
        }

        if len(missing) > 0 {
                return contracts.SourceItem{}, fmt.Errorf("kenya normalizer: missing required field(s): %s", strings.Join(missing, ", "))
        }

        item := contracts.SourceItem{
                SourceID:     getStr("source_id"),
                CountryCode:  "KE",
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

        if item.SourceID == "" {
                item.SourceID = defaultSourceID(item.Type, externalID)
        }

        return item, nil
}

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

// mapSourceType maps Kenyan source-specific type strings to the platform's
// SourceItemType enum.
func mapSourceType(s string) contracts.SourceItemType {
        switch strings.ToLower(strings.TrimSpace(s)) {
        case "bill":
                return contracts.SourceItemBill
        case "hansard", "hansard_report", "official_report":
                return contracts.SourceItemHansard
        case "committee_report", "committee":
                return contracts.SourceItemCommittee
        case "gazette_notice", "gazette", "legal_notice":
                return contracts.SourceItemGazette
        case "act", "act_of_parliament":
                return contracts.SourceItemAct
        case "regulation", "subsidiary_legislation", "subsidiary legislation":
                return contracts.SourceItemRegulation
        case "policy":
                return contracts.SourceItemPolicy
        case "":
                return contracts.SourceItemUnknown
        default:
                return contracts.SourceItemUnknown
        }
}

// mapHouse maps the Kenyan house string ("National Assembly", "Senate", or
// their codes "NA"/"SEN") into the adapter's canonical house code.
func mapHouse(s string) string {
        switch strings.ToLower(strings.TrimSpace(s)) {
        case "national assembly", "nationalassembly", "na":
                return HouseCodeNationalAssembly
        case "senate", "sen":
                return HouseCodeSenate
        default:
                return ""
        }
}

// defaultSourceID generates a fallback SourceID for a record when the
// upstream source does not provide one.
func defaultSourceID(t contracts.SourceItemType, externalID string) string {
        prefix := "ke"
        switch t {
        case contracts.SourceItemBill:
                prefix = "ke-bill"
        case contracts.SourceItemHansard:
                prefix = "ke-hansard"
        case contracts.SourceItemCommittee:
                prefix = "ke-committee"
        case contracts.SourceItemGazette:
                prefix = "ke-gazette"
        case contracts.SourceItemAct:
                prefix = "ke-act"
        case contracts.SourceItemRegulation:
                prefix = "ke-reg"
        case contracts.SourceItemPolicy:
                prefix = "ke-policy"
        case contracts.SourceItemUnknown:
                prefix = "ke"
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

// timeParseFlexible tries the most common date layouts used by Kenyan
// parliamentary sites. Returns an error if none match.
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
        return time.Time{}, fmt.Errorf("kenya normalizer: cannot parse time %q", s)
}
