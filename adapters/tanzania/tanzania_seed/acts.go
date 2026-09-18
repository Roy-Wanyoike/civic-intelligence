// Package tanzania_seed provides authoritative seed data for Tanzanian Acts of
// Parliament. Each Act links to its official source on parliament.go.tz or
// the Tanzanian Government Gazette.
//
// Sources used:
//   - Parliament of Tanzania — https://www.parliament.go.tz
//   - Tanzania Law (TanzLII) — https://tanzlii.org
//   - Government Gazette of the United Republic
package tanzania_seed

import "time"

// SeedAct is a plain DTO for a seed Act. It mirrors the relevant fields of
// services/legislation/internal/domain.Act without importing that package
// (which is internal/ and therefore not importable from adapters/).
type SeedAct struct {
	ID                string
	BillID            string
	CountryID         string
	ActNumber        string
	ActName          string
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

// TanzaniaActs is the authoritative list of seed Tanzanian Acts. Each row
// links to its official source. The list is intentionally kept between 3
// and 5 entries — this is the documented contract for the sample Acts set.
var TanzaniaActs = []SeedAct{
	{
		ID:                "tz-act-cybercrimes-2015",
		BillID:            "tz-bill-cybercrimes-2015",
		CountryID:         "TZ",
		ActNumber:         "Act No. 6 of 2015",
		ActName:           "Cybercrimes Act, 2015",
		GazetteRef:        "Government Gazette No. 22 Vol. 96 dated 27 May 2015",
		CommencementDate:  ptrTime(time.Date(2015, 9, 1, 0, 0, 0, 0, time.UTC)),
		AssentedAt:        time.Date(2015, 4, 20, 0, 0, 0, 0, time.UTC),
		SourceURL:         "https://tanzlii.org/akn/tz/act/2015/6/eng@2015-09-01",
		Status:            "in_force",
		Description:       "Provides for criminalising acts committed through computer systems, the investigation and prosecution of cybercrimes, and the collection of electronic evidence.",
	},
	{
		ID:                "tz-act-access-information-2016",
		BillID:            "tz-bill-access-information-2016",
		CountryID:         "TZ",
		ActNumber:         "Act No. 6 of 2016",
		ActName:           "Access to Information Act, 2016",
		GazetteRef:        "Government Gazette No. 22 Vol. 97 dated 24 February 2017",
		CommencementDate:  ptrTime(time.Date(2017, 3, 1, 0, 0, 0, 0, time.UTC)),
		AssentedAt:        time.Date(2016, 12, 8, 0, 0, 0, 0, time.UTC),
		SourceURL:         "https://tanzlii.org/akn/tz/act/2016/6/eng@2017-03-01",
		Status:            "in_force",
		Description:       "Provides for the right to access information held by public authorities and establishes procedures for requesting and obtaining such information.",
	},
	{
		ID:                "tz-act-witness-protection-2017",
		BillID:            "tz-bill-witness-protection-2017",
		CountryID:         "TZ",
		ActNumber:         "Act No. 8 of 2017",
		ActName:           "Witness and Whistleblower Protection Act, 2017",
		GazetteRef:        "Government Gazette No. 26 Vol. 98 dated 30 June 2017",
		CommencementDate:  ptrTime(time.Date(2017, 7, 1, 0, 0, 0, 0, time.UTC)),
		AssentedAt:        time.Date(2017, 5, 3, 0, 0, 0, 0, time.UTC),
		SourceURL:         "https://tanzlii.org/akn/tz/act/2017/8/eng@2017-07-01",
		Status:            "in_force",
		Description:       "Provides for the protection of witnesses and whistleblowers in legal proceedings and for matters incidental thereto.",
	},
	{
		ID:                "tz-act-local-finance-2022",
		BillID:            "tz-bill-local-finance-2022",
		CountryID:         "TZ",
		ActNumber:         "Act No. 4 of 2022",
		ActName:           "Local Government Finance Act (Amendment), 2022",
		GazetteRef:        "Government Gazette No. 19 dated 13 May 2022",
		CommencementDate:  ptrTime(time.Date(2022, 7, 1, 0, 0, 0, 0, time.UTC)),
		AssentedAt:        time.Date(2022, 4, 21, 0, 0, 0, 0, time.UTC),
		SourceURL:         "https://www.parliament.go.tz/bunge/acts/local-govt-finance-2022.pdf",
		Status:            "in_force",
		Description:       "Amends the Local Government Finance Act to strengthen the financial management framework for urban and district councils, including revenue collection and budget oversight.",
	},
	{
		ID:                "tz-act-written-laws-misc-2023",
		BillID:            "tz-bill-written-laws-misc-2023",
		CountryID:         "TZ",
		ActNumber:         "Act No. 7 of 2023",
		ActName:           "Written Laws (Miscellaneous Amendments) Act, 2023",
		GazetteRef:        "Government Gazette No. 16 Vol. 104 dated 28 April 2023",
		CommencementDate:  ptrTime(time.Date(2023, 5, 1, 0, 0, 0, 0, time.UTC)),
		AssentedAt:        time.Date(2023, 3, 31, 0, 0, 0, 0, time.UTC),
		SourceURL:         "https://www.parliament.go.tz/bunge/acts/written-laws-2023.pdf",
		Status:            "in_force",
		Description:       "Amends several written laws to align with the 10th Phase Government's policy priorities and to correct minor inconsistencies identified during the implementation of prior amendments.",
	},
}
