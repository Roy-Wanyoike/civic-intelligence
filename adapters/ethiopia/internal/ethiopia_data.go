// Package internal holds Ethiopia-specific legislative data.
//
// Ethiopia is a parliamentary federal republic under the 1995 Constitution
// (as revised in 2018). Its bicameral Parliament comprises:
//   - House of Peoples' Representatives (lower house, 547 members, directly
//     elected for 5-year terms).
//   - House of Federation (upper house, 153 members, elected by state
//     councils for 5-year terms; represents the nations, nationalities and
//     peoples of Ethiopia).
//
// The Prime Minister (currently Abiy Ahmed, since 2018) is head of
// government, elected by the House of Peoples' Representatives. The President
// (currently Sahle-Work Zewde, since 2018) is the head of state but the role
// is largely ceremonial — the President promulgates Acts on the advice of the
// Prime Minister.
//
// Bills (called "Proclamations" once enacted) originate in the House of
// Peoples' Representatives. The House of Federation may review Bills that
// affect the federal structure or the rights of nations and nationalities.
package internal

import "github.com/Roy-Wanyoike/civic-intelligence/packages/contracts"

// EthiopiaBillStages defines Ethiopia's Bill lifecycle per Articles 55–67 of
// the 1995 Constitution and the Internal Rules of both houses.
//
// Stages: Proposal → Committee Review → First Reading → Second Reading →
// House of Federation Review → Final Vote → Promulgation
//
// Note: The President promulgates Acts on the advice of the Prime Minister.
// For Bills that affect the federal structure (constitutional matters, fiscal
// allocation to states, etc.), the House of Federation reviews them.
var EthiopiaBillStages = []contracts.StageDefinition{
	{
		Code:              "PROPOSAL",
		Name:              "Proposal",
		SimpleExplanation: "The Bill is proposed by the Council of Ministers or by members of the House of Peoples' Representatives, and is registered with the Bureau of the House.",
		Country:           "ET",
		AllowedNext:       []string{"COMMITTEE_REVIEW"},
	},
	{
		Code:              "COMMITTEE_REVIEW",
		Name:              "Committee Review",
		SimpleExplanation: "The Bill is referred to the relevant standing committee, which examines its substance, holds public consultations, and proposes amendments.",
		Country:           "ET",
		AllowedNext:       []string{"FIRST_READING"},
	},
	{
		Code:              "FIRST_READING",
		Name:              "First Reading",
		SimpleExplanation: "The Bill is debated and voted on at first reading in the House of Peoples' Representatives. A simple majority is required to proceed.",
		Country:           "ET",
		AllowedNext:       []string{"SECOND_READING", "REJECTED"},
	},
	{
		Code:              "SECOND_READING",
		Name:              "Second Reading",
		SimpleExplanation: "The Bill (with any committee amendments) is debated at second reading in the House of Peoples' Representatives and voted on the entire text.",
		Country:           "ET",
		AllowedNext:       []string{"FEDERATION_REVIEW", "FINAL_VOTE", "REJECTED"},
	},
	{
		Code:              "FEDERATION_REVIEW",
		Name:              "House of Federation Review",
		SimpleExplanation: "Bills affecting the federal structure, the rights of nations/nationalities, or inter-state fiscal allocation are reviewed by the House of Federation.",
		Country:           "ET",
		AllowedNext:       []string{"FINAL_VOTE", "REJECTED"},
	},
	{
		Code:              "FINAL_VOTE",
		Name:              "Final Vote",
		SimpleExplanation: "The Bill is put to a final vote by the House of Peoples' Representatives. If passed, it is transmitted to the President for promulgation.",
		Country:           "ET",
		AllowedNext:       []string{"PROMULGATION", "REJECTED"},
	},
	{
		Code:              "PROMULGATION",
		Name:              "Promulgation",
		SimpleExplanation: "The President promulgates the Proclamation within 15 days, on the advice of the Prime Minister. The Act is then published in the Official Gazette (Negarit Gazeta).",
		Country:           "ET",
		AllowedNext:       []string{"COMMENCEMENT"},
	},
	{
		Code:              "COMMENCEMENT",
		Name:              "Commencement",
		SimpleExplanation: "The Proclamation comes into force on the date of its publication in the Negarit Gazeta, unless a later date is specified in the Proclamation.",
		Country:           "ET",
		IsTerminal:        true,
	},
	{
		Code:              "REJECTED",
		Name:              "Rejected",
		SimpleExplanation: "The Bill was defeated at a vote in either house and cannot proceed further in this Parliament.",
		Country:           "ET",
		IsTerminal:        true,
	},
	{
		Code:              "WITHDRAWN",
		Name:              "Withdrawn",
		SimpleExplanation: "The Bill was withdrawn by its sponsor before passage.",
		Country:           "ET",
		IsTerminal:        true,
	},
}

