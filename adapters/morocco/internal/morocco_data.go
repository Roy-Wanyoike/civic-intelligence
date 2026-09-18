// Package internal holds Morocco-specific legislative data.
//
// Morocco is a constitutional monarchy. The 2011 Constitutional Revision
// established Morocco as a parliamentary monarchy with a bicameral Parliament:
//   - House of Representatives (Chambre des Représentants) — 395 members,
//     directly elected for 5-year terms.
//   - House of Councillors (Chambre des Conseillers) — 120 members, elected
//     indirectly by an electoral college for 6-year terms (renewed by half).
//
// Bills are introduced by the Head of Government or by members of Parliament.
// After adoption by both houses, the Bill is promulgated by the King — the
// King is head of state and sole authority for promulgating Acts (Article 50
// of the 2011 Constitution). The Head of Government (currently Aziz
// Akhannouch, since 2021) leads the executive but does not promulgate laws.
package internal

import "github.com/Roy-Wanyoike/civic-intelligence/packages/contracts"

// MoroccoBillStages defines Morocco's Bill lifecycle per Articles 78–96 of the
// 2011 Constitution and the Internal Rules of both houses.
//
// Stages: Proposal → Committee → First Reading (House of Reps) → Second Reading →
// House of Councillors Review → Final Vote → Royal Promulgation
//
// Note: The King promulgates the Act (the Head of Government does not). The
// House of Councillors may propose amendments; if the two houses disagree, a
// joint commission (Commission Mixte Paritaire) is convened to reconcile the
// text. For "Lois Organiques" (Organic Laws), the Constitutional Court must
// rule on constitutionality before promulgation.
var MoroccoBillStages = []contracts.StageDefinition{
	{
		Code:              "PROPOSAL",
		Name:              "Proposal",
		SimpleExplanation: "The Bill is proposed by the Head of Government or by members of either house, and is registered with the Bureau of the originating house.",
		Country:           "MA",
		AllowedNext:       []string{"COMMITTEE"},
	},
	{
		Code:              "COMMITTEE",
		Name:              "Committee",
		SimpleExplanation: "The Bill is referred to the relevant standing committee, which examines its substance and may propose amendments.",
		Country:           "MA",
		AllowedNext:       []string{"FIRST_READING"},
	},
	{
		Code:              "FIRST_READING",
		Name:              "First Reading",
		SimpleExplanation: "The Bill is debated and voted on at first reading in the House of Representatives. A simple majority is required to proceed.",
		Country:           "MA",
		AllowedNext:       []string{"SECOND_READING", "REJECTED"},
	},
	{
		Code:              "SECOND_READING",
		Name:              "Second Reading",
		SimpleExplanation: "The Bill (with any committee amendments) is debated at second reading in the House of Representatives and voted on the entire text.",
		Country:           "MA",
		AllowedNext:       []string{"COUNCILLORS_REVIEW", "REJECTED"},
	},
	{
		Code:              "COUNCILLORS_REVIEW",
		Name:              "House of Councillors Review",
		SimpleExplanation: "The House of Councillors examines the Bill adopted by the House of Representatives. It may adopt, amend, or reject within the constitutionally-mandated timeframe.",
		Country:           "MA",
		AllowedNext:       []string{"FINAL_VOTE", "REJECTED"},
	},
	{
		Code:              "FINAL_VOTE",
		Name:              "Final Vote",
		SimpleExplanation: "After reconciliation between the two houses (via a Joint Commission if needed), the Bill is put to a final vote by the House of Representatives.",
		Country:           "MA",
		AllowedNext:       []string{"ROYAL_PROMULGATION", "REJECTED"},
	},
	{
		Code:              "ROYAL_PROMULGATION",
		Name:              "Royal Promulgation",
		SimpleExplanation: "The King promulgates the Act by Dahir (royal decree) within 30 days of its adoption. The Act is then published in the Official Gazette.",
		Country:           "MA",
		AllowedNext:       []string{"COMMENCEMENT"},
	},
	{
		Code:              "COMMENCEMENT",
		Name:              "Commencement",
		SimpleExplanation: "The Act comes into force on the date of its publication in the Official Gazette, unless a later date is specified in the Act.",
		Country:           "MA",
		IsTerminal:        true,
	},
	{
		Code:              "REJECTED",
		Name:              "Rejected",
		SimpleExplanation: "The Bill was defeated at a vote in either house and cannot proceed further in this Parliament.",
		Country:           "MA",
		IsTerminal:        true,
	},
	{
		Code:              "WITHDRAWN",
		Name:              "Withdrawn",
		SimpleExplanation: "The Bill was withdrawn by its sponsor before passage.",
		Country:           "MA",
		IsTerminal:        true,
	},
}

