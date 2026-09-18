// Package postgres contains the evidence service's Postgres repositories.
package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/Roy-Wanyoike/civic-intelligence/packages/contracts"
	"github.com/Roy-Wanyoike/civic-intelligence/services/evidence/internal/domain"
)

type querier interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

// ClaimRepository implements domain.ClaimRepository against Postgres.
type ClaimRepository struct{ db querier }

// NewClaimRepository constructs the repository.
func NewClaimRepository(db querier) *ClaimRepository { return &ClaimRepository{db: db} }

// Save implements domain.ClaimRepository.
func (r *ClaimRepository) Save(ctx context.Context, c domain.Claim) error {
	_, err := r.db.ExecContext(ctx, `
INSERT INTO evidence.claims
  (id, subject, subject_type, text, source, confidence, created_at)
VALUES ($1,$2,$3,$4,$5,$6,$7)
ON CONFLICT (id) DO UPDATE SET
   text = EXCLUDED.text,
   confidence = EXCLUDED.confidence
`, c.ID, c.Subject, c.SubjectType, c.Text, string(c.Source), c.Confidence, c.CreatedAt)
	return mapErr(err)
}

// Get implements domain.ClaimRepository.
func (r *ClaimRepository) Get(ctx context.Context, id string) (*domain.Claim, error) {
	row := r.db.QueryRowContext(ctx, `
SELECT id, subject, subject_type, text, source, confidence, created_at
  FROM evidence.claims WHERE id=$1`, id)
	var (
		c          domain.Claim
		sourceStr  string
	)
	if err := row.Scan(&c.ID, &c.Subject, &c.SubjectType, &c.Text, &sourceStr, &c.Confidence, &c.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, contracts.ErrNotFound{Kind: "claim", ID: id}
		}
		return nil, fmt.Errorf("scan claim: %w", err)
	}
	c.Source = domain.ClaimSource(sourceStr)
	return &c, nil
}

// ListBySubject implements domain.ClaimRepository.
func (r *ClaimRepository) ListBySubject(ctx context.Context, subject string) ([]domain.Claim, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT id, subject, subject_type, text, source, confidence, created_at
  FROM evidence.claims WHERE subject=$1 ORDER BY created_at DESC`, subject)
	if err != nil {
		return nil, mapErr(err)
	}
	defer rows.Close()
	out := make([]domain.Claim, 0)
	for rows.Next() {
		var (
			c         domain.Claim
			sourceStr string
		)
		if err := rows.Scan(&c.ID, &c.Subject, &c.SubjectType, &c.Text, &sourceStr, &c.Confidence, &c.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan claim row: %w", err)
		}
		c.Source = domain.ClaimSource(sourceStr)
		out = append(out, c)
	}
	return out, nil
}

// CitationRepository implements domain.CitationRepository against Postgres.
type CitationRepository struct{ db querier }

// NewCitationRepository constructs the repository.
func NewCitationRepository(db querier) *CitationRepository { return &CitationRepository{db: db} }

// Save implements domain.CitationRepository.
func (r *CitationRepository) Save(ctx context.Context, c domain.Citation) error {
	_, err := r.db.ExecContext(ctx, `
INSERT INTO evidence.citations
  (id, claim_id, document_id, page_number, section_offset, quote, relationship, confidence, created_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
ON CONFLICT (id) DO UPDATE SET
   quote = EXCLUDED.quote,
   relationship = EXCLUDED.relationship,
   confidence = EXCLUDED.confidence
`, c.ID, c.ClaimID, c.DocumentID, c.PageNumber, c.SectionOffset, c.Quote,
		string(c.Relationship), c.Confidence, c.CreatedAt)
	return mapErr(err)
}

// ListByClaim implements domain.CitationRepository.
func (r *CitationRepository) ListByClaim(ctx context.Context, claimID string) ([]domain.Citation, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT id, claim_id, document_id, page_number, section_offset, quote, relationship, confidence, created_at
  FROM evidence.citations WHERE claim_id=$1 ORDER BY created_at ASC`, claimID)
	if err != nil {
		return nil, mapErr(err)
	}
	defer rows.Close()
	out := make([]domain.Citation, 0)
	for rows.Next() {
		var (
			c       domain.Citation
			rel     string
		)
		if err := rows.Scan(&c.ID, &c.ClaimID, &c.DocumentID, &c.PageNumber, &c.SectionOffset,
			&c.Quote, &rel, &c.Confidence, &c.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan citation: %w", err)
		}
		c.Relationship = domain.CitationRelationship(rel)
		out = append(out, c)
	}
	return out, nil
}

// ConflictRepository implements domain.SourceConflictRepository.
type ConflictRepository struct{ db querier }

// NewConflictRepository constructs the repository.
func NewConflictRepository(db querier) *ConflictRepository { return &ConflictRepository{db: db} }

