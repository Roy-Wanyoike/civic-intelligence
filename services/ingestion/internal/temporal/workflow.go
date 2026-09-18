//
// Package temporal implements the durable Bill Processing Pipeline as a
// Temporal workflow.
//
// Pipeline (per ENG-F3): Discover → Fetch → Parse → Extract → Validate → Publish
//
// Each step is an Activity. Temporal handles retries, timeouts, and replay
// after crashes. Activities own all side-effects (HTTP, DB, NATS, AI gateway);
// the workflow itself is deterministic and only orchestrates activity calls.
//
// Architectural contract (ADR-0007): Temporal never directly mutates canonical
// state in `legislation.*` / `government.*`. The Publish activity calls the
// legislation service's application layer — it never writes SQL directly.
package temporal

import (
	"context"
	"fmt"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// ---------------------------------------------------------------------------
// Workflow input / output
// ---------------------------------------------------------------------------

// BillProcessingWorkflowInput is the durable argument to the workflow.
// WorkflowID is derived from the (country_code, bill_identifier) tuple so
// duplicate runs collapse on Temporal's server-side dedup.
type BillProcessingWorkflowInput struct {
	CountryCode string    `json:"country_code"`
	BillID      string    `json:"bill_id"`        // platform bill UUID (if known) or source identifier
	SourceURL   string    `json:"source_url"`     // canonical source page discovered by the crawler
	Identifier  string    `json:"identifier"`     // e.g. "Bill No. 14 of 2024"
	Year        int       `json:"year"`
	StartedAt   time.Time `json:"started_at"`
}

// BillProcessingWorkflowOutput is the durable result. It is stored by
// Temporal alongside the workflow history and surfaced in the UI.
type BillProcessingWorkflowOutput struct {
	BillID       string    `json:"bill_id"`
	DocumentID   string    `json:"document_id"`   // raw document stored in object storage
	ParsedTitle  string    `json:"parsed_title"`
	Topics       []string  `json:"topics"`
	Stage        string    `json:"stage"`
	Validated    bool      `json:"validated"`
	PublishedAt  time.Time `json:"published_at"`
	DurationMS   int64     `json:"duration_ms"`
}

// ---------------------------------------------------------------------------
// Workflow interface (implemented by BillProcessingWorkflow.Workflow)
// ---------------------------------------------------------------------------

// BillProcessingWorkflow is the durable entrypoint. The implementation lives
// in Workflow() below; this interface exists so callers (tests, API handlers)
// can use typed client.ExecuteWorkflow(...).
type BillProcessingWorkflow interface {
	Workflow(ctx workflow.Context, in BillProcessingWorkflowInput) (BillProcessingWorkflowOutput, error)
}

// ---------------------------------------------------------------------------
// Activity signatures
// ---------------------------------------------------------------------------

// DiscoverActivity enumerates the source endpoint and returns a list of
// source items that need to be fetched. For a single-bill workflow it
// typically returns one item, but the activity is general-purpose so the
// same workflow can be reused for batch discovery.
type DiscoverActivity struct {
	Registry SourceRegistry
}

// DiscoverInput is the activity argument.
type DiscoverInput struct {
	CountryCode string
	SourceURL   string
	Identifier  string
}

// DiscoverResult is the activity output. It is serialisable JSON.
type DiscoverResult struct {
	Items []SourceItem `json:"items"`
}

// SourceItem is a normalised descriptor of one piece of content found at the
// source. It mirrors contracts.SourceItem but is duplicated to keep this
// package dependency-light (Temporal activity arguments must be JSON-able).
type SourceItem struct {
	URL         string            `json:"url"`
	Identifier  string            `json:"identifier"`
	Kind        string            `json:"kind"` // "bill", "act", "gazette", ...
	Title       string            `json:"title"`
	PublishedAt time.Time         `json:"published_at,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// FetchActivity downloads the raw bytes for one SourceItem and stores them
// in object storage (S3/MinIO via packages/storage). Returns the document ID.
type FetchActivity struct {
	Fetcher Fetcher
	Storage ObjectStorage
	IDGen   IDGenerator
}

// FetchInput is the activity argument.
type FetchInput struct {
	CountryCode string
	Item        SourceItem
	UserAgent   string
}

// FetchResult is the activity output.
type FetchResult struct {
	DocumentID  string `json:"document_id"`
	ContentHash string `json:"content_hash"` // SHA-256 hex
	Bytes       int64  `json:"bytes"`
	ContentType string `json:"content_type"`
}

// ParseActivity turns raw bytes into structured bill sections. It delegates
// to the documents service's parser pipeline.
type ParseActivity struct {
	Documents DocumentsClient
}

// ParseInput is the activity argument.
type ParseInput struct {
	DocumentID string
	ParserName string
}

// ParseResult is the activity output.
type ParseResult struct {
	Title    string         `json:"title"`
	Sections []ParsedSection `json:"sections"`
	Text     string         `json:"text"` // concatenated plaintext
}

// ParsedSection is one section of a bill extracted by the parser.
type ParsedSection struct {
	Number string `json:"number"`
	Title  string `json:"title"`
	Text   string `json:"text"`
}

// ExtractActivity calls the AI gateway to extract topics, entities, and a
// one-paragraph summary. The AI never mutates canonical state; it returns
// candidate facts which Validate may reject.
type ExtractActivity struct {
	AI AIGatewayClient
}

// ExtractInput is the activity argument.
type ExtractInput struct {
	BillID   string
	Title    string
	Text     string
	Audience string
}

// ExtractResult is the activity output.
type ExtractResult struct {
	Summary   string   `json:"summary"`
	Topics    []string `json:"topics"`
	Stage     string   `json:"stage"`      // current_stage proposed by AI
	Entities  []string `json:"entities"`
	ModelID   string   `json:"model_id"`
}

// ValidateActivity checks that every claim extracted by the AI is backed by
// at least one citation from the evidence service. Rejected claims are
// dropped; accepted claims flow through to Publish. This is the gate that
// enforces "AI proposes, evidence disposes" (ADR-0011).
type ValidateActivity struct {
	Validator CandidateFactValidator
	Evidence  EvidenceLookupClient
}

// ValidateInput is the activity argument.
type ValidateInput struct {
	BillID    string
	Summary   string
	Topics    []string
	Stage     string
	SourceURL string
}

// ValidateResult is the activity output.
type ValidateResult struct {
	Validated      bool     `json:"validated"`
	AcceptedTopics []string `json:"accepted_topics"`
	RejectedCount  int      `json:"rejected_count"`
}

// PublishActivity emits a `bill.processed` event via NATS and calls the
// legislation service's application layer to materialise the bill. This is
// the only activity allowed to write to `legislation.*`.
type PublishActivity struct {
	Publisher EventPublisher
	Legislation LegislationClient
}

// PublishInput is the activity argument.
type PublishInput struct {
	BillID      string
	CountryCode string
	Identifier  string
	Year        int
	Title       string
	Topics      []string
	Stage       string
	Summary     string
	DocumentID  string
	SourceURL   string
	Validated   bool
}

// PublishResult is the activity output.
type PublishResult struct {
	BillID      string    `json:"bill_id"`
	PublishedAt time.Time `json:"published_at"`
}

// ---------------------------------------------------------------------------
// Dependencies (interfaces — injected by the worker bootstrap)
// ---------------------------------------------------------------------------

// SourceRegistry resolves the country adapter for a given source URL.
type SourceRegistry interface {
	AdapterFor(countryCode string) (string, error) // returns adapter code
	Discover(ctx context.Context, countryCode, sourceURL, identifier string) ([]SourceItem, error)
}

// Fetcher downloads raw bytes from a URL.
type Fetcher interface {
	Fetch(ctx context.Context, url, userAgent string) (body []byte, contentType string, err error)
}

// ObjectStorage uploads raw bytes and returns a document ID/key.
type ObjectStorage interface {
	Upload(ctx context.Context, bucket, key string, body []byte, contentType string) (documentID string, err error)
}

// DocumentsClient parses raw bytes into structured sections.
type DocumentsClient interface {
	Parse(ctx context.Context, documentID, parserName string) (ParseResult, error)
}

// AIGatewayClient mirrors the intelligence service's domain interface.
type AIGatewayClient interface {
	GenerateSummary(ctx context.Context, billID, title, text, audience string) (summary string, modelID string, err error)
	Classify(ctx context.Context, billID, title, text string) (topics []string, err error)
	DetectStage(ctx context.Context, billID, title, text string) (stage string, err error)
}

// CandidateFactValidator checks proposed facts against the evidence service.
type CandidateFactValidator interface {
	Validate(ctx context.Context, billID, claim, sourceURL string) (accepted bool, err error)
}

// EvidenceLookupClient cross-checks claims against the evidence corpus.
type EvidenceLookupClient interface {
	Lookup(ctx context.Context, claimText string) (supports int, refutes int, err error)
}

// EventPublisher emits domain events onto the bus (NATS).
type EventPublisher interface {
	Publish(ctx context.Context, subject string, payload []byte) error
}

// LegislationClient is the only writer to canonical bill state.
type LegislationClient interface {
	UpsertBill(ctx context.Context, in PublishInput) (billID string, err error)
}

// IDGenerator produces string IDs (UUIDs).
type IDGenerator interface {
	New() string
}

// ---------------------------------------------------------------------------
// Workflow implementation
// ---------------------------------------------------------------------------

// BillProcessingWorkflowImpl is the concrete implementation. It holds only
// activity structs (each of which owns its own dependencies); the workflow
// function itself is deterministic.
type BillProcessingWorkflowImpl struct {
	Discover  *DiscoverActivity
	Fetch     *FetchActivity
	Parse     *ParseActivity
	Extract   *ExtractActivity
	Validate  *ValidateActivity
	Publish   *PublishActivity
}

// NewBillProcessingWorkflow constructs the workflow with all activities wired.
func NewBillProcessingWorkflow(
	discover *DiscoverActivity,
	fetch *FetchActivity,
	parse *ParseActivity,
	extract *ExtractActivity,
	validate *ValidateActivity,
	publish *PublishActivity,
) *BillProcessingWorkflowImpl {
	return &BillProcessingWorkflowImpl{
		Discover: discover, Fetch: fetch, Parse: parse,
		Extract: extract, Validate: validate, Publish: publish,
	}
}

// ActivityOptionsFor returns the standard activity options used by the
// Bill Processing Pipeline. Each activity gets retry + timeout policies that
// reflect its SLA (network calls get longer timeouts; AI gets a hard cap).
func ActivityOptionsFor(stage string) workflow.ActivityOptions {
	base := workflow.ActivityOptions{
		StartToCloseTimeout: 2 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    1 * time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    30 * time.Second,
			MaximumAttempts:    5,
			NonRetryableErrorTypes: []string{
				"ErrValidation",
				"ErrEvidenceMissing",
			},
		},
	}
	switch stage {
	case "fetch":
		base.StartToCloseTimeout = 90 * time.Second // large PDFs
	case "parse":
		base.StartToCloseTimeout = 5 * time.Minute // OCR can be slow
	case "extract":
		base.StartToCloseTimeout = 60 * time.Second // AI gateway SLA
		base.RetryPolicy.MaximumAttempts = 3
	case "validate":
		base.StartToCloseTimeout = 30 * time.Second
	case "publish":
		// Publish must not retry forever — duplicate events are worse than
		// a missed one (the run is idempotent on RunID so a replay is safe).
		base.RetryPolicy.MaximumAttempts = 10
	}
	return base
}

// Workflow is the Temporal entrypoint. It is deterministic: it only calls
// activities and uses workflow-side time (workflow.Now(ctx)).
//
// Pipeline:
//   Discover  → list source items (usually 1 for a single bill)
//   Fetch     → download + store raw bytes
//   Parse     → structured sections + plaintext
//   Extract   → AI summary, topics, stage (candidate facts)
//   Validate  → evidence-backed gate
//   Publish   → emit bill.processed event + legislation upsert
func (w *BillProcessingWorkflowImpl) Workflow(
	ctx workflow.Context,
	in BillProcessingWorkflowInput,
) (BillProcessingWorkflowOutput, error) {
	logger := workflow.GetLogger(ctx)
	workflowID := workflow.GetInfo(ctx).WorkflowExecution.ID
	logger.Info("bill processing pipeline starting",
		"workflow_id", workflowID,
		"country", in.CountryCode,
		"identifier", in.Identifier,
		"source_url", in.SourceURL)

	startedAt := workflow.Now(ctx)

	// 1. Discover
	discCtx := workflow.WithActivityOptions(ctx, ActivityOptionsFor("discover"))
	var discOut DiscoverResult
	if err := workflow.ExecuteActivity(
		discCtx, w.Discover.Discover, DiscoverInput{
			CountryCode: in.CountryCode,
			SourceURL:   in.SourceURL,
			Identifier:  in.Identifier,
		},
	).Get(discCtx, &discOut); err != nil {
		return BillProcessingWorkflowOutput{}, fmt.Errorf("discover: %w", err)
	}
	if len(discOut.Items) == 0 {
		return BillProcessingWorkflowOutput{}, temporal.NewNonRetryableApplicationError(
			"discover returned no items", "ErrNoItems", nil)
	}
	item := discOut.Items[0]

	// 2. Fetch
	fetchCtx := workflow.WithActivityOptions(ctx, ActivityOptionsFor("fetch"))
	var fetchOut FetchResult
	if err := workflow.ExecuteActivity(
		fetchCtx, w.Fetch.Fetch, FetchInput{
			CountryCode: in.CountryCode,
			Item:        item,
			UserAgent:   "civic-intelligence/ingestion",
		},
	).Get(fetchCtx, &fetchOut); err != nil {
		return BillProcessingWorkflowOutput{}, fmt.Errorf("fetch: %w", err)
	}

	// 3. Parse
	parseCtx := workflow.WithActivityOptions(ctx, ActivityOptionsFor("parse"))
	var parseOut ParseResult
	if err := workflow.ExecuteActivity(
		parseCtx, w.Parse.Parse, ParseInput{
			DocumentID: fetchOut.DocumentID,
			ParserName: "kenya_bill_v1",
		},
	).Get(parseCtx, &parseOut); err != nil {
		return BillProcessingWorkflowOutput{}, fmt.Errorf("parse: %w", err)
	}

	// 4. Extract (AI — candidate facts only)
	extractCtx := workflow.WithActivityOptions(ctx, ActivityOptionsFor("extract"))
	var extractOut ExtractResult
	if err := workflow.ExecuteActivity(
		extractCtx, w.Extract.Extract, ExtractInput{
			BillID:   in.BillID,
			Title:    parseOut.Title,
			Text:     parseOut.Text,
			Audience: "general",
		},
	).Get(extractCtx, &extractOut); err != nil {
		return BillProcessingWorkflowOutput{}, fmt.Errorf("extract: %w", err)
	}

	// 5. Validate (evidence gate — drops unsupported claims)
	validateCtx := workflow.WithActivityOptions(ctx, ActivityOptionsFor("validate"))
	var validateOut ValidateResult
	if err := workflow.ExecuteActivity(
		validateCtx, w.Validate.Validate, ValidateInput{
			BillID:    in.BillID,
			Summary:   extractOut.Summary,
			Topics:    extractOut.Topics,
			Stage:     extractOut.Stage,
			SourceURL: in.SourceURL,
		},
	).Get(validateCtx, &validateOut); err != nil {
		return BillProcessingWorkflowOutput{}, fmt.Errorf("validate: %w", err)
	}

	// 6. Publish (only writer to canonical state)
	publishCtx := workflow.WithActivityOptions(ctx, ActivityOptionsFor("publish"))
	var publishOut PublishResult
	if err := workflow.ExecuteActivity(
		publishCtx, w.Publish.Publish, PublishInput{
			BillID:      in.BillID,
			CountryCode: in.CountryCode,
			Identifier:  in.Identifier,
			Year:        in.Year,
			Title:       parseOut.Title,
			Topics:      validateOut.AcceptedTopics,
			Stage:       extractOut.Stage,
			Summary:     extractOut.Summary,
			DocumentID:  fetchOut.DocumentID,
			SourceURL:   in.SourceURL,
			Validated:   validateOut.Validated,
		},
	).Get(publishCtx, &publishOut); err != nil {
		return BillProcessingWorkflowOutput{}, fmt.Errorf("publish: %w", err)
	}

	out := BillProcessingWorkflowOutput{
		BillID:      publishOut.BillID,
		DocumentID:  fetchOut.DocumentID,
		ParsedTitle: parseOut.Title,
		Topics:      validateOut.AcceptedTopics,
		Stage:       extractOut.Stage,
		Validated:   validateOut.Validated,
		PublishedAt: publishOut.PublishedAt,
		DurationMS:  workflow.Now(ctx).Sub(startedAt).Milliseconds(),
	}
	logger.Info("bill processing pipeline complete",
		"workflow_id", workflowID,
		"bill_id", out.BillID,
		"validated", out.Validated,
		"duration_ms", out.DurationMS)
	return out, nil
}

// Compile-time assertion that the workflow implementation matches the
// interface (catches signature drift at build time, not runtime).
var _ BillProcessingWorkflow = (*BillProcessingWorkflowImpl)(nil)