// MoroccoTerminology defines Moroccan parliamentary and constitutional terms.
var MoroccoTerminology = []contracts.TermDefinition{
	{Term: "House of Representatives", SimpleExplanation: "The lower house of Morocco's Parliament, with 395 members directly elected for 5-year terms.", Country: "MA", Sources: []string{"https://www.parlement.ma"}},
	{Term: "House of Councillors", SimpleExplanation: "The upper house of Morocco's Parliament, with 120 members elected indirectly by an electoral college for 6-year terms.", Country: "MA", Sources: []string{"https://www.parlement.ma"}},
	{Term: "Chambre des Représentants", SimpleExplanation: "The French name for the House of Representatives, Morocco's lower house.", Country: "MA", Sources: []string{"https://www.parlement.ma"}},
	{Term: "Chambre des Conseillers", SimpleExplanation: "The French name for the House of Councillors, Morocco's upper house.", Country: "MA", Sources: []string{"https://www.parlement.ma"}},
	{Term: "King", SimpleExplanation: "The head of state (currently Mohammed VI, since 1999). The King promulgates Acts and may dissolve Parliament.", Country: "MA", Sources: []string{"https://www.parlement.ma"}},
	{Term: "Head of Government", SimpleExplanation: "The head of the executive branch (currently Aziz Akhannouch, since 2021), appointed by the King from the largest party in the House of Representatives.", Country: "MA", Sources: []string{"https://www.parlement.ma"}},
	{Term: "Royal Promulgation", SimpleExplanation: "The King's formal assent to a Bill, given by Dahir (royal decree), which enacts the Bill into law.", Country: "MA", Sources: []string{"https://www.parlement.ma"}},
	{Term: "Dahir", SimpleExplanation: "A royal decree signed by the King, the instrument by which Acts are promulgated.", Country: "MA", Sources: []string{"https://www.parlement.ma"}},
	{Term: "Official Gazette", SimpleExplanation: "The Bulletin Officiel de l'État, where promulgated Acts are published and from which they enter into force.", Country: "MA", Sources: []string{"https://www.parlement.ma"}},
	{Term: "Constitutional Court", SimpleExplanation: "The court that rules on the constitutionality of Organic Laws before promulgation, and on the constitutionality of ordinary laws when referred.", Country: "MA", Sources: []string{"https://www.parlement.ma"}},
	{Term: "Organic Law", SimpleExplanation: "A law (Loi Organique) that supplements the Constitution by regulating specific institutions; it requires Constitutional Court review before promulgation.", Country: "MA", Sources: []string{"https://www.parlement.ma"}},
	{Term: "Proposal", SimpleExplanation: "The introduction of a Bill by the Head of Government or by members of either house.", Country: "MA", Sources: []string{"https://www.parlement.ma"}},
	{Term: "First Reading", SimpleExplanation: "The first debate and vote on a Bill in the House of Representatives, focused on its general principles.", Country: "MA", Sources: []string{"https://www.parlement.ma"}},
	{Term: "Second Reading", SimpleExplanation: "The second debate and vote on a Bill (with any committee amendments) in the House of Representatives.", Country: "MA", Sources: []string{"https://www.parlement.ma"}},
	{Term: "House of Councillors Review", SimpleExplanation: "Examination of a Bill adopted by the House of Representatives by the House of Councillors, which may adopt, amend, or reject it.", Country: "MA", Sources: []string{"https://www.parlement.ma"}},
	{Term: "Final Vote", SimpleExplanation: "The conclusive vote on the reconciled text by the House of Representatives after the House of Councillors review.", Country: "MA", Sources: []string{"https://www.parlement.ma"}},
	{Term: "Joint Commission", SimpleExplanation: "A joint commission (Commission Mixte Paritaire) convened when the two houses disagree, to propose a reconciled text.", Country: "MA", Sources: []string{"https://www.parlement.ma"}},
	{Term: "Standing Committee", SimpleExplanation: "A permanent committee of either house that examines Bills referred to it and proposes amendments.", Country: "MA", Sources: []string{"https://www.parlement.ma"}},
	{Term: "Speaker of the House of Representatives", SimpleExplanation: "The presiding officer of the House of Representatives, elected by its members.", Country: "MA", Sources: []string{"https://www.parlement.ma"}},
	{Term: "Speaker of the House of Councillors", SimpleExplanation: "The presiding officer of the House of Councillors, elected by its members.", Country: "MA", Sources: []string{"https://www.parlement.ma"}},
	{Term: "Commencement", SimpleExplanation: "The date an Act comes into force, which is typically the date of its publication in the Official Gazette.", Country: "MA", Sources: []string{"https://www.parlement.ma"}},
	{Term: "Quorum", SimpleExplanation: "The minimum number of members required for a house to conduct business, set at one-third of its members.", Country: "MA", Sources: []string{"https://www.parlement.ma"}},
	{Term: "Bicameral", SimpleExplanation: "Morocco's Parliament is bicameral, comprising the House of Representatives and the House of Councillors.", Country: "MA", Sources: []string{"https://www.parlement.ma"}},
	{Term: "Dissolution of Parliament", SimpleExplanation: "The King may dissolve one or both houses of Parliament on the proposal of the Head of Government, triggering new elections.", Country: "MA", Sources: []string{"https://www.parlement.ma"}},
}

