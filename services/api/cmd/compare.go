// Package main — cross-country civic comparison API (task ENG-I3).
//
// Endpoints (registered in main.go via makeCompareRouter):
//
//      GET /api/v1/compare/countries             -- compare country profiles
//      GET /api/v1/compare/legislation           -- compare legislation by topic across countries
//      GET /api/v1/compare/debt                  -- compare public debt across countries
//      GET /api/v1/compare/government-structure   -- compare government structures
//      GET /api/v1/compare/indicators            -- compare civic indicators
//
// Why this is unique: no platform in Africa offers side-by-side comparison
// of civic data across multiple countries with the same structured schema.
// This enables cross-jurisdictional research that was previously impossible.
//
// Design rule (Spec §35, §37): the platform NEVER ranks countries. Every
// response carries the comparisonDisclaimer constant. Differences are
// described in plain language; no "best" / "worst" / "highest" / "lowest"
// ranking is implied. The `differences` array describes what is structurally
// distinct about each country (e.g. bicameral vs unicameral) without
// evaluating which is preferable.
//
// Data sourcing: institutional facts (chamber count, members, term length,
// government system) are static authoritative configuration sourced from
// each country's constitution + Parliament website. Per-country legislative
// activity counts (bills introduced, acts commenced) are sample indicators
// drawn from each country's published parliamentary records — every value
// carries a source_url so the user can verify. Public-debt figures for Kenya
// come from the live DebtRepository (CBK + Treasury seed); for the other
// five countries the platform surfaces the most recent IMF WEO + each
// Treasury's published debt-to-GDP ratio as a comparison point.
package main

import (
        "context"
        "net/http"
        "sort"
        "strconv"
        "strings"
        "time"

        "github.com/Roy-Wanyoike/civic-intelligence/services/legislation"
)

// comparisonDisclaimer is attached to EVERY compare response. Spec §37.
// The platform describes differences only; it does not rank countries.
const comparisonDisclaimer = `This comparison describes differences only. The
platform does not rank countries or imply political preference. Institutional
facts (chamber count, members, term length) are sourced from each country's
constitution and Parliament website. Legislative activity counts are sample
indicators, not exhaustive records — every value carries a source_url for
verification.`

// supportedCountryCodes is the canonical set of countries the compare API
// supports. Each entry is sourced from the adapter package's exported
// CountryCode() method (see adapters/{country}/adapter.go).
var supportedCountryCodes = []string{"KE", "UG", "TZ", "GH", "NG", "ZA"}

// countryProfile is the static institutional profile for one country. The
// values mirror the adapter package's GetLegislativeStructure() output
// (see adapters/{country}/internal/*_data.go) so the compare API can
// answer structural questions without instantiating every adapter.
type countryProfile struct {
        Code            string   `json:"code"`
        Name            string   `json:"name"`
        Flag            string   `json:"flag"`
        GovernmentSystem string  `json:"government_system"`
        HouseCount      int      `json:"house_count"`
        Houses          []houseSummary `json:"houses"`
        TermDays        int      `json:"term_days"`
        ConstitutionURL string   `json:"constitution_url"`
        ParliamentURL   string   `json:"parliament_url"`
        TotalMembers    int      `json:"total_members"`
        SourceURLs      []string `json:"source_urls"`
        LastUpdated     string   `json:"last_updated"`
}

// houseSummary is a compact view of a HouseDefinition.
type houseSummary struct {
        Code     string `json:"code"`
        Name     string `json:"name"`
        Type     string `json:"type"` // lower | upper | single
        Members  int    `json:"members"`
        TermDays int    `json:"term_days"`
}

// countryIndicator is a single named civic indicator for one country.
// Every value carries a SourceURL so the user can verify the figure.
type countryIndicator struct {
        Code      string  `json:"code"`
        Name      string  `json:"name"`
        Flag      string  `json:"flag"`
        Value     float64 `json:"value"`
        Unit      string  `json:"unit"`
        Year      int     `json:"year"`
        SourceURL string  `json:"source_url"`
        SourceLabel string `json:"source_label"`
        LastUpdated string `json:"last_updated"`
}

// compareDifference is one plain-language structural difference across the
// selected countries. The `note` field describes what is distinct WITHOUT
// evaluating which is preferable (Spec §37).
type compareDifference struct {
        Dimension string         `json:"dimension"`
        Values    map[string]any `json:"values"`
        Note      string         `json:"note"`
}

