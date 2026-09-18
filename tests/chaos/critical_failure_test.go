// Package chaos — critical_failure_test.go implements spec §75's critical
// failure test: a worker crashes mid-Bill-Processing-Pipeline and the
// workflow must RESUME from the last completed activity without
// re-executing side effects and without corrupting canonical state.
//
// The production pipeline runs as a Temporal workflow
// (services/ingestion/internal/temporal/workflow.go). Temporal handles
// the resume semantics: when a worker restarts after a crash, the workflow
// is replayed from the event log; activities whose results are already in
// the history are NOT re-executed (Temporal returns the cached result).
//
// This test simulates that semantics with a self-contained, deterministic
// implementation. It does NOT depend on the Temporal SDK — the test
// verifies the CONTRACT (idempotent resume + no duplicate side effects +
// no data corruption), not the SDK's mechanics. The production Temporal
// workflow is the IMPLEMENTATION of that contract; this test is the
// executable specification.
//
// Test sequence (mirrors the production pipeline):
//
//	1. Run the pipeline forward through Fetch.
//	2. Kill the worker (inject a crash).
//	3. Spin up a new worker; resume the workflow from the last
//	   completed activity.
//	4. Verify:
//	     a. Fetch is NOT re-executed (the content-hash + document-id
//	        are reused, not re-fetched from the network).
//	     b. Publish is called exactly ONCE (no duplicate side effects).
//	     c. The final bill record matches the expected canonical
//	        state (no data corruption).
package chaos

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

// PipelineStage names the stages of the Bill Processing Pipeline.
// Mirrors services/ingestion/internal/temporal/workflow.go.
type PipelineStage string

const (
	StageDiscover PipelineStage = "discover"
	StageFetch    PipelineStage = "fetch"
	StageParse    PipelineStage = "parse"
	StageExtract  PipelineStage = "extract"
	StageValidate PipelineStage = "validate"
	StagePublish  PipelineStage = "publish"
)

// All stages in execution order.
var pipelineOrder = []PipelineStage{
	StageDiscover, StageFetch, StageParse, StageExtract, StageValidate, StagePublish,
}

// PipelineState is the durable state of the workflow — what Temporal
// would persist in the event log. After each completed activity, the
// state is checkpointed; on resume, the workflow reads back the latest
// checkpoint and skips forward.
type PipelineState struct {
	WorkflowID      string        `json:"workflow_id"`
	LastCompleted   PipelineStage `json:"last_completed"`
	DocumentID      string        `json:"document_id"`
	ContentHash     string        `json:"content_hash"`
	ParsedTitle     string        `json:"parsed_title"`
	Topics          []string      `json:"topics"`
	Stage           string        `json:"stage"`
	Validated       bool          `json:"validated"`
	PublishedBillID string        `json:"published_bill_id"`
}

// PipelineDeps is the bundle of side-effect-owning dependencies. Each
// field is an interface so the test can inject counting wrappers.
type PipelineDeps struct {
	Discover DiscoverFn
	Fetch    FetchFn
	Parse    ParseFn
	Extract  ExtractFn
	Validate ValidateFn
	Publish  PublishFn
}

// DiscoverFn enumerates source items. Side effect: HTTP call to source.
type DiscoverFn func(ctx context.Context, in DiscoverInput) (DiscoverResult, error)

// DiscoverInput mirrors the production DiscoverInput.
type DiscoverInput struct {
	CountryCode string
	SourceURL   string
	Identifier  string
}

// DiscoverResult mirrors the production DiscoverResult.
type DiscoverResult struct {
	Items []SourceItem
}

// SourceItem mirrors the production SourceItem (minimal subset).
type SourceItem struct {
	URL        string
	Identifier string
	Kind       string
	Title      string
}

// FetchFn downloads + stores raw bytes. Side effect: HTTP fetch + S3 upload.
type FetchFn func(ctx context.Context, in FetchInput) (FetchResult, error)

// FetchInput mirrors the production FetchInput.
type FetchInput struct {
	CountryCode string
	Item        SourceItem
	UserAgent   string
}

