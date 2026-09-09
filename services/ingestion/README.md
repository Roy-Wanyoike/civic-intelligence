# ingestion service

The **ingestion** service acquires new content from country parliamentary
sources and notifies the rest of the platform via events.

## Bounded context

Owns:
- Sources, SourceEndpoints
- CrawlJobs, FetchJobs, IngestionRuns
- Raw documents (bytes + SHA-256 hash)
- The content-hash dedup logic that decides whether an item is new, changed,
  or unchanged.

Does NOT own:
- Stage decisions (→ legislation)
- Parsed text / chunks (→ documents)
- Country-specific field mapping (→ adapters/kenya)
- Notifications (→ notifications)

## Architectural rules enforced by this service

1. **No stage decisions.** Ingestion only knows how to fetch bytes and hash
   them. The country adapter's `NormalizeSourceItem` produces a
   `contracts.SourceItem`; what that item *means* is decided downstream.
2. **SHA-256 content hashing is the dedup key.** Two FetchJobs producing the
   same hash are treated as the same content; the second is not a new item.
3. **Country routing via SourceRegistry.** FetchJobs carry a `CountryCode`;
   the registry resolves the right adapter.
4. **Always publish events.** New items emit `source.discovered`; changed
   items emit `source.changed`. The documents service consumes
   `document.discovered` to start parsing.

## Running

```bash
go build ./...
go test ./...
```

The service exposes `/healthz` and `/readyz`. Production wiring (pgxpool,
NATS, adapter registration) is performed by the main agent.
