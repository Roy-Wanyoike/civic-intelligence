// Package contracts defines the shared types and interfaces used across all
// Go services of the Civic Intelligence Platform.
//
// This package is PURE: it imports only the standard library and a small set
// of well-known utility libraries (uuid, zerolog). It must NEVER import
// infrastructure packages (database/sql, net/http, NATS, etc.) — those are
// the responsibility of each service's infrastructure layer.
package contracts

import (
        "context"
        "fmt"
        "time"
)

// Country is a 2-letter ISO 3166-1 alpha-2 country code (e.g., "KE").
type Country string

// ID is a UUID v4 string. We use a string alias instead of importing uuid
// directly in domain code to keep the domain layer dependency-free.
type ID string

// AuthorityLevel indicates how authoritative a source is.
type AuthorityLevel string

const (
        AuthorityPrimary    AuthorityLevel = "primary"     // official government source
        AuthorityOfficial   AuthorityLevel = "official"    // government-affiliated
        AuthorityAggregator AuthorityLevel = "aggregator"  // third-party aggregating official sources
)

// Confidence indicates how strongly the evidence supports a claim.
type Confidence string

const (
        ConfidenceHigh    Confidence = "high"
        ConfidenceMedium  Confidence = "medium"
        ConfidenceLow     Confidence = "low"
        ConfidenceUnknown Confidence = "unknown"
)

// HouseType classifies a chamber of a bicameral (or unicameral) legislature.
type HouseType string

const (
        HouseTypeLower HouseType = "lower"
        HouseTypeUpper HouseType = "upper"
        HouseTypeSingle HouseType = "single" // for unicameral legislatures
)

// SourceItemType is the platform-wide enum for the kind of document a
// SourceItem points to. Used by country adapters to classify discovered URLs.
type SourceItemType string

const (
        SourceItemBill       SourceItemType = "bill"
        SourceItemHansard    SourceItemType = "hansard"
        SourceItemCommittee  SourceItemType = "committee_report"
        SourceItemGazette    SourceItemType = "gazette_notice"
        SourceItemAct        SourceItemType = "act"
        SourceItemRegulation SourceItemType = "regulation"
        SourceItemPolicy     SourceItemType = "policy"
        SourceItemUnknown    SourceItemType = "unknown"
)

// SourceItem is a discovered item from a country adapter's discovery phase.
type SourceItem struct {
        URL            string
        SourceType     SourceItemType
        DocumentType   string    // "bill", "hansard", "order_paper", etc.
        Title          string
        Description    string
        PublishedAt    time.Time
        ContentHash    string    // if already known
        LastModified   time.Time
        SourceID       string    // adapter-specific stable ID (e.g., "ke-bill-2024-23")
        ExternalID     string    // the upstream source's own identifier, if any
        CountryCode   string    // ISO 3166-1 alpha-2
        Type           SourceItemType // alias for SourceType
        House          string    // house code if known
        DiscoveredAt   time.Time
        RawMetadata    map[string]string
        Metadata       map[string]string
}

// RawDocument is the result of fetching a SourceItem — raw bytes + metadata.
type RawDocument struct {
        URL            string
        Bytes          []byte
        MimeType       string
        RetrievedAt    time.Time
        ContentHash    string // SHA-256 of Bytes
        SourceID       ID
}

// ExtractedRecord is a parsed, normalized record extracted from a RawDocument.
// The adapter is responsible for mapping country-specific source data into
// this generic shape. The adapter MUST set SourceURL, RetrievedAt, and ContentHash.
type ExtractedRecord struct {
        Kind         string                 // "bill", "act", "regulation", "committee_report", etc.
        Identifier   string                 // e.g., "NA Bill No. 23 of 2024"
        Title        string
        Description  string
        PublishedAt  time.Time
        House        string                 // resolved via the adapter's GetLegislativeStructure
        Stage        string                 // stage CODE (e.g., "SECOND_READING"); never a localized name
        Sponsor      string
        Topics       []string
        Confidence   float64                // 0.0-1.0 — how confident the adapter is in this extraction
        ExtractedAt  time.Time
        ExtractorName string                // e.g., "kenya.parliament.BillsHTMLParser"
        Pages        []int                  // page numbers in the source document this record came from
        Extra        map[string]interface{} // country-specific extras
        SourceURL    string
        RetrievedAt  time.Time
        ContentHash  string
}

// StageDefinition describes a single Bill stage for a country. The country
// adapter supplies these; the global domain model never hard-codes them.
type StageDefinition struct {
        Code                string   // "SECOND_READING" — country-specific
        Name                string   // "Second Reading"
        Description         string   // longer description (may duplicate SimpleExplanation)
        SimpleExplanation   string   // plain-language explanation
        OfficialDefinition  string   // from the country's Standing Orders if available
        Country             Country  // "KE"
        AllowedNext         []string // codes of stages that may follow
        // AllowedTransitions is an alias for AllowedNext used by the richer
        // StageGraphValidator. Adapters may populate either; the validator
        // normalizes both to the same internal representation.
        AllowedTransitions []string
        Order              int      // sequence number for sorting
        IsTerminal         bool     // true for REJECTED, WITHDRAWN, LAPSED, etc.
        RequiresEvidence   bool     // does the stage transition require evidence?
        RequiresVote       bool     // does the stage transition require a vote?
        TypicalDurationDays int     // typical number of days a Bill spends in this stage
}

// TermDefinition describes a parliamentary term for a country.
type TermDefinition struct {
        Term                string
        CanonicalKey        string   // normalized lookup key (e.g., "second_reading")
        SimpleExplanation   string
        Description         string   // longer description (may duplicate SimpleExplanation)
        OfficialDefinition  string
        StageCode           string   // optional; if the term is a stage
        Country             Country
        Sources             []string // URLs
}