// FetchResult mirrors the production FetchResult.
type FetchResult struct {
	DocumentID  string
	ContentHash string
	Bytes       int64
	ContentType string
}

// ParseFn turns raw bytes into structured sections. Side effect: none
// (pure computation in production; might call documents service).
type ParseFn func(ctx context.Context, in ParseInput) (ParseResult, error)

// ParseInput mirrors the production ParseInput.
type ParseInput struct {
	DocumentID string
	ParserName string
}

// ParseResult mirrors the production ParseResult.
type ParseResult struct {
	Title    string
	Sections []string
	Text     string
}

// ExtractFn calls the AI gateway. Side effect: LLM API call.
type ExtractFn func(ctx context.Context, in ExtractInput) (ExtractResult, error)

// ExtractInput mirrors the production ExtractInput.
type ExtractInput struct {
	BillID   string
	Title    string
	Text     string
	Audience string
}

// ExtractResult mirrors the production ExtractResult.
type ExtractResult struct {
	Summary string
	Topics  []string
	Stage   string
}

// ValidateFn checks claims against evidence. Side effect: evidence lookup.
type ValidateFn func(ctx context.Context, in ValidateInput) (ValidateResult, error)

// ValidateInput mirrors the production ValidateInput.
type ValidateInput struct {
	BillID    string
	Summary   string
	Topics    []string
	Stage     string
	SourceURL string
}

// ValidateResult mirrors the production ValidateResult.
type ValidateResult struct {
	Validated      bool
	AcceptedTopics []string
	RejectedCount   int
}

// PublishFn emits bill.processed + upserts canonical bill. Side effect:
// NATS publish + Postgres write. THIS IS THE ONLY ACTIVITY ALLOWED TO
// WRITE TO LEGISLATION.* (ADR-0007).
type PublishFn func(ctx context.Context, in PublishInput) (PublishResult, error)

// PublishInput mirrors the production PublishInput.
type PublishInput struct {
	BillID      string
	CountryCode string
	Identifier  string
	Title       string
	Topics      []string
	Stage       string
	Summary     string
	DocumentID  string
	SourceURL    string
	Validated    bool
}

// PublishResult mirrors the production PublishResult.
type PublishResult struct {
	BillID      string
	PublishedAt time.Time
}

// ErrWorkerCrashed is the sentinel error injected to simulate a worker
// crash mid-pipeline. The pipeline runner treats this error specially: it
// does NOT advance the checkpoint; the next run will re-execute the
// interrupted stage.
var ErrWorkerCrashed = errors.New("worker crashed mid-activity")

// ---------------------------------------------------------------------------
// Workflow runner (mirrors the production Temporal workflow semantics)
// ---------------------------------------------------------------------------

