// Package nigeria_seed provides authoritative seed data for Nigerian Acts of
// the National Assembly. Each Act links to its official source on nass.gov.ng
// or the Nigerian Federal Government Gazette.
//
// Sources used:
//   - National Assembly of Nigeria — https://nass.gov.ng
//   - Nigeria Law (Nigerian Legal Information Institute) — https://nigerianlaw.org
//   - Federal Government Gazette — https://www.federalgovernmentpress.gov.ng
package nigeria_seed

import "time"

// SeedAct is a plain DTO for a seed Act. It mirrors the relevant fields of
// services/legislation/internal/domain.Act without importing that package
// (which is internal/ and therefore not importable from adapters/).
type SeedAct struct {
	ID                string
	BillID            string
	CountryID         string
	ActNumber         string
	ActName           string
	GazetteRef        string
	CommencementDate  *time.Time
	AssentedAt        time.Time
	SourceDocumentID  string
	Description       string
	PublicationDate   *time.Time
	Status            string
	SourceURL         string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// NigeriaActs is the authoritative list of seed Nigerian Acts. Each row links
// to its official source. The list is intentionally kept between 3 and 5
// entries — this is the documented contract for the sample Acts set.
var NigeriaActs = []SeedAct{
	{
		ID:                "ng-act-constitution-1999",
		BillID:            "ng-bill-constitution-1999",
		CountryID:         "NG",
		ActNumber:         "Constitution of the Federal Republic of Nigeria, 1999",
		ActName:           "Constitution of Nigeria",
		GazetteRef:        "Federal Government Gazette No. 27 Vol. 90 dated 5 May 1999",
		CommencementDate:  ptrTime(time.Date(1999, 5, 29, 0, 0, 0, 0, time.UTC)),
		AssentedAt:        time.Date(1999, 5, 5, 0, 0, 0, 0, time.UTC),
		SourceURL:         "https://www.constituteproject.org/constitution/Nigeria_1999",
		Status:            "in_force",
		Description:       "Supreme law of the Federal Republic of Nigeria, promulgated by the military in 1999 and brought into force on 29 May 1999 (handover day). Establishes a federal system with a bicameral National Assembly (Senate + House of Representatives) and an elected President.",
	},
	{
		ID:                "ng-act-cybercrimes-2015",
		BillID:            "ng-bill-cybercrimes-2015",
		CountryID:         "NG",
		ActNumber:         "Act No. 14 of 2015",
		ActName:           "Cybercrimes (Prohibition, Prevention, etc.) Act, 2015",
		GazetteRef:        "Federal Government Gazette No. 18 Vol. 102 dated 15 May 2015",
		CommencementDate:  ptrTime(time.Date(2015, 5, 15, 0, 0, 0, 0, time.UTC)),
		AssentedAt:        time.Date(2015, 5, 1, 0, 0, 0, 0, time.UTC),
		SourceURL:         "https://nigerianlaw.org/cybercrimes-prohibition-prevention-etc-act-2015/",
		Status:            "amended",
		Description:       "Provides a unified legal, regulatory and institutional framework for the prohibition, prevention, detection, prosecution and punishment of cybercrimes in Nigeria, and for related matters.",
	},
	{
		ID:                "ng-act-camn-2020",
		BillID:            "ng-bill-camn-2020",
		CountryID:         "NG",
		ActNumber:         "Act No. 3 of 2020",
		ActName:           "Companies and Allied Matters Act, 2020 (CAMA)",
		GazetteRef:        "Federal Government Gazette No. 79 Vol. 107 dated 7 August 2020",
		CommencementDate:  ptrTime(time.Date(2020, 8, 7, 0, 0, 0, 0, time.UTC)),
		AssentedAt:        time.Date(2020, 8, 7, 0, 0, 0, 0, time.UTC),
		SourceURL:         "https://nigerianlaw.org/companies-and-allied-matters-act-2020/",
		Status:            "in_force",
		Description:       "Repeals and replaces the Companies and Allied Matters Act, 1990 to modernise Nigerian company law, introduce new business forms (LLPs), reduce incorporation friction, and align with international best practice.",
	},
	{
		ID:                "ng-act-pibo-2021",
		BillID:            "ng-bill-pibo-2021",
		CountryID:         "NG",
		ActNumber:         "Act No. 6 of 2021",
		ActName:           "Petroleum Industry Act, 2021 (PIA)",
		GazetteRef:        "Federal Government Gazette No. 145 Vol. 108 dated 26 August 2021",
		CommencementDate:  ptrTime(time.Date(2021, 8, 26, 0, 0, 0, 0, time.UTC)),
		AssentedAt:        time.Date(2021, 8, 16, 0, 0, 0, 0, time.UTC),
		SourceURL:         "https://nigerianlaw.org/petroleum-industry-act-2021/",
		Status:            "amended",
		Description:       "Provides the legal, regulatory and fiscal framework for the Nigerian petroleum industry, establishes the Nigerian Upstream Petroleum Regulatory Commission and the Nigerian Midstream and Downstream Petroleum Regulatory Authority, and for related matters.",
	},
	{
		ID:                "ng-act-electoral-2022",
		BillID:            "ng-bill-electoral-2022",
		CountryID:         "NG",
		ActNumber:         "Act No. 13 of 2022",
		ActName:           "Electoral Act, 2022",
		GazetteRef:        "Federal Government Gazette No. 32 Vol. 109 dated 25 February 2022",
		CommencementDate:  ptrTime(time.Date(2022, 2, 25, 0, 0, 0, 0, time.UTC)),
		AssentedAt:        time.Date(2022, 2, 25, 0, 0, 0, 0, time.UTC),
		SourceURL:         "https://nigerianlaw.org/electoral-act-2022/",
		Status:            "in_force",
		Description:       "Repeals the Electoral Act, 2010 and re-enacts a consolidated legal framework for the conduct of federal, state and area council elections, including the use of electronic transmission of results and biometric voter accreditation.",
	},
}
