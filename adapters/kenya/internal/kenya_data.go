// Package internal holds Kenya-specific constants — Bill stages, terminology,
// and the institutional structure. These are VALUES supplied to the global
// domain model; they never become code in services/legislation/.
package internal

import "github.com/Roy-Wanyoike/civic-intelligence/packages/contracts"

// KenyaBillStages returns the canonical Bill stages for Kenya per the 2010
// Constitution and Parliament's Standing Orders.
//
// Order: First Reading → Second Reading → Committee Stage → (Committee of
// the Whole House for money-related matters) → Report Stage → Third Reading
// → Presidential Assent → Commencement.
//
// Terminal states: REJECTED, WITHDRAWN, LAPSED.
func KenyaBillStages() []contracts.StageDefinition {
	return []contracts.StageDefinition{
		{
			Code:              "FIRST_READING",
			Name:              "First Reading",
			SimpleExplanation: "The Bill is read for the first time in the House. No debate on the substance yet — the Bill is simply introduced.",
			OfficialDefinition: "Per Standing Order 91 (National Assembly) / 102 (Senate).",
			Country:           "KE",
			AllowedNext:       []string{"SECOND_READING"},
		},
		{
			Code:              "SECOND_READING",
			Name:              "Second Reading",
			SimpleExplanation: "MPs debate the principles and broad policy of the Bill. A vote is taken on whether the Bill should proceed.",
			OfficialDefinition: "Per Standing Order 95.",
			Country:           "KE",
			AllowedNext:       []string{"COMMITTEE_STAGE", "REJECTED"},
		},
		{
			Code:              "COMMITTEE_STAGE",
			Name:              "Committee Stage",
			SimpleExplanation: "A committee of the House examines the Bill clause-by-clause and may propose amendments. For most Bills this is a Departmental Committee.",
			OfficialDefinition: "Per Standing Order 117.",
			Country:           "KE",
			AllowedNext:       []string{"COMMITTEE_OF_WHOLE_HOUSE", "REPORT_STAGE"},
		},
		{
			Code:              "COMMITTEE_OF_WHOLE_HOUSE",
			Name:              "Committee of the Whole House",
			SimpleExplanation: "When the whole House (not a committee) examines the Bill clause-by-clause. Used for Money Bills and constitutional amendments.",
			OfficialDefinition: "Per Standing Order 124.",
			Country:           "KE",
			AllowedNext:       []string{"REPORT_STAGE"},
		},
		{
			Code:              "REPORT_STAGE",
			Name:              "Report Stage",
			SimpleExplanation: "The committee reports back to the House. MPs can propose further amendments to the committee's version.",
			OfficialDefinition: "Per Standing Order 130.",
			Country:           "KE",
			AllowedNext:       []string{"THIRD_READING", "COMMITTEE_STAGE"},
		},
		{
			Code:              "THIRD_READING",
			Name:              "Third Reading",
			SimpleExplanation: "Final debate on the Bill as amended. The House votes on whether to pass the Bill.",
			OfficialDefinition: "Per Standing Order 132.",
			Country:           "KE",
			AllowedNext:       []string{"PRESIDENTIAL_ASSENT", "REJECTED"},
		},
		{
			Code:              "PRESIDENTIAL_ASSENT",
			Name:              "Presidential Assent",
			SimpleExplanation: "The President signs the Bill into law. Within 14 days of receipt. The President may refer it back once.",
			OfficialDefinition: "Per Article 115 of the Constitution of Kenya, 2010.",
			Country:           "KE",
			AllowedNext:       []string{"COMMENCEMENT"},
		},
		{
			Code:              "COMMENCEMENT",
			Name:              "Commencement",
			SimpleExplanation: "The Act comes into force. Either on the date of assent, on a date specified in the Act, or by a separate commencement notice in the Kenya Gazette.",
			OfficialDefinition: "Per Article 116 of the Constitution of Kenya, 2010.",
			Country:           "KE",
			AllowedNext:       nil,
			IsTerminal:        true,
		},
		{
			Code:              "REJECTED",
			Name:              "Rejected",
			SimpleExplanation: "The Bill was defeated at a vote (typically Second or Third Reading).",
			Country:           "KE",
			AllowedNext:       nil,
			IsTerminal:        true,
		},
		{
			Code:              "WITHDRAWN",
			Name:              "Withdrawn",
			SimpleExplanation: "The sponsor withdrew the Bill.",
			Country:           "KE",
			AllowedNext:       nil,
			IsTerminal:        true,
		},
		{
			Code:              "LAPSED",
			Name:              "Lapsed",
			SimpleExplanation: "The Bill lapsed at the end of a Parliament without completing its passage.",
			Country:           "KE",
			AllowedNext:       nil,
			IsTerminal:        true,
		},
	}
}

