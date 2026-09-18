// Package application contains the evidence service's use cases.
package application

import (
	"context"
	"fmt"
	"time"

	"github.com/Roy-Wanyoike/civic-intelligence/packages/contracts"
	"github.com/Roy-Wanyoike/civic-intelligence/services/evidence/internal/domain"
)

// AttachClaimHandler attaches citations to a claim, persists them, and
// publishes an evidence.attached event.
type AttachClaimHandler struct {
	svc      domain.EvidenceService
	publisher domain.EventPublisher
	clock    domain.Clock
	idGen    IDGenerator
}

// IDGenerator generates string IDs.
type IDGenerator interface {
	New() string
}

// NewAttachClaimHandler constructs the handler.
func NewAttachClaimHandler(svc domain.EvidenceService, publisher domain.EventPublisher, clock domain.Clock, idGen IDGenerator) *AttachClaimHandler {
	return &AttachClaimHandler{svc: svc, publisher: publisher, clock: clock, idGen: idGen}
}

// AttachClaimCommand is the input.
type AttachClaimCommand struct {
	Claim     domain.Claim
	Citations []domain.Citation
}

// Handle executes the command.
func (h *AttachClaimHandler) Handle(ctx context.Context, cmd AttachClaimCommand) (domain.Evidence, error) {
	ev, err := h.svc.AttachClaim(ctx, cmd.Claim, cmd.Citations)
	if err != nil {
		return ev, err
	}
	now := h.clock.Now()
	_ = h.publisher.Publish(ctx, contracts.Event{
		ID:         h.idGen.New(),
		Type:       contracts.EventEvidenceAttached,
		Source:     "evidence",
		Subject:    "civic.evidence.attached",
		OccurredAt: now,
		Data: map[string]any{
			"claim_id":  ev.ClaimID,
			"supports":  ev.Supports,
			"refutes":   ev.Refutes,
		},
	})
	return ev, nil
}

// ValidateAIResponseHandler runs the citation validator over an AI
// response. If any claim is missing evidence, it publishes an
// ai.validation.failed event so the intelligence service can record the
// failure.
type ValidateAIResponseHandler struct {
	validator domain.CitationValidator
	detector  domain.SourceConflictDetector
	conflicts domain.SourceConflictRepository
	publisher domain.EventPublisher
	clock     domain.Clock
	idGen     IDGenerator
}

// NewValidateAIResponseHandler constructs the handler.
func NewValidateAIResponseHandler(
	validator domain.CitationValidator,
	detector domain.SourceConflictDetector,
	conflicts domain.SourceConflictRepository,
	publisher domain.EventPublisher,
	clock domain.Clock,
	idGen IDGenerator,
) *ValidateAIResponseHandler {
	return &ValidateAIResponseHandler{
		validator: validator, detector: detector, conflicts: conflicts,
		publisher: publisher, clock: clock, idGen: idGen,
	}
}

// Handle executes the validation. It returns the EvidenceSet and the list
// of claim IDs that lacked supporting evidence.
func (h *ValidateAIResponseHandler) Handle(ctx context.Context, req domain.ValidationRequest) (domain.EvidenceSet, []string, error) {
	set, missing, err := h.validator.Validate(ctx, req)
	if err != nil {
		return set, missing, err
	}
	// Detect conflicts for each item.
	for i := range set.Items {
		confs, err := h.detector.Detect(ctx, set.Items[i])
		if err != nil {
			return set, missing, err
		}
		set.Items[i].Conflicts = confs
	}
	if len(missing) > 0 {
		now := h.clock.Now()
		_ = h.publisher.Publish(ctx, contracts.Event{
			ID:         h.idGen.New(),
			Type:       contracts.EventAIValidationFailed,
			Source:     "evidence",
			Subject:    "civic.ai.validation.failed",
			OccurredAt: now,
			Data: map[string]any{
				"request_id":     req.RequestID,
				"bill_id":        req.BillID,
				"missing_claims": missing,
				"reason":         "claims lack supporting citations",
			},
		})
	}
	return set, missing, nil
}

// ResolveConflictHandler allows a human to resolve a SourceConflict. The
// handler records the resolution; it never auto-resolves from AI input.
type ResolveConflictHandler struct {
	repo domain.SourceConflictRepository
}

// NewResolveConflictHandler constructs the handler.
func NewResolveConflictHandler(repo domain.SourceConflictRepository) *ResolveConflictHandler {
	return &ResolveConflictHandler{repo: repo}
}

// ResolveConflictCommand is the input.
type ResolveConflictCommand struct {
	ConflictID string
	Resolution domain.ConflictResolution
	ResolvedBy string
}

// Handle executes the command.
func (h *ResolveConflictHandler) Handle(ctx context.Context, cmd ResolveConflictCommand) error {
	if cmd.ConflictID == "" {
		return contracts.ErrValidation{Kind: "conflict", Field: "id", Reason: "must not be empty"}
	}
	now := time.Now().UTC()
	c := domain.SourceConflict{
		ID:         cmd.ConflictID,
		Resolution: cmd.Resolution,
		ResolvedBy: cmd.ResolvedBy,
		DetectedAt: now,
	}
	if cmd.Resolution != domain.ConflictOpen {
		c.ResolvedAt = &now
	}
	if err := h.repo.Save(ctx, c); err != nil {
		return fmt.Errorf("save conflict: %w", err)
	}
	return nil
}