// EthiopiaTerminology defines Ethiopian parliamentary and constitutional terms.
var EthiopiaTerminology = []contracts.TermDefinition{
	{Term: "House of Peoples' Representatives", SimpleExplanation: "The lower house of Ethiopia's Parliament, with 547 members directly elected for 5-year terms.", Country: "ET", Sources: []string{"https://www.parliament.gov.et"}},
	{Term: "House of Federation", SimpleExplanation: "The upper house of Ethiopia's Parliament, with 153 members elected by state councils to represent the nations, nationalities and peoples of Ethiopia.", Country: "ET", Sources: []string{"https://www.parliament.gov.et"}},
	{Term: "Prime Minister", SimpleExplanation: "The head of government (currently Abiy Ahmed, since 2018), elected by the House of Peoples' Representatives from among its members.", Country: "ET", Sources: []string{"https://www.parliament.gov.et"}},
	{Term: "President", SimpleExplanation: "The head of state (currently Sahle-Work Zewde, since 2018), elected by a joint session of both houses for a 6-year term. The role is largely ceremonial.", Country: "ET", Sources: []string{"https://www.parliament.gov.et"}},
	{Term: "Proclamation", SimpleExplanation: "An Act of Parliament enacted by the House of Peoples' Representatives and promulgated by the President. The equivalent of a statute in other jurisdictions.", Country: "ET", Sources: []string{"https://www.parliament.gov.et"}},
	{Term: "Promulgation", SimpleExplanation: "The President's formal assent to a Bill, on the advice of the Prime Minister, enacting it into a Proclamation.", Country: "ET", Sources: []string{"https://www.parliament.gov.et"}},
	{Term: "Negarit Gazeta", SimpleExplanation: "The Official Gazette of Ethiopia, where promulgated Proclamations are published and from which they enter into force.", Country: "ET", Sources: []string{"https://www.parliament.gov.et"}},
	{Term: "Council of Ministers", SimpleExplanation: "The cabinet chaired by the Prime Minister, which may propose Bills to the House of Peoples' Representatives.", Country: "ET", Sources: []string{"https://www.parliament.gov.et"}},
	{Term: "Council of Constitutional Inquiry", SimpleExplanation: "A council that interprets the Constitution and may refer constitutional questions to the House of Federation.", Country: "ET", Sources: []string{"https://www.parliament.gov.et"}},
	{Term: "Constitutional Amendment", SimpleExplanation: "A constitutional amendment under Article 105 of the Constitution, requiring a two-thirds majority in both houses and approval by a two-thirds majority of state councils.", Country: "ET", Sources: []string{"https://www.parliament.gov.et"}},
	{Term: "Proposal", SimpleExplanation: "The introduction of a Bill by the Council of Ministers or by members of the House of Peoples' Representatives.", Country: "ET", Sources: []string{"https://www.parliament.gov.et"}},
	{Term: "Committee Review", SimpleExplanation: "Examination of a Bill by the relevant standing committee, which may propose amendments after public consultation.", Country: "ET", Sources: []string{"https://www.parliament.gov.et"}},
	{Term: "First Reading", SimpleExplanation: "The first debate and vote on a Bill in the House of Peoples' Representatives, focused on its general principles.", Country: "ET", Sources: []string{"https://www.parliament.gov.et"}},
	{Term: "Second Reading", SimpleExplanation: "The second debate and vote on a Bill (with any committee amendments) in the House of Peoples' Representatives.", Country: "ET", Sources: []string{"https://www.parliament.gov.et"}},
	{Term: "House of Federation Review", SimpleExplanation: "Review of a Bill by the House of Federation when it affects the federal structure or the rights of nations/nationalities.", Country: "ET", Sources: []string{"https://www.parliament.gov.et"}},
	{Term: "Final Vote", SimpleExplanation: "The conclusive vote on a Bill by the House of Peoples' Representatives, after which it is transmitted to the President for promulgation.", Country: "ET", Sources: []string{"https://www.parliament.gov.et"}},
	{Term: "Standing Committee", SimpleExplanation: "A permanent committee of either house that examines Bills referred to it and proposes amendments.", Country: "ET", Sources: []string{"https://www.parliament.gov.et"}},
	{Term: "Speaker of the House of Peoples' Representatives", SimpleExplanation: "The presiding officer of the House of Peoples' Representatives, elected by its members.", Country: "ET", Sources: []string{"https://www.parliament.gov.et"}},
	{Term: "Speaker of the House of Federation", SimpleExplanation: "The presiding officer of the House of Federation, elected by its members.", Country: "ET", Sources: []string{"https://www.parliament.gov.et"}},
	{Term: "Commencement", SimpleExplanation: "The date a Proclamation comes into force, typically the date of its publication in the Negarit Gazeta.", Country: "ET", Sources: []string{"https://www.parliament.gov.et"}},
	{Term: "Quorum", SimpleExplanation: "The minimum number of members required for a house to conduct business, set at more than half of its members.", Country: "ET", Sources: []string{"https://www.parliament.gov.et"}},
	{Term: "Bicameral", SimpleExplanation: "Ethiopia's Parliament is bicameral, comprising the House of Peoples' Representatives and the House of Federation.", Country: "ET", Sources: []string{"https://www.parliament.gov.et"}},
	{Term: "Dissolution of Parliament", SimpleExplanation: "The Prime Minister may dissolve the House of Peoples' Representatives before the end of its term, triggering new elections.", Country: "ET", Sources: []string{"https://www.parliament.gov.et"}},
}

