// Package tanzania_seed provides authoritative seed data for the Tanzania
// government, Acts of Parliament, and public debt observations.
//
// All data is sourced from authoritative references:
//   - Parliament of Tanzania (https://www.parliament.go.tz)
//   - Bank of Tanzania (https://www.bot.go.tz)
//   - Ministry of Finance and Planning (https://www.mof.go.tz)
//   - Tanzania Communications Regulatory Authority (TCRA)
//
// CRITICAL ATTRIBUTION RULE (Spec section 35): the platform does NOT say
// "President X borrowed TZS X". It says "The Government of Tanzania
// recorded TZS X in borrowing during this period." The legal borrower is
// the United Republic of Tanzania, not a person.
//
// The data is configuration-driven — no president is hard-coded into
// application logic. Summaries are attached to AdministrationID, not to a
// person.
package tanzania_seed

import (
	"time"

	"github.com/Roy-Wanyoike/civic-intelligence/services/legislation/government"
)

// TanzaniaPresidents is the authoritative list of Tanzanian presidents since
// the Union of Tanganyika and Zanzibar (1964). Source: Parliament of Tanzania.
var TanzaniaPresidents = []government.President{
	{ID: "president-julius-nyerere", CountryCode: "TZ", FullName: "Julius Kambarage Nyerere", DisplayName: "Julius Nyerere"},
	{ID: "president-ali-hassan-mwinyi", CountryCode: "TZ", FullName: "Ali Hassan Mwinyi", DisplayName: "Ali Hassan Mwinyi"},
	{ID: "president-benjamin-mkapa", CountryCode: "TZ", FullName: "Benjamin William Mkapa", DisplayName: "Benjamin Mkapa"},
	{ID: "president-jakaya-kikwete", CountryCode: "TZ", FullName: "Jakaya Mrisho Kikwete", DisplayName: "Jakaya Kikwete"},
	{ID: "president-john-magufuli", CountryCode: "TZ", FullName: "John Pombe Joseph Magufuli", DisplayName: "John Magufuli"},
	{ID: "president-samia-suluhu-hassan", CountryCode: "TZ", FullName: "Samia Suluhu Hassan", DisplayName: "Samia Suluhu Hassan"},
}

// TanzaniaAdministrations is the authoritative list of administrations.
// Spec section 7.
var TanzaniaAdministrations = []government.Administration{
	{
		ID:                "admin-jakaya-kikwete",
		CountryCode:       "TZ",
		PresidentID:       "president-jakaya-kikwete",
		Name:              "Jakaya Kikwete Administration",
		StartDate:         time.Date(2005, 12, 21, 0, 0, 0, 0, time.UTC),
		EndDate:           ptrTime(time.Date(2015, 11, 5, 0, 0, 0, 0, time.UTC)),
		GovernmentSystem:  government.GovernmentSystemPresidential,
	},
	{
		ID:                "admin-john-magufuli",
		CountryCode:       "TZ",
		PresidentID:       "president-john-magufuli",
		Name:              "John Magufuli Administration",
		StartDate:         time.Date(2015, 11, 5, 0, 0, 0, 0, time.UTC),
		EndDate:           ptrTime(time.Date(2021, 3, 19, 0, 0, 0, 0, time.UTC)),
		GovernmentSystem:  government.GovernmentSystemPresidential,
	},
	{
		ID:                "admin-samia-suluhu-hassan",
		CountryCode:       "TZ",
		PresidentID:       "president-samia-suluhu-hassan",
		Name:              "Samia Suluhu Hassan Administration",
		StartDate:         time.Date(2021, 3, 19, 0, 0, 0, 0, time.UTC),
		EndDate:           nil,
		GovernmentSystem:  government.GovernmentSystemPresidential,
	},
}

