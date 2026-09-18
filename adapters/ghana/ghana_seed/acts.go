// Package ghana_seed provides authoritative seed data for Ghanaian Acts of
// Parliament. Each Act links to its official source on parliament.gh or the
// Ghana Gazette.
//
// Sources used:
//   - Parliament of Ghana — https://parliament.gh
//   - GhanaLII — https://ghanalii.org
//   - Government Gazette of the Republic of Ghana
package ghana_seed

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

// GhanaActs is the authoritative list of seed Ghanaian Acts. Each row links
// to its official source. The list is intentionally kept between 3 and 5
// entries — this is the documented contract for the sample Acts set.
var GhanaActs = []SeedAct{
	{
		ID:                "gh-act-constitution-1992",
		BillID:            "gh-bill-constitution-1992",
		CountryID:         "GH",
		ActNumber:         "Constitution of the Republic of Ghana, 1992",
		ActName:           "Constitution of Ghana",
		GazetteRef:        "Ghana Gazette, 28 April 1992 (referendum date)",
		CommencementDate:  ptrTime(time.Date(1993, 1, 7, 0, 0, 0, 0, time.UTC)),
		AssentedAt:        time.Date(1992, 4, 28, 0, 0, 0, 0, time.UTC),
		SourceURL:         "https://www.wipo.int/wipolex/en/text/129013",
		Status:            "in_force",
		Description:       "Supreme law of the Fourth Republic of Ghana, adopted by referendum on 28 April 1992 and brought into force on 7 January 1993. Establishes a unitary state with a unicameral Parliament and an elected President.",
	},
	{
		ID:                "gh-act-right-to-information-2019",
		BillID:            "gh-bill-right-to-information-2018",
		CountryID:         "GH",
		ActNumber:         "Act 989",
		ActName:           "Right to Information Act, 2019 (Act 989)",
		GazetteRef:        "Ghana Gazette, 31 January 2019",
		CommencementDate:  ptrTime(time.Date(2020, 1, 6, 0, 0, 0, 0, time.UTC)),
		AssentedAt:        time.Date(2019, 5, 21, 0, 0, 0, 0, time.UTC),
		SourceURL:         "https://ghanalii.org/akn/gh/act/2019/989/eng@2019-05-21",
		Status:            "in_force",
		Description:       "Provides for the right to information held by public institutions, the procedures for obtaining that information, and the establishment of the Right to Information Commission.",
	},
	{
		ID:                "gh-act-data-protection-2012",
		BillID:            "gh-bill-data-protection-2012",
		CountryID:         "GH",
		ActNumber:         "Act 843",
		ActName:           "Data Protection Act, 2012 (Act 843)",
		GazetteRef:        "Ghana Gazette, 18 October 2012",
		CommencementDate:  ptrTime(time.Date(2012, 10, 18, 0, 0, 0, 0, time.UTC)),
		AssentedAt:        time.Date(2012, 5, 16, 0, 0, 0, 0, time.UTC),
		SourceURL:         "https://ghanalii.org/akn/gh/act/2012/843/eng@2012-10-16",
		Status:            "in_force",
		Description:       "Establishes the Data Protection Commission and regulates the processing of personal data to safeguard the privacy of the individual.",
	},
	{
		ID:                "gh-act-public-financial-mgmt-2016",
		BillID:            "gh-bill-public-financial-mgmt-2016",
		CountryID:         "GH",
		ActNumber:         "Act 921",
		ActName:           "Public Financial Management Act, 2016 (Act 921)",
		GazetteRef:        "Ghana Gazette, 22 August 2016",
		CommencementDate:  ptrTime(time.Date(2016, 8, 22, 0, 0, 0, 0, time.UTC)),
		AssentedAt:        time.Date(2016, 8, 19, 0, 0, 0, 0, time.UTC),
		SourceURL:         "https://ghanalii.org/akn/gh/act/2016/921/eng@2016-08-19",
		Status:            "amended",
		Description:       "Provides for the regulation of the financial management of the public sector, the management of public funds, and the audit of public accounts within the framework of an integrated financial management system.",
	},
	{
		ID:                "gh-act-anti-money-laundering-2020",
		BillID:            "gh-bill-anti-money-laundering-2020",
		CountryID:         "GH",
		ActNumber:         "Act 1044",
		ActName:           "Anti-Money Laundering Act, 2020 (Act 1044)",
		GazetteRef:        "Ghana Gazette, 18 December 2020",
		CommencementDate:  ptrTime(time.Date(2020, 12, 18, 0, 0, 0, 0, time.UTC)),
		AssentedAt:        time.Date(2020, 8, 14, 0, 0, 0, 0, time.UTC),
		SourceURL:         "https://ghanalii.org/akn/gh/act/2020/1044/eng@2020-12-18",
		Status:            "in_force",
		Description:       "Consolidates the law on anti-money laundering and countering the financing of terrorism, gives effect to the Financial Action Task Force (FATF) Recommendations, and establishes the Centre for the Coordination of the Fight against Money Laundering.",
	},
}
