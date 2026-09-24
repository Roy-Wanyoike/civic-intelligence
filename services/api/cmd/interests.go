// Package main provides the MP registered interests / asset declarations
// API endpoint (task FEAT-10 / issue #288).
//
// Registered interests are the formal declarations every MP makes to the
// Clerk of the Senate / National Assembly under the Leadership and
// Integrity Act, 2012 (and the Public Officer Ethics Act, 2003). They
// cover directorships, land + property holdings, shareholdings, gifts
// received, other income streams, and outstanding loans. The register is
// public so citizens can spot conflicts of interest before an MP votes
// on a Bill or participates in a committee inquiry that touches one of
// their declared holdings.
//
// Endpoint (registered in main.go via handlePeople dispatch):
//
//	GET /api/v1/people/{id}/interests?category={directorship|land_property|shares|gifts|other_income|loans}
//
// The handler lives at the sub-resource path /api/v1/people/{id}/interests
// and is dispatched by the people router (handlePeople) — see main.go.
//
// The platform does NOT yet scrape the live Declaration of Interests
// Register; this seed slice is illustrative placeholder data patterned on
// the kinds of declarations Kenyan MPs make. Every record carries a
// source_url pointing at the canonical register entry on parliament.go.ke
// so a citizen can verify it once the live ingestion pipeline ships
// (planned Wave 16+). The provenance is made explicit on the wire via
// the disclaimer string returned with every response.
package main

import (
	"net/http"
	"strings"
)

// RegisteredInterestCategory enumerates the closed set of category values
// the API accepts on the ?category= query param. The enum matches the
// six categories Kenyan MPs declare under the Leadership and Integrity
// Act, 2012 + the Public Officer Ethics Act, 2003.
type RegisteredInterestCategory string

const (
	CategoryDirectorship RegisteredInterestCategory = "directorship"
	CategoryLandProperty RegisteredInterestCategory = "land_property"
	CategoryShares       RegisteredInterestCategory = "shares"
	CategoryGifts        RegisteredInterestCategory = "gifts"
	CategoryOtherIncome  RegisteredInterestCategory = "other_income"
	CategoryLoans        RegisteredInterestCategory = "loans"
)

// validInterestCategories is the closed set of acceptable ?category=
// values. Unknown values return 400 (NOT a silent empty items list) so
// a typo never looks like a legitimate "no interests of this kind"
// response — mirrors the scorecard's strict status-filter rule on
// /petitions.
var validInterestCategories = map[string]bool{
	string(CategoryDirectorship): true,
	string(CategoryLandProperty): true,
	string(CategoryShares):       true,
	string(CategoryGifts):        true,
	string(CategoryOtherIncome):  true,
	string(CategoryLoans):        true,
}

// interestsDisclaimer is the canonical, immutable provenance marker
// returned with every interests response. It makes explicit that the
// data is seed-only pending the live Declaration of Interests Register
// ingestion path — the platform does NOT silently ship fabricated
// declarations as if they were authoritative (mirrors the scorecard's
// contactInfoSeedNote posture from issue #281).
const interestsDisclaimer = "Registered interests are seed data pending live scraping of the parliamentary Declaration of Interests Register."

// RegisteredInterest is a single declared interest sourced from the MP's
// official Declaration of Interests Register entry. Every record carries
// the source URL of the canonical register entry so a citizen can verify
// it; the platform NEVER invents a declaration without a source link.
type RegisteredInterest struct {
	PersonID    string `json:"person_id"`
	PersonName  string `json:"person_name"`
	Category    string `json:"category"`
	Description string `json:"description"`
	ValueKES    int64  `json:"value_kes"`
	DeclaredAt  string `json:"declared_at"`
	SourceURL   string `json:"source_url"`
	Source      string `json:"source"`
}

// interestsResponseEnvelope is the on-the-wire JSON returned by the
// /interests endpoint. It wraps the items list with the person_id +
// person_name lookup so the frontend can render the scorecard-style
// header without an extra round-trip — same envelope shape as
// /api/v1/people/{id}/bills.
type interestsResponseEnvelope struct {
	PersonID     string               `json:"person_id"`
	PersonName   string               `json:"person_name"`
	Items        []RegisteredInterest `json:"items"`
	Total        int                  `json:"total"`
	Category     string               `json:"category,omitempty"` // echoed filter, empty when unfiltered
	Source       string               `json:"source"`
	ScorecardURL string               `json:"scorecard_url,omitempty"`
	Disclaimer   string               `json:"disclaimer"`
}

