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
package kenya_seed

import (
	"fmt"
	"strings"
	"time"

	"github.com/Roy-Wanyoike/civic-intelligence/adapters/kenya/kenya_law"
)

// sampleBillSpec is the raw input used to build a BillCandidate. It is
// private to this file — callers consume kenya_seed.SampleBills directly.
type sampleBillSpec struct {
	houseCode string // "na" or "senate"
	date      string // YYYY-MM-DD
	slug      string // URL slug, e.g. "the-housing-bill-2024"
}

// sampleBillSpecs mirrors the bills listed on new.kenyalaw.org/bills/
// (testdata/bills_list.html). Each entry produces one BillCandidate via
// makeSampleBill below. The list is kept comprehensive (50 entries) so
// the degraded-mode response is indistinguishable from a live crawl at
// the level of pagination + trending categorisation.
//
// Source: https://new.kenyalaw.org/bills/ (snapshot 2026-09).
var sampleBillSpecs = []sampleBillSpec{
	{"na", "2026-09-07", "the-local-authorities-provident-fund-amendment-bill-2026"},
	{"na", "2026-09-04", "the-national-coroners-service-bill-2026"},
	{"na", "2026-08-19", "the-county-governments-retirement-scheme-bill-2026"},
	{"na", "2026-08-19", "the-public-finance-management-amendment-bill-2026"},
	{"na", "2026-08-19", "the-state-corporations-amendment-bill-2026"},
	{"na", "2026-08-12", "the-air-passenger-service-charge-amendment-bill-2026"},
	{"na", "2026-08-07", "the-alcoholic-drinks-control-amendment-bill-2026"},
	{"na", "2026-07-24", "the-basic-education-bill-2026"},
	{"na", "2026-07-24", "the-kenya-institute-of-curriculum-development-amendment-bill-2026"},
	{"na", "2026-07-24", "the-kenya-national-qualifications-framework-amendment-bill-2026"},
	{"na", "2026-07-24", "the-pre-service-education-and-in-service-training-bill-2026"},
	{"na", "2026-07-24", "the-tertiary-education-placement-and-funding-bill-2026"},
	{"na", "2026-07-22", "the-architectural-and-quantity-surveying-practitioners-bill-2026"},
	{"senate", "2026-07-20", "the-national-transport-and-safety-authority-amendment-bill-2026"},
	{"na", "2026-07-09", "the-heraldry-bill-2026"},
	{"senate", "2026-07-09", "the-statutory-instruments-amendment-bill-2026"},
	{"na", "2026-07-02", "the-kenya-economic-zones-bill-2026"},
	{"na", "2026-07-02", "the-kenya-plant-health-inspectorate-services-amendment-bill-2026"},
	{"na", "2026-07-01", "the-legal-metrology-bill-2026"},
	{"na", "2026-06-19", "the-kenya-intellectual-property-bill-2026"},
	{"na", "2026-06-19", "the-tourism-amendment-bill-2026"},
	{"na", "2026-06-19", "the-trade-description-amendment-bill-2026"},
	{"na", "2026-06-15", "the-equalisation-fund-appropriation-bill-2026"},
	{"na", "2026-06-05", "the-appropriation-bill-2026"},
	{"na", "2026-06-05", "the-crops-laws-amendment-bill-2026"},
	{"na", "2026-06-05", "the-east-african-development-bank-amendment-bill-2026"},
	{"na", "2026-06-05", "the-films-and-stage-plays-amendment-bill-2026"},
	{"na", "2026-06-05", "the-investment-and-export-promotion-authority-bill-2026"},
	{"na", "2026-06-05", "the-regional-development-authorities-laws-repeal-bill-2026"},
	{"na", "2026-06-05", "the-water-amendment-bill-2026"},
	{"na", "2026-05-20", "the-trust-administration-bill-2026"},
	{"na", "2026-05-19", "the-kenya-revenue-authority-amendment-bill-2026"},
	{"na", "2026-05-05", "the-foreign-service-amendment-bill-2026"},
	{"senate", "2026-04-30", "the-county-allocation-of-revenue-bill-2026"},
	{"na", "2026-04-14", "the-commission-of-inquiry-amendment-bill-2026"},
	{"na", "2026-04-16", "the-national-coroners-service-amendment-bill-2026"},
	{"na", "2026-04-10", "the-value-added-tax-amendment-bill-2026"},
	{"na", "2026-04-08", "the-pension-amendment-bill-2026"},
	{"na", "2026-04-05", "the-livestock-bill-2026"},
	{"na", "2026-04-02", "the-income-tax-amendment-bill-2026"},
	{"na", "2026-04-02", "the-kenya-blood-cells-tissues-and-organs-bill-2026"},
	{"na", "2026-03-26", "the-certified-governance-secretaries-bill-2026"},
	{"na", "2026-03-25", "the-supplementary-appropriation-bill-2026"},
	{"na", "2026-03-24", "the-strategic-goods-control-bill-2026"},
	{"senate", "2026-03-17", "the-environmental-management-and-co-ordination-amendment-bill-2026"},
	{"na", "2026-03-16", "the-traffic-amendment-bill-2026"},
	{"na", "2026-03-13", "the-criminal-procedure-code-amendment-bill-2026"},
	{"na", "2026-03-13", "the-penal-code-amendment-bill-2026"},
	{"na", "2026-03-11", "the-medical-practitioners-and-dentists-amendment-bill-2026"},
	{"na", "2026-03-10", "the-microfinance-bill-2026"},
}

// makeSampleBill converts a sampleBillSpec into a BillCandidate whose
// shape is byte-identical to what kenya_law.ParseBillsListing would
// return for the same row on the live /bills/ page.
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
// without any conversion logic.
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
