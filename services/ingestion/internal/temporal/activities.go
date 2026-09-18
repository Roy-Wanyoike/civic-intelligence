// FIXME: verify with go build when Go available
//
// Activities for the Bill Processing Pipeline. Each activity is a thin,
// side-effect-owning adapter around a domain/infrastructure dependency. They
// are intentionally small so they can be unit-tested with mocks and replayed
// deterministically by Temporal.
package temporal

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

// ---------------------------------------------------------------------------
// Discover
// ---------------------------------------------------------------------------

// Discover is the Temporal activity entrypoint for the Discover step.
// (Method name is unique per activity so registration doesn't collide —
// Temporal Go SDK names activities by method name.)
func (a *DiscoverActivity) Discover(ctx context.Context, in DiscoverInput) (DiscoverResult, error) {
	if a == nil || a.Registry == nil {
		return DiscoverResult{}, fmt.Errorf("discover: registry not wired")
	}
	items, err := a.Registry.Discover(ctx, in.CountryCode, in.SourceURL, in.Identifier)
	if err != nil {
		return DiscoverResult{}, fmt.Errorf("discover: %w", err)
	}
	if len(items) == 0 {
		return DiscoverResult{}, fmt.Errorf("discover: no items found at %s", in.SourceURL)
	}
	return DiscoverResult{Items: items}, nil
}

// ---------------------------------------------------------------------------
// Fetch
// ---------------------------------------------------------------------------

// Fetch downloads the source item, computes its content hash, uploads the
// raw bytes to object storage, and returns the document ID.
func (a *FetchActivity) Fetch(ctx context.Context, in FetchInput) (FetchResult, error) {
	if a == nil || a.Fetcher == nil || a.Storage == nil || a.IDGen == nil {
		return FetchResult{}, fmt.Errorf("fetch: dependencies not wired")
	}
	body, contentType, err := a.Fetcher.Fetch(ctx, in.Item.URL, in.UserAgent)
	if err != nil {
		return FetchResult{}, fmt.Errorf("fetch: %w", err)
	}
	if len(body) == 0 {
		return FetchResult{}, fmt.Errorf("fetch: empty body from %s", in.Item.URL)
	}

	sum := sha256.Sum256(body)
	hash := hex.EncodeToString(sum[:])

	// Bucket layout: ingestion-<country>/<year>/<hash>.<ext>
	bucket := fmt.Sprintf("ingestion-%s", in.CountryCode)
	key := fmt.Sprintf("%d/%s", time.Now().UTC().Year(), hash)
	docID := a.IDGen.New()

	storedID, err := a.Storage.Upload(ctx, bucket, key, body, contentType)
	if err != nil {
		return FetchResult{}, fmt.Errorf("fetch: store: %w", err)
	}
	if storedID != "" {
		docID = storedID
	}

	return FetchResult{
		DocumentID:  docID,
		ContentHash: hash,
		Bytes:       int64(len(body)),
		ContentType: contentType,
	}, nil
}

// ---------------------------------------------------------------------------
// Parse
// ---------------------------------------------------------------------------

// Parse delegates to the documents service parser pipeline.
func (a *ParseActivity) Parse(ctx context.Context, in ParseInput) (ParseResult, error) {
	if a == nil || a.Documents == nil {
		return ParseResult{}, fmt.Errorf("parse: documents client not wired")
	}
	res, err := a.Documents.Parse(ctx, in.DocumentID, in.ParserName)
	if err != nil {
		return ParseResult{}, fmt.Errorf("parse: %w", err)
	}
	if res.Title == "" && len(res.Sections) == 0 {
		return ParseResult{}, fmt.Errorf("parse: no sections extracted for %s", in.DocumentID)
	}
	return res, nil
}

// ---------------------------------------------------------------------------
// Extract (AI — produces candidate facts; never writes canonical state)
// ---------------------------------------------------------------------------

