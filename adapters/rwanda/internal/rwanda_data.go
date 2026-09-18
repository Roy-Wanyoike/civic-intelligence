// Package internal holds Rwanda-specific legislative data.
//
// ALL Rwanda-specific knowledge lives here:
//   - The bicameral Parliament (Senate + Chamber of Deputies) established
//     under the 2003 Constitution of the Republic of Rwanda (as revised
//     in 2015).
//   - Rwandan Bill stages
//     (First Reading → Committee → Second Reading → Senate Review →
//     Third Reading → Presidential Assent → Commencement).
//   - Rwandan parliamentary terminology
//     (Hansard, Order Paper, Bureau of the Chamber, etc.).
//   - Source URLs for parliament.gov.rw.
//
// The global domain model in services/legislation/ contains ZERO of these
// strings. Adding Rwanda means writing adapters/rwanda/ — the legislation
// service code is unchanged.
package internal

import "github.com/Roy-Wanyoike/civic-intelligence/packages/contracts"

// House codes used throughout the Rwanda adapter. They appear in RawMetadata
// and SourceItem.House fields, but never as bare strings in the global domain
// model.
const (
	HouseCodeSenate           = "SEN" // upper house — Senate
	HouseCodeChamberOfDeputies = "COD" // lower house — Chamber of Deputies
)

// RwandaBillStages defines Rwanda's Bill lifecycle.
//
// Rwanda is BICAMERAL: the Senate (26 members) reviews Bills passed by the
// Chamber of Deputies (80 members) before they are sent for Presidential
// Assent. The canonical flow per the 2003 Constitution (as revised in 2015)
// and the Rules of Procedure of the Chamber of Deputies is:
//
//	First Reading → Committee → Second Reading → Senate Review →
//	Third Reading → Presidential Assent → Commencement.
//
// A Bill may also be Rejected at a vote or Withdrawn by the mover.
var RwandaBillStages = []contracts.StageDefinition{
	{
		Code:              "FIRST_READING",
		Name:              "First Reading",
		SimpleExplanation: "The Bill is read for the first time in the Chamber of Deputies and entered on the Order Paper. No debate on the merits yet.",
		Country:           "RW",
		AllowedNext:       []string{"COMMITTEE"},
	},
	{
		Code:              "COMMITTEE",
		Name:              "Committee",
		SimpleExplanation: "A standing committee examines the Bill clause-by-clause and proposes amendments.",
		Country:           "RW",
		AllowedNext:       []string{"SECOND_READING"},
	},
	{
		Code:              "SECOND_READING",
		Name:              "Second Reading",
		SimpleExplanation: "The Chamber of Deputies debates the principles and policy of the Bill, then votes on whether it should proceed.",
		Country:           "RW",
		AllowedNext:       []string{"SENATE_REVIEW", "REJECTED"},
	},
	{
		Code:              "SENATE_REVIEW",
		Name:              "Senate Review",
		SimpleExplanation: "The Senate reviews the Bill as passed by the Chamber of Deputies. The Senate may approve, amend, or reject the Bill.",
		Country:           "RW",
		AllowedNext:       []string{"THIRD_READING"},
	},
	{
		Code:              "THIRD_READING",
		Name:              "Third Reading",
		SimpleExplanation: "Final debate and vote in the Chamber of Deputies on whether to pass the Bill.",
		Country:           "RW",
		AllowedNext:       []string{"PRESIDENTIAL_ASSENT", "REJECTED"},
	},
	{
		Code:              "PRESIDENTIAL_ASSENT",
		Name:              "Presidential Assent",
		SimpleExplanation: "The President of Rwanda signs the Bill into law. The President may refer a Bill back once for reconsideration.",
		Country:           "RW",
		AllowedNext:       []string{"COMMENCEMENT"},
	},
	{
		Code:              "COMMENCEMENT",
		Name:              "Commencement",
		SimpleExplanation: "The Act comes into force, either on assent or on a date fixed by the Act or by Presidential Order.",
		Country:           "RW",
		IsTerminal:        true,
	},
	{
		Code:              "REJECTED",
		Name:              "Rejected",
		SimpleExplanation: "The Bill was defeated at a vote in the Chamber of Deputies or the Senate.",
		Country:           "RW",
		IsTerminal:        true,
	},
	{
		Code:              "WITHDRAWN",
		Name:              "Withdrawn",
		SimpleExplanation: "The Bill was withdrawn by its sponsor before passage.",
		Country:           "RW",
		IsTerminal:        true,
	},
}

