package internal

import "github.com/Roy-Wanyoike/civic-intelligence/packages/contracts"

// House codes used throughout the Kenya adapter. They appear in RawMetadata
// and SourceItem.House fields, but never as bare strings in the global
// domain model.
const (
	HouseCodeNationalAssembly = "NA"
	HouseCodeSenate           = "SEN"
)

// Committee codes are stable identifiers used by the adapter only. The
// contracts.CommitteeDefinition exposes them via the Code field, which the
// legislation service uses as an opaque key.
const (
	CommitteeJusticeLegalAffairs    = "JLA"
	CommitteeFinance                = "FIN"
	CommitteeHealth                 = "HLT"
	CommitteeEducation              = "EDU"
	CommitteeTrade                  = "TRD"
	CommitteeAgriculture            = "AGR"
	CommitteeDefence                = "DEF"
	CommitteeEnergy                 = "ENG"
	CommitteeCommunication          = "COM"
	CommitteePublicAccounts         = "PAC"
	CommitteePublicInvestments      = "PIC"
	CommitteeDelegatedLegislation   = "DEL"
	CommitteeImplementation         = "IMP"
	CommitteeProcedureAndRules      = "PCR"
	CommitteeSelection              = "SEL"
	CommitteeHouse                  = "HSE"
	CommitteeLibrary                = "LIB"
	CommitteeMembersServices        = "MSS"
)

// KenyaLegislativeStructure returns the structural description of Kenya's
// Parliament as established by the 2010 Constitution: a bicameral
// legislature comprising the National Assembly (lower house) and the Senate
// (upper house), each with its standing committees.
//
// Sources:
//   - Constitution of Kenya, 2010, Articles 93–104 (Parliament)
//     https://www.constituteproject.org/constitution/Kenya_2010
//   - Parliament of Kenya
//     https://www.parliament.go.ke/
//   - The National Assembly
//     https://www.parliament.go.ke/the-national-assembly
//   - The Senate
//     https://www.parliament.go.ke/the-senate
func KenyaLegislativeStructure() contracts.LegislativeStructure {
	return contracts.LegislativeStructure{
		CountryCode: "KE",
		CountryName: "Kenya",
		Houses: []contracts.HouseDefinition{
			{
				Code:     HouseCodeNationalAssembly,
				Name:     "National Assembly",
				Type:     contracts.HouseTypeLower,
				Members:  349, // 290 elected + 47 women reps + 12 nominated = 349 per Article 97
				TermDays: 5 * 365,
			},
			{
				Code:     HouseCodeSenate,
				Name:     "Senate",
				Type:     contracts.HouseTypeUpper,
				Members:  67, // 47 elected + 16 nominated women + 2 youth + 2 PWDs = 67 per Article 98
				TermDays: 5 * 365,
			},
		},
		Committees: kenyaStandingCommittees(),
	}
}

