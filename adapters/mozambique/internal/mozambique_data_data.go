// Package internal holds Mozambique-specific legislative data.
//
// Mozambique is a presidential republic under the 1990 Constitution (as revised in 2004 and 2018) (as revised).
// The Parliament of Mozambique is UNICAMERAL — the National Assembly, with 193
// members directly elected for 5-year terms under a first-past-the-post
// system in single-member constituencies.
//
// The President (currently Filipe Nyusi, since 2015) is both head of
// state and head of government, elected directly by universal suffrage for a
// 5-year term. The President is also a member of the National Assembly.
// Bills passed by Parliament must be assented to by the President within
// 21 days before becoming Acts.
package internal

import "github.com/Roy-Wanyoike/civic-intelligence/packages/contracts"

// MozambiqueBillStages defines Mozambique's Bill lifecycle.
// Mozambique is unicameral — the National Assembly is the sole legislative body.
// Stages: First Reading → Second Reading → Committee Stage → Report Stage →
// Third Reading → Presidential Assent → Commencement
//
// Note: Mozambique follows the Westminster model. The "Committee Stage" is taken
// either by a standing committee or by a Committee of the Whole House, where
// the Bill is examined clause-by-clause. The "Report Stage" allows members
// to consider and vote on the committee's amendments on the floor of the
// House before Third Reading.
var MozambiqueBillStages = []contracts.StageDefinition{
	{
		Code:              "FIRST_READING",
		Name:              "First Reading",
		SimpleExplanation: "The Bill is read for the first time in the National Assembly and a date is set for Second Reading. No debate yet.",
		Country:           "MZ",
		AllowedNext:       []string{"SECOND_READING"},
	},
	{
		Code:              "SECOND_READING",
		Name:              "Second Reading",
		SimpleExplanation: "Members debate the principles and policy of the Bill. A vote decides whether it proceeds to the Committee Stage.",
		Country:           "MZ",
		AllowedNext:       []string{"COMMITTEE_STAGE", "REJECTED"},
	},
	{
		Code:              "COMMITTEE_STAGE",
		Name:              "Committee Stage",
		SimpleExplanation: "The National Assembly, sitting as a Committee of the Whole (or a standing committee), examines the Bill clause-by-clause and considers amendments.",
		Country:           "MZ",
		AllowedNext:       []string{"REPORT_STAGE"},
	},
	{
		Code:              "REPORT_STAGE",
		Name:              "Report Stage",
		SimpleExplanation: "Members consider and vote on the committee's amendments on the floor of the House, and may propose further amendments.",
		Country:           "MZ",
		AllowedNext:       []string{"THIRD_READING"},
	},
	{
		Code:              "THIRD_READING",
		Name:              "Third Reading",
		SimpleExplanation: "Final debate and vote on whether to pass the Bill. No substantive amendments may be made at this stage.",
		Country:           "MZ",
		AllowedNext:       []string{"PRESIDENTIAL_ASSENT", "REJECTED"},
	},
	{
		Code:              "PRESIDENTIAL_ASSENT",
		Name:              "Presidential Assent",
		SimpleExplanation: "The President assents to the Bill within 21 days of its passage, signifying agreement to enact it into law. The President may refer the Bill back once.",
		Country:           "MZ",
		AllowedNext:       []string{"COMMENCEMENT"},
	},
	{
		Code:              "COMMENCEMENT",
		Name:              "Commencement",
		SimpleExplanation: "The Act comes into force, either on the date of assent or on a date specified in the Act or by a commencement notice.",
		Country:           "MZ",
		IsTerminal:        true,
	},
	{
		Code:              "REJECTED",
		Name:              "Rejected",
		SimpleExplanation: "The Bill was defeated at a vote.",
		Country:           "MZ",
		IsTerminal:        true,
	},
	{
		Code:              "WITHDRAWN",
		Name:              "Withdrawn",
		SimpleExplanation: "The Bill was withdrawn by its sponsor before passage.",
		Country:           "MZ",
		IsTerminal:        true,
	},
}

