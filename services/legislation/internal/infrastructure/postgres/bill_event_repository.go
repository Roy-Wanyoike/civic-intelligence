package postgres

import (
        "context"
        "database/sql"
        "fmt"
        "time"

        "github.com/Roy-Wanyoike/civic-intelligence/services/legislation/internal/domain"
)

// BillEventRepository implements domain.BillEventRepository against Postgres.
type BillEventRepository struct {
        db querier
}

// NewBillEventRepository constructs the repository.
func NewBillEventRepository(db querier) *BillEventRepository {
        return &BillEventRepository{db: db}
}

// Save implements domain.BillEventRepository.
func (r *BillEventRepository) Save(ctx context.Context, e domain.BillEvent) error {
        _, err := r.db.ExecContext(ctx, `
INSERT INTO legislation.bill_events
  (id, bill_id, kind, from_stage, to_stage, reason, occurred_at, actor, metadata)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
ON CONFLICT (id) DO NOTHING
`,
                string(e.ID()), string(e.BillID()), string(e.Kind()),
                e.FromStage(), e.ToStage(), e.Reason(), e.OccurredAt(),
                e.Actor(), e.Metadata(),
        )
        return mapPgErr(err)
}

// ListByBill implements domain.BillEventRepository.
func (r *BillEventRepository) ListByBill(ctx context.Context, billID domain.ID) ([]domain.BillEvent, error) {
        rows, err := r.db.QueryContext(ctx, `
SELECT id, bill_id, kind, from_stage, to_stage, reason, occurred_at, actor, metadata
  FROM legislation.bill_events
 WHERE bill_id = $1
 ORDER BY occurred_at ASC, id ASC`, string(billID))
        if err != nil {
                return nil, mapPgErr(err)
        }
        defer rows.Close()
        out := make([]domain.BillEvent, 0)
        for rows.Next() {
                var (
                        id, billIDStr, kind, from, to, reason, actor string
                        occurredAt                                   time.Time
                        meta                                         map[string]string
                )
                if err := rows.Scan(&id, &billIDStr, &kind, &from, &to, &reason,
                        &occurredAt, &actor, &meta); err != nil {
                        return nil, fmt.Errorf("scan event row: %w", err)
                }
                e := domain.NewBillEvent(
                        domain.ID(id), domain.ID(billIDStr), domain.BillEventKind(kind),
                        from, to, reason, actor, occurredAt,
                )
                out = append(out, e)
        }
        return out, nil
}

// Compile-time assertion.
var _ domain.BillEventRepository = (*BillEventRepository)(nil)
