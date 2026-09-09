// Package application holds the ingestion service's use cases. Each handler
// orchestrates domain logic plus repository/event-bus interactions; none of
// them know about Postgres or NATS directly.
package application

import (
        "context"
        "fmt"

        "github.com/Roy-Wanyoike/civic-intelligence/packages/contracts"
        "github.com/Roy-Wanyoike/civic-intelligence/services/ingestion/internal/domain"
)

// CrawlSchedulerHandler schedules a CrawlJob for each enabled endpoint of a
// Source. It is invoked by a cron tick (every N minutes per Source.PollPeriod)
// or manually via the API.
type CrawlSchedulerHandler struct {
        sources domain.SourceRepository
        crawls  domain.CrawlJobRepository
        runs    domain.RunRepository
        clock   domain.Clock
        idGen   IDGenerator
}

// NewCrawlSchedulerHandler constructs the handler.
func NewCrawlSchedulerHandler(
        sources domain.SourceRepository,
        crawls domain.CrawlJobRepository,
        runs domain.RunRepository,
        clock domain.Clock,
        idGen IDGenerator,
) *CrawlSchedulerHandler {
        return &CrawlSchedulerHandler{sources: sources, crawls: crawls, runs: runs, clock: clock, idGen: idGen}
}

// IDGenerator generates string IDs. Domain defines it here so the application
// layer is testable.
type IDGenerator interface {
        New() string
}

// ScheduleCrawlCommand triggers a crawl for the given source.
type ScheduleCrawlCommand struct {
        SourceID string
}

// Handle returns the CrawlJob ID.
func (h *CrawlSchedulerHandler) Handle(ctx context.Context, cmd ScheduleCrawlCommand) (string, error) {
        src, err := h.sources.Get(ctx, cmd.SourceID)
        if err != nil {
                return "", err
        }
        if !src.Enabled {
                return "", contracts.ErrValidation{Kind: "source", Field: "enabled", Reason: "source is disabled"}
        }
        runID := h.idGen.New()
        now := h.clock.Now()
        if err := h.runs.Save(ctx, domain.IngestionRun{
                ID:        runID,
                StartedAt: now,
        }); err != nil {
                return "", err
        }
        crawlID := h.idGen.New()
        job := domain.CrawlJob{
                ID:         crawlID,
                SourceID:   src.ID,
                StartedAt:  now,
                Status:     domain.JobStatusRunning,
        }
        if err := h.crawls.Save(ctx, job); err != nil {
                return "", err
        }
        return crawlID, nil
}

// FetchWorkerHandler is invoked for each pending FetchJob. It downloads the
// URL, computes the content hash, stores the raw document, and publishes a
// source.discovered (new) or source.changed (different hash) event. The
// worker has no idea what the item IS — that decision belongs to the
// country adapter (via NormalizeSourceItem) and downstream services.
type FetchWorkerHandler struct {
        fetches   domain.FetchJobRepository
        docs      domain.DocumentRepository
        registry  domain.SourceRegistry
        hasher    domain.ContentHasher
        publisher domain.EventPublisher
        clock     domain.Clock
        idGen     IDGenerator
        fetcher   HTTPFetcher
}

// HTTPFetcher abstracts the actual HTTP client. In production it's net/http;
// in tests it's a stub.
type HTTPFetcher interface {
        Fetch(ctx context.Context, url string) (body []byte, mimeType string, err error)
}

// NewFetchWorkerHandler constructs the handler.
func NewFetchWorkerHandler(
        fetches domain.FetchJobRepository,
        docs domain.DocumentRepository,
        registry domain.SourceRegistry,
        hasher domain.ContentHasher,
        publisher domain.EventPublisher,
        clock domain.Clock,
        idGen IDGenerator,
        fetcher HTTPFetcher,
) *FetchWorkerHandler {
        return &FetchWorkerHandler{
                fetches: fetches, docs: docs, registry: registry,
                hasher: hasher, publisher: publisher,
                clock: clock, idGen: idGen, fetcher: fetcher,
        }
}

