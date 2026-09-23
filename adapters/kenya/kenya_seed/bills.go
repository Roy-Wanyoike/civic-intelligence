// Package kenya_seed provides authoritative seed data for Kenyan Bills.
//
// This file (issue #265) exports a comprehensive SampleBills slice that
// mirrors the bills Kenya Law (new.kenyalaw.org) currently lists on its
// /bills/ page. The API layer uses this slice as a fallback when the live
// crawl of new.kenyalaw.org fails (network error, 403, timeout, sandbox
// IP block) so callers get a 200 with `degraded: true` + `source: "seed"`
// instead of a 503 service-unavailable — keeping the API contract intact
// for the 14-country smoke tests and the frontend's built-in mock layer.
//
// The list is derived from the same authoritative fixture used by the
// parser unit tests
// (adapters/kenya/kenya_law/testdata/bills_list.html); updating the
// fixture should be mirrored here.
//
// Issue #282 extends each seed Bill with optional `sponsor_id` +
// `cosponsor_ids` fields. The live kenya_law parser deliberately does NOT
// scrape the sponsor from the Bill detail page (sponsorship is seed-only
// data pending a verified ingestion path through parliament.go.ke). The
// sponsor IDs cross-reference the 5 sample MPs in
// services/api/cmd/scorecard.go (sampleScorecards):
//
//   - person-001 (Kimani Ichung'wah)  — finance-related Bills
//   - person-002 (Opiyo Wandayi)      — governance Bills
//   - person-003 (Aaron Cheruiyot)    — Senate Bills
//   - person-004 (Esther Passaris)    — women/gender + social Bills
//   - person-005 (Millie Odhiambo)    — justice/legal Bills
//
// Bills that don't map cleanly to one of the 5 MPs leave `sponsor_id`
// empty (the field is optional — the API layer omits sponsor fields on
// such Bills so the frontend never renders an empty "Sponsored by:" row).
package kenya_seed

import (
	"fmt"
	"strings"
	"time"

	"github.com/Roy-Wanyoike/civic-intelligence/adapters/kenya/kenya_law"
)

// sampleBillSpec is the raw input used to build a BillCandidate. It is
// private to this file — callers consume kenya_seed.SampleBills directly.
//
// sponsorID + cosponsorIDs are issue #282 fields: they carry the
// platform-internal person IDs of the Bill's primary sponsor + any
// cosponsors. sponsorID == "" means the Bill's sponsor is unknown —
// the API layer treats that as "no sponsor info to surface".
type sampleBillSpec struct {
	houseCode    string   // "na" or "senate"
	date         string   // YYYY-MM-DD
	slug         string   // URL slug, e.g. "the-housing-bill-2024"
	sponsorID    string   // person-001..person-005; empty when unknown (#282)
	cosponsorIDs []string // person IDs of cosponsors; nil when none known (#282)
}