// HouseDefinition is the rich form of a House record — includes member count,
// term length, and chamber type. Country adapters populate this; the global
// domain stores it as values.
type HouseDefinition struct {
        Code     string   // adapter-specific, e.g., "NA" / "SEN" for Kenya
        Name     string   // "National Assembly" | "Senate" (country-specific value)
        Type     HouseType
        Members  int
        TermDays int
}

// CommitteeDefinition is the rich form of a Committee record.
type CommitteeDefinition struct {
        Code    string
        Name    string
        House   string   // house code this committee belongs to
        Type    string   // "departmental", "select", "standing", "sessional"
        Members int
}

// LegislativeStructure describes a country's legislature, houses, and committees.
// This is adapter data — the global domain model stores it as values, not as code.
type LegislativeStructure struct {
        Country       Country
        // CountryCode / CountryName are convenience fields for adapters that
        // work with string codes rather than the typed Country alias.
        CountryCode   string
        CountryName   string
        Institutions  []InstitutionRecord
        Legislatures  []LegislatureRecord
        // Houses is the rich form: includes member count, term length, chamber type.
        // Country adapters populate this; the global domain stores it as values.
        Houses        []HouseDefinition
        // Committees is the rich form.
        Committees    []CommitteeDefinition
}

type InstitutionRecord struct {
        Name           string
        Type           string // "legislature" | "executive" | "judiciary" | ...
        Jurisdiction   string // "national" | "county:Nairobi" | "sector:finance"
        OfficialURLs   []string
}

type LegislatureRecord struct {
        Name           string
        InstitutionName string
}

type HouseRecord struct {
        LegislatureName string
        Name            string // "National Assembly" | "Senate" (Kenya-specific)
        SortOrder       int
}

type CommitteeRecord struct {
        HouseName string
        Name      string
        Type      string
}

// Event is the structured event type published through EventPublisher.
type Event struct {
        ID         string      `json:"id"`          // unique event ID (UUID)
        Type       EventType   `json:"type"`        // dotted event name, e.g. "bill.stage_changed"
        Subject    string      `json:"subject"`     // NATS subject (usually == Type prefixed with "civic.")
        Source     string      `json:"source"`      // service that published ("documents", "evidence", ...)
        OccurredAt time.Time   `json:"occurred_at"` // when the event was emitted
        Payload    interface{} `json:"payload"`     // event-specific data (alias for Data)
        Data       map[string]any `json:"data"`     // event-specific data
}

// EventType returns the dotted event name. Satisfies the legacy interface form.
func (e Event) EventType() string { return string(e.Type) }

// LegislativeSourceAdapter is the interface every country adapter implements.
// The ingestion service calls Discover/Fetch/Parse during ingestion. The
// legislation service calls GetStages/GetTerminology at startup to seed the
// global stage + terminology tables.
//
// IMPORTANT: the adapter is a plugin, not a service. Country-specific code
// stays inside the adapter package. The global domain model contains ZERO
// country-specific strings.
type LegislativeSourceAdapter interface {
        // CountryCode returns the ISO 3166-1 alpha-2 country code this adapter serves.
        CountryCode() string

        // Supports reports whether this adapter can handle the given URL.
        Supports(url string) bool

        // Discover returns the items currently available from this adapter's sources.
        Discover(ctx context.Context) ([]SourceItem, error)

        // Fetch downloads a single item's raw bytes.
        Fetch(ctx context.Context, item SourceItem) (*RawDocument, error)

        // Parse converts a raw document into normalized ExtractedRecords.
        // The adapter is responsible for country-specific HTML/PDF/DOCX parsing.
        Parse(ctx context.Context, doc RawDocument) ([]ExtractedRecord, error)

        // NormalizeSourceItem converts country-specific raw metadata (as produced
        // by the adapter's discovery + fetch steps) into the platform's canonical
        // SourceItem shape. This is the boundary where Kenya-specific field names
        // ("bill_no", "house_name") get mapped to platform fields.
        NormalizeSourceItem(raw map[string]any) (SourceItem, error)

        // GetLegislativeStructure returns the country's institutions, houses, and committees.
        GetLegislativeStructure(ctx context.Context) (*LegislativeStructure, error)

        // GetStages returns the country's Bill stages (with allowed transitions).
        GetStages(ctx context.Context) ([]StageDefinition, error)

        // GetTerminology returns the country's parliamentary terminology registry.
        GetTerminology(ctx context.Context) ([]TermDefinition, error)
}

// ErrValidation is a structured validation error. Returned by domain
// validators to give callers enough context to surface a useful message.
type ErrValidation struct {
        Kind   string // "bill_stage", "amendment", etc.
        Field  string // field name that failed validation
        Reason string // human-readable explanation
}

func (e ErrValidation) Error() string {
        return fmt.Sprintf("validation error [%s/%s]: %s", e.Kind, e.Field, e.Reason)
}

// ErrStageTransition is returned when a Bill attempts an invalid stage
// transition. It includes the from/to stages and the allowed set so callers
// can produce a helpful error message.
type ErrStageTransition struct {
        From    string
        To      string
        Allowed []string
}

func (e ErrStageTransition) Error() string {
        return fmt.Sprintf("invalid stage transition %q -> %q (allowed: %v)", e.From, e.To, e.Allowed)
}

// ErrEvidenceMissing is returned when a candidate fact lacks supporting
// evidence. The Claim field identifies which claim failed validation.
type ErrEvidenceMissing struct {
        Claim string
}

func (e ErrEvidenceMissing) Error() string {
        return fmt.Sprintf("evidence missing for claim %q", e.Claim)
}
