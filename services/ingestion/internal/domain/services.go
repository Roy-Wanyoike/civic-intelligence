package domain

import (
        "context"
        "crypto/sha256"
        "encoding/hex"
        "fmt"
        "io"
        "time"

        "github.com/Roy-Wanyoike/civic-intelligence/packages/contracts"
)

// SourceRegistry resolves the LegislativeSourceAdapter for a given country
// code. The ingestion service injects concrete adapters at startup; the
// domain code depends only on this interface.
type SourceRegistry interface {
        // AdapterFor returns the adapter for the country code, or an error if
        // no adapter is registered.
        AdapterFor(countryCode string) (contracts.LegislativeSourceAdapter, error)
        // Supports returns true if any registered adapter can handle the URL.
        Supports(url string) bool
        // AdapterForURL returns the adapter whose Supports() returns true for
        // the URL.
        AdapterForURL(url string) (contracts.LegislativeSourceAdapter, error)
}

// AdapterRegistry is the in-memory default implementation of SourceRegistry.
// Adapters are registered at startup; lookups are O(n) on the registered
// adapter count, which is tiny (one per country).
type AdapterRegistry struct {
        adapters []contracts.LegislativeSourceAdapter
}

// NewAdapterRegistry constructs a registry pre-populated with adapters.
func NewAdapterRegistry(adapters ...contracts.LegislativeSourceAdapter) *AdapterRegistry {
        return &AdapterRegistry{adapters: adapters}
}

// Register adds an adapter to the registry.
func (r *AdapterRegistry) Register(a contracts.LegislativeSourceAdapter) {
        r.adapters = append(r.adapters, a)
}

// AdapterFor implements SourceRegistry.
func (r *AdapterRegistry) AdapterFor(countryCode string) (contracts.LegislativeSourceAdapter, error) {
        for _, a := range r.adapters {
                if a.CountryCode() == countryCode {
                        return a, nil
                }
        }
        return nil, fmt.Errorf("ingestion: no adapter for country %q", countryCode)
}

// Supports implements SourceRegistry.
func (r *AdapterRegistry) Supports(url string) bool {
        for _, a := range r.adapters {
                if a.Supports(url) {
                        return true
                }
        }
        return false
}

// AdapterForURL implements SourceRegistry.
func (r *AdapterRegistry) AdapterForURL(url string) (contracts.LegislativeSourceAdapter, error) {
        for _, a := range r.adapters {
                if a.Supports(url) {
                        return a, nil
                }
        }
        return nil, fmt.Errorf("ingestion: no adapter supports URL %q", url)
}

// ContentHasher computes a stable SHA-256 content hash for any byte slice.
// The hash is the deduplication key: if two FetchJobs produce the same hash,
// the second is treated as a duplicate (not a new item).
type ContentHasher interface {
        Hash(content []byte) string
}

// SHA256Hasher is the canonical implementation.
type SHA256Hasher struct{}

// Hash implements ContentHasher.
func (SHA256Hasher) Hash(content []byte) string {
        if len(content) == 0 {
                return ""
        }
        sum := sha256.Sum256(content)
        return hex.EncodeToString(sum[:])
}

// HashReader hashes a streaming reader without buffering the whole body.
// Useful when downloading large gazettes.
func (SHA256Hasher) HashReader(r io.Reader) (string, error) {
        h := sha256.New()
        if _, err := io.Copy(h, r); err != nil {
                return "", fmt.Errorf("hash reader: %w", err)
        }
        return hex.EncodeToString(h.Sum(nil)), nil
}

// Deduplicator compares new items' content hashes against previously-seen
// ones and partitions them into new vs. changed vs. unchanged.
type Deduplicator interface {
        // Partition returns the index partitioning of items into the three
        // buckets. Items without an entry in known are "new"; items whose
        // hash differs are "changed"; items whose hash matches are "unchanged".
        Partition(items []contracts.SourceItem, known map[string]string) (newItems, changed, unchanged []contracts.SourceItem)
}

// HashDeduplicator is the canonical implementation.
type HashDeduplicator struct{}

// Partition implements Deduplicator.
func (HashDeduplicator) Partition(items []contracts.SourceItem, known map[string]string) (newItems, changed, unchanged []contracts.SourceItem) {
        for _, it := range items {
                if it.ContentHash == "" {
                        // Without a hash we cannot dedup; treat as new.
                        newItems = append(newItems, it)
                        continue
                }
                prev, ok := known[it.ExternalID]
                if !ok {
                        newItems = append(newItems, it)
                        continue
                }
                if prev == it.ContentHash {
                        unchanged = append(unchanged, it)
                        continue
                }
                changed = append(changed, it)
        }
        return
}

// SourceRepository persists Source aggregates.
type SourceRepository interface {
        Save(ctx context.Context, s Source) error
        Get(ctx context.Context, id string) (*Source, error)
        List(ctx context.Context, country string) ([]Source, error)
        GetByBaseURL(ctx context.Context, baseURL string) (*Source, error)
}

// FetchJobRepository persists FetchJobs.
type FetchJobRepository interface {
        Save(ctx context.Context, j FetchJob) error
        Get(ctx context.Context, id string) (*FetchJob, error)
        ListPending(ctx context.Context, limit int) ([]FetchJob, error)
        MarkRunning(ctx context.Context, id string) error
        MarkDone(ctx context.Context, id string, hash string, bytes int64, mime string) error
        MarkFailed(ctx context.Context, id string, reason string) error
}

// CrawlJobRepository persists CrawlJobs.
type CrawlJobRepository interface {
        Save(ctx context.Context, j CrawlJob) error
        ListBySource(ctx context.Context, sourceID string, limit int) ([]CrawlJob, error)
}

// DocumentRepository persists raw documents (without bytes; bytes live in
// object storage and are referenced by ContentHash).
type DocumentRepository interface {
        Save(ctx context.Context, d RawDocument) error
        Get(ctx context.Context, id string) (*RawDocument, error)
        GetByContentHash(ctx context.Context, hash string) (*RawDocument, error)
        GetByExternalID(ctx context.Context, countryCode, externalID string) (*RawDocument, error)
}

// RunRepository persists IngestionRuns.
type RunRepository interface {
        Save(ctx context.Context, r IngestionRun) error
}

// EventPublisher is the abstract event bus.
type EventPublisher interface {
        Publish(ctx context.Context, event contracts.Event) error
}

// Clock abstracts time so handlers can be deterministic in tests.
type Clock interface {
        Now() time.Time
}

// SystemClock is the production Clock.
type SystemClock struct{}

// Now implements Clock.
func (SystemClock) Now() time.Time { return time.Now().UTC() }

// Compile-time assertion.
var _ Clock = SystemClock{}