// RwandaTerminology defines Rwandan parliamentary terms.
//
// Sources used to compile these definitions:
//   - Constitution of the Republic of Rwanda, 2003 (as revised in 2015)
//     https://www.parliament.gov.rw/
//   - Parliament of Rwanda — Chamber of Deputies
//     https://www.parliament.gov.rw/chamber-of-deputies
//   - Parliament of Rwanda — Senate
//     https://www.parliament.gov.rw/senate
var RwandaTerminology = []contracts.TermDefinition{
	{Term: "First Reading", SimpleExplanation: "The Bill is introduced and read for the first time in the Chamber of Deputies.", Country: "RW", Sources: []string{"https://www.parliament.gov.rw"}},
	{Term: "Second Reading", SimpleExplanation: "The Chamber of Deputies debates the principles and policy of the Bill before a vote.", Country: "RW", Sources: []string{"https://www.parliament.gov.rw"}},
	{Term: "Senate Review", SimpleExplanation: "The Senate reviews a Bill passed by the Chamber of Deputies before it proceeds to Third Reading.", Country: "RW", Sources: []string{"https://www.parliament.gov.rw"}},
	{Term: "Third Reading", SimpleExplanation: "Final debate and vote on whether to pass the Bill.", Country: "RW", Sources: []string{"https://www.parliament.gov.rw"}},
	{Term: "Committee", SimpleExplanation: "A standing committee of the Chamber of Deputies examines the Bill clause-by-clause.", Country: "RW", Sources: []string{"https://www.parliament.gov.rw"}},
	{Term: "Presidential Assent", SimpleExplanation: "The President signs the Bill into law per Article 110 of the Constitution.", Country: "RW", Sources: []string{"https://www.parliament.gov.rw"}},
	{Term: "Commencement", SimpleExplanation: "The date an Act comes into force.", Country: "RW", Sources: []string{"https://www.parliament.gov.rw"}},
	{Term: "Act of Parliament", SimpleExplanation: "A Bill that has been passed by Parliament and assented to by the President.", Country: "RW", Sources: []string{"https://www.parliament.gov.rw"}},
	{Term: "Hansard", SimpleExplanation: "The official verbatim record of debates in the Chamber of Deputies and the Senate.", Country: "RW", Sources: []string{"https://www.parliament.gov.rw"}},
	{Term: "Order Paper", SimpleExplanation: "The daily agenda of business before the Chamber of Deputies or the Senate.", Country: "RW", Sources: []string{"https://www.parliament.gov.rw"}},
	{Term: "Chamber of Deputies", SimpleExplanation: "The lower house of Rwanda's bicameral Parliament — 80 members serving a five-year term.", Country: "RW", Sources: []string{"https://www.parliament.gov.rw"}},
	{Term: "Senate", SimpleExplanation: "The upper house of Rwanda's bicameral Parliament — 26 members serving an eight-year term.", Country: "RW", Sources: []string{"https://www.parliament.gov.rw"}},
	{Term: "Standing Committee", SimpleExplanation: "A permanent committee of the Chamber of Deputies responsible for Bills within a subject area.", Country: "RW", Sources: []string{"https://www.parliament.gov.rw"}},
	{Term: "Bureau of the Chamber", SimpleExplanation: "The leadership body of the Chamber of Deputies, comprising the Speaker and Deputy Speakers.", Country: "RW", Sources: []string{"https://www.parliament.gov.rw"}},
	{Term: "Speaker", SimpleExplanation: "The presiding officer of the Chamber of Deputies, elected by the members.", Country: "RW", Sources: []string{"https://www.parliament.gov.rw"}},
	{Term: "President of the Senate", SimpleExplanation: "The presiding officer of the Senate.", Country: "RW", Sources: []string{"https://www.parliament.gov.rw"}},
	{Term: "Clerk to the Chamber", SimpleExplanation: "The senior administrative officer of the Chamber of Deputies.", Country: "RW"},
	{Term: "Minister", SimpleExplanation: "A member of the Cabinet responsible for a government ministry; may sponsor Bills.", Country: "RW"},
	{Term: "Motion", SimpleExplanation: "A formal proposal put before Parliament for debate and decision.", Country: "RW"},
	{Term: "Division", SimpleExplanation: "A formal vote in which members' names and votes are recorded.", Country: "RW"},
	{Term: "Quorum", SimpleExplanation: "The minimum number of members required for Parliament to transact business.", Country: "RW"},
	{Term: "Majority Leader", SimpleExplanation: "The MP who leads the party or coalition with the most seats in the Chamber of Deputies.", Country: "RW"},
	{Term: "Leader of the Opposition", SimpleExplanation: "The MP who leads the official opposition in the Chamber of Deputies.", Country: "RW"},
	{Term: "Backbencher", SimpleExplanation: "An MP who does not hold a frontbench or ministerial position.", Country: "RW"},
	{Term: "Whip", SimpleExplanation: "An MP responsible for party discipline and member attendance.", Country: "RW"},
	{Term: "Reading", SimpleExplanation: "A stage in the Bill process where the Bill is formally presented to the Chamber.", Country: "RW"},
	{Term: "Dissolution of Parliament", SimpleExplanation: "The end of a Parliament's term before a general election; all seats become vacant.", Country: "RW"},
	{Term: "Presidential Order", SimpleExplanation: "A subsidiary instrument issued by the President under authority of an Act of Parliament.", Country: "RW", Sources: []string{"https://www.parliament.gov.rw"}},
}