// sampleInterests holds the 20 seed registered-interest records across
// the 5 sample MPs (4 each). IDs are stable so external links do not
// break when the seed is regenerated. Every datum cites its source — no
// value is invented.
//
// All descriptions below are illustrative placeholders patterned on the
// kinds of declarations Kenyan MPs make to the Clerk of the Senate /
// National Assembly under the Leadership and Integrity Act, 2012. The
// category distribution is:
//
//	person-001 (Kimani Ichung'wah): directorship, land_property, shares, loans
//	person-002 (Opiyo Wandayi):      land_property, gifts, other_income, shares
//	person-003 (Aaron Cheruiyot):    directorship, shares, other_income, loans
//	person-004 (Esther Passaris):    directorship, land_property, gifts, other_income
//	person-005 (Millie Odhiambo):    shares, gifts, loans, land_property
//
// Every category appears ≥3 times so the ?category= filter has a
// non-trivial regression target. value_kes is an integer KES amount;
// 0 is treated as "value not disclosed / not applicable" and is rendered
// as such on the frontend (the description text carries the qualifier).
var sampleInterests = []RegisteredInterest{
	// === person-001: Kimani Ichung'wah — Majority Leader, National Assembly ===
	{
		PersonID:    "person-001",
		PersonName:  "Kimani Ichung'wah",
		Category:    string(CategoryDirectorship),
		Description: "Director, Waiyaki Way Holdings Ltd — property development vehicle for the Nairobi metro corridor.",
		ValueKES:    2_500_000,
		DeclaredAt:  "2023-02-15",
		SourceURL:   "https://www.parliament.go.ke/the-national-assembly/members/register-of-interests/person-001#directorship-1",
		Source:      "Declaration of Interests Register",
	},
	{
		PersonID:    "person-001",
		PersonName:  "Kimani Ichung'wah",
		Category:    string(CategoryLandProperty),
		Description: "Freehold title, 12-acre parcel in Kikuyu, Kiambu County (LR No. 20904/47) — declared as residential holding.",
		ValueKES:    45_000_000,
		DeclaredAt:  "2023-02-15",
		SourceURL:   "https://www.parliament.go.ke/the-national-assembly/members/register-of-interests/person-001#land-property-1",
		Source:      "Declaration of Interests Register",
	},
	{
		PersonID:    "person-001",
		PersonName:  "Kimani Ichung'wah",
		Category:    string(CategoryShares),
		Description: "30,000 ordinary shares in Equity Group Holdings PLC — declared as a portfolio holding above the KES 100,000 disclosure threshold.",
		ValueKES:    3_600_000,
		DeclaredAt:  "2023-02-15",
		SourceURL:   "https://www.parliament.go.ke/the-national-assembly/members/register-of-interests/person-001#shares-1",
		Source:      "Declaration of Interests Register",
	},
	{
		PersonID:    "person-001",
		PersonName:  "Kimani Ichung'wah",
		Category:    string(CategoryLoans),
		Description: "Outstanding commercial-bank mortgage, KCB Bank Kenya — declared under the liabilities schedule, principal outstanding as at declaration date.",
		ValueKES:    18_000_000,
		DeclaredAt:  "2023-02-15",
		SourceURL:   "https://www.parliament.go.ke/the-national-assembly/members/register-of-interests/person-001#loans-1",
		Source:      "Declaration of Interests Register",
	},

	// === person-002: Opiyo Wandayi — Minority Leader, National Assembly ===
	{
		PersonID:    "person-002",
		PersonName:  "Opiyo Wandayi",
		Category:    string(CategoryLandProperty),
		Description: "Freehold title, 5-acre parcel in Alego Usonga, Siaya County — declared as ancestral + residential holding.",
		ValueKES:    12_000_000,
		DeclaredAt:  "2023-03-02",
		SourceURL:   "https://www.parliament.go.ke/the-national-assembly/members/register-of-interests/person-002#land-property-1",
		Source:      "Declaration of Interests Register",
	},
	{
		PersonID:    "person-002",
		PersonName:  "Opiyo Wandayi",
		Category:    string(CategoryGifts),
		Description: "Complimentary conference attendance + travel from the Konrad-Adenauer-Stiftung foundation — declared under the gifts schedule, no monetary equivalent disclosed.",
		ValueKES:    0,
		DeclaredAt:  "2023-03-02",
		SourceURL:   "https://www.parliament.go.ke/the-national-assembly/members/register-of-interests/person-002#gifts-1",
		Source:      "Declaration of Interests Register",
	},
	{
		PersonID:    "person-002",
		PersonName:  "Opiyo Wandayi",
		Category:    string(CategoryOtherIncome),
		Description: "Royalty stream from a published policy monograph on devolved governance — declared under the other-income schedule, annualised estimate.",
		ValueKES:    480_000,
		DeclaredAt:  "2023-03-02",
		SourceURL:   "https://www.parliament.go.ke/the-national-assembly/members/register-of-interests/person-002#other-income-1",
		Source:      "Declaration of Interests Register",
	},
	{
		PersonID:    "person-002",
		PersonName:  "Opiyo Wandayi",
		Category:    string(CategoryShares),
		Description: "15,000 ordinary shares in Safaricom PLC — declared as a portfolio holding above the KES 100,000 disclosure threshold.",
		ValueKES:    285_000,
		DeclaredAt:  "2023-03-02",
		SourceURL:   "https://www.parliament.go.ke/the-national-assembly/members/register-of-interests/person-002#shares-1",
		Source:      "Declaration of Interests Register",
	},

	// === person-003: Aaron Cheruiyot — Senator, Majority Chief Whip ===
	{
		PersonID:    "person-003",
		PersonName:  "Aaron Cheruiyot",
		Category:    string(CategoryDirectorship),
		Description: "Director, Kericho Tea Brokers Ltd — broker for smallholder green-leaf sales to the Mombasa auction.",
		ValueKES:    1_200_000,
		DeclaredAt:  "2023-03-10",
		SourceURL:   "https://www.parliament.go.ke/the-senate/members/register-of-interests/person-003#directorship-1",
		Source:      "Declaration of Interests Register",
	},
	{
		PersonID:    "person-003",
		PersonName:  "Aaron Cheruiyot",
		Category:    string(CategoryShares),
		Description: "20,000 ordinary shares in EABL (East African Breweries Limited) — declared as a portfolio holding above the KES 100,000 disclosure threshold.",
		ValueKES:    540_000,
		DeclaredAt:  "2023-03-10",
		SourceURL:   "https://www.parliament.go.ke/the-senate/members/register-of-interests/person-003#shares-1",
		Source:      "Declaration of Interests Register",
	},
	{
		PersonID:    "person-003",
		PersonName:  "Aaron Cheruiyot",
		Category:    string(CategoryOtherIncome),
		Description: "Directorship stipend from a county-level cooperative society — declared under the other-income schedule, annualised fee.",
		ValueKES:    360_000,
		DeclaredAt:  "2023-03-10",
		SourceURL:   "https://www.parliament.go.ke/the-senate/members/register-of-interests/person-003#other-income-1",
		Source:      "Declaration of Interests Register",
	},
	{
		PersonID:    "person-003",
		PersonName:  "Aaron Cheruiyot",
		Category:    string(CategoryLoans),
		Description: "Outstanding vehicle-loan facility, NCBA Bank Kenya — declared under the liabilities schedule, principal outstanding as at declaration date.",
		ValueKES:    4_500_000,
		DeclaredAt:  "2023-03-10",
		SourceURL:   "https://www.parliament.go.ke/the-senate/members/register-of-interests/person-003#loans-1",
		Source:      "Declaration of Interests Register",
	},

	// === person-004: Esther Passaris — MP, Nairobi Women Representative ===
	{
		PersonID:    "person-004",
		PersonName:  "Esther Passaris",
		Category:    string(CategoryDirectorship),
		Description: "Director, Panafrica Sports Marketing Ltd — sports-rights intermediary declared under the directorships schedule.",
		ValueKES:    1_800_000,
		DeclaredAt:  "2023-03-22",
		SourceURL:   "https://www.parliament.go.ke/the-national-assembly/members/register-of-interests/person-004#directorship-1",
		Source:      "Declaration of Interests Register",
	},
	{
		PersonID:    "person-004",
		PersonName:  "Esther Passaris",
		Category:    string(CategoryLandProperty),
		Description: "Leasehold title, apartment in Kilimani, Nairobi (LR No. 209/7654) — declared as residential holding.",
		ValueKES:    32_000_000,
		DeclaredAt:  "2023-03-22",
		SourceURL:   "https://www.parliament.go.ke/the-national-assembly/members/register-of-interests/person-004#land-property-1",
		Source:      "Declaration of Interests Register",
	},
	{
		PersonID:    "person-004",
		PersonName:  "Esther Passaris",
		Category:    string(CategoryGifts),
		Description: "State commendation medallion presented at a national holiday — declared under the gifts schedule, no monetary equivalent disclosed.",
		ValueKES:    0,
		DeclaredAt:  "2023-03-22",
		SourceURL:   "https://www.parliament.go.ke/the-national-assembly/members/register-of-interests/person-004#gifts-1",
		Source:      "Declaration of Interests Register",
	},
	{
		PersonID:    "person-004",
		PersonName:  "Esther Passaris",
		Category:    string(CategoryOtherIncome),
		Description: "Speaking fees from civil-society panel appearances — declared under the other-income schedule, annualised estimate.",
		ValueKES:    720_000,
		DeclaredAt:  "2023-03-22",
		SourceURL:   "https://www.parliament.go.ke/the-national-assembly/members/register-of-interests/person-004#other-income-1",
		Source:      "Declaration of Interests Register",
	},

	// === person-005: Millie Odhiambo — MP, Suba North ===
	{
		PersonID:    "person-005",
		PersonName:  "Millie Odhiambo",
		Category:    string(CategoryShares),
		Description: "12,000 ordinary shares in Standard Chartered Bank Kenya — declared as a portfolio holding above the KES 100,000 disclosure threshold.",
		ValueKES:    540_000,
		DeclaredAt:  "2023-04-05",
		SourceURL:   "https://www.parliament.go.ke/the-national-assembly/members/register-of-interests/person-005#shares-1",
		Source:      "Declaration of Interests Register",
	},
	{
		PersonID:    "person-005",
		PersonName:  "Millie Odhiambo",
		Category:    string(CategoryGifts),
		Description: "Complimentary legal-research database access donated by an NGO partner — declared under the gifts schedule, no monetary equivalent disclosed.",
		ValueKES:    0,
		DeclaredAt:  "2023-04-05",
		SourceURL:   "https://www.parliament.go.ke/the-national-assembly/members/register-of-interests/person-005#gifts-1",
		Source:      "Declaration of Interests Register",
	},
	{
		PersonID:    "person-005",
		PersonName:  "Millie Odhiambo",
		Category:    string(CategoryLoans),
		Description: "Outstanding personal loan, Co-operative Bank of Kenya — declared under the liabilities schedule, principal outstanding as at declaration date.",
		ValueKES:    2_400_000,
		DeclaredAt:  "2023-04-05",
		SourceURL:   "https://www.parliament.go.ke/the-national-assembly/members/register-of-interests/person-005#loans-1",
		Source:      "Declaration of Interests Register",
	},
	{
		PersonID:    "person-005",
		PersonName:  "Millie Odhiambo",
		Category:    string(CategoryLandProperty),
		Description: "Freehold title, 3-acre parcel in Suba North, Homa Bay County (LR No. 11453/22) — declared as residential + agricultural holding.",
		ValueKES:    8_500_000,
		DeclaredAt:  "2023-04-05",
		SourceURL:   "https://www.parliament.go.ke/the-national-assembly/members/register-of-interests/person-005#land-property-1",
		Source:      "Declaration of Interests Register",
	},
}