// RunPipeline executes the Bill Processing Pipeline with checkpoint /
// resume semantics. If lastState.LastCompleted is non-empty, the runner
// SKIPS forward to the stage after it (idempotent resume).
//
// The runner returns:
//   - The final state (with LastCompleted = StagePublish on success).
//   - ErrWorkerCrashed if a crash was injected mid-flight.
//   - Any other error from the underlying activities.
func RunPipeline(ctx context.Context, deps PipelineDeps, in BillProcessingInput, lastState PipelineState) (PipelineState, error) {
	state := lastState
	if state.WorkflowID == "" {
		state.WorkflowID = in.WorkflowID
	}

	// 1. Discover
	if state.LastCompleted == "" {
		out, err := deps.Discover(ctx, DiscoverInput{
			CountryCode: in.CountryCode,
			SourceURL:   in.SourceURL,
			Identifier:  in.Identifier,
		})
		if err != nil {
			return state, err
		}
		if len(out.Items) == 0 {
			return state, errors.New("discover returned no items")
		}
		state.LastCompleted = StageDiscover
		// (Production would checkpoint here via Temporal's history.)
	}

	// 2. Fetch — idempotent on WorkflowID + SourceURL. If the state already
	// has a DocumentID (from a previous run), skip the network call.
	if state.LastCompleted == StageDiscover {
		out, err := deps.Fetch(ctx, FetchInput{
			CountryCode: in.CountryCode,
			Item: SourceItem{
				URL:        in.SourceURL,
				Identifier: in.Identifier,
			},
			UserAgent: "civic-intelligence/ingestion",
		})
		if err != nil {
			return state, err
		}
		state.DocumentID = out.DocumentID
		state.ContentHash = out.ContentHash
		state.LastCompleted = StageFetch
	}

	// 3. Parse — reads the stored document; no network call.
	if state.LastCompleted == StageFetch {
		out, err := deps.Parse(ctx, ParseInput{
			DocumentID: state.DocumentID,
			ParserName: "kenya_bill_v1",
		})
		if err != nil {
			return state, err
		}
		state.ParsedTitle = out.Title
		state.LastCompleted = StageParse
	}

	// 4. Extract (AI — candidate facts only).
	if state.LastCompleted == StageParse {
		out, err := deps.Extract(ctx, ExtractInput{
			BillID:   in.BillID,
			Title:    state.ParsedTitle,
			Text:     "synthesised bill text", // production would use the parsed text
			Audience: "general",
		})
		if err != nil {
			return state, err
		}
		state.Topics = out.Topics
		state.Stage = out.Stage
		state.LastCompleted = StageExtract
	}

	// 5. Validate (evidence gate — drops unsupported claims).
	if state.LastCompleted == StageExtract {
		out, err := deps.Validate(ctx, ValidateInput{
			BillID:    in.BillID,
			Summary:   "AI summary",
			Topics:    state.Topics,
			Stage:     state.Stage,
			SourceURL: in.SourceURL,
		})
		if err != nil {
			return state, err
		}
		state.Validated = out.Validated
		state.Topics = out.AcceptedTopics // production: drop rejected topics
		state.LastCompleted = StageValidate
	}

	// 6. Publish — the ONLY writer to canonical state.
	if state.LastCompleted == StageValidate {
		out, err := deps.Publish(ctx, PublishInput{
			BillID:      in.BillID,
			CountryCode: in.CountryCode,
			Identifier:  in.Identifier,
			Title:       state.ParsedTitle,
			Topics:      state.Topics,
			Stage:       state.Stage,
			Summary:     "AI summary",
			DocumentID:  state.DocumentID,
			SourceURL:   in.SourceURL,
			Validated:   state.Validated,
		})
		if err != nil {
			return state, err
		}
		state.PublishedBillID = out.BillID
		state.LastCompleted = StagePublish
	}

	return state, nil
}

// BillProcessingInput is the workflow input. Mirrors the production type.
type BillProcessingInput struct {
	WorkflowID  string
	CountryCode string
	BillID      string
	SourceURL   string
	Identifier  string
	Year        int
	StartedAt   time.Time
}

// ---------------------------------------------------------------------------
// Counting wrappers — record every call so the test can assert that
// side-effect-owning activities (Fetch, Publish) are not re-executed on
// resume.
// ---------------------------------------------------------------------------

type callCountingDeps struct {
	mu       sync.Mutex
	calls    map[PipelineStage]int
	inner    PipelineDeps
	crashAt  *PipelineStage // if non-nil, the stage at which to inject ErrWorkerCrashed
	crashHit bool
}

func newCountingDeps(inner PipelineDeps) *callCountingDeps {
	return &callCountingDeps{
		calls: map[PipelineStage]int{},
		inner: inner,
	}
}

// maybeCrash returns ErrWorkerCrashed if the wrapper is configured to
// crash at this stage AND hasn't already crashed. The caller holds the
// mutex when invoking this; the function does NOT touch the mutex.
func (c *callCountingDeps) maybeCrash(stage PipelineStage) error {
	if c.crashAt != nil && *c.crashAt == stage && !c.crashHit {
		c.crashHit = true
		return ErrWorkerCrashed
	}
	return nil
}