// KenyaTerminology returns a registry of ≥25 Kenyan parliamentary terms with
// plain-language explanations and (where available) source URLs.
func KenyaTerminology() []contracts.TermDefinition {
	return []contracts.TermDefinition{
		{Term: "First Reading", SimpleExplanation: "The Bill is read out in the House for the first time — its title and sponsor are announced, and a date is set for the Second Reading.", Country: "KE", Sources: []string{"https://parliament.go.ke/standing-orders"}},
		{Term: "Second Reading", SimpleExplanation: "MPs debate the principles and policy of the Bill, not the detailed text. A vote decides whether the Bill proceeds.", Country: "KE", Sources: []string{"https://parliament.go.ke/standing-orders"}},
		{Term: "Third Reading", SimpleExplanation: "The final debate on the Bill as amended. The House votes on whether to pass it.", Country: "KE", Sources: []string{"https://parliament.go.ke/standing-orders"}},
		{Term: "Committee Stage", SimpleExplanation: "A committee examines the Bill clause-by-clause and may propose amendments.", Country: "KE", Sources: []string{"https://parliament.go.ke/standing-orders"}},
		{Term: "Committee of the Whole House", SimpleExplanation: "The entire House sits as a committee to examine a Bill clause-by-clause. Used for Money Bills and constitutional amendments.", Country: "KE", Sources: []string{"https://parliament.go.ke/standing-orders"}},
		{Term: "Report Stage", SimpleExplanation: "After committee consideration, the committee reports back to the House. MPs can propose further amendments.", Country: "KE", Sources: []string{"https://parliament.go.ke/standing-orders"}},
		{Term: "Presidential Assent", SimpleExplanation: "The President signs a Bill passed by Parliament, making it an Act of Parliament. The President has 14 days to assent or refer it back.", OfficialDefinition: "Article 115, Constitution of Kenya, 2010.", Country: "KE", Sources: []string{"https://www.kenyalaw.org/kl/index.php?id=398"}},
		{Term: "Commencement", SimpleExplanation: "The date on which an Act of Parliament comes into force. May be the date of assent, a date specified in the Act, or by separate gazette notice.", Country: "KE", Sources: []string{"https://www.kenyalaw.org/kl/index.php?id=398"}},
		{Term: "Hansard", SimpleExplanation: "The official verbatim record of parliamentary debates. Named after the original publisher.", Country: "KE", Sources: []string{"https://parliament.go.ke/hansard"}},
		{Term: "Order Paper", SimpleExplanation: "The official daily agenda of the House — what will be discussed, in what order.", Country: "KE", Sources: []string{"https://parliament.go.ke/order-papers"}},
		{Term: "Votes and Proceedings", SimpleExplanation: "The official record of decisions taken in the House on a given sitting day — divisions, vote outcomes, and rulings.", Country: "KE", Sources: []string{"https://parliament.go.ke/votes-and-proceedings"}},
		{Term: "Gazette Notice", SimpleExplanation: "An official publication in the Kenya Gazette — the official record of government notices, including legal notices, regulations, and appointments.", Country: "KE", Sources: []string{"https://www.kenyalaw.org/kl/index.php?id=589"}},
		{Term: "Government Bill", SimpleExplanation: "A Bill sponsored by the executive (typically a Cabinet Secretary). Most significant legislation is introduced this way.", Country: "KE"},
		{Term: "Private Member's Bill", SimpleExplanation: "A Bill sponsored by an MP or Senator who is not in the executive. Less common than Government Bills.", Country: "KE"},
		{Term: "Money Bill", SimpleExplanation: "A Bill that concerns taxation, public debt, or public expenditure. Originates only in the National Assembly. Goes through the Committee of the Whole House.", OfficialDefinition: "Article 114, Constitution of Kenya, 2010.", Country: "KE", Sources: []string{"https://www.kenyalaw.org/kl/index.php?id=398"}},
		{Term: "Mediation Committee", SimpleExplanation: "A committee formed when the National Assembly and Senate disagree on a Bill. It produces a mediated version both houses must pass.", OfficialDefinition: "Article 113, Constitution of Kenya, 2010.", Country: "KE", Sources: []string{"https://www.kenyalaw.org/kl/index.php?id=398"}},
		{Term: "County Government", SimpleExplanation: "One of the 47 devolved governments under the 2010 Constitution, each with its own assembly and governor.", OfficialDefinition: "Chapter 11, Constitution of Kenya, 2010.", Country: "KE", Sources: []string{"https://www.kenyalaw.org/kl/index.php?id=398"}},
		{Term: "Constituency Development Fund (CDF)", SimpleExplanation: "A fund (now the National Government Constituencies Development Fund) channeling national resources to constituency-level projects.", Country: "KE"},
		{Term: "Bicameral", SimpleExplanation: "A two-chamber legislature. Kenya's Parliament is bicameral: National Assembly + Senate, per the 2010 Constitution.", Country: "KE"},
		{Term: "Prorogation", SimpleExplanation: "The formal end of a parliamentary session. All pending Bills lapse unless carried over.", Country: "KE"},
		{Term: "Sine Die", SimpleExplanation: "Latin for 'without a day'. When the House adjourns without setting a date to return.", Country: "KE"},
		{Term: "Quorum", SimpleExplanation: "The minimum number of members who must be present for the House to conduct business. Per Standing Orders.", Country: "KE"},
		{Term: "Division", SimpleExplanation: "A formal vote in which members physically divide into 'Aye' and 'Noe' lobbies. Recorded individually.", Country: "KE"},
		{Term: "Mover", SimpleExplanation: "The MP or Senator who introduces a Bill or motion.", Country: "KE"},
		{Term: "Seconder", SimpleExplanation: "The member who formally supports a motion or Bill, seconding the mover.", Country: "KE"},
		{Term: "Clerk of the Senate", SimpleExplanation: "The senior administrative officer of the Senate, responsible for procedural advice and records.", Country: "KE", Sources: []string{"https://senate.go.ke/clerk"}},
		{Term: "Speaker of the National Assembly", SimpleExplanation: "The presiding officer of the National Assembly, elected by members.", Country: "KE", Sources: []string{"https://nationalassembly.go.ke/speaker"}},
		{Term: "Attorney General", SimpleExplanation: "The government's chief legal advisor, an ex-officio member of Parliament.", OfficialDefinition: "Article 156, Constitution of Kenya, 2010.", Country: "KE", Sources: []string{"https://www.kenyalaw.org/kl/index.php?id=398"}},
		{Term: "Solicitor General", SimpleExplanation: "The deputy to the Attorney General, responsible for government litigation.", Country: "KE"},
		{Term: "Public Participation", SimpleExplanation: "Constitutionally required process where the public is invited to submit views on proposed legislation.", OfficialDefinition: "Article 118, Constitution of Kenya, 2010.", Country: "KE", Sources: []string{"https://www.kenyalaw.org/kl/index.php?id=398"}},
	}
}