// TanzaniaPresidentialTerms is the authoritative list of presidential terms.
// Source: Parliament of Tanzania — the President serves a 5-year term, with a
// two-term constitutional limit (Article 40 of the Constitution of the United
// Republic of Tanzania, 1977).
var TanzaniaPresidentialTerms = []government.PresidentialTerm{
	// Kikwete administration (2 terms).
	{ID: "term-kikwete-1", AdministrationID: "admin-jakaya-kikwete", PresidentID: "president-jakaya-kikwete", TermNumber: 1, StartDate: time.Date(2005, 12, 21, 0, 0, 0, 0, time.UTC), EndDate: ptrTime(time.Date(2010, 11, 5, 0, 0, 0, 0, time.UTC)), Status: "COMPLETED"},
	{ID: "term-kikwete-2", AdministrationID: "admin-jakaya-kikwete", PresidentID: "president-jakaya-kikwete", TermNumber: 2, StartDate: time.Date(2010, 11, 5, 0, 0, 0, 0, time.UTC), EndDate: ptrTime(time.Date(2015, 11, 5, 0, 0, 0, 0, time.UTC)), Status: "COMPLETED"},
	// Magufuli administration (term interrupted by death in office on 17 March 2021).
	{ID: "term-magufuli-1", AdministrationID: "admin-john-magufuli", PresidentID: "president-john-magufuli", TermNumber: 1, StartDate: time.Date(2015, 11, 5, 0, 0, 0, 0, time.UTC), EndDate: ptrTime(time.Date(2020, 11, 5, 0, 0, 0, 0, time.UTC)), Status: "COMPLETED"},
	{ID: "term-magufuli-2", AdministrationID: "admin-john-magufuli", PresidentID: "president-john-magufuli", TermNumber: 2, StartDate: time.Date(2020, 11, 5, 0, 0, 0, 0, time.UTC), EndDate: ptrTime(time.Date(2021, 3, 19, 0, 0, 0, 0, time.UTC)), Status: "INTERRUPTED"},
	// Samia Suluhu Hassan — completing Magufuli's second term.
	{ID: "term-samia-1", AdministrationID: "admin-samia-suluhu-hassan", PresidentID: "president-samia-suluhu-hassan", TermNumber: 1, StartDate: time.Date(2021, 3, 19, 0, 0, 0, 0, time.UTC), EndDate: ptrTime(time.Date(2025, 11, 5, 0, 0, 0, 0, time.UTC)), Status: "COMPLETED"},
	{ID: "term-samia-2", AdministrationID: "admin-samia-suluhu-hassan", PresidentID: "president-samia-suluhu-hassan", TermNumber: 2, StartDate: time.Date(2025, 11, 5, 0, 0, 0, 0, time.UTC), EndDate: nil, Status: "CURRENT"},
}

// TanzaniaGovernmentTransitions records the formal handover events.
var TanzaniaGovernmentTransitions = []government.GovernmentTransition{
	{
		ID:                  "transition-tz-2015",
		CountryCode:         "TZ",
		OutgoingAdminID:     ptrID("admin-jakaya-kikwete"),
		IncomingAdminID:     "admin-john-magufuli",
		TransitionDate:      time.Date(2015, 11, 5, 0, 0, 0, 0, time.UTC),
		OutgoingPresidentID: ptrID("president-jakaya-kikwete"),
		IncomingPresidentID: "president-john-magufuli",
	},
	{
		ID:                  "transition-tz-2021",
		CountryCode:         "TZ",
		OutgoingAdminID:     ptrID("admin-john-magufuli"),
		IncomingAdminID:     "admin-samia-suluhu-hassan",
		TransitionDate:      time.Date(2021, 3, 19, 0, 0, 0, 0, time.UTC),
		OutgoingPresidentID: ptrID("president-john-magufuli"),
		IncomingPresidentID: "president-samia-suluhu-hassan",
	},
}

func ptrTime(t time.Time) *time.Time { return &t }
func ptrID(s string) *government.ID  { id := government.ID(s); return &id }