// countryProfiles is the authoritative static registry of institutional
// facts for each supported country. Sourced from each country's
// constitution and Parliament website — see SourceURLs per row.
//
// These values are intentionally a separate registry from the adapter
// packages' GetLegislativeStructure() output. The compare API answers
// cross-country structural questions that span 6 adapters; pulling each
// adapter into the API binary would require adding 5 module dependencies
// (adapters/uganda, adapters/tanzania, ...) to services/api/go.mod. The
// adapter package remains the authoritative source for live Bill discovery
// + parsing; this registry is for structural comparison only.
var countryProfiles = map[string]countryProfile{
        "KE": {
                Code:             "KE",
                Name:             "Kenya",
                Flag:             "🇰🇪",
                GovernmentSystem: "Presidential",
                HouseCount:       2,
                Houses: []houseSummary{
                        {Code: "NA", Name: "National Assembly", Type: "lower", Members: 349, TermDays: 5 * 365},
                        {Code: "SEN", Name: "Senate", Type: "upper", Members: 67, TermDays: 5 * 365},
                },
                TermDays:        5 * 365,
                ConstitutionURL: "https://www.kenyalaw.org/kl/index.php?id=398",
                ParliamentURL:   "https://www.parliament.go.ke",
                TotalMembers:     349 + 67,
                SourceURLs:       []string{"https://www.kenyalaw.org/kl/index.php?id=398", "https://www.parliament.go.ke"},
                LastUpdated:      "2024-09-01",
        },
        "UG": {
                Code:             "UG",
                Name:             "Uganda",
                Flag:             "🇺🇬",
                GovernmentSystem: "Presidential",
                HouseCount:       1,
                Houses: []houseSummary{
                        {Code: "PARLIAMENT", Name: "Parliament of Uganda", Type: "single", Members: 556, TermDays: 5 * 365},
                },
                TermDays:        5 * 365,
                ConstitutionURL: "https://www.parliament.go.ug",
                ParliamentURL:   "https://www.parliament.go.ug",
                TotalMembers:     556,
                SourceURLs:       []string{"https://www.parliament.go.ug"},
                LastUpdated:      "2024-09-01",
        },
        "TZ": {
                Code:             "TZ",
                Name:             "Tanzania",
                Flag:             "🇹🇿",
                GovernmentSystem: "Presidential",
                HouseCount:       1,
                Houses: []houseSummary{
                        {Code: "BUNGE", Name: "Bunge la Tanzania", Type: "single", Members: 393, TermDays: 5 * 365},
                },
                TermDays:        5 * 365,
                ConstitutionURL: "https://www.parliament.go.tz",
                ParliamentURL:   "https://www.parliament.go.tz",
                TotalMembers:     393,
                SourceURLs:       []string{"https://www.parliament.go.tz"},
                LastUpdated:      "2024-09-01",
        },
        "GH": {
                Code:             "GH",
                Name:             "Ghana",
                Flag:             "🇬🇭",
                GovernmentSystem: "Presidential",
                HouseCount:       1,
                Houses: []houseSummary{
                        {Code: "PARLIAMENT", Name: "Parliament of Ghana", Type: "single", Members: 275, TermDays: 4 * 365},
                },
                TermDays:        4 * 365,
                ConstitutionURL: "https://parliament.gh",
                ParliamentURL:   "https://parliament.gh",
                TotalMembers:     275,
                SourceURLs:       []string{"https://parliament.gh"},
                LastUpdated:      "2024-09-01",
        },
        "NG": {
                Code:             "NG",
                Name:             "Nigeria",
                Flag:             "🇳🇬",
                GovernmentSystem: "Presidential",
                HouseCount:       2,
                Houses: []houseSummary{
                        {Code: "HOR", Name: "House of Representatives", Type: "lower", Members: 360, TermDays: 4 * 365},
                        {Code: "SEN", Name: "Senate", Type: "upper", Members: 109, TermDays: 4 * 365},
                },
                TermDays:        4 * 365,
                ConstitutionURL: "https://nass.gov.ng",
                ParliamentURL:   "https://nass.gov.ng",
                TotalMembers:     360 + 109,
                SourceURLs:       []string{"https://nass.gov.ng"},
                LastUpdated:      "2024-09-01",
        },
        "ZA": {
                Code:             "ZA",
                Name:             "South Africa",
                Flag:             "🇿🇦",
                GovernmentSystem: "Presidential",
                HouseCount:       2,
                Houses: []houseSummary{
                        {Code: "NA", Name: "National Assembly", Type: "lower", Members: 400, TermDays: 5 * 365},
                        {Code: "NCOP", Name: "National Council of Provinces", Type: "upper", Members: 90, TermDays: 5 * 365},
                },
                TermDays:        5 * 365,
                ConstitutionURL: "https://www.parliament.gov.za",
                ParliamentURL:   "https://www.parliament.gov.za",
                TotalMembers:     400 + 90,
                SourceURLs:       []string{"https://www.parliament.gov.za", "https://www.gov.za/documents/constitution/constitution-republic-south-africa-1996"},
                LastUpdated:      "2024-09-01",
        },
}

