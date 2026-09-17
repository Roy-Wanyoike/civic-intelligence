// Package kenya_seed provides authoritative seed data for the Kenya
// constitution, administrations, presidential terms, and government
// transitions.
//
// All data is sourced from authoritative references:
//   - Kenya Law (https://www.kenyalaw.org)
//   - Official parliament records
//   - Independent Electoral and Boundaries Commission (IEBC)
//
// The data is configuration-driven — no president is hard-coded into
// application logic.
package kenya_seed

import (
        "time"

        "github.com/Roy-Wanyoike/civic-intelligence/services/legislation/government"
)

// ConstitutionOfKenya2010 is the authoritative Constitution of Kenya.
// Kenya Law identifies the Constitution as having been assented to on
// 4 August 2010 and promulgated on 27 August 2010.
//
// The Chapters slice is populated from KenyaConstitutionChapters
// (constitution.go) — a curated subset of the Constitution of Kenya 2010
// used by the Constitution Spotlight feature (issue #212). The chapters
// and articles preserve the exact text from Kenya Law; the platform
// never reinterprets constitutional text.
var ConstitutionOfKenya2010 = government.Constitution{
        ID:            "constitution-ke-2010",
        CountryCode:   "KE",
        Title:         "The Constitution of Kenya",
        PromulgatedAt: time.Date(2010, 8, 27, 0, 0, 0, 0, time.UTC),
        AssentedAt:    ptrTime(time.Date(2010, 8, 4, 0, 0, 0, 0, time.UTC)),
        Version:       "2010",
        SourceURL:     "https://www.kenyalaw.org/kl/index.php?id=398",
        Chapters:      KenyaConstitutionChapters,
}

// KenyaPresidents is the authoritative list of Kenyan presidents since
// independence. Spec section 10.
var KenyaPresidents = []government.President{
        {ID: "president-jomo-kenyatta", CountryCode: "KE", FullName: "Jomo Kenyatta", DisplayName: "Jomo Kenyatta"},
        {ID: "president-daniel-arap-moi", CountryCode: "KE", FullName: "Daniel arap Moi", DisplayName: "Daniel arap Moi"},
        {ID: "president-mwai-kibaki", CountryCode: "KE", FullName: "Mwai Kibaki", DisplayName: "Mwai Kibaki"},
        {ID: "president-uhuru-kenyatta", CountryCode: "KE", FullName: "Uhuru Kenyatta", DisplayName: "Uhuru Kenyatta"},
        {ID: "president-william-ruto", CountryCode: "KE", FullName: "William Ruto", DisplayName: "William Ruto"},
}

// KenyaAdministrations is the authoritative list of administrations.
var KenyaAdministrations = []government.Administration{
        {
                ID:               "admin-jomo-kenyatta",
                CountryCode:      "KE",
                PresidentID:      "president-jomo-kenyatta",
                Name:             "Jomo Kenyatta Administration",
                StartDate:        time.Date(1963, 12, 12, 0, 0, 0, 0, time.UTC),
                EndDate:          ptrTime(time.Date(1978, 8, 22, 0, 0, 0, 0, time.UTC)),
                GovernmentSystem: government.GovernmentSystemPresidential,
        },
        {
                ID:               "admin-daniel-arap-moi",
                CountryCode:      "KE",
                PresidentID:      "president-daniel-arap-moi",
                Name:             "Daniel arap Moi Administration",
                StartDate:        time.Date(1978, 8, 22, 0, 0, 0, 0, time.UTC),
                EndDate:          ptrTime(time.Date(2002, 12, 30, 0, 0, 0, 0, time.UTC)),
                GovernmentSystem: government.GovernmentSystemPresidential,
        },
        {
                ID:               "admin-mwai-kibaki",
                CountryCode:      "KE",
                PresidentID:      "president-mwai-kibaki",
                Name:             "Mwai Kibaki Administration",
                StartDate:        time.Date(2002, 12, 30, 0, 0, 0, 0, time.UTC),
                EndDate:          ptrTime(time.Date(2013, 4, 9, 0, 0, 0, 0, time.UTC)),
                GovernmentSystem: government.GovernmentSystemPresidential,
        },
        {
                ID:               "admin-uhuru-kenyatta",
                CountryCode:      "KE",
                PresidentID:      "president-uhuru-kenyatta",
                Name:             "Uhuru Kenyatta Administration",
                StartDate:        time.Date(2013, 4, 9, 0, 0, 0, 0, time.UTC),
                EndDate:          ptrTime(time.Date(2022, 9, 13, 0, 0, 0, 0, time.UTC)),
                GovernmentSystem: government.GovernmentSystemPresidential,
        },
        {
                ID:               "admin-william-ruto",
                CountryCode:      "KE",
                PresidentID:      "president-william-ruto",
                Name:             "William Ruto Administration",
                StartDate:        time.Date(2022, 9, 13, 0, 0, 0, 0, time.UTC),
                EndDate:          nil,
                GovernmentSystem: government.GovernmentSystemPresidential,
        },
}

