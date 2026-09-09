package internal

import "github.com/Roy-Wanyoike/civic-intelligence/packages/contracts"

// KenyaTerm is the richer, Kenya-specific glossary entry. It carries
// country, sources (URLs to official Kenyan documents) and an optional
// stage association. The contracts.TermDefinition type only carries a
// canonical key, term and short description — adapter.go projects the
// richer record down to that shape.
type KenyaTerm struct {
	Term                string
	CanonicalKey        string
	SimpleExplanation   string
	OfficialDefinition  string
	Stage               string // optional stage code, "" if general
	Country             string // always "KE"
	Sources             []string
}

// KenyaTerminology is the registry of Kenyan parliamentary and legal terms
// used by the adapter when normalising extracted records.
//
// Sources used to compile these definitions:
//   - Constitution of Kenya, 2010
//     https://www.constituteproject.org/constitution/Kenya_2010
//   - Parliament of Kenya — glossary and standing orders
//     https://www.parliament.go.ke/
//   - Kenya Law (Kenya Law Reports / National Council for Law Reporting)
//     https://www.kenyalaw.org/
//   - Kenya Gazette
//     https://gazettes.africa/kenya
var KenyaTerminology = []KenyaTerm{
	{
		Term: "First Reading",
		CanonicalKey: "first_reading",
		SimpleExplanation: "The formal introduction of a Bill in a house; the Bill is read by title only and published.",
		OfficialDefinition: "Publication and first reading per Standing Order 117 (National Assembly) / 134 (Senate).",
		Stage:    string(StageFirstReading),
		Country:  "KE",
		Sources:  []string{"https://www.parliament.go.ke/"},
	},
	{
		Term: "Second Reading",
		CanonicalKey: "second_reading",
		SimpleExplanation: "Debate on the principles and policy of a Bill, followed by a vote on whether it should proceed.",
		OfficialDefinition: "Per Standing Order 118 (NA) / 135 (Senate). The Mover outlines the policy; the Seconder supports; debate follows.",
		Stage:    string(StageSecondReading),
		Country:  "KE",
		Sources:  []string{"https://www.parliament.go.ke/"},
	},
	{
		Term: "Committee Stage",
		CanonicalKey: "committee_stage",
		SimpleExplanation: "Scrutiny of a Bill by a departmental or select committee, including public participation.",
		OfficialDefinition: "Per Standing Orders 119 and 136; committee reports back to the House.",
		Stage:    string(StageCommitteeStage),
		Country:  "KE",
		Sources:  []string{"https://www.parliament.go.ke/"},
	},
	{
		Term: "Committee of the Whole House",
		CanonicalKey: "committee_of_whole_house",
		SimpleExplanation: "The house sitting as a committee, chaired by the Chair of Committees, to consider a Bill clause-by-clause.",
		OfficialDefinition: "Per Standing Orders 120 and 137.",
		Stage:    string(StageCommitteeOfWholeHouse),
		Country:  "KE",
		Sources:  []string{"https://www.parliament.go.ke/"},
	},
	{
		Term: "Report Stage",
		CanonicalKey: "report_stage",
		SimpleExplanation: "The committee reports its proposed amendments back to the full house for decision.",
		OfficialDefinition: "Per Standing Orders 121 and 138.",
		Stage:    string(StageReportStage),
		Country:  "KE",
		Sources:  []string{"https://www.parliament.go.ke/"},
	},
	{
		Term: "Third Reading",
		CanonicalKey: "third_reading",
		SimpleExplanation: "Final reading and vote on a Bill in a house.",
		OfficialDefinition: "Per Standing Orders 122 and 139.",
		Stage:    string(StageThirdReading),
		Country:  "KE",
		Sources:  []string{"https://www.parliament.go.ke/"},
	},
	{
		Term: "Presidential Assent",
		CanonicalKey: "presidential_assent",
		SimpleExplanation: "The President signs a Bill passed by Parliament into law.",
		OfficialDefinition: "Article 115 of the Constitution of Kenya, 2010.",
		Stage:    string(StagePresidentialAssent),
		Country:  "KE",
		Sources:  []string{
			"https://www.constituteproject.org/constitution/Kenya_2010",
			"https://www.kenyalaw.org/",
		},
	},
	{
		Term: "Commencement",
		CanonicalKey: "commencement",
		SimpleExplanation: "The date an Act comes into force, fixed by the Act or by a Gazette Legal Notice.",
		OfficialDefinition: "Per the Interpretation and General Provisions Act (Cap 2) and the Act's own commencement provision.",
		Stage:    string(StageCommencement),
		Country:  "KE",
		Sources:  []string{"https://www.kenyalaw.org/", "https://gazettes.africa/kenya"},
	},
	{
		Term: "Order Paper",
		CanonicalKey: "order_paper",
		SimpleExplanation: "The daily published agenda of business for a house.",
		OfficialDefinition: "The Order Paper sets out the business of a sitting; it is published by the Clerk of each house before each sitting.",
		Country:  "KE",
		Sources:  []string{
			"https://www.parliament.go.ke/the-national-assembly/order-paper",
			"https://www.parliament.go.ke/the-senate/order-paper",
		},
	},
	{
		Term: "Hansard",
		CanonicalKey: "hansard",
		SimpleExplanation: "The official verbatim report of debates in a house.",
		OfficialDefinition: "The official report of parliamentary debates, prepared by the Hansard department of each house.",
		Country:  "KE",
		Sources:  []string{
			"https://www.parliament.go.ke/the-national-assembly/hansard",
			"https://www.parliament.go.ke/the-senate/hansard",
		},
	},
	{
		Term: "Votes and Proceedings",
		CanonicalKey: "votes_and_proceedings",
		SimpleExplanation: "The official record of decisions and divisions taken at each sitting.",
		OfficialDefinition: "The Votes and Proceedings record every decision of the House; it is the authoritative source for vote outcomes.",
		Country:  "KE",
		Sources:  []string{
			"https://www.parliament.go.ke/the-national-assembly/votes-and-proceedings",
			"https://www.parliament.go.ke/the-senate/votes-and-proceedings",
		},
	},
	{
		Term: "Gazette Notice",
		CanonicalKey: "gazette_notice",
		SimpleExplanation: "An official announcement published in the Kenya Gazette by the Government Printer.",
		OfficialDefinition: "A notice published in the Kenya Gazette (or a supplement) under the Kenya Gazette Act (Cap 8). Gazette notices include Legal Notices, Acts commencement notices, appointments and statutory instruments.",
		Country:  "KE",
		Sources:  []string{
			"https://gazettes.africa/kenya",
			"https://www.kenyalaw.org/kenya_gazette",
		},
	},
	{
		Term: "Government Bill",
		CanonicalKey: "government_bill",
		SimpleExplanation: "A Bill introduced by a Cabinet Secretary or the Attorney General on behalf of the Executive.",
		OfficialDefinition: "A Bill sponsored by the Government (the Executive), as distinguished from a Private Member's Bill.",
		Country:  "KE",
		Sources:  []string{"https://www.parliament.go.ke/"},
	},
	{
		Term: "Private Member's Bill",
		CanonicalKey: "private_members_bill",
		SimpleExplanation: "A Bill introduced by an individual member of Parliament rather than the Government.",
		OfficialDefinition: "A Bill sponsored by a member (or members) in their individual capacity. The member's name appears as Mover.",
		Country:  "KE",
		Sources:  []string{"https://www.parliament.go.ke/"},
	},
	{
		Term: "Money Bill",
		CanonicalKey: "money_bill",
		SimpleExplanation: "A Bill dealing with taxation, public debt or appropriation of public funds, requiring special procedures per Article 114.",
		OfficialDefinition: "A Bill within the meaning of Article 114 of the Constitution. Money Bills may only originate in the National Assembly and require a recommendation from the Cabinet Secretary responsible for finance.",
		Country:  "KE",
		Sources:  []string{
			"https://www.constituteproject.org/constitution/Kenya_2010",
			"https://www.parliament.go.ke/",
		},
	},
	{
		Term: "Bill Concerning Counties",
		CanonicalKey: "bill_concerning_counties",
		SimpleExplanation: "A Bill that affects the functions or finances of county governments; it must pass both houses.",
		OfficialDefinition: "Article 109 and 110 of the Constitution. A Bill concerning county government requires a vote in both houses; if the houses disagree, a Mediation Committee is convened.",
		Country:  "KE",
		Sources:  []string{"https://www.constituteproject.org/constitution/Kenya_2010"},
	},
	{
		Term: "Mediation Committee",
		CanonicalKey: "mediation_committee",
		SimpleExplanation: "A joint committee of both houses formed when they disagree on a Bill, to agree a compromise version.",
		OfficialDefinition: "Article 113 of the Constitution. A Mediation Committee prepares a version of the Bill for both houses to consider.",
		Stage:    string(StageMediation),
		Country:  "KE",
		Sources:  []string{"https://www.constituteproject.org/constitution/Kenya_2010"},
	},
	{
		Term: "County Government",
		CanonicalKey: "county_government",
		SimpleExplanation: "One of the 47 devolved governments established under Chapter 11 of the Constitution.",
		OfficialDefinition: "Chapter 11 of the Constitution of Kenya, 2010 establishes 47 county governments with executive and legislative arms.",
		Country:  "KE",
		Sources:  []string{"https://www.constituteproject.org/constitution/Kenya_2010"},
	},
	{
		Term: "Constituency Development Fund",
		CanonicalKey: "cdf",
		SimpleExplanation: "The National Government Constituency Development Fund, established to allocate development funding per constituency.",
		OfficialDefinition: "Established under the NG-CDF Act, 2015. Allocations to constituencies are managed by a Constituency Development Fund Committee.",
		Country:  "KE",
		Sources:  []string{"https://www.ng-cdf.go.ke/"},
	},
	{
		Term: "Bicameral",
		CanonicalKey: "bicameral",
		SimpleExplanation: "A legislature with two houses; Kenya's Parliament has the National Assembly (lower) and Senate (upper).",
		OfficialDefinition: "Article 93 of the Constitution establishes Parliament as two houses: the National Assembly and the Senate.",
		Country:  "KE",
		Sources:  []string{"https://www.constituteproject.org/constitution/Kenya_2010"},
	},
	{
		Term: "Prorogation",
		CanonicalKey: "prorogation",
		SimpleExplanation: "The formal end of a parliamentary session, by proclamation of the President, without dissolving Parliament.",
		OfficialDefinition: "A session of Parliament is prorogued by the President; pending business generally lapses unless carried over.",
		Country:  "KE",
		Sources:  []string{"https://www.parliament.go.ke/"},
	},
	{
		Term: "Sine Die",
		CanonicalKey: "sine_die",
		SimpleExplanation: "Adjournment without fixing a future date of meeting.",
		OfficialDefinition: "Adjournment sine die (without a day): the house adjourns without appointing a date for the next sitting.",
		Country:  "KE",
		Sources:  []string{"https://www.parliament.go.ke/"},
	},
	{
		Term: "Quorum",
		CanonicalKey: "quorum",
		SimpleExplanation: "The minimum number of members (50 in the National Assembly, 15 in the Senate) required to conduct business.",
		OfficialDefinition: "Standing Order 30 (NA) / 30 (Senate). Quorum is 50 members of the National Assembly and 15 of the Senate.",
		Country:  "KE",
		Sources:  []string{"https://www.parliament.go.ke/"},
	},
	{
		Term: "Division",
		CanonicalKey: "division",
		SimpleExplanation: "A formal recorded vote where members' names are entered in the Votes and Proceedings.",
		OfficialDefinition: "A division is the procedure by which the House votes with the ayes and noes recorded individually; results are entered into the Votes and Proceedings.",
		Country:  "KE",
		Sources:  []string{"https://www.parliament.go.ke/"},
	},
	{
		Term: "Mover",
		CanonicalKey: "mover",
		SimpleExplanation: "The member who introduces a Bill or motion and is responsible for steering it through the house.",
		OfficialDefinition: "The Mover of a Bill is the member designated to introduce and pilot the Bill. For a Government Bill this is usually the Cabinet Secretary or the AG; for a Private Member's Bill, the sponsoring member.",
		Country:  "KE",
		Sources:  []string{"https://www.parliament.go.ke/"},
	},
	{
		Term: "Seconder",
		CanonicalKey: "seconder",
		SimpleExplanation: "The member who formally seconds (supports) a motion or Bill after it has been moved.",
		OfficialDefinition: "The Seconder speaks immediately after the Mover. For Government Bills, the Seconder is usually a member of the same department or the Chief Whip's nominee.",
		Country:  "KE",
		Sources:  []string{"https://www.parliament.go.ke/"},
	},
	{
		Term: "Clerk of the Senate",
		CanonicalKey: "clerk_of_the_senate",
		SimpleExplanation: "The chief administrative officer of the Senate, responsible for publishing the Order Paper, Hansard, and Votes and Proceedings.",
		OfficialDefinition: "The Clerk of the Senate is appointed under Article 128 of the Constitution and is the chief procedural advisor to the Senate.",
		Country:  "KE",
		Sources:  []string{
			"https://www.parliament.go.ke/the-senate",
			"https://www.constituteproject.org/constitution/Kenya_2010",
		},
	},
	{
		Term: "Speaker of the National Assembly",
		CanonicalKey: "speaker_of_the_national_assembly",
		SimpleExplanation: "The presiding officer of the National Assembly, elected under Article 106 of the Constitution.",
		OfficialDefinition: "Article 106 of the Constitution establishes the Speaker of the National Assembly as the presiding officer.",
		Country:  "KE",
		Sources:  []string{
			"https://www.parliament.go.ke/the-national-assembly",
			"https://www.constituteproject.org/constitution/Kenya_2010",
		},
	},
	{
		Term: "Attorney General",
		CanonicalKey: "attorney_general",
		SimpleExplanation: "The principal legal advisor to the Government, an ex-officio member of Parliament per Article 156.",
		OfficialDefinition: "Article 156 of the Constitution. The Attorney General is the principal legal advisor to the Government and an ex-officio member of Parliament.",
		Country:  "KE",
		Sources:  []string{
			"https://www.constituteproject.org/constitution/Kenya_2010",
			"https://www.statelaw.go.ke/",
		},
	},
	{
		Term: "Solicitor General",
		CanonicalKey: "solicitor_general",
		SimpleExplanation: "The deputy to the Attorney General, established under the Office of the Attorney General Act.",
		OfficialDefinition: "Established under the Office of the Attorney General Act, 2012. The Solicitor General deputises the Attorney General.",
		Country:  "KE",
		Sources:  []string{"https://www.statelaw.go.ke/"},
	},
	{
		Term: "Public Participation",
		CanonicalKey: "public_participation",
		SimpleExplanation: "Constitutionally-mandated process where the public submits views on Bills and other parliamentary business.",
		OfficialDefinition: "Article 118(1)(b) of the Constitution mandates Parliament to facilitate public participation in its legislative and other business.",
		Country:  "KE",
		Sources:  []string{
			"https://www.constituteproject.org/constitution/Kenya_2010",
			"https://www.parliament.go.ke/",
		},
	},
	{
		Term: "Memorandum of Objects and Reasons",
		CanonicalKey: "memorandum_of_objects_and_reasons",
		SimpleExplanation: "The explanatory statement accompanying a Bill setting out its policy, objects, and reasons for each clause.",
		OfficialDefinition: "Standing Order 116 (NA) / 133 (Senate) requires that a Bill be published with a Memorandum of Objects and Reasons.",
		Country:  "KE",
		Sources:  []string{"https://www.parliament.go.ke/"},
	},
}

// FindTerm looks up a KenyaTerm by its canonical key. Returns nil if not found.
func FindTerm(key string) *KenyaTerm {
	for i := range KenyaTerminology {
		if KenyaTerminology[i].CanonicalKey == key {
			return &KenyaTerminology[i]
		}
	}
	return nil
}

// ToContract projects the richer KenyaTerm down to the platform's
// country-agnostic contracts.TermDefinition. The canonical key is the only
// string that links the platform back to this richer record.
func (t KenyaTerm) ToContract() contracts.TermDefinition {
	return contracts.TermDefinition{
		Term:         t.Term,
		CanonicalKey: t.CanonicalKey,
		Description:  t.SimpleExplanation,
	}
}
