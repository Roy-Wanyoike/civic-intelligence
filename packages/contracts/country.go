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
	AuthorityPrimary     AuthorityLevel = "primary"     // official government source
	AuthorityOfficial    AuthorityLevel = "official"    // government-affiliated
	AuthorityAggregator  AuthorityLevel = "aggregator"  // third-party aggregating official sources
)

// Confidence indicates how strongly the evidence supports a claim.
type Confidence string

const (
	ConfidenceHigh    Confidence = "high"
	ConfidenceMedium  Confidence = "medium"
	ConfidenceLow     Confidence = "low"
	ConfidenceUnknown Confidence = "unknown"
)

// SourceItem is a discovered item from a country adapter's discovery phase.
type SourceItem struct {
	URL           string
	DocumentType  string    // "bill", "hansard", "order_paper", etc.
	Title         string
	PublishedAt   time.Time
	ContentHash   string    // if already known
	LastModified  time.Time
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
	PublishedAt  time.Time
	House        string                 // resolved via the adapter's GetLegislativeStructure
	Stage        string                 // stage CODE (e.g., "SECOND_READING"); never a localized name
	Sponsor      string
	Topics       []string
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
	SimpleExplanation   string   // plain-language explanation
	OfficialDefinition  string   // from the country's Standing Orders if available
	Country             Country  // "KE"
	AllowedNext         []string // codes of stages that may follow
	IsTerminal          bool     // true for REJECTED, WITHDRAWN, LAPSED, etc.
}

// TermDefinition describes a parliamentary term for a country.
type TermDefinition struct {
	Term                string
	SimpleExplanation   string
	OfficialDefinition  string
	StageCode           string // optional; if the term is a stage
	Country             Country
	Sources             []string // URLs
}

// LegislativeStructure describes a country's legislature, houses, and committees.
// This is adapter data — the global domain model stores it as values, not as code.
type LegislativeStructure struct {
	Country       Country
	Institutions  []InstitutionRecord
	Legislatures  []LegislatureRecord
	Houses        []HouseRecord
	Committees    []CommitteeRecord
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

// LegislativeSourceAdapter is the interface every country adapter implements.
// The ingestion service calls Discover/Fetch/Parse during ingestion. The
// legislation service calls GetStages/GetTerminology at startup to seed the
// global stage + terminology tables.
//
// IMPORTANT: the adapter is a plugin, not a service. Country-specific code
// stays inside the adapter package. The global domain model contains ZERO
// country-specific strings.
type LegislativeSourceAdapter interface {
	// Discover returns the items currently available from this adapter's sources.
	Discover(ctx context.Context) ([]SourceItem, error)

	// Fetch downloads a single item's raw bytes.
	Fetch(ctx context.Context, item SourceItem) (*RawDocument, error)

	// Parse converts a raw document into normalized ExtractedRecords.
	// The adapter is responsible for country-specific HTML/PDF/DOCX parsing.
	Parse(ctx context.Context, doc RawDocument) ([]ExtractedRecord, error)

	// GetLegislativeStructure returns the country's institutions, houses, and committees.
	GetLegislativeStructure(ctx context.Context) (*LegislativeStructure, error)

	// GetStages returns the country's Bill stages (with allowed transitions).
	GetStages(ctx context.Context) ([]StageDefinition, error)

	// GetTerminology returns the country's parliamentary terminology registry.
	GetTerminology(ctx context.Context) ([]TermDefinition, error)
}