// Save implements domain.SourceConflictRepository.
func (r *ConflictRepository) Save(ctx context.Context, c domain.SourceConflict) error {
	_, err := r.db.ExecContext(ctx, `
INSERT INTO evidence.source_conflicts
  (id, claim_id, claim_text, source_a_value, source_b_value, detected_at, resolution, resolved_by, resolved_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
ON CONFLICT (id) DO UPDATE SET
   resolution = EXCLUDED.resolution,
   resolved_by = EXCLUDED.resolved_by,
   resolved_at = EXCLUDED.resolved_at
`, c.ID, c.ClaimID, c.ClaimText, c.SourceAValue, c.SourceBValue,
		c.DetectedAt, string(c.Resolution), c.ResolvedBy, c.ResolvedAt)
	return mapErr(err)
}

// ListOpen implements domain.SourceConflictRepository.
func (r *ConflictRepository) ListOpen(ctx context.Context) ([]domain.SourceConflict, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT id, claim_id, claim_text, source_a_value, source_b_value, detected_at, resolution, COALESCE(resolved_by,''), resolved_at
  FROM evidence.source_conflicts WHERE resolution='open' ORDER BY detected_at DESC`)
	if err != nil {
		return nil, mapErr(err)
	}
	defer rows.Close()
	out := make([]domain.SourceConflict, 0)
	for rows.Next() {
		var (
			c    domain.SourceConflict
			rel  string
		)
		if err := rows.Scan(&c.ID, &c.ClaimID, &c.ClaimText, &c.SourceAValue, &c.SourceBValue,
			&c.DetectedAt, &rel, &c.ResolvedBy, &c.ResolvedAt); err != nil {
			return nil, fmt.Errorf("scan conflict: %w", err)
		}
		c.Resolution = domain.ConflictResolution(rel)
		out = append(out, c)
	}
	return out, nil
}

// EvidenceRepository implements domain.EvidenceRepository.
type EvidenceRepository struct{ db querier }

// NewEvidenceRepository constructs the repository.
func NewEvidenceRepository(db querier) *EvidenceRepository { return &EvidenceRepository{db: db} }

// Save implements domain.EvidenceRepository.
func (r *EvidenceRepository) Save(ctx context.Context, e domain.EvidenceSet) error {
	_, err := r.db.ExecContext(ctx, `
INSERT INTO evidence.evidence_sets
  (id, request_id, bill_id, validator_id, validated_at)
VALUES ($1,$2,$3,$4,$5)
ON CONFLICT (id) DO UPDATE SET validator_id = EXCLUDED.validator_id
`, e.ID, e.RequestID, e.BillID, e.ValidatorID, e.ValidatedAt)
	return mapErr(err)
}

// GetByRequest implements domain.EvidenceRepository.
func (r *EvidenceRepository) GetByRequest(ctx context.Context, requestID string) (*domain.EvidenceSet, error) {
	row := r.db.QueryRowContext(ctx, `
SELECT id, request_id, bill_id, validator_id, validated_at
  FROM evidence.evidence_sets WHERE request_id=$1`, requestID)
	var e domain.EvidenceSet
	if err := row.Scan(&e.ID, &e.RequestID, &e.BillID, &e.ValidatorID, &e.ValidatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, contracts.ErrNotFound{Kind: "evidence_set", ID: requestID}
		}
		return nil, fmt.Errorf("scan evidence set: %w", err)
	}
	return &e, nil
}

// SourceReferenceRepository implements domain.SourceReferenceRepository.
type SourceReferenceRepository struct{ db querier }

// NewSourceReferenceRepository constructs the repository.
func NewSourceReferenceRepository(db querier) *SourceReferenceRepository {
	return &SourceReferenceRepository{db: db}
}

// Save implements domain.SourceReferenceRepository.
func (r *SourceReferenceRepository) Save(ctx context.Context, s domain.SourceReference) error {
	_, err := r.db.ExecContext(ctx, `
INSERT INTO evidence.source_references
  (id, document_id, source_id, country_code, url, published_at, content_hash, is_authoritative)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
ON CONFLICT (id) DO UPDATE SET url = EXCLUDED.url, content_hash = EXCLUDED.content_hash
`, s.ID, s.DocumentID, s.SourceID, s.CountryCode, s.URL, s.PublishedAt, s.ContentHash, s.IsAuthoritative)
	return mapErr(err)
}

// GetByDocument implements domain.SourceReferenceRepository.
func (r *SourceReferenceRepository) GetByDocument(ctx context.Context, documentID string) (*domain.SourceReference, error) {
	row := r.db.QueryRowContext(ctx, `
SELECT id, document_id, source_id, country_code, url, published_at, content_hash, is_authoritative
  FROM evidence.source_references WHERE document_id=$1`, documentID)
	var s domain.SourceReference
	if err := row.Scan(&s.ID, &s.DocumentID, &s.SourceID, &s.CountryCode, &s.URL, &s.PublishedAt,
		&s.ContentHash, &s.IsAuthoritative); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, contracts.ErrNotFound{Kind: "source_reference", ID: documentID}
		}
		return nil, fmt.Errorf("scan source ref: %w", err)
	}
	return &s, nil
}

// mapErr is the shared error mapper.
func mapErr(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return contracts.ErrNotFound{Kind: "row", ID: ""}
	}
	return fmt.Errorf("postgres: %w", err)
}