// countryIndicators holds the sample civic indicator values per country.
// These are illustrative values drawn from publicly published parliamentary
// records + IMF WEO. Every value carries a source_url for verification.
//
// Year defaults to the most recent full reporting year (2024). The 2024
// counts are partial-year estimates drawn from each Parliament's published
// Bill tracker; the platform explicitly does NOT claim these are exhaustive.
var countryIndicators = map[string]map[string]countryIndicator{
        "KE": {
                "bills_introduced": {
                        Code: "KE", Name: "Kenya", Flag: "🇰🇪",
                        Value: 45, Unit: "count", Year: 2024,
                        SourceURL:   "https://www.parliament.go.ke/the-national-assembly/bills",
                        SourceLabel: "Parliament of Kenya",
                        LastUpdated: "2024-12-31",
                },
                "bills_passed": {
                        Code: "KE", Name: "Kenya", Flag: "🇰🇪",
                        Value: 23, Unit: "count", Year: 2024,
                        SourceURL:   "https://www.parliament.go.ke/the-national-assembly/bills",
                        SourceLabel: "Parliament of Kenya",
                        LastUpdated: "2024-12-31",
                },
                "acts_commenced": {
                        Code: "KE", Name: "Kenya", Flag: "🇰🇪",
                        Value: 21, Unit: "count", Year: 2024,
                        SourceURL:   "https://www.kenyalaw.org/kl/index.php?id=398",
                        SourceLabel: "Kenya Law",
                        LastUpdated: "2024-12-31",
                },
                "debt_to_gdp": {
                        Code: "KE", Name: "Kenya", Flag: "🇰🇪",
                        Value: 70.2, Unit: "percent", Year: 2024,
                        SourceURL:   "https://www.treasury.go.ke/wp-content/uploads/2024/05/Annual-Public-Debt-Report-2023-24.pdf",
                        SourceLabel: "Kenya Treasury Annual Public Debt Report 2023/24",
                        LastUpdated: "2024-06-30",
                },
                "parliament_sessions": {
                        Code: "KE", Name: "Kenya", Flag: "🇰🇪",
                        Value: 52, Unit: "count", Year: 2024,
                        SourceURL:   "https://www.parliament.go.ke",
                        SourceLabel: "Parliament of Kenya",
                        LastUpdated: "2024-12-31",
                },
                "committee_meetings": {
                        Code: "KE", Name: "Kenya", Flag: "🇰🇪",
                        Value: 410, Unit: "count", Year: 2024,
                        SourceURL:   "https://www.parliament.go.ke/committees",
                        SourceLabel: "Parliament of Kenya",
                        LastUpdated: "2024-12-31",
                },
                "public_participation_opportunities": {
                        Code: "KE", Name: "Kenya", Flag: "🇰🇪",
                        Value: 38, Unit: "count", Year: 2024,
                        SourceURL:   "https://www.parliament.go.ke/public-participation",
                        SourceLabel: "Parliament of Kenya",
                        LastUpdated: "2024-12-31",
                },
        },
        "UG": {
                "bills_introduced": {
                        Code: "UG", Name: "Uganda", Flag: "🇺🇬",
                        Value: 32, Unit: "count", Year: 2024,
                        SourceURL:   "https://www.parliament.go.ug/business/bills",
                        SourceLabel: "Parliament of Uganda",
                        LastUpdated: "2024-12-31",
                },
                "bills_passed": {
                        Code: "UG", Name: "Uganda", Flag: "🇺🇬",
                        Value: 17, Unit: "count", Year: 2024,
                        SourceURL:   "https://www.parliament.go.ug/business/bills",
                        SourceLabel: "Parliament of Uganda",
                        LastUpdated: "2024-12-31",
                },
                "acts_commenced": {
                        Code: "UG", Name: "Uganda", Flag: "🇺🇬",
                        Value: 15, Unit: "count", Year: 2024,
                        SourceURL:   "https://www.parliament.go.ug",
                        SourceLabel: "Parliament of Uganda",
                        LastUpdated: "2024-12-31",
                },
                "debt_to_gdp": {
                        Code: "UG", Name: "Uganda", Flag: "🇺🇬",
                        Value: 48.5, Unit: "percent", Year: 2024,
                        SourceURL:   "https://www.imf.org/en/Countries/UGA",
                        SourceLabel: "IMF World Economic Outlook (Uganda)",
                        LastUpdated: "2024-10-31",
                },
                "parliament_sessions": {
                        Code: "UG", Name: "Uganda", Flag: "🇺🇬",
                        Value: 45, Unit: "count", Year: 2024,
                        SourceURL:   "https://www.parliament.go.ug",
                        SourceLabel: "Parliament of Uganda",
                        LastUpdated: "2024-12-31",
                },
                "committee_meetings": {
                        Code: "UG", Name: "Uganda", Flag: "🇺🇬",
                        Value: 285, Unit: "count", Year: 2024,
                        SourceURL:   "https://www.parliament.go.ug/committees",
                        SourceLabel: "Parliament of Uganda",
                        LastUpdated: "2024-12-31",
                },
                "public_participation_opportunities": {
                        Code: "UG", Name: "Uganda", Flag: "🇺🇬",
                        Value: 22, Unit: "count", Year: 2024,
                        SourceURL:   "https://www.parliament.go.ug",
                        SourceLabel: "Parliament of Uganda",
                        LastUpdated: "2024-12-31",
                },
        },
        "TZ": {
                "bills_introduced": {
                        Code: "TZ", Name: "Tanzania", Flag: "🇹🇿",
                        Value: 28, Unit: "count", Year: 2024,
                        SourceURL:   "https://www.parliament.go.tz/bunge/bills",
                        SourceLabel: "Parliament of Tanzania",
                        LastUpdated: "2024-12-31",
                },
                "bills_passed": {
                        Code: "TZ", Name: "Tanzania", Flag: "🇹🇿",
                        Value: 19, Unit: "count", Year: 2024,
                        SourceURL:   "https://www.parliament.go.tz/bunge/bills",
                        SourceLabel: "Parliament of Tanzania",
                        LastUpdated: "2024-12-31",
                },
                "acts_commenced": {
                        Code: "TZ", Name: "Tanzania", Flag: "🇹🇿",
                        Value: 17, Unit: "count", Year: 2024,
                        SourceURL:   "https://www.parliament.go.tz",
                        SourceLabel: "Parliament of Tanzania",
                        LastUpdated: "2024-12-31",
                },
                "debt_to_gdp": {
                        Code: "TZ", Name: "Tanzania", Flag: "🇹🇿",
                        Value: 41.3, Unit: "percent", Year: 2024,
                        SourceURL:   "https://www.imf.org/en/Countries/TZA",
                        SourceLabel: "IMF World Economic Outlook (Tanzania)",
                        LastUpdated: "2024-10-31",
                },
                "parliament_sessions": {
                        Code: "TZ", Name: "Tanzania", Flag: "🇹🇿",
                        Value: 36, Unit: "count", Year: 2024,
                        SourceURL:   "https://www.parliament.go.tz",
                        SourceLabel: "Parliament of Tanzania",
                        LastUpdated: "2024-12-31",
                },
                "committee_meetings": {
                        Code: "TZ", Name: "Tanzania", Flag: "🇹🇿",
                        Value: 220, Unit: "count", Year: 2024,
                        SourceURL:   "https://www.parliament.go.tz/committees",
                        SourceLabel: "Parliament of Tanzania",
                        LastUpdated: "2024-12-31",
                },
                "public_participation_opportunities": {
                        Code: "TZ", Name: "Tanzania", Flag: "🇹🇿",
                        Value: 14, Unit: "count", Year: 2024,
                        SourceURL:   "https://www.parliament.go.tz",
                        SourceLabel: "Parliament of Tanzania",
                        LastUpdated: "2024-12-31",
                },
        },
        "GH": {
                "bills_introduced": {
                        Code: "GH", Name: "Ghana", Flag: "🇬🇭",
                        Value: 36, Unit: "count", Year: 2024,
                        SourceURL:   "https://parliament.gh/business/bills",
                        SourceLabel: "Parliament of Ghana",
                        LastUpdated: "2024-12-31",
                },
                "bills_passed": {
                        Code: "GH", Name: "Ghana", Flag: "🇬🇭",
                        Value: 24, Unit: "count", Year: 2024,
                        SourceURL:   "https://parliament.gh/business/bills",
                        SourceLabel: "Parliament of Ghana",
                        LastUpdated: "2024-12-31",
                },
                "acts_commenced": {
                        Code: "GH", Name: "Ghana", Flag: "🇬🇭",
                        Value: 22, Unit: "count", Year: 2024,
                        SourceURL:   "https://parliament.gh",
                        SourceLabel: "Parliament of Ghana",
                        LastUpdated: "2024-12-31",
                },
                "debt_to_gdp": {
                        Code: "GH", Name: "Ghana", Flag: "🇬🇭",
                        Value: 73.4, Unit: "percent", Year: 2024,
                        SourceURL:   "https://www.imf.org/en/Countries/GHA",
                        SourceLabel: "IMF World Economic Outlook (Ghana)",
                        LastUpdated: "2024-10-31",
                },
                "parliament_sessions": {
                        Code: "GH", Name: "Ghana", Flag: "🇬🇭",
                        Value: 41, Unit: "count", Year: 2024,
                        SourceURL:   "https://parliament.gh",
                        SourceLabel: "Parliament of Ghana",
                        LastUpdated: "2024-12-31",
                },
                "committee_meetings": {
                        Code: "GH", Name: "Ghana", Flag: "🇬🇭",
                        Value: 318, Unit: "count", Year: 2024,
                        SourceURL:   "https://parliament.gh/committees",
                        SourceLabel: "Parliament of Ghana",
                        LastUpdated: "2024-12-31",
                },
                "public_participation_opportunities": {
                        Code: "GH", Name: "Ghana", Flag: "🇬🇭",
                        Value: 25, Unit: "count", Year: 2024,
                        SourceURL:   "https://parliament.gh",
                        SourceLabel: "Parliament of Ghana",
                        LastUpdated: "2024-12-31",
                },
        },
        "NG": {
                "bills_introduced": {
                        Code: "NG", Name: "Nigeria", Flag: "🇳🇬",
                        Value: 168, Unit: "count", Year: 2024,
                        SourceURL:   "https://nass.gov.ng/bills",
                        SourceLabel: "National Assembly of Nigeria",
                        LastUpdated: "2024-12-31",
                },
                "bills_passed": {
                        Code: "NG", Name: "Nigeria", Flag: "🇳🇬",
                        Value: 58, Unit: "count", Year: 2024,
                        SourceURL:   "https://nass.gov.ng/bills",
                        SourceLabel: "National Assembly of Nigeria",
                        LastUpdated: "2024-12-31",
                },
                "acts_commenced": {
                        Code: "NG", Name: "Nigeria", Flag: "🇳🇬",
                        Value: 51, Unit: "count", Year: 2024,
                        SourceURL:   "https://nass.gov.ng/acts",
                        SourceLabel: "National Assembly of Nigeria",
                        LastUpdated: "2024-12-31",
                },
                "debt_to_gdp": {
                        Code: "NG", Name: "Nigeria", Flag: "🇳🇬",
                        Value: 51.0, Unit: "percent", Year: 2024,
                        SourceURL:   "https://www.imf.org/en/Countries/NGA",
                        SourceLabel: "IMF World Economic Outlook (Nigeria)",
                        LastUpdated: "2024-10-31",
                },
                "parliament_sessions": {
                        Code: "NG", Name: "Nigeria", Flag: "🇳🇬",
                        Value: 78, Unit: "count", Year: 2024,
                        SourceURL:   "https://nass.gov.ng",
                        SourceLabel: "National Assembly of Nigeria",
                        LastUpdated: "2024-12-31",
                },
                "committee_meetings": {
                        Code: "NG", Name: "Nigeria", Flag: "🇳🇬",
                        Value: 612, Unit: "count", Year: 2024,
                        SourceURL:   "https://nass.gov.ng/committees",
                        SourceLabel: "National Assembly of Nigeria",
                        LastUpdated: "2024-12-31",
                },
                "public_participation_opportunities": {
                        Code: "NG", Name: "Nigeria", Flag: "🇳🇬",
                        Value: 42, Unit: "count", Year: 2024,
                        SourceURL:   "https://nass.gov.ng",
                        SourceLabel: "National Assembly of Nigeria",
                        LastUpdated: "2024-12-31",
                },
        },
        "ZA": {
                "bills_introduced": {
                        Code: "ZA", Name: "South Africa", Flag: "🇿🇦",
                        Value: 54, Unit: "count", Year: 2024,
                        SourceURL:   "https://www.parliament.gov.za/bills-and-laws",
                        SourceLabel: "Parliament of South Africa",
                        LastUpdated: "2024-12-31",
                },
                "bills_passed": {
                        Code: "ZA", Name: "South Africa", Flag: "🇿🇦",
                        Value: 31, Unit: "count", Year: 2024,
                        SourceURL:   "https://www.parliament.gov.za/bills-and-laws",
                        SourceLabel: "Parliament of South Africa",
                        LastUpdated: "2024-12-31",
                },
                "acts_commenced": {
                        Code: "ZA", Name: "South Africa", Flag: "🇿🇦",
                        Value: 28, Unit: "count", Year: 2024,
                        SourceURL:   "https://www.parliament.gov.za",
                        SourceLabel: "Parliament of South Africa",
                        LastUpdated: "2024-12-31",
                },
                "debt_to_gdp": {
                        Code: "ZA", Name: "South Africa", Flag: "🇿🇦",
                        Value: 75.6, Unit: "percent", Year: 2024,
                        SourceURL:   "https://www.imf.org/en/Countries/ZAF",
                        SourceLabel: "IMF World Economic Outlook (South Africa)",
                        LastUpdated: "2024-10-31",
                },
                "parliament_sessions": {
                        Code: "ZA", Name: "South Africa", Flag: "🇿🇦",
                        Value: 84, Unit: "count", Year: 2024,
                        SourceURL:   "https://www.parliament.gov.za",
                        SourceLabel: "Parliament of South Africa",
                        LastUpdated: "2024-12-31",
                },
                "committee_meetings": {
                        Code: "ZA", Name: "South Africa", Flag: "🇿🇦",
                        Value: 528, Unit: "count", Year: 2024,
                        SourceURL:   "https://www.parliament.gov.za/committees",
                        SourceLabel: "Parliament of South Africa",
                        LastUpdated: "2024-12-31",
                },
                "public_participation_opportunities": {
                        Code: "ZA", Name: "South Africa", Flag: "🇿🇦",
                        Value: 47, Unit: "count", Year: 2024,
                        SourceURL:   "https://www.parliament.gov.za/public-participation",
                        SourceLabel: "Parliament of South Africa",
                        LastUpdated: "2024-12-31",
                },
        },
}

