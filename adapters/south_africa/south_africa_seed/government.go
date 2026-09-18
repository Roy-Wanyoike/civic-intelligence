// Package south_africa_seed provides authoritative seed data for the South
// Africa government, Acts of Parliament, and public debt observations.
//
// All data is sourced from authoritative references:
//   - Parliament of South Africa (https://www.parliament.gov.za)
//   - South African Reserve Bank (https://www.resbank.co.za)
//   - National Treasury (https://www.treasury.gov.za)
//
// CRITICAL ATTRIBUTION RULE (Spec section 35): the platform does NOT say
// "President X borrowed ZAR X". It says "The Government of South Africa
// recorded ZAR X in borrowing during this period." The legal borrower is
// the Republic of South Africa, not a person.
package south_africa_seed

import (
	"time"

	"github.com/Roy-Wanyoike/civic-intelligence/services/legislation/government"
)

// SouthAfricaPresidents is the authoritative list of South African presidents
// since the end of apartheid (27 April 1994, when the 1993 Constitution came
// into force). Source: Parliament of South Africa.
var SouthAfricaPresidents = []government.President{
	{ID: "president-nelson-mandela", CountryCode: "ZA", FullName: "Nelson Rolihlahla Mandela", DisplayName: "Nelson Mandela"},
	{ID: "president-thabo-mbeki", CountryCode: "ZA", FullName: "Thabo Mvuyelwa Mbeki", DisplayName: "Thabo Mbeki"},
	{ID: "president-kgalema-motlanthe", CountryCode: "ZA", FullName: "Kgalema Petrus Motlanthe", DisplayName: "Kgalema Motlanthe"},
	{ID: "president-jacob-zuma", CountryCode: "ZA", FullName: "Jacob Gedleyihlekisa Zuma", DisplayName: "Jacob Zuma"},
	{ID: "president-cyril-ramaphosa", CountryCode: "ZA", FullName: "Matamela Cyril Ramaphosa", DisplayName: "Cyril Ramaphosa"},
}

// SouthAfricaAdministrations is the authoritative list of administrations
// under the post-apartheid Republic. Spec section 7.
var SouthAfricaAdministrations = []government.Administration{
	{
		ID:                "admin-jacob-zuma",
		CountryCode:       "ZA",
		PresidentID:       "president-jacob-zuma",
		Name:              "Jacob Zuma Administration",
		StartDate:         time.Date(2009, 5, 9, 0, 0, 0, 0, time.UTC),
		EndDate:           ptrTime(time.Date(2018, 2, 15, 0, 0, 0, 0, time.UTC)),
		GovernmentSystem:  government.GovernmentSystemPresidential,
	},
	{
		ID:                "admin-cyril-ramaphosa",
		CountryCode:       "ZA",
		PresidentID:       "president-cyril-ramaphosa",
		Name:              "Cyril Ramaphosa Administration",
		StartDate:         time.Date(2018, 2, 15, 0, 0, 0, 0, time.UTC),
		EndDate:           nil,
		GovernmentSystem:  government.GovernmentSystemPresidential,
	},
}

// SouthAfricaPresidentialTerms is the authoritative list of presidential
// terms. Source: Parliament of South Africa — the President is elected by
// the National Assembly and serves a 5-year term, with a two-term
// constitutional limit (Section 88 of the 1996 Constitution).
var SouthAfricaPresidentialTerms = []government.PresidentialTerm{
	// Zuma administration (2 terms).
	{ID: "term-zuma-1", AdministrationID: "admin-jacob-zuma", PresidentID: "president-jacob-zuma", TermNumber: 1, StartDate: time.Date(2009, 5, 9, 0, 0, 0, 0, time.UTC), EndDate: ptrTime(time.Date(2014, 5, 21, 0, 0, 0, 0, time.UTC)), Status: "COMPLETED"},
	{ID: "term-zuma-2", AdministrationID: "admin-jacob-zuma", PresidentID: "president-jacob-zuma", TermNumber: 2, StartDate: time.Date(2014, 5, 21, 0, 0, 0, 0, time.UTC), EndDate: ptrTime(time.Date(2018, 2, 15, 0, 0, 0, 0, time.UTC)), Status: "INTERRUPTED"},
	// Ramaphosa administration (currently in term 2).
	{ID: "term-ramaphosa-1", AdministrationID: "admin-cyril-ramaphosa", PresidentID: "president-cyril-ramaphosa", TermNumber: 1, StartDate: time.Date(2018, 2, 15, 0, 0, 0, 0, time.UTC), EndDate: ptrTime(time.Date(2019, 5, 25, 0, 0, 0, 0, time.UTC)), Status: "COMPLETED"},
	{ID: "term-ramaphosa-2", AdministrationID: "admin-cyril-ramaphosa", PresidentID: "president-cyril-ramaphosa", TermNumber: 2, StartDate: time.Date(2019, 5, 25, 0, 0, 0, 0, time.UTC), EndDate: ptrTime(time.Date(2024, 6, 14, 0, 0, 0, 0, time.UTC)), Status: "COMPLETED"},
	{ID: "term-ramaphosa-3", AdministrationID: "admin-cyril-ramaphosa", PresidentID: "president-cyril-ramaphosa", TermNumber: 3, StartDate: time.Date(2024, 6, 14, 0, 0, 0, 0, time.UTC), EndDate: nil, Status: "CURRENT"},
}

// SouthAfricaGovernmentTransitions records the formal handover events.
var SouthAfricaGovernmentTransitions = []government.GovernmentTransition{
	{
		ID:                  "transition-za-2018",
		CountryCode:         "ZA",
		OutgoingAdminID:     ptrID("admin-jacob-zuma"),
		IncomingAdminID:     "admin-cyril-ramaphosa",
		TransitionDate:      time.Date(2018, 2, 15, 0, 0, 0, 0, time.UTC),
		OutgoingPresidentID: ptrID("president-jacob-zuma"),
		IncomingPresidentID: "president-cyril-ramaphosa",
	},
}

func ptrTime(t time.Time) *time.Time { return &t }
func ptrID(s string) *government.ID  { id := government.ID(s); return &id }
