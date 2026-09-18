// Package south_africa_seed provides authoritative seed data for South African
// Acts of Parliament. Each Act links to its official source on
// parliament.gov.za or the South African Government Gazette.
//
// Sources used:
//   - Parliament of South Africa — https://www.parliament.gov.za
//   - Southern African Legal Information Institute (SAFLII) — https://www.saflii.org
//   - Government Gazette of the Republic of South Africa
package south_africa_seed

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

// SouthAfricaActs is the authoritative list of seed South African Acts.
// Each row links to its official source. The list is intentionally kept
// between 3 and 5 entries — this is the documented contract for the sample
// Acts set.
var SouthAfricaActs = []SeedAct{
	{
		ID:                "za-act-constitution-1996",
		BillID:            "za-bill-constitution-1996",
		CountryID:         "ZA",
		ActNumber:         "Constitution of the Republic of South Africa, 1996",
		ActName:           "Constitution of South Africa",
		GazetteRef:        "Government Gazette No. 17678 dated 18 December 1996",
		CommencementDate:  ptrTime(time.Date(1997, 2, 4, 0, 0, 0, 0, time.UTC)),
		AssentedAt:        time.Date(1996, 12, 10, 0, 0, 0, 0, time.UTC),
		SourceURL:         "https://www.gov.za/documents/constitution/constitution-republic-south-africa-1996",
		Status:            "in_force",
		Description:       "Supreme law of the Republic of South Africa, certified by the Constitutional Court on 4 December 1996 and brought into force on 4 February 1997. Establishes a constitutional democracy with a bicameral Parliament (National Assembly + National Council of Provinces) and a President elected by the NA.",
	},
	{
		ID:                "za-act-potic-2000",
		BillID:            "za-bill-potic-2000",
		CountryID:         "ZA",
		ActNumber:         "Act No. 4 of 2000",
		ActName:           "Promotion of Access to Information Act, 2000 (PAIA)",
		GazetteRef:        "Government Gazette No. 20852 dated 3 February 2000",
		CommencementDate:  ptrTime(time.Date(2001, 3, 9, 0, 0, 0, 0, time.UTC)),
		AssentedAt:        time.Date(2000, 2, 3, 0, 0, 0, 0, time.UTC),
		SourceURL:         "https://www.saflii.org/za/legis/num_reg/potaia2000318/",
		Status:            "amended",
		Description:       "Gives effect to section 32 of the Constitution (the right of access to information). Establishes the procedures for requesting and obtaining information held by public and private bodies.",
	},
	{
		ID:                "za-act-pipa-2013",
		BillID:            "za-bill-pipa-2013",
		CountryID:         "ZA",
		ActNumber:         "Act No. 4 of 2013",
		ActName:           "Protection of Personal Information Act, 2013 (POPIA)",
		GazetteRef:        "Government Gazette No. 37067 dated 26 November 2013",
		CommencementDate:  ptrTime(time.Date(2021, 7, 1, 0, 0, 0, 0, time.UTC)),
		AssentedAt:        time.Date(2013, 11, 19, 0, 0, 0, 0, time.UTC),
		SourceURL:         "https://www.saflii.org/za/legis/num_reg/popia2013255/",
		Status:            "in_force",
		Description:       "Establishes the Information Regulator and regulates the processing of personal information by public and private bodies, giving effect to the right to privacy in section 14 of the Constitution.",
	},
	{
		ID:                "za-act-financial-sector-2015",
		BillID:            "za-bill-financial-sector-2015",
		CountryID:         "ZA",
		ActNumber:         "Act No. 9 of 2017",
		ActName:           "Financial Sector Regulation Act, 2017",
		GazetteRef:        "Government Gazette No. 41060 dated 22 August 2017",
		CommencementDate:  ptrTime(time.Date(2018, 4, 1, 0, 0, 0, 0, time.UTC)),
		AssentedAt:        time.Date(2017, 8, 21, 0, 0, 0, 0, time.UTC),
		SourceURL:         "https://www.resbank.co.za/RegulationAndSupervision/FinancialSectorLaws/Pages/default.aspx",
		Status:            "in_force",
		Description:       "Implements a Twin Peaks model of financial regulation. Establishes the Financial Sector Conduct Authority (FSCA) and the Prudential Authority (within the SARB) and consolidates the financial regulatory architecture.",
	},
	{
		ID:                "za-act-nhi-2023",
		BillID:            "za-bill-nhi-2023",
		CountryID:         "ZA",
		ActNumber:         "Act No. 13 of 2023",
		ActName:           "National Health Insurance Act, 2023",
		GazetteRef:        "Government Gazette No. 50345 dated 18 December 2024",
		CommencementDate:  nil,
		AssentedAt:        time.Date(2024, 5, 15, 0, 0, 0, 0, time.UTC),
		SourceURL:         "https://www.parliament.gov.za/bills-and-laws/b-11-2022",
		Status:            "in_force",
		Description:       "Establishes the National Health Insurance Fund to provide universal access to quality healthcare services for all South Africans, free at the point of care, financed through mandatory prepayment. The Act commenced upon gazettement of the proclamation.",
	},
}