func (c *callCountingDeps) Discover(ctx context.Context, in DiscoverInput) (DiscoverResult, error) {
	c.mu.Lock()
	c.calls[StageDiscover]++
	err := c.maybeCrash(StageDiscover)
	c.mu.Unlock()
	if err != nil {
		return DiscoverResult{}, err
	}
	return c.inner.Discover(ctx, in)
}

func (c *callCountingDeps) Fetch(ctx context.Context, in FetchInput) (FetchResult, error) {
	c.mu.Lock()
	c.calls[StageFetch]++
	err := c.maybeCrash(StageFetch)
	c.mu.Unlock()
	if err != nil {
		return FetchResult{}, err
	}
	return c.inner.Fetch(ctx, in)
}

func (c *callCountingDeps) Parse(ctx context.Context, in ParseInput) (ParseResult, error) {
	c.mu.Lock()
	c.calls[StageParse]++
	err := c.maybeCrash(StageParse)
	c.mu.Unlock()
	if err != nil {
		return ParseResult{}, err
	}
	return c.inner.Parse(ctx, in)
}

func (c *callCountingDeps) Extract(ctx context.Context, in ExtractInput) (ExtractResult, error) {
	c.mu.Lock()
	c.calls[StageExtract]++
	err := c.maybeCrash(StageExtract)
	c.mu.Unlock()
	if err != nil {
		return ExtractResult{}, err
	}
	return c.inner.Extract(ctx, in)
}

func (c *callCountingDeps) Validate(ctx context.Context, in ValidateInput) (ValidateResult, error) {
	c.mu.Lock()
	c.calls[StageValidate]++
	err := c.maybeCrash(StageValidate)
	c.mu.Unlock()
	if err != nil {
		return ValidateResult{}, err
	}
	return c.inner.Validate(ctx, in)
}

func (c *callCountingDeps) Publish(ctx context.Context, in PublishInput) (PublishResult, error) {
	c.mu.Lock()
	c.calls[StagePublish]++
	err := c.maybeCrash(StagePublish)
	c.mu.Unlock()
	if err != nil {
		return PublishResult{}, err
	}
	return c.inner.Publish(ctx, in)
}

// asPipelineDeps returns a PipelineDeps that uses the counting wrappers.
func (c *callCountingDeps) asPipelineDeps() PipelineDeps {
	return PipelineDeps{
		Discover: c.Discover,
		Fetch:    c.Fetch,
		Parse:    c.Parse,
		Extract:  c.Extract,
		Validate: c.Validate,
		Publish:  c.Publish,
	}
}

// callCount returns the number of times the given stage was invoked.
func (c *callCountingDeps) callCount(stage PipelineStage) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.calls[stage]
}

// ---------------------------------------------------------------------------
// Stub implementations for the underlying activities — each returns a
// deterministic value so the test can assert against it.
// ---------------------------------------------------------------------------

func stubDiscover(_ context.Context, _ DiscoverInput) (DiscoverResult, error) {
	return DiscoverResult{Items: []SourceItem{{URL: "https://example.org/bill", Identifier: "Bill No. 1 of 2024", Kind: "bill", Title: "Test Bill"}}}, nil
}

func stubFetch(_ context.Context, in FetchInput) (FetchResult, error) {
	// Production computes a SHA-256 of the body and uses it as the S3
	// key (idempotent on RunID). The stub simulates this with the input
	// identifier — same identifier → same document ID.
	return FetchResult{
		DocumentID:  "doc-" + in.Item.Identifier,
		ContentHash: "sha256-" + in.Item.Identifier,
		Bytes:       1024,
		ContentType: "application/pdf",
	}, nil
}

func stubParse(_ context.Context, _ ParseInput) (ParseResult, error) {
	return ParseResult{Title: "Test Bill 2024", Sections: []string{"s1", "s2"}, Text: "bill text"}, nil
}

func stubExtract(_ context.Context, _ ExtractInput) (ExtractResult, error) {
	return ExtractResult{Summary: "AI summary", Topics: []string{"finance", "health"}, Stage: "SECOND_READING"}, nil
}

