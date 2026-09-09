package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Roy-Wanyoike/civic-intelligence/packages/contracts"
	"github.com/Roy-Wanyoike/civic-intelligence/services/legislation/internal/domain"
)

// AmendmentRepository implements domain.AmendmentRepository against Postgres.
type AmendmentRepository struct {
	db querier
}

// NewAmendmentRepository constructs the repository.
func NewAmendmentRepository(db querier) *AmendmentRepository {
	return &AmendmentRepository{db: db}
}

// Save implements domain.AmendmentRepository.
func (r *AmendmentRepository) Save(ctx context.Context, a domain.Amendment) error {
	_, err := r.db.ExecContext(ctx, `
INSERT INTO legislation.amendments
  (id, bill_id, version_no, sponsor_id, clause_ref, type, text,
   justification, status, created_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
ON CONFLICT (id) DO UPDATE SET
   status = EXCLUDED.status,
   justification = EXCLUDED.justification
`,
		string(a.ID), string(a.BillID), a.VersionNo, string(a.SponsorID),
		a.ClauseRef, a.Type, a.Text, a.Justification,
		string(a.Status), a.CreatedAt,
	)
	return mapPgErr(err)
}

// Get implements domain.AmendmentRepository.
func (r *AmendmentRepository) Get(ctx context.Context, id domain.ID) (*domain.Amendment, error) {
	row := r.db.QueryRowContext(ctx, `
SELECT id, bill_id, version_no, sponsor_id, clause_ref, type, text,
       justification, status, created_at
  FROM legislation.amendments
 WHERE id = $1`, string(id))
	var (
		a        domain.Amendment
		sponsor  string
		kind     string
		status   string
	)
	if err := row.Scan(&a.ID, &a.BillID, &a.VersionNo, &sponsor,
		&a.ClauseRef, &kind, &a.Text, &a.Justification,
		&status, &a.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, contracts.ErrNotFound{Kind: "amendment", ID: string(id)}
		}
		return nil, fmt.Errorf("scan amendment: %w", err)
	}
	a.SponsorID = domain.ID(sponsor)
	a.Type = kind
	a.Status = domain.AmendmentStatus(status)
	return &a, nil
}

// ListByBill implements domain.AmendmentRepository.
func (r *AmendmentRepository) ListByBill(ctx context.Context, billID domain.ID) ([]domain.Amendment, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT id, bill_id, version_no, sponsor_id, clause_ref, type, text,
       justification, status, created_at
  FROM legislation.amendments
 WHERE bill_id = $1
 ORDER BY created_at ASC`, string(billID))
	if err != nil {
		return nil, mapPgErr(err)
	}
	defer rows.Close()
	out := make([]domain.Amendment, 0)
	for rows.Next() {
		var (
			a       domain.Amendment
			sponsor string
			kind    string
			status  string
		)
		if err := rows.Scan(&a.ID, &a.BillID, &a.VersionNo, &sponsor,
			&a.ClauseRef, &kind, &a.Text, &a.Justification,
			&status, &a.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan amendment row: %w", err)
		}
		a.SponsorID = domain.ID(sponsor)
		a.Type = kind
		a.Status = domain.AmendmentStatus(status)
		out = append(out, a)
	}
	return out, nil
}

// Compile-time assertion.
var _ domain.AmendmentRepository = (*AmendmentRepository)(nil)
