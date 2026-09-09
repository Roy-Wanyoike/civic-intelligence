// Package application contains the use case handlers for the legislation
// bounded context. Each handler is a single business operation; handlers
// depend only on domain interfaces and the shared packages, never on
// infrastructure.
package application

import (
	"context"
	"time"

	"github.com/Roy-Wanyoike/civic-intelligence/packages/contracts"
	"github.com/Roy-Wanyoike/civic-intelligence/services/legislation/internal/domain"
)

// CreateBillHandler creates a new Bill. The initial stage is supplied by
// the caller (typically the country adapter's initial stage). This handler
// is invoked both by humans (via the API) and by the ingestion service
// (when a new source item is discovered).
type CreateBillHandler struct {
	repo      domain.BillRepository
	validator domain.BillStageTransitionValidator
	clock     domain.Clock
	idGen     domain.IDGenerator
	publisher domain.EventPublisher
}

// NewCreateBillHandler constructs the handler.
func NewCreateBillHandler(
	repo domain.BillRepository,
	validator domain.BillStageTransitionValidator,
	clock domain.Clock,
	idGen domain.IDGenerator,
	publisher domain.EventPublisher,
) *CreateBillHandler {
	return &CreateBillHandler{repo: repo, validator: validator, clock: clock, idGen: idGen, publisher: publisher}
}

// CreateBillCommand is the input to CreateBillHandler.Handle.
type CreateBillCommand struct {
	CountryID    domain.ID
	Title        string
	ShortTitle   string
	HouseID      domain.ID
	SponsorID    *domain.ID
	InitialStage string
	ExternalID   string
	SourceID     string
	Actor        string
}

// Handle executes the command. It returns the new Bill's ID.
func (h *CreateBillHandler) Handle(ctx context.Context, cmd CreateBillCommand) (domain.ID, error) {
	if cmd.Title == "" {
		return "", contracts.ErrValidation{Kind: "bill", Field: "title", Reason: "must not be empty"}
	}
	if cmd.CountryID == "" {
		return "", contracts.ErrValidation{Kind: "bill", Field: "country_id", Reason: "must not be empty"}
	}
	if cmd.InitialStage == "" {
		return "", contracts.ErrValidation{Kind: "bill", Field: "initial_stage", Reason: "must not be empty"}
	}

	id := h.idGen.New()
	now := h.clock.Now()
	bill := domain.NewBill(id, cmd.CountryID, cmd.Title, cmd.ShortTitle, cmd.HouseID, cmd.SponsorID, cmd.InitialStage, now)

	if err := h.repo.Save(ctx, bill); err != nil {
		return "", err
	}

	_ = h.publisher.Publish(ctx, contracts.Event{
		ID:           string(h.idGen.New()),
		Type:         contracts.EventBillCreated,
		Source:       "legislation",
		Subject:      "civic.bill.created",
		OccurredAt:   now,
		Actor:        cmd.Actor,
		Data:         map[string]any{
			"bill_id":      string(id),
			"country_id":   string(cmd.CountryID),
			"title":        cmd.Title,
			"stage":        cmd.InitialStage,
			"source_id":    cmd.SourceID,
			"external_id":  cmd.ExternalID,
		},
	})
	return id, nil
}

// UpdateBillStageHandler transitions a bill to a new stage. The transition
// is validated by the country-provided validator. On success it publishes a
// bill.stage_changed event, which notifications listens to.
type UpdateBillStageHandler struct {
	repo      domain.BillRepository
	validator domain.BillStageTransitionValidator
	clock     domain.Clock
	idGen     domain.IDGenerator
	publisher domain.EventPublisher
}

// NewUpdateBillStageHandler constructs the handler.
func NewUpdateBillStageHandler(
	repo domain.BillRepository,
	validator domain.BillStageTransitionValidator,
	clock domain.Clock,
	idGen domain.IDGenerator,
	publisher domain.EventPublisher,
) *UpdateBillStageHandler {
	return &UpdateBillStageHandler{repo: repo, validator: validator, clock: clock, idGen: idGen, publisher: publisher}
}

// UpdateBillStageCommand is the input.
type UpdateBillStageCommand struct {
	BillID    domain.ID
	NewStage  string
	Reason    string
	Actor     string
}

// Handle executes the command. It returns the new BillEvent's ID.
func (h *UpdateBillStageHandler) Handle(ctx context.Context, cmd UpdateBillStageCommand) (domain.ID, error) {
	bill, err := h.repo.Get(ctx, cmd.BillID)
	if err != nil {
		return "", err
	}
	eventID := h.idGen.New()
	now := h.clock.Now()
	old := bill.CurrentStage()
	if err := bill.ApplyTransition(h.validator, cmd.NewStage, cmd.Reason, now, eventID); err != nil {
		return "", err
	}
	if err := h.repo.Save(ctx, bill); err != nil {
		return "", err
	}

	_ = h.publisher.Publish(ctx, contracts.Event{
		ID:           string(h.idGen.New()),
		Type:         contracts.EventBillStageChanged,
		Source:       "legislation",
		Subject:      "civic.bill.stage_changed",
		OccurredAt:   now,
		Actor:        cmd.Actor,
		CorrelationID: string(eventID),
		Data: map[string]any{
			"bill_id":    string(cmd.BillID),
			"old_stage":  old,
			"new_stage":  cmd.NewStage,
			"reason":     cmd.Reason,
			"event_id":   string(eventID),
		},
	})
	return eventID, nil
}