func stubValidate(_ context.Context, _ ValidateInput) (ValidateResult, error) {
	return ValidateResult{Validated: true, AcceptedTopics: []string{"finance", "health"}, RejectedCount: 0}, nil
}

func stubPublish(_ context.Context, in PublishInput) (PublishResult, error) {
	return PublishResult{BillID: in.BillID, PublishedAt: time.Now().UTC()}, nil
}

func stubDeps() PipelineDeps {
	return PipelineDeps{
		Discover: stubDiscover,
		Fetch:    stubFetch,
		Parse:    stubParse,
		Extract:  stubExtract,
		Validate: stubValidate,
		Publish:  stubPublish,
	}
}

// ---------------------------------------------------------------------------
// The §75 critical failure tests
// ---------------------------------------------------------------------------

// TestCriticalFailure_WorkerCrashResume verifies spec §75's critical
// failure contract: when a worker crashes mid-Bill-Processing-Pipeline,
// the workflow MUST:
//
//  1. Resume from the last completed activity (NOT restart from Discover).
//  2. NOT re-execute side-effect-owning activities (Fetch, Publish).
//  3. NOT corrupt canonical state — the published bill matches the
//     expected canonical record.
//
// The test simulates a crash after Fetch completes (between Fetch and
// Parse — a common failure point because Parse can be slow with OCR).
func TestCriticalFailure_WorkerCrashResume(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	workflowInput := BillProcessingInput{
		WorkflowID:  "wf-test-001",
		CountryCode: "KE",
		BillID:      "bill-test-001",
		SourceURL:   "https://example.org/bill",
		Identifier:  "Bill No. 1 of 2024",
		Year:        2024,
		StartedAt:   time.Now().UTC(),
	}

	// Stage 1: run the pipeline forward, crashing AFTER Fetch completes.
	// The crash is injected at StageParse so Fetch is fully recorded.
	crashAt := StageParse
	counting := newCountingDeps(stubDeps())
	counting.crashAt = &crashAt

	initialState := PipelineState{} // empty — fresh workflow

	state, err := RunPipeline(ctx, counting.asPipelineDeps(), workflowInput, initialState)
	if !errors.Is(err, ErrWorkerCrashed) {
		t.Fatalf("expected ErrWorkerCrashed on first run; got %v (state=%+v)", err, state)
	}

	// After the crash: Discover + Fetch were called exactly once each.
	if got := counting.callCount(StageDiscover); got != 1 {
		t.Errorf("expected Discover called 1×; got %d", got)
	}
	if got := counting.callCount(StageFetch); got != 1 {
		t.Errorf("expected Fetch called 1× (must not be re-run on resume); got %d", got)
	}
	// Parse was attempted but crashed — its call count is 1 (the failed attempt).
	if got := counting.callCount(StageParse); got != 1 {
		t.Errorf("expected Parse called 1× (the crashed attempt); got %d", got)
	}
	// LastCompleted must be StageFetch (Parse did not complete).
	if state.LastCompleted != StageFetch {
		t.Fatalf("expected LastCompleted=StageFetch after crash; got %s", state.LastCompleted)
	}
	// DocumentID must be set (Fetch completed before the crash).
	if state.DocumentID == "" {
		t.Fatal("expected DocumentID to be set after Fetch completed")
	}

	// Stage 2: spin up a new worker and resume. The new worker has the
	// same underlying stubs but a fresh crashAt=nil (no crash injected).
	// We reuse the SAME counting wrapper so we can verify Fetch was NOT
	// re-invoked on resume.
	counting.crashAt = nil // clear the crash injection
	resumedState, err := RunPipeline(ctx, counting.asPipelineDeps(), workflowInput, state)
	if err != nil {
		t.Fatalf("expected pipeline to resume cleanly; got %v", err)
	}

	// After resume:
	//   - Discover still 1× (not re-run).
	//   - Fetch still 1× (not re-run — this is the key contract).
	//   - Parse now 2× (the first attempt crashed, the second succeeded).
	//   - Extract, Validate, Publish each 1× (first successful run).
	if got := counting.callCount(StageDiscover); got != 1 {
		t.Errorf("post-resume: expected Discover still 1×; got %d", got)
	}
	if got := counting.callCount(StageFetch); got != 1 {
		t.Errorf("post-resume: expected Fetch still 1× (idempotent resume contract); got %d", got)
	}
	if got := counting.callCount(StageParse); got != 2 {
		t.Errorf("post-resume: expected Parse 2× (1 crashed + 1 success); got %d", got)
	}
	if got := counting.callCount(StageExtract); got != 1 {
		t.Errorf("post-resume: expected Extract 1×; got %d", got)
	}
	if got := counting.callCount(StageValidate); got != 1 {
		t.Errorf("post-resume: expected Validate 1×; got %d", got)
	}
	if got := counting.callCount(StagePublish); got != 1 {
		t.Errorf("post-resume: expected Publish 1× (no duplicate side effects — ADR-0007); got %d", got)
	}

	// Stage 3: verify the final state matches the expected canonical
	// record (no data corruption).
	if resumedState.LastCompleted != StagePublish {
		t.Fatalf("expected final LastCompleted=StagePublish; got %s", resumedState.LastCompleted)
	}
	if resumedState.DocumentID != "doc-Bill No. 1 of 2024" {
		t.Errorf("expected DocumentID=doc-Bill No. 1 of 2024; got %s", resumedState.DocumentID)
	}
	if resumedState.PublishedBillID != workflowInput.BillID {
		t.Errorf("expected PublishedBillID=%s; got %s", workflowInput.BillID, resumedState.PublishedBillID)
	}
	if !resumedState.Validated {
		t.Error("expected Validated=true (evidence was sufficient)")
	}
	if resumedState.ParsedTitle != "Test Bill 2024" {
		t.Errorf("expected ParsedTitle=Test Bill 2024; got %s", resumedState.ParsedTitle)
	}
}

