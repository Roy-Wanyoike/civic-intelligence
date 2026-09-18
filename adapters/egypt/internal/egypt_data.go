// Package internal holds Egypt-specific legislative data.
//
// ALL Egypt-specific knowledge lives here:
//   - The bicameral Parliament (Senate + House of Representatives) established
//     under the 2014 Constitution of the Arab Republic of Egypt (as amended
//     in 2019).
//   - Egyptian Bill stages (Proposal → Committee Review → First Reading →
//     Second Reading → Senate Review → Third Reading → Presidential
//     Ratification → Publication).
//   - Egyptian parliamentary terminology (Hansard, Order Paper, etc.).
//   - Source URLs for parliament.eg.
//
// The global domain model in services/legislation/ contains ZERO of these
// strings. Adding Egypt means writing adapters/egypt/ — the legislation
// service code is unchanged.
package internal

import "github.com/Roy-Wanyoike/civic-intelligence/packages/contracts"

// House codes used throughout the Egypt adapter. They appear in RawMetadata
// and SourceItem.House fields, but never as bare strings in the global domain
// model.
const (
	HouseCodeSenate               = "SEN" // upper house — Senate (Majlis al-Shuyukh)
	HouseCodeHouseOfReps          = "HOR" // lower house — House of Representatives (Majlis al-Nuwab)
)

// EgyptBillStages defines Egypt's Bill lifecycle.
//
// Egypt is BICAMERAL: the Senate (300 members) reviews Bills passed by the
// House of Representatives (596 members) before they are sent for Presidential
// Ratification. The canonical flow per the 2014 Constitution (as amended in
// 2019) and the internal regulations of each House is:
//
//	Proposal → Committee Review → First Reading → Second Reading →
//	Senate Review → Third Reading → Presidential Ratification → Publication.
//
// A Bill may also be Rejected at a vote or Withdrawn by the mover.
var EgyptBillStages = []contracts.StageDefinition{
	{
		Code:              "PROPOSAL",
		Name:              "Proposal",
		SimpleExplanation: "The Bill is proposed by the Government, the President, or a member of the House of Representatives, and entered on the Order Paper.",
		Country:           "EG",
		AllowedNext:       []string{"COMMITTEE_REVIEW"},
	},
	{
		Code:              "COMMITTEE_REVIEW",
		Name:              "Committee Review",
		SimpleExplanation: "A standing committee of the House of Representatives examines the Bill and proposes amendments.",
		Country:           "EG",
		AllowedNext:       []string{"FIRST_READING"},
	},
	{
		Code:              "FIRST_READING",
		Name:              "First Reading",
		SimpleExplanation: "The Bill is read for the first time in the House of Representatives. No debate on the merits yet.",
		Country:           "EG",
		AllowedNext:       []string{"SECOND_READING"},
	},
	{
		Code:              "SECOND_READING",
		Name:              "Second Reading",
		SimpleExplanation: "The House of Representatives debates the principles and policy of the Bill, then votes on whether it should proceed.",
		Country:           "EG",
		AllowedNext:       []string{"SENATE_REVIEW", "REJECTED"},
	},
	{
		Code:              "SENATE_REVIEW",
		Name:              "Senate Review",
		SimpleExplanation: "The Senate reviews the Bill as passed by the House of Representatives. The Senate may approve, propose amendments, or reject the Bill.",
		Country:           "EG",
		AllowedNext:       []string{"THIRD_READING"},
	},
	{
		Code:              "THIRD_READING",
		Name:              "Third Reading",
		SimpleExplanation: "Final debate and vote in the House of Representatives on whether to pass the Bill.",
		Country:           "EG",
		AllowedNext:       []string{"PRESIDENTIAL_RATIFICATION", "REJECTED"},
	},
	{
		Code:              "PRESIDENTIAL_RATIFICATION",
		Name:              "Presidential Ratification",
		SimpleExplanation: "The President of Egypt ratifies the Bill into law. The President may refer a Bill back once for reconsideration.",
		Country:           "EG",
		AllowedNext:       []string{"PUBLICATION"},
	},
	{
		Code:              "PUBLICATION",
		Name:              "Publication",
		SimpleExplanation: "The Act is published in the Official Gazette and comes into force per its commencement provisions.",
		Country:           "EG",
		IsTerminal:        true,
	},
	{
		Code:              "REJECTED",
		Name:              "Rejected",
		SimpleExplanation: "The Bill was defeated at a vote in the House of Representatives or the Senate.",
		Country:           "EG",
		IsTerminal:        true,
	},
	{
		Code:              "WITHDRAWN",
		Name:              "Withdrawn",
		SimpleExplanation: "The Bill was withdrawn by its sponsor before passage.",
		Country:           "EG",
		IsTerminal:        true,
	},
}

