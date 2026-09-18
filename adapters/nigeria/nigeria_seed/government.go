// Package nigeria_seed provides authoritative seed data for the Nigeria
// government, Acts of the National Assembly, and public debt observations.
//
// All data is sourced from authoritative references:
//   - National Assembly of Nigeria (https://nass.gov.ng)
//   - Central Bank of Nigeria (https://www.cbn.gov.ng)
//   - Debt Management Office (https://www.dmo.gov.ng)
//   - Federal Ministry of Finance (https://www.fmf.gov.ng)
//
// CRITICAL ATTRIBUTION RULE (Spec section 35): the platform does NOT say
// "President X borrowed NGN X". It says "The Government of Nigeria
// recorded NGN X in borrowing during this period." The legal borrower is
// the Federal Republic of Nigeria, not a person.
package nigeria_seed

import (
	"time"

	"github.com/Roy-Wanyoike/civic-intelligence/services/legislation/government"
)

// NigeriaPresidents is the authoritative list of Nigerian presidents under
// the Fourth Republic (since 29 May 1999, when the 1999 Constitution came
// into force). Source: National Assembly of Nigeria.
var NigeriaPresidents = []government.President{
	{ID: "president-olusegun-obasanjo", CountryCode: "NG", FullName: "Olusegun Mathew Okikiola Aremu Obasanjo", DisplayName: "Olusegun Obasanjo"},
	{ID: "president-umaru-yaradua", CountryCode: "NG", FullName: "Umaru Musa Yar'Adua", DisplayName: "Umaru Musa Yar'Adua"},
	{ID: "president-goodluck-jonathan", CountryCode: "NG", FullName: "Goodluck Ebele Azikiwe Jonathan", DisplayName: "Goodluck Jonathan"},
	{ID: "president-muhammadu-buhari", CountryCode: "NG", FullName: "Muhammadu Buhari", DisplayName: "Muhammadu Buhari"},
	{ID: "president-bola-tinubu", CountryCode: "NG", FullName: "Bola Ahmed Adekunle Tinubu", DisplayName: "Bola Tinubu"},
}

// NigeriaAdministrations is the authoritative list of administrations under
// the Fourth Republic. Spec section 7.
var NigeriaAdministrations = []government.Administration{
	{
		ID:                "admin-goodluck-jonathan",
		CountryCode:       "NG",
		PresidentID:       "president-goodluck-jonathan",
		Name:              "Goodluck Jonathan Administration",
		StartDate:         time.Date(2010, 5, 6, 0, 0, 0, 0, time.UTC),
		EndDate:           ptrTime(time.Date(2015, 5, 29, 0, 0, 0, 0, time.UTC)),
		GovernmentSystem:  government.GovernmentSystemPresidential,
	},
	{
		ID:                "admin-muhammadu-buhari",
		CountryCode:       "NG",
		PresidentID:       "president-muhammadu-buhari",
		Name:              "Muhammadu Buhari Administration",
		StartDate:         time.Date(2015, 5, 29, 0, 0, 0, 0, time.UTC),
		EndDate:           ptrTime(time.Date(2023, 5, 29, 0, 0, 0, 0, time.UTC)),
		GovernmentSystem:  government.GovernmentSystemPresidential,
	},
	{
		ID:                "admin-bola-tinubu",
		CountryCode:       "NG",
		PresidentID:       "president-bola-tinubu",
		Name:              "Bola Tinubu Administration",
		StartDate:         time.Date(2023, 5, 29, 0, 0, 0, 0, time.UTC),
		EndDate:           nil,
		GovernmentSystem:  government.GovernmentSystemPresidential,
	},
}

// NigeriaPresidentialTerms is the authoritative list of presidential terms.
// Source: National Assembly of Nigeria — the President serves a 4-year term,
// with a two-term constitutional limit (Section 137 of the 1999 Constitution).
var NigeriaPresidentialTerms = []government.PresidentialTerm{
	// Jonathan administration (1½ terms — completed Yar'Adua's term + own 1 term).
	{ID: "term-jonathan-1", AdministrationID: "admin-goodluck-jonathan", PresidentID: "president-goodluck-jonathan", TermNumber: 1, StartDate: time.Date(2010, 5, 6, 0, 0, 0, 0, time.UTC), EndDate: ptrTime(time.Date(2011, 5, 29, 0, 0, 0, 0, time.UTC)), Status: "INTERRUPTED"},
	{ID: "term-jonathan-2", AdministrationID: "admin-goodluck-jonathan", PresidentID: "president-goodluck-jonathan", TermNumber: 2, StartDate: time.Date(2011, 5, 29, 0, 0, 0, 0, time.UTC), EndDate: ptrTime(time.Date(2015, 5, 29, 0, 0, 0, 0, time.UTC)), Status: "COMPLETED"},
	// Buhari administration (2 terms).
	{ID: "term-buhari-1", AdministrationID: "admin-muhammadu-buhari", PresidentID: "president-muhammadu-buhari", TermNumber: 1, StartDate: time.Date(2015, 5, 29, 0, 0, 0, 0, time.UTC), EndDate: ptrTime(time.Date(2019, 5, 29, 0, 0, 0, 0, time.UTC)), Status: "COMPLETED"},
	{ID: "term-buhari-2", AdministrationID: "admin-muhammadu-buhari", PresidentID: "president-muhammadu-buhari", TermNumber: 2, StartDate: time.Date(2019, 5, 29, 0, 0, 0, 0, time.UTC), EndDate: ptrTime(time.Date(2023, 5, 29, 0, 0, 0, 0, time.UTC)), Status: "COMPLETED"},
	// Tinubu administration (current, term 1).
	{ID: "term-tinubu-1", AdministrationID: "admin-bola-tinubu", PresidentID: "president-bola-tinubu", TermNumber: 1, StartDate: time.Date(2023, 5, 29, 0, 0, 0, 0, time.UTC), EndDate: nil, Status: "CURRENT"},
}

// NigeriaGovernmentTransitions records the formal handover events.
var NigeriaGovernmentTransitions = []government.GovernmentTransition{
	{
		ID:                  "transition-ng-2015",
		CountryCode:         "NG",
		OutgoingAdminID:     ptrID("admin-goodluck-jonathan"),
		IncomingAdminID:     "admin-muhammadu-buhari",
		TransitionDate:      time.Date(2015, 5, 29, 0, 0, 0, 0, time.UTC),
		OutgoingPresidentID: ptrID("president-goodluck-jonathan"),
		IncomingPresidentID: "president-muhammadu-buhari",
	},
	{
		ID:                  "transition-ng-2023",
		CountryCode:         "NG",
		OutgoingAdminID:     ptrID("admin-muhammadu-buhari"),
		IncomingAdminID:     "admin-bola-tinubu",
		TransitionDate:      time.Date(2023, 5, 29, 0, 0, 0, 0, time.UTC),
		OutgoingPresidentID: ptrID("president-muhammadu-buhari"),
		IncomingPresidentID: "president-bola-tinubu",
	},
}

func ptrTime(t time.Time) *time.Time { return &t }
func ptrID(s string) *government.ID  { id := government.ID(s); return &id }