// KenyaLegislativeStructure returns Kenya's civic institutions, legislature,
// houses, and committees.
func KenyaLegislativeStructure() *contracts.LegislativeStructure {
	return &contracts.LegislativeStructure{
		Country: "KE",
		Institutions: []contracts.InstitutionRecord{
			{Name: "Parliament of Kenya", Type: "legislature", Jurisdiction: "national", OfficialURLs: []string{"https://parliament.go.ke"}},
			{Name: "Office of the Attorney General", Type: "constitutional_body", Jurisdiction: "national"},
			{Name: "Judiciary", Type: "judiciary", Jurisdiction: "national", OfficialURLs: []string{"https://www.judiciary.go.ke"}},
			{Name: "Independent Electoral and Boundaries Commission", Type: "commission", Jurisdiction: "national", OfficialURLs: []string{"https://www.iebc.or.ke"}},
			{Name: "Ethics and Anti-Corruption Commission", Type: "commission", Jurisdiction: "national", OfficialURLs: []string{"https://www.eacc.go.ke"}},
			{Name: "Controller of Budget", Type: "independent_office", Jurisdiction: "national"},
			{Name: "Auditor General", Type: "independent_office", Jurisdiction: "national"},
		},
		Legislatures: []contracts.LegislatureRecord{
			{Name: "Parliament of Kenya", InstitutionName: "Parliament of Kenya"},
		},
		Houses: []contracts.HouseRecord{
			{LegislatureName: "Parliament of Kenya", Name: "National Assembly", SortOrder: 1},
			{LegislatureName: "Parliament of Kenya", Name: "Senate", SortOrder: 2},
		},
		Committees: []contracts.CommitteeRecord{
			{HouseName: "National Assembly", Name: "Departmental Committee on Lands", Type: "departmental"},
			{HouseName: "National Assembly", Name: "Departmental Committee on Finance and National Planning", Type: "departmental"},
			{HouseName: "National Assembly", Name: "Departmental Committee on Education and Research", Type: "departmental"},
			{HouseName: "National Assembly", Name: "Departmental Committee on Health", Type: "departmental"},
			{HouseName: "National Assembly", Name: "Departmental Committee on Transport, Public Works and Housing", Type: "departmental"},
			{HouseName: "National Assembly", Name: "Departmental Committee on Justice and Legal Affairs", Type: "departmental"},
			{HouseName: "National Assembly", Name: "Public Accounts Committee", Type: "sessional"},
			{HouseName: "National Assembly", Name: "Public Investments Committee", Type: "sessional"},
			{HouseName: "Senate", Name: "Standing Committee on Information, Communication and Technology", Type: "standing"},
			{HouseName: "Senate", Name: "Standing Committee on Finance and Budget", Type: "standing"},
			{HouseName: "Senate", Name: "Standing Committee on Justice, Legal Affairs and Human Rights", Type: "standing"},
			{HouseName: "Senate", Name: "Standing Committee on Health", Type: "standing"},
			{HouseName: "Senate", Name: "Standing Committee on Education", Type: "standing"},
			{HouseName: "Senate", Name: "County Public Accounts and Investments Committee", Type: "sessional"},
		},
	}
}