// Handle processes one FetchJob end-to-end.
func (h *FetchWorkerHandler) Handle(ctx context.Context, jobID string) error {
        job, err := h.fetches.Get(ctx, jobID)
        if err != nil {
                return err
        }
        if err := h.fetches.MarkRunning(ctx, jobID); err != nil {
                return err
        }
        body, mime, err := h.fetcher.Fetch(ctx, job.URL)
        if err != nil {
                _ = h.fetches.MarkFailed(ctx, jobID, err.Error())
                return fmt.Errorf("fetch %s: %w", job.URL, err)
        }
        hash := h.hasher.Hash(body)
        now := h.clock.Now()
        if err := h.fetches.MarkDone(ctx, jobID, hash, int64(len(body)), mime); err != nil {
                return err
        }

        // Resolve the country adapter so we can normalise the item.
        adapter, err := h.registry.AdapterFor(job.CountryCode)
        if err != nil {
                return err
        }
        raw := map[string]any{
                "external_id": job.ExternalID,
                "url":         job.URL,
                "mime_type":   mime,
                "content_hash": hash,
        }
        item, err := adapter.NormalizeSourceItem(raw)
        if err != nil {
                return err
        }

        docID := h.idGen.New()
        doc := domain.RawDocument{
                ID: docID, SourceID: job.SourceID, FetchJobID: jobID,
                ExternalID: job.ExternalID, CountryCode: job.CountryCode,
                URL: job.URL, MimeType: mime, Bytes: body,
                ContentHash: hash, FetchedAt: now,
        }
        if err := h.docs.Save(ctx, doc); err != nil {
                return err
        }

        // Determine if this is new vs. changed by looking up the previous doc
        // with this external ID. If a previous hash exists and differs, we
        // emit a "changed" event; otherwise a "discovered" event.
        prev, err := h.docs.GetByExternalID(ctx, job.CountryCode, job.ExternalID)
        if err == nil && prev != nil && prev.ContentHash != hash {
                return h.publisher.Publish(ctx, contracts.Event{
                        ID:        h.idGen.New(),
                        Type:      contracts.EventSourceChanged,
                        Source:    "ingestion",
                        Subject:   "civic.source.changed",
                        OccurredAt: now,
                        Data: map[string]any{
                                "source_id":       job.SourceID,
                                "country_code":   job.CountryCode,
                                "external_id":    job.ExternalID,
                                "old_content_hash": prev.ContentHash,
                                "new_content_hash": hash,
                                "url":            job.URL,
                        },
                })
        }

        // Default: discovered (new content).
        return h.publisher.Publish(ctx, contracts.Event{
                ID:         h.idGen.New(),
                Type:       contracts.EventSourceDiscovered,
                Source:     "ingestion",
                Subject:    "civic.source.discovered",
                OccurredAt: now,
                Data: map[string]any{
                        "source_id":     job.SourceID,
                        "country_code": job.CountryCode,
                        "external_id":  job.ExternalID,
                        "title":        item.Title,
                        "url":          job.URL,
                        "type":         string(item.Type),
                        "content_hash": hash,
                },
        })
}

// RecordDocumentDiscoveredHandler is invoked after a raw document has been
// persisted to publish the document.discovered event, which the documents
// service consumes to start parsing.
type RecordDocumentDiscoveredHandler struct {
        docs      domain.DocumentRepository
        publisher domain.EventPublisher
        clock     domain.Clock
        idGen     IDGenerator
}

// NewRecordDocumentDiscoveredHandler constructs the handler.
func NewRecordDocumentDiscoveredHandler(
        docs domain.DocumentRepository,
        publisher domain.EventPublisher,
        clock domain.Clock,
        idGen IDGenerator,
) *RecordDocumentDiscoveredHandler {
        return &RecordDocumentDiscoveredHandler{docs: docs, publisher: publisher, clock: clock, idGen: idGen}
}

// Handle executes the command.
func (h *RecordDocumentDiscoveredHandler) Handle(ctx context.Context, docID string) error {
        doc, err := h.docs.Get(ctx, docID)
        if err != nil {
                return err
        }
        now := h.clock.Now()
        if now.IsZero() {
                now = doc.FetchedAt
        }
        return h.publisher.Publish(ctx, contracts.Event{
                ID:         h.idGen.New(),
                Type:       contracts.EventDocumentDiscovered,
                Source:     "ingestion",
                Subject:    "civic.document.discovered",
                OccurredAt: now,
                Data: map[string]any{
                        "document_id":  doc.ID,
                        "source_id":    doc.SourceID,
                        "country_code": doc.CountryCode,
                        "mime_type":    doc.MimeType,
                        "content_hash": doc.ContentHash,
                        "url":          doc.URL,
                },
        })
}

// Ensure time import is used (referenced by handler signatures indirectly
// through Clock.Now(); here it's kept for future expansion).
var _ = context.Background