// MozambiqueTerminology defines Mozambiquean parliamentary terms.
var MozambiqueTerminology = []contracts.TermDefinition{
	{Term: "First Reading", SimpleExplanation: "The Bill is introduced and read for the first time in the National Assembly.", Country: "MZ", Sources: []string{"https://www.parlamento.gov.mz"}},
	{Term: "Second Reading", SimpleExplanation: "Members debate the principles and policy of the Bill before a vote.", Country: "MZ", Sources: []string{"https://www.parlamento.gov.mz"}},
	{Term: "Committee Stage", SimpleExplanation: "The National Assembly, in Committee of the Whole or a standing committee, examines the Bill clause-by-clause and considers amendments.", Country: "MZ", Sources: []string{"https://www.parlamento.gov.mz"}},
	{Term: "Report Stage", SimpleExplanation: "Members consider and vote on the committee's amendments on the floor of the House, and may propose further amendments.", Country: "MZ", Sources: []string{"https://www.parlamento.gov.mz"}},
	{Term: "Third Reading", SimpleExplanation: "Final debate and vote on whether to pass the Bill. No substantive amendments may be made at this stage.", Country: "MZ", Sources: []string{"https://www.parlamento.gov.mz"}},
	{Term: "Presidential Assent", SimpleExplanation: "The President assents to the Bill within 21 days of passage, enacting it into law.", Country: "MZ", Sources: []string{"https://www.parlamento.gov.mz"}},
	{Term: "Commencement", SimpleExplanation: "The date an Act comes into force, either on assent or on a date specified in the Act.", Country: "MZ", Sources: []string{"https://www.parlamento.gov.mz"}},
	{Term: "Act of Parliament", SimpleExplanation: "A Bill that has been passed by the National Assembly and assented to by the President.", Country: "MZ", Sources: []string{"https://www.parlamento.gov.mz"}},
	{Term: "Hansard", SimpleExplanation: "The official verbatim record of debates in the National Assembly.", Country: "MZ", Sources: []string{"https://www.parlamento.gov.mz"}},
	{Term: "Order Paper", SimpleExplanation: "The daily agenda of business before the National Assembly.", Country: "MZ", Sources: []string{"https://www.parlamento.gov.mz"}},
	{Term: "Committee of the Whole", SimpleExplanation: "A committee comprising all members, used during the Committee Stage to examine a Bill clause-by-clause.", Country: "MZ", Sources: []string{"https://www.parlamento.gov.mz"}},
	{Term: "Standing Committee", SimpleExplanation: "A permanent committee of the National Assembly that examines Bills and issues within a specific subject area.", Country: "MZ", Sources: []string{"https://www.parlamento.gov.mz"}},
	{Term: "Ad Hoc Committee", SimpleExplanation: "A committee set up for a specific task and dissolved once it reports.", Country: "MZ"},
	{Term: "Speaker", SimpleExplanation: "The presiding officer of the National Assembly, elected by members from outside or from among themselves.", Country: "MZ", Sources: []string{"https://www.parlamento.gov.mz"}},
	{Term: "Deputy Speaker", SimpleExplanation: "Assists the Speaker and presides in the Speaker's absence.", Country: "MZ", Sources: []string{"https://www.parlamento.gov.mz"}},
	{Term: "Clerk of Parliament", SimpleExplanation: "The senior administrative officer and chief advisor on parliamentary procedure.", Country: "MZ", Sources: []string{"https://www.parlamento.gov.mz"}},
	{Term: "Leader of the House", SimpleExplanation: "The member, usually a Cabinet Minister, responsible for scheduling and guiding government business in the National Assembly.", Country: "MZ", Sources: []string{"https://www.parlamento.gov.mz"}},
	{Term: "Leader of the Opposition", SimpleExplanation: "The MP who leads the largest party not in government.", Country: "MZ", Sources: []string{"https://www.parlamento.gov.mz"}},
	{Term: "Government Chief Whip", SimpleExplanation: "A member of the governing side responsible for party discipline and member attendance.", Country: "MZ", Sources: []string{"https://www.parlamento.gov.mz"}},
	{Term: "Opposition Chief Whip", SimpleExplanation: "A member of the opposition side responsible for party discipline and member attendance.", Country: "MZ", Sources: []string{"https://www.parlamento.gov.mz"}},
	{Term: "Backbencher", SimpleExplanation: "An MP who does not hold a frontbench or ministerial position.", Country: "MZ"},
	{Term: "Minister", SimpleExplanation: "A member of the executive, usually an MP, responsible for a government ministry.", Country: "MZ"},
	{Term: "Motion", SimpleExplanation: "A formal proposal put before the National Assembly for debate and decision.", Country: "MZ"},
	{Term: "Division", SimpleExplanation: "A formal vote in which members' names and votes are recorded.", Country: "MZ"},
	{Term: "Quorum", SimpleExplanation: "The minimum number of members (one-third, excluding the presiding officer) required for the National Assembly to transact business.", Country: "MZ", Sources: []string{"https://www.parlamento.gov.mz"}},
	{Term: "Prorogation", SimpleExplanation: "The end of a parliamentary session, after which the National Assembly must be summoned anew.", Country: "MZ"},
	{Term: "Dissolution of Parliament", SimpleExplanation: "The end of a Parliament's term before a general election; all seats become vacant.", Country: "MZ"},
}