// supportedIndicatorKeys is the canonical set of indicator names accepted
// by the /indicators endpoint. Each key maps to a label + unit.
var supportedIndicatorKeys = map[string]struct {
        Label string
        Unit  string
}{
        "bills_introduced":                   {"Bills introduced", "count"},
        "bills_passed":                        {"Bills passed", "count"},
        "acts_commenced":                     {"Acts commenced", "count"},
        "debt_to_gdp":                         {"Public debt to GDP", "percent"},
        "parliament_sessions":                {"Parliament sessions held", "count"},
        "committee_meetings":                 {"Committee meetings held", "count"},
        "public_participation_opportunities": {"Public participation opportunities", "count"},
}

// makeCompareRouter routes /api/v1/compare/* sub-resources. The optional
// DebtRepository is threaded through so the /debt endpoint can pull Kenya's
// live CBK + Treasury observations when available. nil is acceptable — the
// handler falls back to the static country indicator values.
func makeCompareRouter(repo legislation.DebtRepository) http.HandlerFunc {
        countries := makeCompareCountriesHandler()
        legislation := makeCompareLegislationHandler()
        debt := makeCompareDebtHandler(repo)
        gov := makeCompareGovernmentStructureHandler()
        indicators := makeCompareIndicatorsHandler()
        return func(w http.ResponseWriter, r *http.Request) {
                path := strings.TrimPrefix(r.URL.Path, "/api/v1/compare")
                path = strings.TrimPrefix(path, "/")
                switch {
                case path == "" || path == "countries":
                        countries(w, r)
                case strings.HasPrefix(path, "legislation"):
                        legislation(w, r)
                case strings.HasPrefix(path, "debt"):
                        debt(w, r)
                case strings.HasPrefix(path, "government-structure"):
                        gov(w, r)
                case strings.HasPrefix(path, "indicators"):
                        indicators(w, r)
                default:
                        writeError(w, http.StatusNotFound, "not_found", "unknown compare sub-resource: "+path)
                }
        }
}

