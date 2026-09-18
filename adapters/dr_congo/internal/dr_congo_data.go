// Package internal holds DR Congo-specific legislative data.
//
// DR Congo is a semi-presidential republic. Under the 2006 Constitution (as
// revised in 2011), DR Congo's Parliament is bicameral:
//   - National Assembly (Assemblée Nationale) — 500 members, directly elected
//     for 5-year terms by universal suffrage.
//   - Senate (Sénat) — 109 members, indirectly elected by the 26 provincial
//     legislatures for 5-year terms.
//
// Bills may be introduced by the President of the Republic, by members of
// either house, or by the provincial legislatures (with conditions). After
// adoption by both houses, the Bill is promulgated by the President. The
// President may refer a Bill back to Parliament for reconsideration, or to
// the Constitutional Court for a ruling on constitutionality.
package internal

import "github.com/Roy-Wanyoike/civic-intelligence/packages/contracts"

// DRCongoBillStages defines DR Congo's Bill lifecycle per Articles 100–137 of
// the 2006 Constitution and the Internal Rules of both houses.
//
// Stages: Proposition → Commission → Première Lecture (Assemblée) →
// Lecture au Sénat → Commission Mixte → Deuxième Lecture → Adoption →
// Promulgation
//
// Note: The President promulgates the Act. When the two houses disagree on a
// Bill, a Joint Commission (Commission Mixte Paritaire) is convened to
// propose a reconciled text; the National Assembly has the final say if the
// Joint Commission cannot reach agreement.
var DRCongoBillStages = []contracts.StageDefinition{
	{
		Code:              "PROPOSITION",
		Name:              "Proposition",
		SimpleExplanation: "The Bill is introduced by the President, by a member of either house, or by a provincial legislature, and is registered with the Bureau of the originating house.",
		Country:           "CD",
		AllowedNext:       []string{"COMMISSION"},
	},
	{
		Code:              "COMMISSION",
		Name:              "Commission",
		SimpleExplanation: "The Bill is referred to the relevant standing committee, which examines its substance and may propose amendments.",
		Country:           "CD",
		AllowedNext:       []string{"FIRST_READING"},
	},
	{
		Code:              "FIRST_READING",
		Name:              "Première Lecture",
		SimpleExplanation: "The Bill is debated and voted on at first reading in the National Assembly. A simple majority is required to proceed.",
		Country:           "CD",
		AllowedNext:       []string{"SENATE_READING", "REJECTED"},
	},
	{
		Code:              "SENATE_READING",
		Name:              "Lecture au Sénat",
		SimpleExplanation: "The Senate examines the Bill adopted by the National Assembly. It may adopt, amend, or reject within the constitutionally-mandated timeframe.",
		Country:           "CD",
		AllowedNext:       []string{"SECOND_READING", "JOINT_COMMISSION", "REJECTED"},
	},
	{
		Code:              "JOINT_COMMISSION",
		Name:              "Commission Mixte",
		SimpleExplanation: "When the two houses disagree, a Joint Commission (Commission Mixte Paritaire) is convened to propose a reconciled text.",
		Country:           "CD",
		AllowedNext:       []string{"SECOND_READING", "REJECTED"},
	},
	{
		Code:              "SECOND_READING",
		Name:              "Deuxième Lecture",
		SimpleExplanation: "After reconciliation (or Senate concurrence), the Bill is put to a second reading and final vote by the National Assembly.",
		Country:           "CD",
		AllowedNext:       []string{"ADOPTION", "REJECTED"},
	},
	{
		Code:              "ADOPTION",
		Name:              "Adoption",
		SimpleExplanation: "The Bill is adopted by both houses and transmitted to the President of the Republic for promulgation.",
		Country:           "CD",
		AllowedNext:       []string{"PROMULGATION"},
	},
	{
		Code:              "PROMULGATION",
		Name:              "Promulgation",
		SimpleExplanation: "The President promulgates the Act within 30 days of its adoption. The Act is then published in the Official Gazette (Journal Officiel).",
		Country:           "CD",
		AllowedNext:       []string{"COMMENCEMENT"},
	},
	{
		Code:              "COMMENCEMENT",
		Name:              "Commencement",
		SimpleExplanation: "The Act comes into force on the date of its publication in the Official Gazette, unless a later date is specified in the Act.",
		Country:           "CD",
		IsTerminal:        true,
	},
	{
		Code:              "REJECTED",
		Name:              "Rejet",
		SimpleExplanation: "The Bill was defeated at a vote in either house and cannot proceed further in this Parliament.",
		Country:           "CD",
		IsTerminal:        true,
	},
	{
		Code:              "WITHDRAWN",
		Name:              "Withdrawn",
		SimpleExplanation: "The Bill was withdrawn by its sponsor before passage.",
		Country:           "CD",
		IsTerminal:        true,
	},
}