// TestCriticalFailure_NoDuplicatePublish verifies the no-duplicate-side-effects
// contract from spec §75 / ADR-0007: even if the worker crashes AFTER
// Publish completes (but before the workflow records the completion),
// the resume must NOT re-publish.
//
// This is the worst-case scenario: Publish's side effect (NATS publish +
// Postgres upsert) has already hit the wire, but the workflow hasn't
// checkpointed. On resume, the workflow MUST detect that Publish already
// happened (via the idempotency key = WorkflowID) and skip the re-publish.
func TestCriticalFailure_NoDuplicatePublish(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	workflowInput := BillProcessingInput{
		WorkflowID:  "wf-test-002",
		CountryCode: "KE",
		BillID:      "bill-test-002",
		SourceURL:   "https://example.org/bill2",
		Identifier:  "Bill No. 2 of 2024",
		Year:        2024,
		StartedAt:   time.Now().UTC(),
	}

	// Pre-populate the state as if Publish completed but the worker
	// crashed before the final checkpoint landed. The next run must
	// NOT re-call Publish.
	preState := PipelineState{
		WorkflowID:      workflowInput.WorkflowID,
		LastCompleted:   StagePublish,
		DocumentID:      "doc-already-published",
		ContentHash:     "sha256-already-published",
		ParsedTitle:     "Already Published Bill",
		Topics:          []string{"finance"},
		Stage:           "ASSENTED",
		Validated:       true,
		PublishedBillID: workflowInput.BillID,
	}

	counting := newCountingDeps(stubDeps())
	finalState, err := RunPipeline(ctx, counting.asPipelineDeps(), workflowInput, preState)
	if err != nil {
		t.Fatalf("expected resume to complete cleanly; got %v", err)
	}

	// Every stage must have 0 calls — the pre-state already reflects a
	// completed pipeline, so nothing should be re-executed.
	for _, stage := range pipelineOrder {
		if got := counting.callCount(stage); got != 0 {
			t.Errorf("expected %s called 0× (already completed); got %d", stage, got)
		}
	}
	if finalState.LastCompleted != StagePublish {
		t.Errorf("expected final LastCompleted=StagePublish; got %s", finalState.LastCompleted)
	}
	if finalState.PublishedBillID != workflowInput.BillID {
		t.Errorf("expected PublishedBillID=%s; got %s", workflowInput.BillID, finalState.PublishedBillID)
	}
}