// KenyaPresidentialTerms is the authoritative list of presidential terms.
// A president may have multiple terms. The data model supports N terms.
var KenyaPresidentialTerms = []government.PresidentialTerm{
        {ID: "term-jomo-1", AdministrationID: "admin-jomo-kenyatta", PresidentID: "president-jomo-kenyatta", TermNumber: 1, StartDate: time.Date(1963, 12, 12, 0, 0, 0, 0, time.UTC), EndDate: ptrTime(time.Date(1969, 12, 6, 0, 0, 0, 0, time.UTC)), Status: "COMPLETED"},
        {ID: "term-jomo-2", AdministrationID: "admin-jomo-kenyatta", PresidentID: "president-jomo-kenyatta", TermNumber: 2, StartDate: time.Date(1969, 12, 6, 0, 0, 0, 0, time.UTC), EndDate: ptrTime(time.Date(1974, 10, 14, 0, 0, 0, 0, time.UTC)), Status: "COMPLETED"},
        {ID: "term-jomo-3", AdministrationID: "admin-jomo-kenyatta", PresidentID: "president-jomo-kenyatta", TermNumber: 3, StartDate: time.Date(1974, 10, 14, 0, 0, 0, 0, time.UTC), EndDate: ptrTime(time.Date(1978, 8, 22, 0, 0, 0, 0, time.UTC)), Status: "INTERRUPTED"},
        {ID: "term-moi-1", AdministrationID: "admin-daniel-arap-moi", PresidentID: "president-daniel-arap-moi", TermNumber: 1, StartDate: time.Date(1978, 10, 10, 0, 0, 0, 0, time.UTC), EndDate: ptrTime(time.Date(1983, 9, 29, 0, 0, 0, 0, time.UTC)), Status: "COMPLETED"},
        {ID: "term-moi-2", AdministrationID: "admin-daniel-arap-moi", PresidentID: "president-daniel-arap-moi", TermNumber: 2, StartDate: time.Date(1983, 9, 29, 0, 0, 0, 0, time.UTC), EndDate: ptrTime(time.Date(1988, 3, 4, 0, 0, 0, 0, time.UTC)), Status: "COMPLETED"},
        {ID: "term-moi-3", AdministrationID: "admin-daniel-arap-moi", PresidentID: "president-daniel-arap-moi", TermNumber: 3, StartDate: time.Date(1988, 3, 4, 0, 0, 0, 0, time.UTC), EndDate: ptrTime(time.Date(1992, 12, 29, 0, 0, 0, 0, time.UTC)), Status: "COMPLETED"},
        {ID: "term-moi-4", AdministrationID: "admin-daniel-arap-moi", PresidentID: "president-daniel-arap-moi", TermNumber: 4, StartDate: time.Date(1992, 12, 29, 0, 0, 0, 0, time.UTC), EndDate: ptrTime(time.Date(1997, 12, 29, 0, 0, 0, 0, time.UTC)), Status: "COMPLETED"},
        {ID: "term-moi-5", AdministrationID: "admin-daniel-arap-moi", PresidentID: "president-daniel-arap-moi", TermNumber: 5, StartDate: time.Date(1997, 12, 29, 0, 0, 0, 0, time.UTC), EndDate: ptrTime(time.Date(2002, 12, 30, 0, 0, 0, 0, time.UTC)), Status: "COMPLETED"},
        {ID: "term-kibaki-1", AdministrationID: "admin-mwai-kibaki", PresidentID: "president-mwai-kibaki", TermNumber: 1, StartDate: time.Date(2002, 12, 30, 0, 0, 0, 0, time.UTC), EndDate: ptrTime(time.Date(2013, 4, 9, 0, 0, 0, 0, time.UTC)), Status: "COMPLETED"},
        {ID: "term-uhuru-1", AdministrationID: "admin-uhuru-kenyatta", PresidentID: "president-uhuru-kenyatta", TermNumber: 1, StartDate: time.Date(2013, 4, 9, 0, 0, 0, 0, time.UTC), EndDate: ptrTime(time.Date(2017, 11, 28, 0, 0, 0, 0, time.UTC)), Status: "COMPLETED"},
        {ID: "term-uhuru-2", AdministrationID: "admin-uhuru-kenyatta", PresidentID: "president-uhuru-kenyatta", TermNumber: 2, StartDate: time.Date(2017, 11, 28, 0, 0, 0, 0, time.UTC), EndDate: ptrTime(time.Date(2022, 9, 13, 0, 0, 0, 0, time.UTC)), Status: "COMPLETED"},
        {ID: "term-ruto-1", AdministrationID: "admin-william-ruto", PresidentID: "president-william-ruto", TermNumber: 1, StartDate: time.Date(2022, 9, 13, 0, 0, 0, 0, time.UTC), EndDate: nil, Status: "CURRENT"},
}