// MozambiqueSampleBills is a representative sample of recent Mozambique Parliament
// Bills, used to seed local fixtures (testdata/bills.html) and contract tests
// without touching the network. Titles are realistic Mozambiquean Bill titles
// drawn from publicly published Bills.
//
// These records are values, not constants — they exist so test fixtures can
// be regenerated deterministically and so the adapter's Parse step has a
// known-shape input to extract from.
var MozambiqueSampleBills = []MozambiqueSampleBill{
	{
		Title:      "Electronic Transactions and Cyber Security Bill, 2024",
		BillNumber: "Bill No. 12 of 2024",
		Sponsor:    "Minister of Information and Digitalisation",
		Stage:      "Committee Stage",
		Date:       "12 March 2024",
		URL:        "https://www.parlamento.gov.mz/bills/electronic-transactions-cyber-security-2024.pdf",
	},
	{
		Title:      "Access to Information (Amendment) Bill, 2024",
		BillNumber: "Bill No. 18 of 2024",
		Sponsor:    "Minister of Information",
		Stage:      "Second Reading",
		Date:       "27 February 2024",
		URL:        "https://www.parlamento.gov.mz/bills/access-to-information-amendment-2024.pdf",
	},
	{
		Title:      "Public Universities (Amendment) Bill, 2024",
		BillNumber: "Bill No. 22 of 2024",
		Sponsor:    "Minister of Education",
		Stage:      "First Reading",
		Date:       "06 May 2024",
		URL:        "https://www.parlamento.gov.mz/bills/public-universities-2024.pdf",
	},
	{
		Title:      "Cyber Security (Amendment) Bill, 2024",
		BillNumber: "Bill No. 27 of 2024",
		Sponsor:    "Minister of Information and Digitalisation",
		Stage:      "Third Reading",
		Date:       "18 July 2024",
		URL:        "https://www.parlamento.gov.mz/bills/cyber-security-amendment-2024.pdf",
	},
	{
		Title:      "Companies (Amendment) Bill, 2024",
		BillNumber: "Bill No. 31 of 2024",
		Sponsor:    "Minister of Justice",
		Stage:      "Assent",
		Date:       "02 October 2024",
		URL:        "https://www.parlamento.gov.mz/bills/companies-amendment-2024.pdf",
	},
}

// MozambiqueSampleBill is a single sample Bill record. Field names mirror the
// bill-card HTML structure parsed by parliament.ParseBillsListing.
type MozambiqueSampleBill struct {
	Title      string
	BillNumber string
	Sponsor    string
	Stage      string
	Date       string
	URL        string
}

// MozambiqueLegislativeStructure returns Mozambique's institutional structure.
// KEY: Mozambique is UNICAMERAL — only one House (the National Assembly), with
// 193 elected members serving five-year terms under Article 50 of the 1994
// Constitution.
func MozambiqueLegislativeStructure() contracts.LegislativeStructure {
	return contracts.LegislativeStructure{
		Country:     "MZ",
		CountryCode: "MZ",
		CountryName: "Mozambique",
		Houses: []contracts.HouseDefinition{
			{
				Code:     "PARLIAMENT",
				Name:     "National Assembly of Mozambique",
				Type:     contracts.HouseTypeSingle,
				Members:  193,
				TermDays: 5 * 365,
			},
		},
	}
}