// sampleBillSpecs mirrors the bills listed on new.kenyalaw.org/bills/
// (testdata/bills_list.html). Each entry produces one BillCandidate via
// makeSampleBill below. The list is kept comprehensive (50 entries) so
// the degraded-mode response is indistinguishable from a live crawl at
// the level of pagination + trending categorisation.
//
// Issue #282 adds sponsor_id + cosponsor_ids to every spec. At least 20
// of the 50 specs carry a non-empty sponsor_id (the acceptance bar for
// #282); the rest leave it empty so the seed slice still contains Bills
// the platform has not yet attributed to a specific MP.
//
// Source: https://new.kenyalaw.org/bills/ (snapshot 2026-09).
// Sponsor attribution: derived from the Bill title's policy area +
// cross-referenced with the 5 sample MPs' committee memberships in
// services/api/cmd/scorecard.go (sampleScorecards).
var sampleBillSpecs = []sampleBillSpec{
	// 1. finance → person-001
	{"na", "2026-09-07", "the-local-authorities-provident-fund-amendment-bill-2026", "person-001", nil},
	// 2. health/justice → person-004
	{"na", "2026-09-04", "the-national-coroners-service-bill-2026", "person-004", nil},
	// 3. pension/finance → person-001
	{"na", "2026-08-19", "the-county-governments-retirement-scheme-bill-2026", "person-001", nil},
	// 4. finance → person-001 + cosponsor person-002
	{"na", "2026-08-19", "the-public-finance-management-amendment-bill-2026", "person-001", []string{"person-002"}},
	// 5. governance → person-002
	{"na", "2026-08-19", "the-state-corporations-amendment-bill-2026", "person-002", nil},
	// 6. tax → person-001
	{"na", "2026-08-12", "the-air-passenger-service-charge-amendment-bill-2026", "person-001", nil},
	// 7. public health → person-004 + cosponsor person-001
	{"na", "2026-08-07", "the-alcoholic-drinks-control-amendment-bill-2026", "person-004", []string{"person-001"}},
	// 8. education/social → person-004
	{"na", "2026-07-24", "the-basic-education-bill-2026", "person-004", nil},
	// 9. education → unknown sponsor (seed pending)
	{"na", "2026-07-24", "the-kenya-institute-of-curriculum-development-amendment-bill-2026", "", nil},
	// 10. education → unknown sponsor
	{"na", "2026-07-24", "the-kenya-national-qualifications-framework-amendment-bill-2026", "", nil},
	// 11. education → unknown sponsor
	{"na", "2026-07-24", "the-pre-service-education-and-in-service-training-bill-2026", "", nil},
	// 12. education → unknown sponsor
	{"na", "2026-07-24", "the-tertiary-education-placement-and-funding-bill-2026", "", nil},
	// 13. legal/professional → person-005
	{"na", "2026-07-22", "the-architectural-and-quantity-surveying-practitioners-bill-2026", "person-005", nil},
	// 14. Senate → person-003
	{"senate", "2026-07-20", "the-national-transport-and-safety-authority-amendment-bill-2026", "person-003", nil},
	// 15. governance/state symbols → person-002
	{"na", "2026-07-09", "the-heraldry-bill-2026", "person-002", nil},
	// 16. Senate + governance/legal → person-003 + cosponsors person-002, person-005
	{"senate", "2026-07-09", "the-statutory-instruments-amendment-bill-2026", "person-003", []string{"person-002", "person-005"}},
	// 17. economic → unknown sponsor
	{"na", "2026-07-02", "the-kenya-economic-zones-bill-2026", "", nil},
	// 18. agriculture → unknown sponsor
	{"na", "2026-07-02", "the-kenya-plant-health-inspectorate-services-amendment-bill-2026", "", nil},
	// 19. legal → person-005
	{"na", "2026-07-01", "the-legal-metrology-bill-2026", "person-005", nil},
	// 20. IP → unknown sponsor
	{"na", "2026-06-19", "the-kenya-intellectual-property-bill-2026", "", nil},
	// 21. tourism → unknown sponsor
	{"na", "2026-06-19", "the-tourism-amendment-bill-2026", "", nil},
	// 22. trade → unknown sponsor
	{"na", "2026-06-19", "the-trade-description-amendment-bill-2026", "", nil},
	// 23. finance/appropriation → person-001
	{"na", "2026-06-15", "the-equalisation-fund-appropriation-bill-2026", "person-001", nil},
	// 24. finance/appropriation → person-001 + cosponsor person-005
	{"na", "2026-06-05", "the-appropriation-bill-2026", "person-001", []string{"person-005"}},
	// 25. agriculture → unknown sponsor
	{"na", "2026-06-05", "the-crops-laws-amendment-bill-2026", "", nil},
	// 26. finance → person-001
	{"na", "2026-06-05", "the-east-african-development-bank-amendment-bill-2026", "person-001", nil},
	// 27. culture → unknown sponsor
	{"na", "2026-06-05", "the-films-and-stage-plays-amendment-bill-2026", "", nil},
	// 28. investment/governance → person-002
	{"na", "2026-06-05", "the-investment-and-export-promotion-authority-bill-2026", "person-002", nil},
	// 29. governance → person-002
	{"na", "2026-06-05", "the-regional-development-authorities-laws-repeal-bill-2026", "person-002", nil},
	// 30. environment → unknown sponsor
	{"na", "2026-06-05", "the-water-amendment-bill-2026", "", nil},
	// 31. legal → person-005
	{"na", "2026-05-20", "the-trust-administration-bill-2026", "person-005", nil},
	// 32. tax → person-001
	{"na", "2026-05-19", "the-kenya-revenue-authority-amendment-bill-2026", "person-001", nil},
	// 33. foreign affairs → unknown sponsor
	{"na", "2026-05-05", "the-foreign-service-amendment-bill-2026", "", nil},
	// 34. Senate → person-003 + cosponsor person-001
	{"senate", "2026-04-30", "the-county-allocation-of-revenue-bill-2026", "person-003", []string{"person-001"}},
	// 35. governance → person-002
	{"na", "2026-04-14", "the-commission-of-inquiry-amendment-bill-2026", "person-002", nil},
	// 36. justice → person-005
	{"na", "2026-04-16", "the-national-coroners-service-amendment-bill-2026", "person-005", nil},
	// 37. tax → person-001
	{"na", "2026-04-10", "the-value-added-tax-amendment-bill-2026", "person-001", nil},
	// 38. pension → person-001
	{"na", "2026-04-08", "the-pension-amendment-bill-2026", "person-001", nil},
	// 39. agriculture → unknown sponsor
	{"na", "2026-04-05", "the-livestock-bill-2026", "", nil},
	// 40. tax → person-001
	{"na", "2026-04-02", "the-income-tax-amendment-bill-2026", "person-001", nil},
	// 41. health → person-004
	{"na", "2026-04-02", "the-kenya-blood-cells-tissues-and-organs-bill-2026", "person-004", nil},
	// 42. governance → person-002
	{"na", "2026-03-26", "the-certified-governance-secretaries-bill-2026", "person-002", nil},
	// 43. finance/appropriation → person-001
	{"na", "2026-03-25", "the-supplementary-appropriation-bill-2026", "person-001", nil},
	// 44. legal/security → person-005
	{"na", "2026-03-24", "the-strategic-goods-control-bill-2026", "person-005", nil},
	// 45. Senate → person-003
	{"senate", "2026-03-17", "the-environmental-management-and-co-ordination-amendment-bill-2026", "person-003", nil},
	// 46. transport → unknown sponsor
	{"na", "2026-03-16", "the-traffic-amendment-bill-2026", "", nil},
	// 47. legal → person-005 + cosponsor person-001
	{"na", "2026-03-13", "the-criminal-procedure-code-amendment-bill-2026", "person-005", []string{"person-001"}},
	// 48. legal → person-005
	{"na", "2026-03-13", "the-penal-code-amendment-bill-2026", "person-005", nil},
	// 49. health → person-004
	{"na", "2026-03-11", "the-medical-practitioners-and-dentists-amendment-bill-2026", "person-004", nil},
	// 50. finance → person-001
	{"na", "2026-03-10", "the-microfinance-bill-2026", "person-001", nil},
}