// DRCongoTerminology defines DR Congo's parliamentary and constitutional terms.
var DRCongoTerminology = []contracts.TermDefinition{
	{Term: "National Assembly", SimpleExplanation: "The lower house of DR Congo's Parliament, with 500 members directly elected for 5-year terms.", Country: "CD", Sources: []string{"https://www.assemblee-nationale.cd"}},
	{Term: "Assemblée Nationale", SimpleExplanation: "The French name for the National Assembly, DR Congo's lower house.", Country: "CD", Sources: []string{"https://www.assemblee-nationale.cd"}},
	{Term: "Senate", SimpleExplanation: "The upper house of DR Congo's Parliament, with 109 members elected indirectly by provincial legislatures for 5-year terms.", Country: "CD", Sources: []string{"https://www.senat.cd"}},
	{Term: "Sénat", SimpleExplanation: "The French name for the Senate, DR Congo's upper house.", Country: "CD", Sources: []string{"https://www.senat.cd"}},
	{Term: "President of the Republic", SimpleExplanation: "The head of state (currently Félix Tshisekedi, since 2019), directly elected for 5-year terms. Promulgates Acts and may dissolve the National Assembly.", Country: "CD", Sources: []string{"https://www.assemblee-nationale.cd"}},
	{Term: "Prime Minister", SimpleExplanation: "The head of government, appointed by the President from the majority coalition in the National Assembly.", Country: "CD", Sources: []string{"https://www.assemblee-nationale.cd"}},
	{Term: "Promulgation", SimpleExplanation: "The President's formal assent to a Bill, enacting it into law. Must occur within 30 days of adoption.", Country: "CD", Sources: []string{"https://www.assemblee-nationale.cd"}},
	{Term: "Official Gazette", SimpleExplanation: "The Journal Officiel, where promulgated Acts are published and from which they enter into force.", Country: "CD", Sources: []string{"https://www.assemblee-nationale.cd"}},
	{Term: "Constitutional Court", SimpleExplanation: "The court that rules on the constitutionality of Acts and international treaties, and on the constitutionality of Bills when referred.", Country: "CD", Sources: []string{"https://www.assemblee-nationale.cd"}},
	{Term: "Proposition", SimpleExplanation: "The introduction of a Bill by the President, a member of either house, or a provincial legislature.", Country: "CD", Sources: []string{"https://www.assemblee-nationale.cd"}},
	{Term: "Première Lecture", SimpleExplanation: "The first debate and vote on a Bill in the National Assembly, focused on its general principles.", Country: "CD", Sources: []string{"https://www.assemblee-nationale.cd"}},
	{Term: "Lecture au Sénat", SimpleExplanation: "Examination of a Bill adopted by the National Assembly by the Senate, which may adopt, amend, or reject it.", Country: "CD", Sources: []string{"https://www.senat.cd"}},
	{Term: "Commission Mixte", SimpleExplanation: "A Joint Commission (Commission Mixte Paritaire) convened when the two houses disagree, to propose a reconciled text.", Country: "CD", Sources: []string{"https://www.assemblee-nationale.cd"}},
	{Term: "Deuxième Lecture", SimpleExplanation: "The second reading and final vote on a Bill (with any reconciled amendments) in the National Assembly.", Country: "CD", Sources: []string{"https://www.assemblee-nationale.cd"}},
	{Term: "Adoption", SimpleExplanation: "The Bill is adopted by both houses and transmitted to the President for promulgation.", Country: "CD", Sources: []string{"https://www.assemblee-nationale.cd"}},
	{Term: "Standing Committee", SimpleExplanation: "A permanent committee of either house that examines Bills referred to it and proposes amendments.", Country: "CD", Sources: []string{"https://www.assemblee-nationale.cd"}},
	{Term: "Speaker of the National Assembly", SimpleExplanation: "The presiding officer of the National Assembly, elected by its members.", Country: "CD", Sources: []string{"https://www.assemblee-nationale.cd"}},
	{Term: "Speaker of the Senate", SimpleExplanation: "The presiding officer of the Senate, elected by its members.", Country: "CD", Sources: []string{"https://www.senat.cd"}},
	{Term: "Commencement", SimpleExplanation: "The date an Act comes into force, which is typically the date of its publication in the Official Gazette.", Country: "CD", Sources: []string{"https://www.assemblee-nationale.cd"}},
	{Term: "Quorum", SimpleExplanation: "The minimum number of members required for a house to conduct business, set at one-half of its members.", Country: "CD", Sources: []string{"https://www.assemblee-nationale.cd"}},
	{Term: "Bicameral", SimpleExplanation: "DR Congo's Parliament is bicameral, comprising the National Assembly and the Senate.", Country: "CD", Sources: []string{"https://www.assemblee-nationale.cd"}},
	{Term: "Dissolution of the National Assembly", SimpleExplanation: "The President may dissolve the National Assembly after consulting the Prime Minister and the Bureaus of both houses, triggering new elections within 90 days.", Country: "CD", Sources: []string{"https://www.assemblee-nationale.cd"}},
	{Term: "Organic Law", SimpleExplanation: "A law (Loi Organique) that supplements the Constitution by regulating specific institutions; it requires Constitutional Court review before promulgation.", Country: "CD", Sources: []string{"https://www.assemblee-nationale.cd"}},
}