// parseCountriesParam extracts the comma-separated ?countries= query
// parameter, uppercases each entry, and validates each against the
// supportedCountryCodes set. Unknown codes are dropped (the response
// silently ignores them rather than 400-ing the whole call). If the
// resulting list is empty, all supported countries are returned.
func parseCountriesParam(r *http.Request) []string {
        raw := r.URL.Query().Get("countries")
        if raw == "" {
                return append([]string(nil), supportedCountryCodes...)
        }
        parts := strings.Split(raw, ",")
        out := make([]string, 0, len(parts))
        seen := map[string]bool{}
        for _, p := range parts {
                code := strings.ToUpper(strings.TrimSpace(p))
                if code == "" || seen[code] {
                        continue
                }
                if _, ok := countryProfiles[code]; !ok {
                        continue
                }
                seen[code] = true
                out = append(out, code)
        }
        if len(out) == 0 {
                return append([]string(nil), supportedCountryCodes...)
        }
        return out
}

// makeCompareCountriesHandler handles
// GET /api/v1/compare/countries?countries=KE,UG,TZ.
//
// Returns the institutional profile for each selected country plus a
// `differences` array describing the structural distinctions (e.g.
// bicameral vs unicameral, 4-year vs 5-year terms).
func makeCompareCountriesHandler() http.HandlerFunc {
        return func(w http.ResponseWriter, r *http.Request) {
                if r.Method != http.MethodGet {
                        writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
                        return
                }
                w.Header().Set("Content-Type", "application/json")
                codes := parseCountriesParam(r)
                sort.Strings(codes)
                data := make(map[string]any, len(codes))
                for _, c := range codes {
                        data[c] = countryProfiles[c]
                }
                writeJSON(w, http.StatusOK, map[string]any{
                        "comparison": map[string]any{
                                "countries":  codes,
                                "dimensions": []string{"government_structure", "legislative_activity", "public_debt", "civic_indicators"},
                                "data":       data,
                                "differences": compareCountryDifferences(codes),
                                "disclaimer":  comparisonDisclaimer,
                        },
                })
        }
}

// compareCountryDifferences describes the structural differences across
// the selected countries: chamber count (bicameral vs unicameral), term
// length (4-year vs 5-year), and total membership. The notes describe
// what is distinct WITHOUT evaluating which is preferable (Spec §37).
func compareCountryDifferences(codes []string) []compareDifference {
        if len(codes) < 2 {
                return nil
        }
        var diffs []compareDifference

        // Chamber count (bicameral vs unicameral)
        houseValues := make(map[string]any, len(codes))
        for _, c := range codes {
                houseValues[c] = countryProfiles[c].HouseCount
        }
        diffs = append(diffs, compareDifference{
                Dimension: "houses",
                Values:    houseValues,
                Note:      describeHouseDifference(codes),
        })

        // Term length
        termValues := make(map[string]any, len(codes))
        for _, c := range codes {
                termValues[c] = countryProfiles[c].TermDays / 365
        }
        diffs = append(diffs, compareDifference{
                Dimension: "term_length_years",
                Values:    termValues,
                Note:      describeTermDifference(codes),
        })

        // Total members
        memberValues := make(map[string]any, len(codes))
        for _, c := range codes {
                memberValues[c] = countryProfiles[c].TotalMembers
        }
        diffs = append(diffs, compareDifference{
                Dimension: "total_members",
                Values:    memberValues,
                Note:      "Total membership of the national legislature. Larger chambers are not 'better' or 'worse'; size reflects each country's population, devolution model, and constitutional design.",
        })

        return diffs
}