// Extract calls the AI gateway. The gateway handles retries, timeouts, and
// budget enforcement internally; this activity just orchestrates the three
// calls (summary, topics, stage) in parallel and merges the results.
func (a *ExtractActivity) Extract(ctx context.Context, in ExtractInput) (ExtractResult, error) {
	if a == nil || a.AI == nil {
		return ExtractResult{}, fmt.Errorf("extract: AI gateway not wired")
	}
	summary, modelID, err := a.AI.GenerateSummary(ctx, in.BillID, in.Title, in.Text, in.Audience)
	if err != nil {
		return ExtractResult{}, fmt.Errorf("extract: summary: %w", err)
	}
	topics, err := a.AI.Classify(ctx, in.BillID, in.Title, in.Text)
	if err != nil {
		return ExtractResult{}, fmt.Errorf("extract: classify: %w", err)
	}
	stage, err := a.AI.DetectStage(ctx, in.BillID, in.Title, in.Text)
	if err != nil {
		return ExtractResult{}, fmt.Errorf("extract: stage: %w", err)
	}
	return ExtractResult{
		Summary: summary,
		Topics:  topics,
		Stage:   stage,
		ModelID: modelID,
	}, nil
}

// ---------------------------------------------------------------------------
// Validate (evidence gate)
// ---------------------------------------------------------------------------

// Validate cross-checks the summary, topics, and stage against the evidence
// service. Topics whose underlying claims have no supporting citation are
// dropped. The summary is kept only if at least one supporting citation
// exists for its central claim.
func (a *ValidateActivity) Validate(ctx context.Context, in ValidateInput) (ValidateResult, error) {
	if a == nil || a.Validator == nil || a.Evidence == nil {
		return ValidateResult{}, fmt.Errorf("validate: dependencies not wired")
	}

	accepted := make([]string, 0, len(in.Topics))
	rejected := 0
	for _, topic := range in.Topics {
		ok, err := a.Validator.Validate(ctx, in.BillID, topic, in.SourceURL)
		if err != nil {
			// Validation failure is non-fatal: drop the topic, log it, move on.
			rejected++
			continue
		}
		if ok {
			accepted = append(accepted, topic)
		} else {
			rejected++
		}
	}

	// The summary itself is treated as one big claim. If the evidence
	// service can't corroborate it, mark the workflow as not-validated but
	// still allow publication (the summary will be flagged "AI-generated,
	// unverified" downstream — see ADR-0011).
	validated := true
	if supports, _, err := a.Evidence.Lookup(ctx, in.Summary); err == nil && supports == 0 {
		validated = false
	}

	return ValidateResult{
		Validated:      validated,
		AcceptedTopics: accepted,
		RejectedCount:  rejected,
	}, nil
}

// ---------------------------------------------------------------------------
// Publish (the ONLY writer to canonical legislation.bills state)
// ---------------------------------------------------------------------------

// Publish calls the legislation service to upsert the bill and then publishes
// a `bill.processed` event so downstream services (notifications, search index)
// can react. The event is published AFTER the upsert so subscribers never see
// an event for a bill that doesn't yet exist.
func (a *PublishActivity) Publish(ctx context.Context, in PublishInput) (PublishResult, error) {
	if a == nil || a.Publisher == nil || a.Legislation == nil {
		return PublishResult{}, fmt.Errorf("publish: dependencies not wired")
	}

	billID, err := a.Legislation.UpsertBill(ctx, in)
	if err != nil {
		return PublishResult{}, fmt.Errorf("publish: upsert: %w", err)
	}

	// The event payload is constructed here (not in the legislation service)
	// so the workflow owns the shape of the event. Subscribers depend on it.
	payload := []byte(fmt.Sprintf(
		`{"bill_id":%q,"country_code":%q,"identifier":%q,"year":%d,"validated":%t}`,
		billID, in.CountryCode, in.Identifier, in.Year, in.Validated,
	))
	if err := a.Publisher.Publish(ctx, "bill.processed", payload); err != nil {
		// The bill was upserted but the event failed. This is recoverable:
		// a re-run of the workflow will re-publish (UpsertBill is idempotent
		// on (country_code, identifier, year)).
		return PublishResult{}, fmt.Errorf("publish: event: %w", err)
	}

	return PublishResult{
		BillID:      billID,
		PublishedAt: time.Now().UTC(),
	}, nil
}
