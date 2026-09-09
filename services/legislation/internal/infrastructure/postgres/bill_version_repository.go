package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/Roy-Wanyoike/civic-intelligence/packages/contracts"
	"github.com/Roy-Wanyoike/civic-intelligence/services/legislation/internal/domain"
)

// BillVersionRepository implements domain.BillVersionRepository against Postgres.
type BillVersionRepository struct {
	db querier
}

// NewBillVersionRepository constructs the repository.
func NewBillVersionRepository(db querier) *BillVersionRepository {
	return &BillVersionRepository{db: db}
}

// Save implements domain.BillVersionRepository.
func (r *BillVersionRepository) Save(ctx context.Context, v domain.BillVersion) error {
	_, err := r.db.ExecContext(ctx, `
INSERT INTO legislation.bill_versions
  (id, bill_id, version_number, title, text_hash, source_document_id,
   published_at, changes_summary)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
ON CONFLICT (id) DO UPDATE SET
   title = EXCLUDED.title,
   text_hash = EXCLUDED.text_hash,
   source_document_id = EXCLUDED.source_document_id,
   changes_summary = EXCLUDED.changes_summary
`,
		string(v.ID), string(v.BillID), v.VersionNumber, v.Title,
		v.TextHash, v.SourceDocumentID, v.PublishedAt, v.ChangesSummary,
	)
	return mapPgErr(err)
}

// Get implements domain.BillVersionRepository.
func (r *BillVersionRepository) Get(ctx context.Context, id domain.ID) (*domain.BillVersion, error) {
	row := r.db.QueryRowContext(ctx, `
SELECT id, bill_id, version_number, title, text_hash,
       source_document_id, published_at, changes_summary
  FROM legislation.bill_versions
 WHERE id = $1`, string(id))
	var v domain.BillVersion
	if err := row.Scan(&v.ID, &v.BillID, &v.VersionNumber, &v.Title,
		&v.TextHash, &v.SourceDocumentID, &v.PublishedAt, &v.ChangesSummary); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, contracts.ErrNotFound{Kind: "bill_version", ID: string(id)}
		}
		return nil, fmt.Errorf("scan version: %w", err)
	}
	return &v, nil
}

// ListByBill implements domain.BillVersionRepository.
func (r *BillVersionRepository) ListByBill(ctx context.Context, billID domain.ID) ([]domain.BillVersion, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT id, bill_id, version_number, title, text_hash,
       source_document_id, published_at, changes_summary
  FROM legislation.bill_versions
 WHERE bill_id = $1
 ORDER BY version_number ASC`, string(billID))
	if err != nil {
		return nil, mapPgErr(err)
	}
	defer rows.Close()
	out := make([]domain.BillVersion, 0)
	for rows.Next() {
		var v domain.BillVersion
		if err := rows.Scan(&v.ID, &v.BillID, &v.VersionNumber, &v.Title,
			&v.TextHash, &v.SourceDocumentID, &v.PublishedAt, &v.ChangesSummary); err != nil {
			return nil, fmt.Errorf("scan version row: %w", err)
		}
		out = append(out, v)
	}
	return out, nil
}

// CurrentForBill implements domain.BillVersionRepository.
func (r *BillVersionRepository) CurrentForBill(ctx context.Context, billID domain.ID) (*domain.BillVersion, error) {
	row := r.db.QueryRowContext(ctx, `
SELECT id, bill_id, version_number, title, text_hash,
       source_document_id, published_at, changes_summary
  FROM legislation.bill_versions
 WHERE bill_id = $1
 ORDER BY version_number DESC
 LIMIT 1`, string(billID))
	var v domain.BillVersion
	if err := row.Scan(&v.ID, &v.BillID, &v.VersionNumber, &v.Title,
		&v.TextHash, &v.SourceDocumentID, &v.PublishedAt, &v.ChangesSummary); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, contracts.ErrNotFound{Kind: "bill_version", ID: "current"}
		}
		return nil, fmt.Errorf("scan current version: %w", err)
	}
	return &v, nil
}

// Compile-time assertion.
var _ domain.BillVersionRepository = (*BillVersionRepository)(nil)