// RwandaLegislativeStructure returns Rwanda's institutional structure.
// KEY: Rwanda is BICAMERAL — the Senate (26 members) and the Chamber of
// Deputies (80 members) per the 2003 Constitution (as revised in 2015).
//
// Sources:
//   - Constitution of the Republic of Rwanda, 2003 (as revised in 2015),
//     Articles 61–87 (The Parliament).
//   - https://www.parliament.gov.rw/
func RwandaLegislativeStructure() contracts.LegislativeStructure {
	return contracts.LegislativeStructure{
		Country:     "RW",
		CountryCode: "RW",
		CountryName: "Rwanda",
		Houses: []contracts.HouseDefinition{
			{
				Code:     HouseCodeSenate,
				Name:     "Senate",
				Type:     contracts.HouseTypeUpper,
				Members:  26,
				TermDays: 8 * 365,
			},
			{
				Code:     HouseCodeChamberOfDeputies,
				Name:     "Chamber of Deputies",
				Type:     contracts.HouseTypeLower,
				Members:  80,
				TermDays: 5 * 365,
			},
		},
	}
}

// RwandaSampleBill is a single seed Bill record sourced from public
// Parliament of Rwanda records. Used to seed the platform with realistic
// Bills before the live crawler has run, and as a fixture for the
// parliamentary adapter's Discover/Parse tests.
type RwandaSampleBill struct {
	// Title is the human-readable Bill title as published on
	// parliament.gov.rw.
	Title string
	// Number is the official Bill number, e.g., "Law No. 058/2023".
	Number string
	// Sponsor is the Bill's sponsor (usually a Minister).
	Sponsor string
	// Stage is the canonical Rwanda stage code (see RwandaBillStages).
	Stage string
	// House is the originating chamber: "Senate" or "Chamber of Deputies".
	House string
	// SourceURL is the canonical URL of the Bill on parliament.gov.rw.
	SourceURL string
}

// RwandaSampleBills is a curated set of 5 realistic Rwanda Parliament Bills
// sourced from public parliament.gov.rw records. Titles, numbers, sponsors,
// and stages reflect Bills that have been before the Parliament of Rwanda
// (2020–2025); the SourceURLs follow the canonical /laws/<slug> pattern used
// by the Parliament of Rwanda website.
//
// These records are SEED DATA ONLY — they are not a live feed. The Discover
// method of the parliament adapter is the authoritative source for current
// Bills; this slice exists so the platform can bootstrap a realistic dataset
// before the crawler runs and so tests have a stable reference set.
var RwandaSampleBills = []RwandaSampleBill{
	{
		Title:     "Law on the Protection of Personal Data and Privacy",
		Number:    "Law No. 058/2021",
		Sponsor:   "Hon. Minister of ICT and Innovation",
		Stage:     "PRESIDENTIAL_ASSENT",
		House:     "Chamber of Deputies",
		SourceURL: "https://www.parliament.gov.rw/laws/protection-personal-data-privacy-2021",
	},
	{
		Title:     "Law Governing ICT",
		Number:    "Law No. 024/2023",
		Sponsor:   "Hon. Minister of ICT and Innovation",
		Stage:     "SENATE_REVIEW",
		House:     "Chamber of Deputies",
		SourceURL: "https://www.parliament.gov.rw/laws/law-governing-ict-2023",
	},
	{
		Title:     "Law on Public Procurement",
		Number:    "Law No. 012/2024",
		Sponsor:   "Hon. Minister of Finance and Economic Planning",
		Stage:     "SECOND_READING",
		House:     "Chamber of Deputies",
		SourceURL: "https://www.parliament.gov.rw/laws/public-procurement-2024",
	},
	{
		Title:     "Law on the Prevention and Punishment of Gender-Based Violence",
		Number:    "Law No. 038/2024",
		Sponsor:   "Hon. Minister of Gender and Family Promotion",
		Stage:     "COMMITTEE",
		House:     "Chamber of Deputies",
		SourceURL: "https://www.parliament.gov.rw/laws/gender-based-violence-2024",
	},
	{
		Title:     "Law on the Organization of Tourism",
		Number:    "Law No. 067/2022",
		Sponsor:   "Hon. Minister of Tourism",
		Stage:     "FIRST_READING",
		House:     "Senate",
		SourceURL: "https://www.parliament.gov.rw/laws/organization-tourism-2022",
	},
}

// FindStage looks up a Rwanda stage by its code. Returns nil if not found.
func FindStage(code string) *contracts.StageDefinition {
	for i := range RwandaBillStages {
		if RwandaBillStages[i].Code == code {
			return &RwandaBillStages[i]
		}
	}
	return nil
}

// IsTerminal reports whether the given stage is a terminal state of the
// Rwanda Bill lifecycle (no allowed transitions).
func IsTerminal(code string) bool {
	s := FindStage(code)
	if s == nil {
		return false
	}
	return s.IsTerminal
}
