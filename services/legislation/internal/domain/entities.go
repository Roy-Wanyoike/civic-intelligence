package legislation

import (
	"time"

	"github.com/Roy-Wanyoike/civic-intelligence/packages/contracts"
)

// Country is a 2-letter ISO code. Stored in the countries table.
type Country struct {
	ISOCode contracts.Country
	Name    string
	Active  bool
}

// Institution is a civic institution (legislature, executive, judiciary, etc.).
type Institution struct {
	ID              contracts.ID
	CountryID       contracts.Country
	Name            string
	Type            string // "legislature" | "executive" | "judiciary" | "ministry" | "agency" | "regulator" | "county" | "commission" | "constitutional_body" | "independent_office"
	ParentID        *contracts.ID
	Jurisdiction    string
	OfficialSources []string
	Active          bool
}

type Legislature struct {
	ID             contracts.ID
	CountryID      contracts.Country
	InstitutionID  contracts.ID
	Name           string
	Active         bool
}

// House is a chamber of a legislature. In Kenya: National Assembly, Senate.
// The strings "National Assembly" and "Senate" are VALUES supplied by the
// Kenya adapter, never constants in this file.
type House struct {
	ID            contracts.ID
	LegislatureID contracts.ID
	Name          string
	SortOrder     int
	Active        bool
}

type Committee struct {
	ID        contracts.ID
	HouseID   contracts.ID
	Name      string
	Type      string // "departmental", "select", "standing", "sessional"
	Active    bool
}

type Person struct {
	ID            contracts.ID
	CountryID     contracts.Country
	Name          string
	PrimaryRole   string // "mp", "senator", "cs", "ag", "mcvp"
	InstitutionID *contracts.ID
	PartyID       *contracts.ID
	ConstituencyID *contracts.ID
	CountyID      *contracts.ID
	StartDate     *time.Time
	EndDate       *time.Time
	OfficialURL   string
}

// Bill is the canonical Bill record. It owns the metadata; BillVersion owns
// the immutable text. BillEvent owns the timeline.
type Bill struct {
	ID           contracts.ID
	CountryID    contracts.Country
	HouseID      contracts.ID
	Identifier   string // e.g., "NA Bill No. 23 of 2024"
	Year         int
	Title        string
	SponsorID    *contracts.ID
	CommitteeID  *contracts.ID
	Status       BillStatus
	CurrentStage string // stage CODE resolved via the adapter; e.g., "SECOND_READING"
	Purpose      string
	Description  string
	Topics       []string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type BillStatus string

const (
	BillStatusInProgress BillStatus = "in_progress"
	BillStatusEnacted    BillStatus = "enacted"
	BillStatusWithdrawn  BillStatus = "withdrawn"
	BillStatusRejected   BillStatus = "rejected"
	BillStatusLapsed     BillStatus = "lapsed"
)

// BillVersion is IMMUTABLE. Never UPDATE; only INSERT.
type BillVersion struct {
	ID          contracts.ID
	BillID      contracts.ID
	VersionNo   int
	ContentHash string
	RetrievedAt time.Time
	SourceURL   string
	DocumentID  *contracts.ID
	IsCurrent   bool
}

// BillEvent is one verified event in a Bill's timeline.
// If EventDate is nil, the date is unknown — preserve that uncertainty.
type BillEvent struct {
	ID                contracts.ID
	BillID            contracts.ID
	EventType         string // "first_reading", "second_reading", etc.
	EventDate         *time.Time
	DateIsApproximate bool
	House             string
	Description       string
	SourceDocumentID  *contracts.ID
	SourceURL         string
	Confidence        contracts.Confidence
	Note              string
}

type Amendment struct {
	ID            contracts.ID
	BillID        contracts.ID
	BillVersionID *contracts.ID
	MoverID       *contracts.ID
	Title         string
	Text          string
	ProposedAt    time.Time
	Status        string // "proposed", "accepted", "rejected", "withdrawn"
	SourceURL     string
}

type Clause struct {
	ID            contracts.ID
	BillVersionID contracts.ID
	ClauseNo      int
	Heading        string
	Text          string
	PageNumber    *int
}