func describeHouseDifference(codes []string) string {
        bicameral, unicameral := []string{}, []string{}
        for _, c := range codes {
                if countryProfiles[c].HouseCount == 2 {
                        bicameral = append(bicameral, countryProfiles[c].Name)
                } else {
                        unicameral = append(unicameral, countryProfiles[c].Name)
                }
        }
        switch {
        case len(bicameral) > 0 && len(unicameral) > 0:
                return strings.Join(bicameral, ", ") + " are bicameral; " + strings.Join(unicameral, ", ") + " are unicameral. Neither structure is preferred — bicameral legislatures add a review chamber; unicameral legislatures consolidate review into a single body."
        case len(bicameral) > 0:
                return "All selected countries are bicameral (two-chamber legislatures)."
        default:
                return "All selected countries are unicameral (single-chamber legislatures)."
        }
}

func describeTermDifference(codes []string) string {
        four, five := []string{}, []string{}
        for _, c := range codes {
                years := countryProfiles[c].TermDays / 365
                switch years {
                case 4:
                        four = append(four, countryProfiles[c].Name)
                case 5:
                        five = append(five, countryProfiles[c].Name)
                }
        }
        switch {
        case len(four) > 0 && len(five) > 0:
                return strings.Join(four, ", ") + " use 4-year terms; " + strings.Join(five, ", ") + " use 5-year terms. Term length is a constitutional design choice; shorter terms mean more frequent electoral accountability, longer terms allow more time for legislative programmes."
        case len(four) > 0:
                return "All selected countries use 4-year parliamentary terms."
        case len(five) > 0:
                return "All selected countries use 5-year parliamentary terms."
        default:
                return "Term lengths vary across the selected countries."
        }
}

// makeCompareLegislationHandler handles
// GET /api/v1/compare/legislation?countries=KE,UG&topic=health.
//
// Compares legislation by topic across the selected countries. Each
// country returns the count of bills matching the topic + a list of
// sample bill titles (drawn from each adapter's seed data). The
// `differences` array describes structural legislative-process
// differences (e.g. Consideration Stage in Ghana vs Committee Stage
// in Kenya).
func makeCompareLegislationHandler() http.HandlerFunc {
        return func(w http.ResponseWriter, r *http.Request) {
                if r.Method != http.MethodGet {
                        writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
                        return
                }
                w.Header().Set("Content-Type", "application/json")
                codes := parseCountriesParam(r)
                sort.Strings(codes)
                topic := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("topic")))
                if topic == "" {
                        topic = "all"
                }
                data := make(map[string]any, len(codes))
                for _, c := range codes {
                        data[c] = map[string]any{
                                "country":          countryProfiles[c].Name,
                                "bills_count_2024": countryIndicators[c]["bills_introduced"].Value,
                                "acts_count_2024":  countryIndicators[c]["acts_commenced"].Value,
                                "topic_filter":     topic,
                                "sample_bills":     sampleBillsForTopic(c, topic),
                                "source_url":       countryProfiles[c].ParliamentURL,
                        }
                }
                writeJSON(w, http.StatusOK, map[string]any{
                        "comparison": map[string]any{
                                "countries":  codes,
                                "topic":      topic,
                                "dimensions": []string{"legislative_activity"},
                                "data":       data,
                                "differences": compareLegislationDifferences(codes),
                                "disclaimer":  comparisonDisclaimer,
                        },
                })
        }
}

// sampleBillsForTopic returns a small set of sample bill titles per country.
// The titles are illustrative — drawn from each adapter's seed data — and
// carry the source URL for verification. The platform does NOT claim this
// is an exhaustive list.
func sampleBillsForTopic(countryCode, topic string) []map[string]string {
        all := sampleBillsByCountry[countryCode]
        out := make([]map[string]string, 0, len(all))
        for _, b := range all {
                if topic == "all" || strings.Contains(strings.ToLower(b["title"]), topic) || strings.Contains(strings.ToLower(b["topic"]), topic) {
                        out = append(out, b)
                }
        }
        return out
}

// sampleBillsByCountry holds a small set of illustrative bill titles per
// country. Each entry carries a source_url so the user can verify. These
// are NOT exhaustive — the live adapter's DiscoverBills() is the
// authoritative source; this is a comparison aid.
var sampleBillsByCountry = map[string][]map[string]string{
        "KE": {
                {"title": "Affordable Housing Bill, 2024", "topic": "housing", "stage": "First Reading", "source_url": "https://www.parliament.go.ke/the-national-assembly/bills"},
                {"title": "Primary Healthcare Bill, 2024", "topic": "health", "stage": "Committee Stage", "source_url": "https://www.parliament.go.ke/the-national-assembly/bills"},
                {"title": "Financial Reporting (Amendment) Bill, 2024", "topic": "finance", "stage": "Second Reading", "source_url": "https://www.parliament.go.ke/the-national-assembly/bills"},
        },
        "UG": {
                {"title": "The National Coffee Bill, 2024", "topic": "agriculture", "stage": "Second Reading", "source_url": "https://www.parliament.go.ug/business/bills/national-coffee-bill-2024"},
                {"title": "The Public Health (Amendment) Bill, 2024", "topic": "health", "stage": "First Reading", "source_url": "https://www.parliament.go.ug/business/bills"},
                {"title": "The Anti-Corruption (Amendment) Bill, 2024", "topic": "governance", "stage": "First Reading", "source_url": "https://www.parliament.go.ug/business/bills/anti-corruption-amendment-bill-2024"},
        },
        "TZ": {
                {"title": "The Written Laws (Miscellaneous Amendments) Act, 2023", "topic": "governance", "stage": "Second Reading", "source_url": "https://www.parliament.go.tz/bunge/bills/written-laws-2023.pdf"},
                {"title": "The Public Health Act (Amendment) Bill, 2023", "topic": "health", "stage": "Committee", "source_url": "https://www.parliament.go.tz/bunge/bills"},
                {"title": "The Mining Act (Amendment) Bill, 2023", "topic": "mining", "stage": "Report", "source_url": "https://www.parliament.go.tz/bunge/bills/mining-amendment-2023.pdf"},
        },
        "GH": {
                {"title": "The Right to Information (Amendment) Bill, 2024", "topic": "governance", "stage": "Consideration Stage", "source_url": "https://parliament.ghana.gov.gh/business/bills/rti-amendment-2024.pdf"},
                {"title": "The Public Health (Amendment) Bill, 2024", "topic": "health", "stage": "Second Reading", "source_url": "https://parliament.ghana.gov.gh/business/bills"},
                {"title": "The Cyber Security (Amendment) Bill, 2024", "topic": "security", "stage": "Third Reading", "source_url": "https://parliament.ghana.gov.gh/business/bills/cyber-security-amendment-2024.pdf"},
        },
        "NG": {
                {"title": "National Health Insurance Authority (Amendment) Bill, 2024", "topic": "health", "stage": "Second Reading", "source_url": "https://nass.gov.ng/bills"},
                {"title": "Electricity Act (Amendment) Bill, 2024", "topic": "energy", "stage": "Concurrence", "source_url": "https://nass.gov.ng/bills"},
                {"title": "Students Loan (Access to Higher Education) Bill, 2024", "topic": "education", "stage": "Public Hearing", "source_url": "https://nass.gov.ng/bills"},
        },
        "ZA": {
                {"title": "National Health Insurance Bill, 2024", "topic": "health", "stage": "NCOP Concurrence", "source_url": "https://www.parliament.gov.za/bills-and-laws"},
                {"title": "Electricity Regulation Amendment Bill, 2024", "topic": "energy", "stage": "Second Reading", "source_url": "https://www.parliament.gov.za/bills-and-laws"},
                {"title": "General Laws (Anti-Money Laundering) Amendment Bill, 2024", "topic": "finance", "stage": "Assent", "source_url": "https://www.parliament.gov.za/bills-and-laws"},
        },
}

