// Package ingestion is responsible for acquiring new content from country
// parliamentary sources and notifying the rest of the platform. It owns
// Source, SourceEndpoint, CrawlJob, FetchJob, Document (raw bytes), and
// IngestionRun entities. It performs content hashing (SHA-256) and dedup,
// then publishes source.discovered / source.changed / document.discovered
// events.
//
// # Critical invariant
//
// Ingestion NEVER decides whether a discovered source is a "bill", "act" or
// anything else: it normalises raw bytes into contracts.SourceItem (via the
// country adapter) and publishes events. The downstream services (legislation,
// documents) decide what each item means in the platform.
package ingestion

import "time"

// Source is a registered external source of civic information: a parliamentary
// website, a gazette portal, etc. Sources are configured per-country and
// route crawl jobs to the right country adapter.
type Source struct {
        ID          string
        CountryCode string
        Name        string
        BaseURL     string
        Type        SourceType
        Endpoints   []SourceEndpoint
        AdapterCode string // country adapter that handles normalisation
        PollPeriod  time.Duration
        LastCrawledAt *time.Time
        Enabled     bool
        CreatedAt   time.Time
        UpdatedAt   time.Time
}

// SourceType enumerates the kinds of sources the platform can ingest from.
type SourceType string

const (
        SourceTypeParliament    SourceType = "parliament_site"
        SourceTypeGazette      SourceType = "gazette"
        SourceTypeKenyaLaw      SourceType = "kenya_law"
        SourceTypeCommittee    SourceType = "committee_site"
        SourceTypeCustom        SourceType = "custom"
)

// SourceEndpoint describes one URL pattern within a Source that the crawler
// should hit on each crawl. Templates may use Go's text/template syntax to
// interpolate pagination cursors.
type SourceEndpoint struct {
        ID         string
        SourceID   string
        URLPattern string
        Type       SourceItemType
        Headers    map[string]string
        ParserName string // parser the documents service should use
}

// SourceItemType mirrors contracts.SourceItemType but is duplicated here so
// the ingestion domain does not need to import the contracts package for the
// type alone (it imports contracts for the SourceItem struct anyway).
type SourceItemType string

// CrawlJob represents a scheduled visit to a SourceEndpoint to enumerate
// new items. CrawlJobs produce FetchJobs for each newly-discovered item.
type CrawlJob struct {
        ID           string
        SourceID     string
        EndpointID   string
        StartedAt    time.Time
        FinishedAt   *time.Time
        Status       JobStatus
        ItemsFound   int
        ItemsNew     int
        ItemsChanged int
        Error        string
}

// FetchJob represents a single URL the fetcher must download.
type FetchJob struct {
        ID          string
        SourceID    string
        EndpointID  string
        URL         string
        ExternalID  string
        CountryCode string
        Status      JobStatus
        Attempts    int
        Bytes       int64
        ContentHash string
        MimeType    string
        FetchedAt   *time.Time
        Error       string
}

// JobStatus enumerates crawl/fetch job lifecycle states.
type JobStatus string

const (
        JobStatusPending   JobStatus = "pending"
        JobStatusRunning   JobStatus = "running"
        JobStatusSucceeded JobStatus = "succeeded"
        JobStatusFailed    JobStatus = "failed"
        JobStatusSkipped   JobStatus = "skipped"
)

// RawDocument is the bytes-on-disk representation of fetched content. It is
// distinct from contracts.RawDocument in that it carries a reference to the
// FetchJob that produced it, which is internal to ingestion.
type RawDocument struct {
        ID            string
        SourceID      string
        FetchJobID    string
        ExternalID    string
        CountryCode   string
        URL           string
        MimeType      string
        Bytes         []byte
        ContentHash   string
        FetchedAt     time.Time
}

// IngestionRun is a single end-to-end pass over one or more Sources. Each
// run records aggregate statistics for observability.
type IngestionRun struct {
        ID            string
        StartedAt     time.Time
        FinishedAt    *time.Time
        SourcesSeen   int
        ItemsDiscovered int
        ItemsNew       int
        ItemsChanged   int
        ItemsFailed    int
}