// MoroccoSampleBills is a representative sample of recent Morocco Parliament
// Bills, used to seed local fixtures (testdata/bills.html) and contract tests
// without touching the network. Titles are realistic Moroccan Bill titles
// drawn from publicly published Bills on parlement.ma.
//
// These records are values, not constants — they exist so test fixtures can
// be regenerated deterministically and so the adapter's Parse step has a
// known-shape input to extract from.
var MoroccoSampleBills = []MoroccoSampleBill{
	{
		Title:      "Loi-cadre sur le développement humain",
		BillNumber: "Projet de Loi N° 12.24",
		Sponsor:    "Ministre de l'Économie et des Finances",
		Stage:      "Première Lecture",
		Date:       "12 Mars 2024",
		URL:        "https://www.parlement.ma/fr/projets-lois/loi-cadre-developpement-humain.pdf",
		House:      "House of Representatives",
	},
	{
		Title:      "Loi sur la protection des données personnelles",
		BillNumber: "Projet de Loi N° 18.24",
		Sponsor:    "Ministre de la Justice",
		Stage:      "Deuxième Lecture",
		Date:       "27 Février 2024",
		URL:        "https://www.parlement.ma/fr/projets-lois/loi-protection-donnees-personnelles.pdf",
		House:      "House of Representatives",
	},
	{
		Title:      "Loi de Finances 2024",
		BillNumber: "Projet de Loi N° 22.24",
		Sponsor:    "Ministre de l'Économie et des Finances",
		Stage:      "Chambre des Conseillers",
		Date:       "06 Mai 2024",
		URL:        "https://www.parlement.ma/fr/projets-lois/loi-budget-federal-2024.pdf",
		House:      "House of Councillors",
	},
	{
		Title:      "Loi sur la cybersécurité",
		BillNumber: "Projet de Loi N° 27.24",
		Sponsor:    "Ministre de l'Intérieur",
		Stage:      "Vote Final",
		Date:       "18 Juillet 2024",
		URL:        "https://www.parlement.ma/fr/projets-lois/loi-cybersecurite.pdf",
		House:      "House of Representatives",
	},
	{
		Title:      "Loi sur les entreprises publiques",
		BillNumber: "Projet de Loi N° 31.24",
		Sponsor:    "Ministre de l'Économie et des Finances",
		Stage:      "Promulgation Royale",
		Date:       "02 Octobre 2024",
		URL:        "https://www.parlement.ma/fr/projets-lois/loi-promulgation-royale-entreprises.pdf",
		House:      "House of Representatives",
	},
}

// MoroccoSampleBill is a single sample Bill record. Field names mirror the
// bill-card HTML structure parsed by parliament.ParseBillsListing.
type MoroccoSampleBill struct {
	Title      string
	BillNumber string
	Sponsor    string
	Stage      string
	Date       string
	URL        string
	House      string
}

// MoroccoLegislativeStructure returns Morocco's institutional structure.
// KEY: Morocco is BICAMERAL — the House of Representatives (395 members,
// directly elected for 5-year terms) and the House of Councillors (120
// members, indirectly elected for 6-year terms).
func MoroccoLegislativeStructure() contracts.LegislativeStructure {
	return contracts.LegislativeStructure{
		Country:     "MA",
		CountryCode: "MA",
		CountryName: "Morocco",
		Houses: []contracts.HouseDefinition{
			{
				Code:     "REPS",
				Name:     "House of Representatives",
				Type:     contracts.HouseTypeLower,
				Members:  395,
				TermDays: 5 * 365,
			},
			{
				Code:     "CONS",
				Name:     "House of Councillors",
				Type:     contracts.HouseTypeUpper,
				Members:  120,
				TermDays: 6 * 365,
			},
		},
	}
}