// kenyaStandingCommittees returns the standing committees that the National
// Assembly and Senate publish on their official sites. Joint committees
// (e.g. the Parliamentary Service Commission) are out of scope here.
//
// Committee definitions are derived from the published Committee Lists:
//   - National Assembly departmental committees
//     https://www.parliament.go.ke/the-national-assembly/departmental-committees
//   - Senate standing committees
//     https://www.parliament.go.ke/the-senate/standing-committees
func kenyaStandingCommittees() []contracts.CommitteeDefinition {
	return []contracts.CommitteeDefinition{
		// National Assembly departmental & select committees.
		{Code: CommitteeJusticeLegalAffairs, Name: "Justice and Legal Affairs", House: HouseCodeNationalAssembly, Type: "departmental", Members: 29},
		{Code: CommitteeFinance, Name: "Finance and National Planning", House: HouseCodeNationalAssembly, Type: "departmental", Members: 29},
		{Code: CommitteeHealth, Name: "Health", House: HouseCodeNationalAssembly, Type: "departmental", Members: 29},
		{Code: CommitteeEducation, Name: "Education and Research", House: HouseCodeNationalAssembly, Type: "departmental", Members: 29},
		{Code: CommitteeTrade, Name: "Trade, Industry and Cooperatives", House: HouseCodeNationalAssembly, Type: "departmental", Members: 29},
		{Code: CommitteeAgriculture, Name: "Agriculture and Livestock", House: HouseCodeNationalAssembly, Type: "departmental", Members: 29},
		{Code: CommitteeDefence, Name: "Defence, Foreign Relations and Intelligence", House: HouseCodeNationalAssembly, Type: "departmental", Members: 29},
		{Code: CommitteeEnergy, Name: "Energy", House: HouseCodeNationalAssembly, Type: "departmental", Members: 29},
		{Code: CommitteeCommunication, Name: "Communication, Information and Innovation", House: HouseCodeNationalAssembly, Type: "departmental", Members: 29},
		{Code: CommitteePublicAccounts, Name: "Public Accounts", House: HouseCodeNationalAssembly, Type: "sessional", Members: 29},
		{Code: CommitteePublicInvestments, Name: "Public Investments", House: HouseCodeNationalAssembly, Type: "sessional", Members: 29},
		{Code: CommitteeDelegatedLegislation, Name: "Delegated Legislation", House: HouseCodeNationalAssembly, Type: "sessional", Members: 21},
		{Code: CommitteeImplementation, Name: "Implementation", House: HouseCodeNationalAssembly, Type: "sessional", Members: 29},
		{Code: CommitteeProcedureAndRules, Name: "Procedure and Rules (House Business)", House: HouseCodeNationalAssembly, Type: "select", Members: 27},
		{Code: CommitteeSelection, Name: "Selection", House: HouseCodeNationalAssembly, Type: "select", Members: 19},
		{Code: CommitteeHouse, Name: "House", House: HouseCodeNationalAssembly, Type: "select", Members: 12},
		{Code: CommitteeLibrary, Name: "Library", House: HouseCodeNationalAssembly, Type: "select", Members: 12},
		{Code: CommitteeMembersServices, Name: "Members' Services and Facilities", House: HouseCodeNationalAssembly, Type: "select", Members: 12},

		// Senate standing committees.
		{Code: CommitteeJusticeLegalAffairs, Name: "Justice, Legal Affairs and Human Rights", House: HouseCodeSenate, Type: "standing", Members: 13},
		{Code: CommitteeFinance, Name: "Finance and Budget", House: HouseCodeSenate, Type: "standing", Members: 13},
		{Code: CommitteeHealth, Name: "Health", House: HouseCodeSenate, Type: "standing", Members: 13},
		{Code: CommitteeEducation, Name: "Education", House: HouseCodeSenate, Type: "standing", Members: 13},
		{Code: CommitteeTrade, Name: "Trade, Industrialization and Tourism", House: HouseCodeSenate, Type: "standing", Members: 13},
		{Code: CommitteeAgriculture, Name: "Agriculture, Livestock and Fisheries", House: HouseCodeSenate, Type: "standing", Members: 13},
		{Code: CommitteeDefence, Name: "National Security, Defence and Foreign Relations", House: HouseCodeSenate, Type: "standing", Members: 13},
		{Code: CommitteeEnergy, Name: "Energy", House: HouseCodeSenate, Type: "standing", Members: 13},
		{Code: CommitteeCommunication, Name: "Information, Communication and Technology", House: HouseCodeSenate, Type: "standing", Members: 13},
		{Code: CommitteePublicAccounts, Name: "County Public Accounts", House: HouseCodeSenate, Type: "standing", Members: 13},
		{Code: CommitteePublicInvestments, Name: "County Public Investments and Special Funds", House: HouseCodeSenate, Type: "standing", Members: 13},
		{Code: CommitteeDelegatedLegislation, Name: "Delegated Legislation", House: HouseCodeSenate, Type: "standing", Members: 13},
		{Code: CommitteeImplementation, Name: "County Public Investments", House: HouseCodeSenate, Type: "standing", Members: 13},
		{Code: CommitteeProcedureAndRules, Name: "House Business (Procedure and Rules)", House: HouseCodeSenate, Type: "standing", Members: 19},
		{Code: CommitteeSelection, Name: "Selection", House: HouseCodeSenate, Type: "standing", Members: 11},
		{Code: CommitteeHouse, Name: "House", House: HouseCodeSenate, Type: "standing", Members: 12},
		{Code: CommitteeLibrary, Name: "Library", House: HouseCodeSenate, Type: "standing", Members: 12},
		{Code: CommitteeMembersServices, Name: "Members' Services and Facilities", House: HouseCodeSenate, Type: "standing", Members: 12},
	}
}