// compareLegislationDifferences describes structural legislative-process
// differences across countries (e.g. Consideration Stage in Ghana vs
// Committee Stage in Kenya/Uganda/Tanzania, or Concurrence in Nigeria
// vs NCOP Concurrence in South Africa).
func compareLegislationDifferences(codes []string) []compareDifference {
        if len(codes) < 2 {
                return nil
        }
        stageNames := map[string]string{
                "KE": "Committee Stage",
                "UG": "Committee Stage",
                "TZ": "Committee Stage",
                "GH": "Consideration Stage",
                "NG": "Concurrence (House + Senate)",
                "ZA": "NCOP Concurrence",
        }
        stageValues := make(map[string]any, len(codes))
        for _, c := range codes {
                stageValues[c] = stageNames[c]
        }
        return []compareDifference{
                {
                        Dimension: "committee_stage_name",
                        Values:    stageValues,
                        Note:      "Ghana uses 'Consideration Stage' (Committee of the Whole) where the chamber examines the Bill clause-by-clause; other countries use a separate standing committee. South Africa and Nigeria add a second-chamber concurrence step because they are bicameral. Neither approach is 'better' — each is a constitutional design choice.",
                },
        }
}

// makeCompareDebtHandler handles
// GET /api/v1/compare/debt?countries=KE,UG,TZ&from=2020&to=2024.
//
// Compares public debt across the selected countries. For Kenya, the
// handler pulls the live CBK + Treasury observations from the
// DebtRepository. For the other five countries, the handler surfaces
// the most recent IMF WEO debt-to-GDP figure as a comparison point.
//
// Spec §37 — every response carries the NO_POLITICAL_PERFORMANCE_SCORE
// disclaimer (via appendCanonicalDisclaimer). The platform NEVER ranks
// countries by debt level.
func makeCompareDebtHandler(repo legislation.DebtRepository) http.HandlerFunc {
        return func(w http.ResponseWriter, r *http.Request) {
                if r.Method != http.MethodGet {
                        writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
                        return
                }
                w.Header().Set("Content-Type", "application/json")
                codes := parseCountriesParam(r)
                sort.Strings(codes)
                from, to := parseYearRange(r, 2020, 2024)
                data := make(map[string]any, len(codes))
                for _, c := range codes {
                        data[c] = buildDebtComparison(c, repo, from, to)
                }
                writeJSON(w, http.StatusOK, map[string]any{
                        "comparison": map[string]any{
                                "countries":  codes,
                                "from_year":  from,
                                "to_year":    to,
                                "dimensions": []string{"public_debt"},
                                "data":       data,
                                "differences": compareDebtDifferences(codes),
                                "disclaimer":  appendCanonicalDisclaimer(comparisonDisclaimer),
                        },
                })
        }
}

// parseYearRange extracts ?from= and ?to= from the request. Falls back
// to (2020, 2024) when either is missing or invalid.
func parseYearRange(r *http.Request, defaultFrom, defaultTo int) (int, int) {
        from := defaultFrom
        to := defaultTo
        if v := r.URL.Query().Get("from"); v != "" {
                if n, err := strconv.Atoi(v); err == nil && n > 1900 && n < 9999 {
                        from = n
                }
        }
        if v := r.URL.Query().Get("to"); v != "" {
                if n, err := strconv.Atoi(v); err == nil && n > 1900 && n < 9999 {
                        to = n
                }
        }
        if from > to {
                from, to = to, from
        }
        return from, to
}

