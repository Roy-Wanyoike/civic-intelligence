// Package ghana_seed provides authoritative seed data for the Ghana
// government, Acts of Parliament, and public debt observations.
//
// All data is sourced from authoritative references:
//   - Parliament of Ghana (https://parliament.gh)
//   - Bank of Ghana (https://www.bog.gov.gh)
//   - Ministry of Finance (https://mofep.gov.gh)
//
// CRITICAL ATTRIBUTION RULE (Spec section 35): the platform does NOT say
// "President X borrowed GHS X". It says "The Government of Ghana recorded
// GHS X in borrowing during this period." The legal borrower is the
// Republic of Ghana, not a person.
package ghana_seed

import (
	"time"

	"github.com/Roy-Wanyoike/civic-intelligence/services/legislation/government"
)

// GhanaPresidents is the authoritative list of Ghanaian presidents under
// the Fourth Republic (since 7 January 1993, when the 1992 Constitution came
// into force). Source: Parliament of Ghana.
var GhanaPresidents = []government.President{
	{ID: "president-jj-rawlings", CountryCode: "GH", FullName: "Jerry John Rawlings", DisplayName: "Jerry John Rawlings"},
	{ID: "president-john-kufuor", CountryCode: "GH", FullName: "John Agyekum Kufuor", DisplayName: "John Kufuor"},
	{ID: "president-john-atta-mills", CountryCode: "GH", FullName: "John Evans Atta Mills", DisplayName: "John Atta Mills"},
	{ID: "president-john-mahama", CountryCode: "GH", FullName: "John Dramani Mahama", DisplayName: "John Mahama"},
	{ID: "president-nana-akufo-addo", CountryCode: "GH", FullName: "Nana Addo Dankwa Akufo-Addo", DisplayName: "Nana Akufo-Addo"},
}

// GhanaAdministrations is the authoritative list of administrations under
// the Fourth Republic. Spec section 7.
var GhanaAdministrations = []government.Administration{
	{
		ID:                "admin-john-mahama",
		CountryCode:       "GH",
		PresidentID:       "president-john-mahama",
		Name:              "John Mahama Administration",
		StartDate:         time.Date(2012, 7, 24, 0, 0, 0, 0, time.UTC),
		EndDate:           ptrTime(time.Date(2017, 1, 7, 0, 0, 0, 0, time.UTC)),
		GovernmentSystem:  government.GovernmentSystemPresidential,
	},
	{
		ID:                "admin-nana-akufo-addo",
		CountryCode:       "GH",
		PresidentID:       "president-nana-akufo-addo",
		Name:              "Nana Akufo-Addo Administration",
		StartDate:         time.Date(2017, 1, 7, 0, 0, 0, 0, time.UTC),
		EndDate:           nil,
		GovernmentSystem:  government.GovernmentSystemPresidential,
	},
}

// GhanaPresidentialTerms is the authoritative list of presidential terms.
// Source: Parliament of Ghana — the President serves a 4-year term, with a
// two-term constitutional limit (Article 66 of the 1992 Constitution).
var GhanaPresidentialTerms = []government.PresidentialTerm{
	// Mahama administration (one full term).
	{ID: "term-mahama-1", AdministrationID: "admin-john-mahama", PresidentID: "president-john-mahama", TermNumber: 1, StartDate: time.Date(2012, 7, 24, 0, 0, 0, 0, time.UTC), EndDate: ptrTime(time.Date(2017, 1, 7, 0, 0, 0, 0, time.UTC)), Status: "COMPLETED"},
	// Akufo-Addo administration (2 terms; term 2 ends 7 January 2025).
	{ID: "term-akufo-addo-1", AdministrationID: "admin-nana-akufo-addo", PresidentID: "president-nana-akufo-addo", TermNumber: 1, StartDate: time.Date(2017, 1, 7, 0, 0, 0, 0, time.UTC), EndDate: ptrTime(time.Date(2021, 1, 7, 0, 0, 0, 0, time.UTC)), Status: "COMPLETED"},
	{ID: "term-akufo-addo-2", AdministrationID: "admin-nana-akufo-addo", PresidentID: "president-nana-akufo-addo", TermNumber: 2, StartDate: time.Date(2021, 1, 7, 0, 0, 0, 0, time.UTC), EndDate: ptrTime(time.Date(2025, 1, 7, 0, 0, 0, 0, time.UTC)), Status: "COMPLETED"},
}

// GhanaGovernmentTransitions records the formal handover events.
var GhanaGovernmentTransitions = []government.GovernmentTransition{
	{
		ID:                  "transition-gh-2017",
		CountryCode:         "GH",
		OutgoingAdminID:     ptrID("admin-john-mahama"),
		IncomingAdminID:     "admin-nana-akufo-addo",
		TransitionDate:      time.Date(2017, 1, 7, 0, 0, 0, 0, time.UTC),
		OutgoingPresidentID: ptrID("president-john-mahama"),
		IncomingPresidentID: "president-nana-akufo-addo",
	},
}

func ptrTime(t time.Time) *time.Time { return &t }
func ptrID(s string) *government.ID  { id := government.ID(s); return &id }