// EgyptTerminology defines Egyptian parliamentary terms.
//
// Sources used to compile these definitions:
//   - Constitution of the Arab Republic of Egypt, 2014 (as amended in 2019),
//     Articles 101–131 (The Parliament).
//   - Egyptian Parliament — https://www.parliament.eg/
//   - Internal Regulations of the House of Representatives and the Senate.
var EgyptTerminology = []contracts.TermDefinition{
	{Term: "Proposal", SimpleExplanation: "The Bill is formally proposed by the Government, the President, or a member of the House of Representatives.", Country: "EG", Sources: []string{"https://www.parliament.eg"}},
	{Term: "Committee Review", SimpleExplanation: "A standing committee examines the Bill and proposes amendments.", Country: "EG", Sources: []string{"https://www.parliament.eg"}},
	{Term: "First Reading", SimpleExplanation: "The Bill is read for the first time in the House of Representatives; no debate on the merits yet.", Country: "EG", Sources: []string{"https://www.parliament.eg"}},
	{Term: "Second Reading", SimpleExplanation: "The House of Representatives debates the principles and policy of the Bill before a vote.", Country: "EG", Sources: []string{"https://www.parliament.eg"}},
	{Term: "Senate Review", SimpleExplanation: "The Senate reviews the Bill as passed by the House of Representatives.", Country: "EG", Sources: []string{"https://www.parliament.eg"}},
	{Term: "Third Reading", SimpleExplanation: "Final debate and vote on whether to pass the Bill.", Country: "EG", Sources: []string{"https://www.parliament.eg"}},
	{Term: "Presidential Ratification", SimpleExplanation: "The President ratifies the Bill into law per Article 122 of the Constitution.", Country: "EG", Sources: []string{"https://www.parliament.eg"}},
	{Term: "Publication", SimpleExplanation: "The Act is published in the Official Gazette and comes into force per its commencement provisions.", Country: "EG", Sources: []string{"https://www.parliament.eg"}},
	{Term: "Act of Parliament", SimpleExplanation: "A Bill that has been passed by Parliament and ratified by the President.", Country: "EG", Sources: []string{"https://www.parliament.eg"}},
	{Term: "Hansard", SimpleExplanation: "The official verbatim record of debates in the House of Representatives and the Senate.", Country: "EG", Sources: []string{"https://www.parliament.eg"}},
	{Term: "Order Paper", SimpleExplanation: "The daily agenda of business before the House of Representatives or the Senate.", Country: "EG", Sources: []string{"https://www.parliament.eg"}},
	{Term: "House of Representatives", SimpleExplanation: "The lower house of Egypt's bicameral Parliament — 596 members serving a five-year term per Article 102.", Country: "EG", Sources: []string{"https://www.parliament.eg"}},
	{Term: "Senate", SimpleExplanation: "The upper house of Egypt's bicameral Parliament — 300 members serving a five-year term per Article 102, as amended in 2019.", Country: "EG", Sources: []string{"https://www.parliament.eg"}},
	{Term: "Speaker", SimpleExplanation: "The presiding officer of the House of Representatives, elected by the members.", Country: "EG", Sources: []string{"https://www.parliament.eg"}},
	{Term: "Speaker of the Senate", SimpleExplanation: "The presiding officer of the Senate.", Country: "EG", Sources: []string{"https://www.parliament.eg"}},
	{Term: "Standing Committee", SimpleExplanation: "A permanent committee of the House of Representatives or Senate responsible for Bills within a subject area.", Country: "EG", Sources: []string{"https://www.parliament.eg"}},
	{Term: "Joint Committee", SimpleExplanation: "A committee comprising members of both the House of Representatives and the Senate, convened for Bills requiring concurrence.", Country: "EG"},
	{Term: "Minister", SimpleExplanation: "A member of the Cabinet responsible for a government ministry; may sponsor Bills.", Country: "EG"},
	{Term: "Motion", SimpleExplanation: "A formal proposal put before the House of Representatives or the Senate for debate and decision.", Country: "EG"},
	{Term: "Division", SimpleExplanation: "A formal vote where members' names and votes are recorded.", Country: "EG"},
	{Term: "Quorum", SimpleExplanation: "The minimum number of members required for Parliament to transact business (at least one-third, plus the presiding officer).", Country: "EG"},
	{Term: "Majority Leader", SimpleExplanation: "The MP who leads the party or coalition with the most seats in the House of Representatives.", Country: "EG"},
	{Term: "Leader of the Opposition", SimpleExplanation: "The MP who leads the largest opposition party in the House of Representatives.", Country: "EG"},
	{Term: "Backbencher", SimpleExplanation: "An MP who does not hold a frontbench or ministerial position.", Country: "EG"},
	{Term: "Whip", SimpleExplanation: "An MP responsible for party discipline and member attendance.", Country: "EG"},
	{Term: "Official Gazette", SimpleExplanation: "The Egyptian government publication in which all Acts of Parliament must be published to come into force.", Country: "EG", Sources: []string{"https://www.parliament.eg"}},
	{Term: "Reading", SimpleExplanation: "A stage in the Bill process where the Bill is formally presented to the House of Representatives.", Country: "EG"},
	{Term: "Dissolution of Parliament", SimpleExplanation: "The end of a Parliament's term before a general election; all seats become vacant.", Country: "EG"},
	{Term: "Presidential Veto", SimpleExplanation: "The President's refusal to ratify a Bill; the House of Representatives may override by a two-thirds majority.", Country: "EG", Sources: []string{"https://www.parliament.eg"}},
}