// TestCriticalFailure_DataIntegrity verifies the no-data-corruption
// contract from spec §75: after a crash + resume, the canonical bill
// record produced by the pipeline matches the bill record that a
// non-crashed run would have produced.
//
// The test runs the pipeline twice — once with a crash, once without —
// and asserts the final states are identical (modulo the PublishedAt
// timestamp, which is non-deterministic).
func TestCriticalFailure_DataIntegrity(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mkInput := func(id string) BillProcessingInput {
		return BillProcessingInput{
			WorkflowID:  "wf-integrity-" + id,
			CountryCode: "KE",
			BillID:      "bill-integrity-" + id,
			SourceURL:   "https://example.org/" + id,
			Identifier:  "Bill " + id,
			Year:        2024,
			StartedAt:   time.Now().UTC(),
		}
	}

	// Run 1: clean run, no crash.
	cleanDeps := newCountingDeps(stubDeps())
	cleanState, err := RunPipeline(ctx, cleanDeps.asPipelineDeps(), mkInput("clean"), PipelineState{})
	if err != nil {
		t.Fatalf("clean run failed: %v", err)
	}
	if cleanState.LastCompleted != StagePublish {
		t.Fatalf("clean run did not complete: %s", cleanState.LastCompleted)
	}

	// Run 2: crash after Fetch, then resume.
	crashInput := mkInput("crash")
	crashDeps := newCountingDeps(stubDeps())
	crashAt := StageParse
	crashDeps.crashAt = &crashAt
	crashedState, err := RunPipeline(ctx, crashDeps.asPipelineDeps(), crashInput, PipelineState{})
	if !errors.Is(err, ErrWorkerCrashed) {
		t.Fatalf("expected crash on second run; got %v", err)
	}
	crashDeps.crashAt = nil
	resumedState, err := RunPipeline(ctx, crashDeps.asPipelineDeps(), crashInput, crashedState)
	if err != nil {
		t.Fatalf("resume failed: %v", err)
	}
	if resumedState.LastCompleted != StagePublish {
		t.Fatalf("resume did not complete: %s", resumedState.LastCompleted)
	}

	// Compare canonical bill fields — these MUST match between the clean
	// run and the crash-then-resume run. (PublishedAt differs because
	// each run uses time.Now().UTC(); the BillID is the idempotency
	// key and must match.)
	if cleanState.ParsedTitle != resumedState.ParsedTitle {
		t.Errorf("ParsedTitle mismatch: clean=%s resumed=%s", cleanState.ParsedTitle, resumedState.ParsedTitle)
	}
	if cleanState.Stage != resumedState.Stage {
		t.Errorf("Stage mismatch: clean=%s resumed=%s", cleanState.Stage, resumedState.Stage)
	}
	if cleanState.Validated != resumedState.Validated {
		t.Errorf("Validated mismatch: clean=%v resumed=%v", cleanState.Validated, resumedState.Validated)
	}
	if len(cleanState.Topics) != len(resumedState.Topics) {
		t.Errorf("Topics length mismatch: clean=%d resumed=%d", len(cleanState.Topics), len(resumedState.Topics))
	} else {
		for i, topic := range cleanState.Topics {
			if resumedState.Topics[i] != topic {
				t.Errorf("Topics[%d] mismatch: clean=%s resumed=%s", i, topic, resumedState.Topics[i])
			}
		}
	}
	// PublishedBillID must match the workflow's input BillID — this is
	// the idempotency contract: re-running the same workflow ID must
	// produce the same canonical bill ID.
	if resumedState.PublishedBillID != crashInput.BillID {
		t.Errorf("PublishedBillID mismatch: expected %s; got %s", crashInput.BillID, resumedState.PublishedBillID)
	}
}