// DRCongoSampleBills is a representative sample of recent DR Congo Parliament
// Bills, used to seed local fixtures (testdata/bills.html) and contract tests
// without touching the network. Titles are realistic DR Congo Bill titles
// drawn from publicly published Bills.
//
// These records are values, not constants — they exist so test fixtures can
// be regenerated deterministically and so the adapter's Parse step has a
// known-shape input to extract from.
var DRCongoSampleBills = []DRCongoSampleBill{
	{
		Title:      "Loi portant mesures de protection des données personnelles",
		BillNumber: "Projet de Loi N° 24/012",
		Sponsor:    "Ministre des Télécommunications, Numérique et Nouvelles Technologies",
		Stage:      "Première Lecture",
		Date:       "12 Mars 2024",
		URL:        "https://www.assemblee-nationale.cd/projets-lois/loi-protection-donnees-personnelles.pdf",
		House:      "National Assembly",
	},
	{
		Title:      "Loi sur les hydrocarbures",
		BillNumber: "Projet de Loi N° 24/018",
		Sponsor:    "Ministre des Hydrocarbures",
		Stage:      "Lecture au Sénat",
		Date:       "27 Février 2024",
		URL:        "https://www.assemblee-nationale.cd/projets-lois/loi-hydrocarbures.pdf",
		House:      "National Assembly",
	},
	{
		Title:      "Loi de Finances 2024",
		BillNumber: "Projet de Loi N° 24/022",
		Sponsor:    "Ministre des Finances",
		Stage:      "Commission Mixte",
		Date:       "06 Mai 2024",
		URL:        "https://www.assemblee-nationale.cd/projets-lois/loi-budget-2024.pdf",
		House:      "National Assembly",
	},
	{
		Title:      "Loi sur la cybersécurité",
		BillNumber: "Projet de Loi N° 24/027",
		Sponsor:    "Ministre de l'Intérieur, Sécurité et Affaires Coutumières",
		Stage:      "Adoption",
		Date:       "18 Juillet 2024",
		URL:        "https://www.assemblee-nationale.cd/projets-lois/loi-cybersecurite.pdf",
		House:      "National Assembly",
	},
	{
		Title:      "Loi sur les entreprises publiques",
		BillNumber: "Projet de Loi N° 24/031",
		Sponsor:    "Ministre du Portefeuille des Entreprises Publiques",
		Stage:      "Promulgation",
		Date:       "02 Octobre 2024",
		URL:        "https://www.assemblee-nationale.cd/projets-lois/loi-entreprises-publiques.pdf",
		House:      "Senate",
	},
}

// DRCongoSampleBill is a single sample Bill record. Field names mirror the
// bill-card HTML structure parsed by parliament.ParseBillsListing.
type DRCongoSampleBill struct {
	Title      string
	BillNumber string
	Sponsor    string
	Stage      string
	Date       string
	URL        string
	House      string
}

// DRCongoLegislativeStructure returns DR Congo's institutional structure.
// KEY: DR Congo is BICAMERAL — the National Assembly (500 members, 5-year
// terms) and the Senate (109 members, 5-year terms).
func DRCongoLegislativeStructure() contracts.LegislativeStructure {
	return contracts.LegislativeStructure{
		Country:     "CD",
		CountryCode: "CD",
		CountryName: "DR Congo",
		Houses: []contracts.HouseDefinition{
			{
				Code:     "NA",
				Name:     "National Assembly",
				Type:     contracts.HouseTypeLower,
				Members:  500,
				TermDays: 5 * 365,
			},
			{
				Code:     "SEN",
				Name:     "Senate",
				Type:     contracts.HouseTypeUpper,
				Members:  109,
				TermDays: 5 * 365,
			},
		},
	}
}