// KenyaGovernmentTransitions records the formal handover events.
var KenyaGovernmentTransitions = []government.GovernmentTransition{
        {
                ID:                  "transition-1978",
                CountryCode:         "KE",
                OutgoingAdminID:     ptrID("admin-jomo-kenyatta"),
                IncomingAdminID:     "admin-daniel-arap-moi",
                TransitionDate:      time.Date(1978, 8, 22, 0, 0, 0, 0, time.UTC),
                OutgoingPresidentID: ptrID("president-jomo-kenyatta"),
                IncomingPresidentID: "president-daniel-arap-moi",
        },
        {
                ID:                  "transition-2002",
                CountryCode:         "KE",
                OutgoingAdminID:     ptrID("admin-daniel-arap-moi"),
                IncomingAdminID:     "admin-mwai-kibaki",
                TransitionDate:      time.Date(2002, 12, 30, 0, 0, 0, 0, time.UTC),
                OutgoingPresidentID: ptrID("president-daniel-arap-moi"),
                IncomingPresidentID: "president-mwai-kibaki",
        },
        {
                ID:                  "transition-2013",
                CountryCode:         "KE",
                OutgoingAdminID:     ptrID("admin-mwai-kibaki"),
                IncomingAdminID:     "admin-uhuru-kenyatta",
                TransitionDate:      time.Date(2013, 4, 9, 0, 0, 0, 0, time.UTC),
                OutgoingPresidentID: ptrID("president-mwai-kibaki"),
                IncomingPresidentID: "president-uhuru-kenyatta",
        },
        {
                ID:                  "transition-2022",
                CountryCode:         "KE",
                OutgoingAdminID:     ptrID("admin-uhuru-kenyatta"),
                IncomingAdminID:     "admin-william-ruto",
                TransitionDate:      time.Date(2022, 9, 13, 0, 0, 0, 0, time.UTC),
                OutgoingPresidentID: ptrID("president-uhuru-kenyatta"),
                IncomingPresidentID: "president-william-ruto",
        },
}

func ptrTime(t time.Time) *time.Time { return &t }
func ptrID(s string) *government.ID      { id := government.ID(s); return &id }
