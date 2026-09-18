package domain

import "time"

// ID is the platform-wide identifier type for entities. It is a string so we
// can use UUIDv7, ULIDs or country-specific identifiers without type churn.
type ID string

// Country is the root aggregate of the platform. Every other entity belongs
// to exactly one Country. The platform is multi-country by design; today
// only Kenya is wired up.
type Country struct {
	ID        ID
	Code      string // ISO 3166-1 alpha-2, e.g. "KE"
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Institution is a constitutional or statutory body in a country (e.g. the
// Parliament of Kenya, the Judiciary, the Office of the Auditor General).
type Institution struct {
	ID         ID
	CountryID  ID
	Name       string
	Type       string // "parliament", "judiciary", "executive", etc.
	ParentID   *ID    // optional parent institution
	CreatedAt  time.Time
}

// Legislature is the legislative body of a country. Kenya has one
// Legislature (the Parliament of Kenya) which contains two Houses.
type Legislature struct {
	ID           ID
	CountryID    ID
	Name         string
	StartDate    time.Time
	EndDate      *time.Time // nil if current
	Houses       []House
	CreatedAt    time.Time
}

// House is one chamber of a legislature. Kenya has the National Assembly
// and the Senate. House codes are country-specific and provided by the
// adapter — the legislation service never hard-codes them.
type House struct {
	ID           ID
	LegislatureID ID
	Code          string // country-provided code, e.g. "NA", "SEN"
	Name          string
	Type          string // "lower", "upper", "single"
	Members       int
	TermDays      int
}

// Committee is a standing or select committee of one or both houses.
type Committee struct {
	ID         ID
	HouseID    *ID    // nil for joint committees
	Code       string
	Name       string
	Type       string // "standing", "select", "departmental", "joint"
	Members    int
	CreatedAt  time.Time
}

// Person is any individual the platform tracks: members of parliament,
// sponsors of bills, committee chairs, etc. The identity service owns
// user accounts; this entity owns civic persons.
type Person struct {
	ID            ID
	CountryID     ID
	FullName      string
	DisplayName   string
	HouseID       *ID
	PartyID       *ID
	CommitteeIDs  []ID
	BiographyURL  string
	BornAt        *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// PoliticalParty is a registered political party.
type PoliticalParty struct {
	ID         ID
	CountryID  ID
	Name       string
	Abbreviation string
	Colour     string
	CreatedAt  time.Time
}

// Constituency is an electoral unit returning one member to a House. Kenya
// has 290 constituencies for the National Assembly.
type Constituency struct {
	ID         ID
	CountryID  ID
	Name       string
	CountyID   *ID
	Code       string
	Population int
}

// County is the second-tier administrative division in Kenya. The platform
// tracks them because Senate representation and many bills concern counties.
type County struct {
	ID         ID
	CountryID  ID
	Name       string
	Code       string
	Governor   string
	Population int
}