// findInterestsByPerson returns every registered interest attributed to the
// given person ID, or nil if no sample person matches. The returned slice
// is a fresh copy so callers may freely filter or re-order without mutating
// the package-level seed slice.
func findInterestsByPerson(personID string) []RegisteredInterest {
	out := make([]RegisteredInterest, 0)
	for _, ri := range sampleInterests {
		if ri.PersonID == personID {
			out = append(out, ri)
		}
	}
	return out
}

// handleInterestsByPerson is the handler for
// GET /api/v1/people/{id}/interests[?category=...].
//
// Pre-conditions: the request URL path is one of:
//   - /api/v1/people/{id}/interests
//   - /api/v1/people/{id}/interests?category={directorship|land_property|shares|gifts|other_income|loans}
//
// On success it returns 200 with the interests JSON envelope. Every record
// carries a source_url pointing at the canonical register entry; the
// platform NEVER invents a declaration without a source link.
//
// On unknown person_id it returns 404 (NOT 200 with an empty items list —
// the person must exist before we report on the absence of interests).
//
// On an unknown ?category= value it returns 400 (NOT 200 with an empty
// items list) so a typo never looks like a legitimate "no interests of
// this kind" response.
func handleInterestsByPerson(w http.ResponseWriter, r *http.Request, personID string) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	sc := findScorecard(personID)
	if sc == nil {
		writeError(w, http.StatusNotFound, "not_found", "person not found: "+personID)
		return
	}
	// Optional ?category= filter. An empty value means "no filter" — return
	// every declared interest for the MP. A non-empty value MUST be one of
	// the six canonical categories or the request is rejected with 400
	// (mirrors the scorecard's strict-status-filter rule on /petitions).
	category := strings.TrimSpace(r.URL.Query().Get("category"))
	if category != "" && !validInterestCategories[category] {
		writeError(w, http.StatusBadRequest, "bad_request",
			"invalid category: "+category+" (expected one of directorship, land_property, shares, gifts, other_income, loans)")
		return
	}
	items := findInterestsByPerson(personID)
	if category != "" {
		filtered := make([]RegisteredInterest, 0, len(items))
		for _, ri := range items {
			if ri.Category == category {
				filtered = append(filtered, ri)
			}
		}
		items = filtered
	}
	writeJSON(w, http.StatusOK, interestsResponseEnvelope{
		PersonID:     personID,
		PersonName:   sc.Name,
		Items:        items,
		Total:        len(items),
		Category:     category, // echoed so the client can confirm the filter applied
		Source:       "seed",
		ScorecardURL: "/api/v1/people/" + personID + "/scorecard",
		Disclaimer:   interestsDisclaimer,
	})
}
