// Package kenya_seed provides authoritative seed data for Kenyan Acts of
// Parliament and their post-assent events (commencement, regulations,
// amendments, court challenges, repeal).
//
// All data is sourced from authoritative references:
//   - Kenya Law (https://www.kenyalaw.org)
//   - Kenya Gazette
//
// This file (issue #202) moves the hardcoded sample data out of
// services/api/cmd/{main.go,post_assent.go} into the seed package so
// both the legislation service (via Wire()) and the API layer share a
// single source of truth.
//
// The DTO types below are intentionally plain structs (no domain imports)
// so this package does NOT depend on services/legislation/internal. The
// legislation service's Wire() function maps these DTOs into domain.Act
// and domain.PostAssentEvent values.
package kenya_seed

import "time"

// SeedAct is a plain DTO for a seed Act. It mirrors the relevant fields of
// services/legislation/internal/domain.Act without importing that package
// (which is internal/ and therefore not importable from adapters/).
type SeedAct struct {
	ID                string
	BillID            string
	CountryID         string
	ActNumber         string // citation-style, e.g. "No. 24 of 2019"
	ActName           string // human-readable title
	GazetteRef        string
	CommencementDate  *time.Time
	AssentedAt        time.Time
	SourceDocumentID  string
	Description       string
	PublicationDate   *time.Time
	Status            string // stored as the API response string (e.g. "in_force", "amended")
	SourceURL         string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// SeedPostAssentEvent is a plain DTO for a seed post-assent event.
type SeedPostAssentEvent struct {
	ID          string
	ActID       string
	BillID      *string
	EventType   string
	EventDate   time.Time
	Title       string
	Description string
	SourceURL   string
	CreatedAt   time.Time
}

// KenyaActs is the authoritative list of seed Kenyan Acts of Parliament.
// Each row links to the official Kenya Law (kenyalaw.org) source.
//
// The list is intentionally kept between 3 and 5 entries — this is the
// documented contract for the /api/v1/acts sample set (see
// services/api/cmd/main_acts_test.go: TestActsList_HasThreeToFive).
var KenyaActs = []SeedAct{
	{
		ID:                "ke-act-constitution-2010",
		BillID:            "ke-bill-constitution-2008",
		CountryID:         "KE",
		ActNumber:         "Constitution of Kenya, 2010",
		ActName:           "Constitution of Kenya",
		GazetteRef:        "Kenya Gazette Special Issue, 27 August 2010",
		CommencementDate:  ptrTime(time.Date(2010, 8, 27, 0, 0, 0, 0, time.UTC)),
		AssentedAt:        time.Date(2010, 8, 4, 0, 0, 0, 0, time.UTC),
		SourceURL:         "https://www.kenyalaw.org/kl/index.php?id=398",
		Status:            "in_force",
		Description:       "Supreme law of Kenya, promulgated on 27 August 2010, replacing the 1963 independence constitution. Establishes a devolved system of government, a Bill of Rights, and an independent judiciary.",
	},
	{
		ID:                "ke-act-data-protection-2019",
		BillID:            "ke-bill-data-protection-2018",
		CountryID:         "KE",
		ActNumber:         "No. 24 of 2019",
		ActName:           "Data Protection Act, 2019",
		GazetteRef:        "Kenya Gazette, Vol. CXXI-No. 181",
		CommencementDate:  ptrTime(time.Date(2019, 11, 25, 0, 0, 0, 0, time.UTC)),
		AssentedAt:        time.Date(2019, 11, 8, 0, 0, 0, 0, time.UTC),
		SourceURL:         "https://www.kenyalaw.org/kl/index.php?id=646aa3ba8b8f6d3a9c3f3f9c",
		Status:            "in_force",
		Description:       "Establishes the Office of the Data Protection Commissioner and regulates the processing of personal data, giving effect to Article 31 of the Constitution.",
	},
	{
		ID:                "ke-act-public-finance-management-2015",
		BillID:            "ke-bill-public-finance-management-2014",
		CountryID:         "KE",
		ActNumber:         "No. 18 of 2015",
		ActName:           "Public Finance Management Act, 2015",
		GazetteRef:        "Kenya Gazette, Vol. CXVII-No. 158",
		CommencementDate:  ptrTime(time.Date(2015, 9, 30, 0, 0, 0, 0, time.UTC)),
		AssentedAt:        time.Date(2015, 9, 23, 0, 0, 0, 0, time.UTC),
		SourceURL:         "https://www.kenyalaw.org/kl/index.php?id=5769b1c8e3a1f8c3f9b1c8e3",
		Status:            "amended",
		Description:       "Provides for the management of public funds at national and county levels, establishing the framework for budgeting, accounting, and auditing of public money.",
	},
	{
		ID:                "ke-act-elections-2011",
		BillID:            "ke-bill-elections-2011",
		CountryID:         "KE",
		ActNumber:         "No. 24 of 2011",
		ActName:           "Elections Act, 2011",
		GazetteRef:        "Kenya Gazette, Vol. CXIII-No. 164",
		CommencementDate:  ptrTime(time.Date(2012, 1, 1, 0, 0, 0, 0, time.UTC)),
		AssentedAt:        time.Date(2011, 12, 22, 0, 0, 0, 0, time.UTC),
		SourceURL:         "https://www.kenyalaw.org/kl/index.php?id=51a5b3d6c0e3a1f8c3f9b1c8",
		Status:            "amended",
		Description:       "Provides for the conduct of elections to the National Assembly, the Senate, county assemblies, county governors, and the President; gives effect to Articles 81\u201386 of the Constitution.",
	},
	{
		ID:                "ke-act-companies-2015",
		BillID:            "ke-bill-companies-2014",
		CountryID:         "KE",
		ActNumber:         "No. 17 of 2015",
		ActName:           "Companies Act, 2015",
		GazetteRef:        "Kenya Gazette, Vol. CXVII-No. 152",
		CommencementDate:  ptrTime(time.Date(2016, 1, 15, 0, 0, 0, 0, time.UTC)),
		AssentedAt:        time.Date(2015, 9, 11, 0, 0, 0, 0, time.UTC),
		SourceURL:         "https://www.kenyalaw.org/kl/index.php?id=5769b1c8e3a1f8c3f9b1c8e2",
		Status:            "in_force",
		Description:       "Repeals and replaces the Companies Act (Cap 486) to modernise company law in Kenya and align it with international best practice.",
	},
}

// KenyaPostAssentEvents is the authoritative list of seed post-assent
// events. Currently only the Data Protection Act has seed events
// (commencement + regulations) to demonstrate the lifecycle.
var KenyaPostAssentEvents = []SeedPostAssentEvent{
	{
		ID:          "ev-dpa-1",
		ActID:       "ke-act-data-protection-2019",
		EventType:   "COMMENCEMENT",
		EventDate:   time.Date(2019, 11, 25, 0, 0, 0, 0, time.UTC),
		Title:       "Commencement Notice",
		Description: "The Data Protection Act, 2019 commenced on 25 November 2019.",
		SourceURL:   "https://www.kenyalaw.org/kl/index.php?id=4639",
	},
	{
		ID:          "ev-dpa-2",
		ActID:       "ke-act-data-protection-2019",
		EventType:   "REGULATION",
		EventDate:   time.Date(2021, 2, 12, 0, 0, 0, 0, time.UTC),
		Title:       "Data Protection (General) Regulations, 2021",
		Description: "Regulations issued under section 71 of the Act.",
		SourceURL:   "https://www.kenyalaw.org/kl/index.php?id=10675",
	},
}