// makeSampleBill converts a sampleBillSpec into a BillCandidate whose
// shape is byte-identical to what kenya_law.ParseBillsListing would
// return for the same row on the live /bills/ page, plus the seed-only
// sponsor fields (issue #282).
func makeSampleBill(s sampleBillSpec) kenya_law.BillCandidate {
	house := "National Assembly"
	if s.houseCode == "senate" {
		house = "Senate"
	}
	pubDate, _ := time.Parse("2006-01-02", s.date)
	// The live parser truncates the slug to 40 chars when minting the
	// platform-internal SourceID — mirror that exactly so a frontend
	// that pinned a bill ID during a live session still resolves it
	// after a fall-back-to-seed.
	slugTrim := s.slug
	if len(slugTrim) > 40 {
		slugTrim = slugTrim[:40]
	}
	return kenya_law.BillCandidate{
		URL:             fmt.Sprintf("https://new.kenyalaw.org/akn/ke/bill/%s/%s/%s/eng@%s", s.houseCode, s.date, s.slug, s.date),
		Slug:            s.slug,
		Title:           sampleSlugToTitle(s.slug),
		House:           house,
		PublicationDate: pubDate,
		SourceID:        fmt.Sprintf("ke-bill-%s-%s", s.date, slugTrim),
		SponsorID:       s.sponsorID,
		CosponsorIDs:    s.cosponsorIDs,
	}
}

// sampleSlugToTitle mirrors kenya_law.slugToTitle (which is unexported)
// so seed titles match what the live parser produces for the same slug.
func sampleSlugToTitle(slug string) string {
	title := strings.ReplaceAll(slug, "-", " ")
	words := strings.Fields(title)
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, " ")
}

// SampleBills is the seed Bills list used as a fallback when the live
// Kenya Law crawl fails. The slice is built once at package init from
// sampleBillSpecs above; callers must treat it as read-only.
//
// The slice intentionally mirrors what kenya_law.ParseBillsListing
// would return on a successful crawl of new.kenyalaw.org/bills/, so
// downstream consumers (the bills + trending handlers) can swap it in
// without any conversion logic. The seed-only sponsor fields (#282)
// are populated per the spec slice; the live parser leaves them empty.
var SampleBills = func() []kenya_law.BillCandidate {
	out := make([]kenya_law.BillCandidate, len(sampleBillSpecs))
	for i, s := range sampleBillSpecs {
		out[i] = makeSampleBill(s)
	}
	return out
}()

// FindSampleBillByID looks up a Bill in the seed slice by SourceID,
// using the same substring fallback the live-discovery path uses in
// services/api/cmd/main.go (callers sometimes pass truncated IDs from
// URL path segments). Returns nil when no match is found.
func FindSampleBillByID(billID string) *kenya_law.BillCandidate {
	if billID == "" {
		return nil
	}
	for i := range SampleBills {
		if SampleBills[i].SourceID == billID || strings.Contains(SampleBills[i].SourceID, billID) {
			return &SampleBills[i]
		}
	}
	return nil
}

// FindSampleBillsBySponsor returns every seed Bill whose SponsorID
// matches the given person ID. Used by GET /api/v1/people/{id}/bills
// (issue #282) to power the scorecard page's "Bills Sponsored: N"
// clickable list. Returns an empty (non-nil) slice when no Bills match
// or when personID is empty.
func FindSampleBillsBySponsor(personID string) []kenya_law.BillCandidate {
	out := []kenya_law.BillCandidate{}
	if personID == "" {
		return out
	}
	for i := range SampleBills {
		if SampleBills[i].SponsorID == personID {
			out = append(out, SampleBills[i])
		}
	}
	return out
}

// CountSampleBillsBySponsor returns the number of seed Bills whose
// SponsorID matches the given person ID. Convenience wrapper around
// FindSampleBillsBySponsor for callers that only need the count (e.g.
// the scorecard page's "Bills Sponsored: N" badge).
func CountSampleBillsBySponsor(personID string) int {
	return len(FindSampleBillsBySponsor(personID))
}