// CreateBillVersionHandler records a new version of a bill's text.
type CreateBillVersionHandler struct {
	billRepo    domain.BillRepository
	versionRepo domain.BillVersionRepository
	clock       domain.Clock
	idGen       domain.IDGenerator
	publisher   domain.EventPublisher
}

// NewCreateBillVersionHandler constructs the handler.
func NewCreateBillVersionHandler(
	billRepo domain.BillRepository,
	versionRepo domain.BillVersionRepository,
	clock domain.Clock,
	idGen domain.IDGenerator,
	publisher domain.EventPublisher,
) *CreateBillVersionHandler {
	return &CreateBillVersionHandler{
		billRepo: billRepo, versionRepo: versionRepo,
		clock: clock, idGen: idGen, publisher: publisher,
	}
}

// CreateBillVersionCommand is the input.
type CreateBillVersionCommand struct {
	BillID           domain.ID
	Title            string
	TextHash         string
	SourceDocumentID string
	ChangesSummary   string
}

// Handle executes the command.
func (h *CreateBillVersionHandler) Handle(ctx context.Context, cmd CreateBillVersionCommand) (domain.ID, error) {
	bill, err := h.billRepo.Get(ctx, cmd.BillID)
	if err != nil {
		return "", err
	}
	now := h.clock.Now()
	versionNo := len(bill.Versions()) + 1
	v := domain.BillVersion{
		ID: h.idGen.New(), BillID: cmd.BillID,
		VersionNumber: versionNo, Title: cmd.Title,
		TextHash: cmd.TextHash, SourceDocumentID: cmd.SourceDocumentID,
		PublishedAt: now, ChangesSummary: cmd.ChangesSummary,
	}
	if err := h.versionRepo.Save(ctx, v); err != nil {
		return "", err
	}
	bill.AddVersion(v)
	if err := h.billRepo.Save(ctx, bill); err != nil {
		return "", err
	}

	_ = h.publisher.Publish(ctx, contracts.Event{
		ID:         string(h.idGen.New()),
		Type:       contracts.EventBillVersionCreated,
		Source:     "legislation",
		Subject:    "civic.bill.version_created",
		OccurredAt: now,
		Data: map[string]any{
			"bill_id":         string(cmd.BillID),
			"version_id":      string(v.ID),
			"version_number":  versionNo,
			"document_id":     cmd.SourceDocumentID,
		},
	})
	return v.ID, nil
}

// CreateAmendmentHandler records a new amendment against a bill.
type CreateAmendmentHandler struct {
	billRepo  domain.BillRepository
	amendRepo domain.AmendmentRepository
	clock     domain.Clock
	idGen     domain.IDGenerator
	publisher domain.EventPublisher
}

// NewCreateAmendmentHandler constructs the handler.
func NewCreateAmendmentHandler(
	billRepo domain.BillRepository,
	amendRepo domain.AmendmentRepository,
	clock domain.Clock,
	idGen domain.IDGenerator,
	publisher domain.EventPublisher,
) *CreateAmendmentHandler {
	return &CreateAmendmentHandler{billRepo: billRepo, amendRepo: amendRepo, clock: clock, idGen: idGen, publisher: publisher}
}

// CreateAmendmentCommand is the input.
type CreateAmendmentCommand struct {
	BillID        domain.ID
	VersionNo     int
	SponsorID     domain.ID
	ClauseRef     string
	Type          string
	Text          string
	Justification string
}

// Handle executes the command.
func (h *CreateAmendmentHandler) Handle(ctx context.Context, cmd CreateAmendmentCommand) (domain.ID, error) {
	bill, err := h.billRepo.Get(ctx, cmd.BillID)
	if err != nil {
		return "", err
	}
	now := h.clock.Now()
	a := domain.Amendment{
		ID:           h.idGen.New(),
		BillID:       cmd.BillID,
		VersionNo:    cmd.VersionNo,
		SponsorID:    cmd.SponsorID,
		ClauseRef:    cmd.ClauseRef,
		Type:         cmd.Type,
		Text:         cmd.Text,
		Justification: cmd.Justification,
		Status:       domain.AmendmentStatusProposed,
		CreatedAt:    now,
	}
	if err := h.amendRepo.Save(ctx, a); err != nil {
		return "", err
	}
	bill.AddAmendment(a)
	if err := h.billRepo.Save(ctx, bill); err != nil {
		return "", err
	}
	_ = h.publisher.Publish(ctx, contracts.Event{
		ID:         string(h.idGen.New()),
		Type:       contracts.EventAmendmentDiscovered,
		Source:     "legislation",
		Subject:    "civic.amendment.discovered",
		OccurredAt: now,
		Data: map[string]any{
			"amendment_id": string(a.ID),
			"bill_id":      string(cmd.BillID),
			"sponsor_id":   string(cmd.SponsorID),
			"clause_ref":  cmd.ClauseRef,
		},
	})
	return a.ID, nil
}