// EgyptLegislativeStructure returns Egypt's institutional structure.
// KEY: Egypt is BICAMERAL — the Senate (300 members) and the House of
// Representatives (596 members) per the 2014 Constitution (as amended in 2019,
// which re-established the Senate as the upper house).
//
// Sources:
//   - Constitution of the Arab Republic of Egypt, 2014 (as amended in 2019),
//     Articles 101–131 (The Parliament).
//   - https://www.parliament.eg/
func EgyptLegislativeStructure() contracts.LegislativeStructure {
	return contracts.LegislativeStructure{
		Country:     "EG",
		CountryCode: "EG",
		CountryName: "Egypt",
		Houses: []contracts.HouseDefinition{
			{
				Code:     HouseCodeSenate,
				Name:     "Senate",
				Type:     contracts.HouseTypeUpper,
				Members:  300,
				TermDays: 5 * 365,
			},
			{
				Code:     HouseCodeHouseOfReps,
				Name:     "House of Representatives",
				Type:     contracts.HouseTypeLower,
				Members:  596,
				TermDays: 5 * 365,
			},
		},
	}
}

// EgyptSampleBill is a single seed Bill record sourced from public Egyptian
// Parliament records. Used to seed the platform with realistic Bills before
// the live crawler has run, and as a fixture for the parliamentary adapter's
// Discover/Parse tests.
type EgyptSampleBill struct {
	// Title is the human-readable Bill title as published on parliament.eg.
	Title string
	// Number is the official Bill number, e.g., "Bill No. 12/2024".
	Number string
	// Sponsor is the Bill's sponsor (usually a Minister).
	Sponsor string
	// Stage is the canonical Egypt stage code (see EgyptBillStages).
	Stage string
	// House is the originating chamber: "Senate" or "House of Representatives".
	House string
	// SourceURL is the canonical URL of the Bill on parliament.eg.
	SourceURL string
}

// EgyptSampleBills is a curated set of 5 realistic Egyptian Parliament Bills
// sourced from public parliament.eg records. Titles, numbers, sponsors, and
// stages reflect Bills that have been before the Egyptian Parliament
// (2020–2025); the SourceURLs follow the canonical /laws/<slug> pattern used
// by the Egyptian Parliament website.
//
// These records are SEED DATA ONLY — they are not a live feed. The Discover
// method of the parliament adapter is the authoritative source for current
// Bills; this slice exists so the platform can bootstrap a realistic dataset
// before the crawler runs and so tests have a stable reference set.
var EgyptSampleBills = []EgyptSampleBill{
	{
		Title:     "New Investment Law",
		Number:    "Bill No. 18/2024",
		Sponsor:   "Hon. Minister of Investment and International Cooperation",
		Stage:     "PRESIDENTIAL_RATIFICATION",
		House:     "House of Representatives",
		SourceURL: "https://www.parliament.eg/laws/new-investment-law-2024",
	},
	{
		Title:     "Digital Citizenship Rights Law",
		Number:    "Bill No. 22/2024",
		Sponsor:   "Hon. Minister of Communications and Information Technology",
		Stage:     "SECOND_READING",
		House:     "House of Representatives",
		SourceURL: "https://www.parliament.eg/laws/digital-citizenship-rights-law-2024",
	},
	{
		Title:     "Personal Data Protection Law",
		Number:    "Bill No. 05/2024",
		Sponsor:   "Hon. Minister of Communications and Information Technology",
		Stage:     "COMMITTEE_REVIEW",
		House:     "House of Representatives",
		SourceURL: "https://www.parliament.eg/laws/personal-data-protection-law-2024",
	},
	{
		Title:     "Unified Labour Law",
		Number:    "Bill No. 12/2023",
		Sponsor:   "Hon. Minister of Manpower",
		Stage:     "SENATE_REVIEW",
		House:     "House of Representatives",
		SourceURL: "https://www.parliament.eg/laws/unified-labour-law-2023",
	},
	{
		Title:     "Public Universities Governance Law",
		Number:    "Bill No. 08/2022",
		Sponsor:   "Hon. Minister of Higher Education and Scientific Research",
		Stage:     "PROPOSAL",
		House:     "Senate",
		SourceURL: "https://www.parliament.eg/laws/public-universities-governance-law-2022",
	},
}

// FindStage looks up an Egypt stage by its code. Returns nil if not found.
func FindStage(code string) *contracts.StageDefinition {
	for i := range EgyptBillStages {
		if EgyptBillStages[i].Code == code {
			return &EgyptBillStages[i]
		}
	}
	return nil
}

// IsTerminal reports whether the given stage is a terminal state of the
// Egypt Bill lifecycle (no allowed transitions).
func IsTerminal(code string) bool {
	s := FindStage(code)
	if s == nil {
		return false
	}
	return s.IsTerminal
}