// EthiopiaSampleBills is a representative sample of recent Ethiopia
// Parliament Bills, used to seed local fixtures (testdata/bills.html) and
// contract tests without touching the network. Titles are realistic
// Ethiopian Proclamation titles drawn from publicly published Bills.
//
// These records are values, not constants — they exist so test fixtures can
// be regenerated deterministically and so the adapter's Parse step has a
// known-shape input to extract from.
var EthiopiaSampleBills = []EthiopiaSampleBill{
	{
		Title:      "Proclamation on Hate Speech and Disinformation",
		BillNumber: "Proclamation No. 1234/2024",
		Sponsor:    "Minister of Peace",
		Stage:      "First Reading",
		Date:       "12 March 2024",
		URL:        "https://www.parliament.gov.et/bills/hate-speech-proclamation.pdf",
		House:      "House of Peoples' Representatives",
	},
	{
		Title:      "Federal Government Budget Proclamation",
		BillNumber: "Proclamation No. 1235/2024",
		Sponsor:    "Minister of Finance",
		Stage:      "Second Reading",
		Date:       "27 February 2024",
		URL:        "https://www.parliament.gov.et/bills/federal-budget-proclamation.pdf",
		House:      "House of Peoples' Representatives",
	},
	{
		Title:      "Data Protection Proclamation",
		BillNumber: "Proclamation No. 1236/2024",
		Sponsor:    "Minister of Innovation and Technology",
		Stage:      "House of Federation Review",
		Date:       "06 May 2024",
		URL:        "https://www.parliament.gov.et/bills/data-protection-proclamation.pdf",
		House:      "House of Federation",
	},
	{
		Title:      "Anti-Corruption Proclamation (Amendment)",
		BillNumber: "Proclamation No. 1237/2024",
		Sponsor:    "Attorney General",
		Stage:      "Final Vote",
		Date:       "18 July 2024",
		URL:        "https://www.parliament.gov.et/bills/anti-corruption-proclamation.pdf",
		House:      "House of Peoples' Representatives",
	},
	{
		Title:      "Cooperatives Proclamation",
		BillNumber: "Proclamation No. 1238/2024",
		Sponsor:    "Minister of Agriculture",
		Stage:      "Promulgation",
		Date:       "02 October 2024",
		URL:        "https://www.parliament.gov.et/bills/cooperatives-proclamation.pdf",
		House:      "House of Peoples' Representatives",
	},
}

// EthiopiaSampleBill is a single sample Bill record. Field names mirror the
// bill-card HTML structure parsed by parliament.ParseBillsListing.
type EthiopiaSampleBill struct {
	Title      string
	BillNumber string
	Sponsor    string
	Stage      string
	Date       string
	URL        string
	House      string
}

// EthiopiaLegislativeStructure returns Ethiopia's institutional structure.
// KEY: Ethiopia is BICAMERAL — the House of Peoples' Representatives (547
// members, 5-year terms) and the House of Federation (153 members, 5-year
// terms).
func EthiopiaLegislativeStructure() contracts.LegislativeStructure {
	return contracts.LegislativeStructure{
		Country:     "ET",
		CountryCode: "ET",
		CountryName: "Ethiopia",
		Houses: []contracts.HouseDefinition{
			{
				Code:     "HPR",
				Name:     "House of Peoples' Representatives",
				Type:     contracts.HouseTypeLower,
				Members:  547,
				TermDays: 5 * 365,
			},
			{
				Code:     "HOF",
				Name:     "House of Federation",
				Type:     contracts.HouseTypeUpper,
				Members:  153,
				TermDays: 5 * 365,
			},
		},
	}
}