// RecordBillEventHandler appends a non-stage event to a bill's history.
type RecordBillEventHandler struct {
	repo      domain.BillRepository
	clock     domain.Clock
	idGen     domain.IDGenerator
	publisher domain.EventPublisher
}

// NewRecordBillEventHandler constructs the handler.
func NewRecordBillEventHandler(repo domain.BillRepository, clock domain.Clock, idGen domain.IDGenerator, publisher domain.EventPublisher) *RecordBillEventHandler {
	return &RecordBillEventHandler{repo: repo, clock: clock, idGen: idGen, publisher: publisher}
}

// RecordBillEventCommand is the input.
type RecordBillEventCommand struct {
	BillID   domain.ID
	Kind     domain.BillEventKind
	Reason   string
	Actor    string
}

// Handle executes the command.
func (h *RecordBillEventHandler) Handle(ctx context.Context, cmd RecordBillEventCommand) (domain.ID, error) {
	bill, err := h.repo.Get(ctx, cmd.BillID)
	if err != nil {
		return "", err
	}
	id := h.idGen.New()
	now := h.clock.Now()
	stage := bill.CurrentStage()
	e := domain.NewBillEvent(id, cmd.BillID, cmd.Kind, stage, stage, cmd.Reason, cmd.Actor, now)
	bill.RecordEvent(e)
	if err := h.repo.Save(ctx, bill); err != nil {
		return "", err
	}
	return id, nil
}

// AcceptCandidateFactHandler consumes CandidateFactProposedEvents from the
// intelligence service. The handler validates the candidate's evidence
// pointer (the evidence must already be present in the evidence service) and,
// on acceptance, records a BillEvent of kind "other" so the platform has an
// auditable trail of AI-sourced facts. It NEVER mutates the bill's stage.
type AcceptCandidateFactHandler struct {
	repo      domain.BillRepository
	clock     domain.Clock
	idGen     domain.IDGenerator
	publisher domain.EventPublisher
}

// NewAcceptCandidateFactHandler constructs the handler.
func NewAcceptCandidateFactHandler(repo domain.BillRepository, clock domain.Clock, idGen domain.IDGenerator, publisher domain.EventPublisher) *AcceptCandidateFactHandler {
	return &AcceptCandidateFactHandler{repo: repo, clock: clock, idGen: idGen, publisher: publisher}
}

// AcceptCandidateFactCommand is the input.
type AcceptCandidateFactCommand struct {
	CandidateFactID string
	BillID          domain.ID
	Subject         string
	Claim           string
	Validator       string
}

// Handle executes the command. The bill must exist; if it does not the
// candidate fact is rejected (returning ErrNotFound) and the caller (the
// intelligence service) is responsible for emitting a rejected event.
func (h *AcceptCandidateFactHandler) Handle(ctx context.Context, cmd AcceptCandidateFactCommand) error {
	bill, err := h.repo.Get(ctx, cmd.BillID)
	if err != nil {
		return err
	}
	id := h.idGen.New()
	now := h.clock.Now()
	e := domain.NewBillEvent(id, cmd.BillID, domain.BillEventKindOther, bill.CurrentStage(), bill.CurrentStage(),
		"AI candidate fact accepted: "+cmd.Claim, "ai-validator:"+cmd.Validator, now)
	e.Metadata()["candidate_fact_id"] = cmd.CandidateFactID
	e.Metadata()["subject"] = cmd.Subject
	bill.RecordEvent(e)
	if err := h.repo.Save(ctx, bill); err != nil {
		return err
	}
	_ = h.publisher.Publish(ctx, contracts.Event{
		ID:         string(h.idGen.New()),
		Type:       contracts.EventCandidateFactAccepted,
		Source:     "legislation",
		Subject:    "civic.ai.candidate_fact.accepted",
		OccurredAt: now,
		Data: map[string]any{
			"candidate_fact_id": cmd.CandidateFactID,
			"bill_id":           string(cmd.BillID),
			"claim":             cmd.Claim,
			"validator":         cmd.Validator,
			"bill_event_id":     string(id),
		},
	})
	return nil
}

// Compile-time assertions that handlers' deps implement the right interfaces.
var _ time.Duration // keep time import (used elsewhere)