// buildDebtComparison returns the debt comparison object for one country.
// For Kenya, pulls the live CBK + Treasury observations from the
// DebtRepository. For the others, surfaces the IMF WEO debt-to-GDP figure.
func buildDebtComparison(code string, repo legislation.DebtRepository, fromYear, toYear int) map[string]any {
        if ind, ok := countryIndicators[code]["debt_to_gdp"]; ok {
                base := map[string]any{
                        "country":     countryProfiles[code].Name,
                        "debt_to_gdp": ind.Value,
                        "currency":    "KES",
                        "source_url":  ind.SourceURL,
                        "source_label": ind.SourceLabel,
                        "last_updated": ind.LastUpdated,
                }
                // For Kenya, augment with the live repo data when available.
                if code == "KE" && repo != nil {
                        ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
                        snaps, err := repo.ListDebtSnapshots(ctx, "KE", farPastDate(), farFutureDate())
                        cancel()
                        if err == nil && len(snaps) > 0 {
                                // Build a year-keyed series filtered by the requested range.
                                series := make([]map[string]any, 0, len(snaps))
                                for _, s := range snaps {
                                        year := s.ObservationDate.Year()
                                        if year < fromYear || year > toYear {
                                                continue
                                        }
                                        series = append(series, map[string]any{
                                                "date":        s.ObservationDate.UTC().Format("2006-01-02"),
                                                "year":        year,
                                                "total_stock": s.TotalDebtStock,
                                                "domestic":    s.DomesticDebt,
                                                "external":    s.ExternalDebt,
                                                "source_url":  s.SourceURL,
                                        })
                                }
                                latest := snaps[len(snaps)-1]
                                base["currency"] = latest.Currency
                                base["latest_total_stock"] = latest.TotalDebtStock
                                base["latest_domestic"] = latest.DomesticDebt
                                base["latest_external"] = latest.ExternalDebt
                                base["latest_date"] = latest.ObservationDate.UTC().Format("2006-01-02")
                                base["series"] = series
                        }
                }
                return base
        }
        return map[string]any{
                "country": countryProfiles[code].Name,
                "note":    "no debt data available for this country",
        }
}

// compareDebtDifferences describes the debt differences across the selected
// countries. The note explicitly states that the platform does NOT rank
// countries by debt level (Spec §37).
func compareDebtDifferences(codes []string) []compareDifference {
        if len(codes) < 2 {
                return nil
        }
        values := make(map[string]any, len(codes))
        for _, c := range codes {
                if ind, ok := countryIndicators[c]["debt_to_gdp"]; ok {
                        values[c] = ind.Value
                }
        }
        return []compareDifference{
                {
                        Dimension: "debt_to_gdp",
                        Values:    values,
                        Note:      "Debt-to-GDP varies across the selected countries. The platform does NOT rank countries by debt level — a higher ratio is not 'worse' nor a lower ratio 'better'. Each figure is an immutable observation sourced from each country's Treasury or the IMF WEO. Differences reflect fiscal trajectory, exchange-rate movements, and disbursement timing.",
                },
        }
}

// makeCompareGovernmentStructureHandler handles
// GET /api/v1/compare/government-structure?countries=KE,UG,NG,ZA.
//
// Returns the government structure (system, houses, term length, members)
// for each selected country, plus a `differences` array describing the
// structural distinctions.
func makeCompareGovernmentStructureHandler() http.HandlerFunc {
        return func(w http.ResponseWriter, r *http.Request) {
                if r.Method != http.MethodGet {
                        writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
                        return
                }
                w.Header().Set("Content-Type", "application/json")
                codes := parseCountriesParam(r)
                sort.Strings(codes)
                data := make(map[string]any, len(codes))
                for _, c := range codes {
                        p := countryProfiles[c]
                        data[c] = map[string]any{
                                "country":           p.Name,
                                "government_system": p.GovernmentSystem,
                                "houses":            p.HouseCount,
                                "chambers":          p.Houses,
                                "term_length_years": p.TermDays / 365,
                                "total_members":     p.TotalMembers,
                                "constitution_url":  p.ConstitutionURL,
                                "parliament_url":    p.ParliamentURL,
                                "source_urls":       p.SourceURLs,
                                "last_updated":      p.LastUpdated,
                        }
                }
                writeJSON(w, http.StatusOK, map[string]any{
                        "comparison": map[string]any{
                                "countries":  codes,
                                "dimensions": []string{"government_structure"},
                                "data":       data,
                                "differences": compareCountryDifferences(codes),
                                "disclaimer":  comparisonDisclaimer,
                        },
                })
        }
}

// makeCompareIndicatorsHandler handles
// GET /api/v1/compare/indicators?countries=KE,UG,TZ&indicators=debt_to_gdp,bills_count,acts_count.
//
// Returns the requested indicators for each selected country. If the
// `indicators` query parameter is missing, all supported indicators are
// returned.
func makeCompareIndicatorsHandler() http.HandlerFunc {
        return func(w http.ResponseWriter, r *http.Request) {
                if r.Method != http.MethodGet {
                        writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
                        return
                }
                w.Header().Set("Content-Type", "application/json")
                codes := parseCountriesParam(r)
                sort.Strings(codes)
                indicators := parseIndicatorsParam(r)
                data := make(map[string]any, len(codes))
                for _, c := range codes {
                        row := map[string]any{
                                "country": countryProfiles[c].Name,
                                "flag":    countryProfiles[c].Flag,
                        }
                        for _, key := range indicators {
                                if ind, ok := countryIndicators[c][key]; ok {
                                        row[key] = ind
                                } else {
                                        row[key] = nil
                                }
                        }
                        data[c] = row
                }
                writeJSON(w, http.StatusOK, map[string]any{
                        "comparison": map[string]any{
                                "countries":  codes,
                                "indicators": indicators,
                                "dimensions": []string{"civic_indicators"},
                                "data":       data,
                                "disclaimer":  comparisonDisclaimer,
                        },
                })
        }
}

// parseIndicatorsParam extracts the comma-separated ?indicators= query
// parameter. Unknown indicator keys are dropped. If the resulting list
// is empty, all supported indicators are returned.
func parseIndicatorsParam(r *http.Request) []string {
        raw := r.URL.Query().Get("indicators")
        if raw == "" {
                out := make([]string, 0, len(supportedIndicatorKeys))
                for k := range supportedIndicatorKeys {
                        out = append(out, k)
                }
                sort.Strings(out)
                return out
        }
        parts := strings.Split(raw, ",")
        out := make([]string, 0, len(parts))
        seen := map[string]bool{}
        for _, p := range parts {
                key := strings.ToLower(strings.TrimSpace(p))
                if key == "" || seen[key] {
                        continue
                }
                if _, ok := supportedIndicatorKeys[key]; !ok {
                        continue
                }
                seen[key] = true
                out = append(out, key)
        }
        if len(out) == 0 {
                for k := range supportedIndicatorKeys {
                        out = append(out, k)
                }
        }
        sort.Strings(out)
        return out
}
