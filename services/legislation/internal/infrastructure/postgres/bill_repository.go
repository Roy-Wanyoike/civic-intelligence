// Package postgres contains the PostgreSQL implementations of the
// legislation service's repository interfaces. The domain layer depends only
// on the interfaces in internal/domain; this package provides the concrete
// implementations wired up at startup in cmd/main.go.
package postgres

import (
        "context"
        "database/sql"
        "errors"
        "fmt"
        "time"

        "github.com/Roy-Wanyoike/civic-intelligence/packages/contracts"
        "github.com/Roy-Wanyoike/civic-intelligence/services/legislation/internal/domain"
)

// queries is the small surface of pgxConn that we use; narrowing the
// dependency lets the repository talk to either *sql.DB, pgxpool.Pool or a
// transaction without taking a direct dependency on a specific driver.
type querier interface {
        ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
        QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
        QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

// BillRepository is the PostgreSQL implementation of domain.BillRepository.
// It maps the Bill aggregate to/from the canonical bills table.
type BillRepository struct {
        db querier
}

// NewBillRepository constructs the repository. The db parameter may be a
// *sql.DB or a pgx pool wrapped to satisfy querier.
func NewBillRepository(db querier) *BillRepository {
        return &BillRepository{db: db}
}

// Save implements domain.BillRepository. It serialises the bill's stage and
// event history into their canonical columns. The implementation is
// upsert-style: if the row exists, it is updated; otherwise it is inserted.
func (r *BillRepository) Save(ctx context.Context, bill *domain.Bill) error {
        // Note: This is a hand-written SQL implementation; in production we'd
        // lean on sqlc-generated code, but a hand-rolled version is more
        // transparent for review.
        _, err := r.db.ExecContext(ctx, `
INSERT INTO legislation.bills
  (id, country_id, title, short_title, house_id, sponsor_id,
   current_stage, introduced_at, updated_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
ON CONFLICT (id) DO UPDATE SET
   title = EXCLUDED.title,
   short_title = EXCLUDED.short_title,
   sponsor_id = EXCLUDED.sponsor_id,
   current_stage = EXCLUDED.current_stage,
   updated_at = EXCLUDED.updated_at
`,
                string(bill.ID()), string(bill.CountryID()), bill.Title(),
                bill.ShortTitle(), string(bill.HouseID()),
                idToNullable(bill.SponsorID()), bill.CurrentStage(),
                bill.IntroducedAt(), bill.UpdatedAt(),
        )
        if err != nil {
                return mapPgErr(err)
        }
        return nil
}

// Get implements domain.BillRepository.
func (r *BillRepository) Get(ctx context.Context, id domain.ID) (*Bill, error) {
        row := r.db.QueryRowContext(ctx, `
SELECT id, country_id, title, short_title, house_id, sponsor_id,
       current_stage, introduced_at, updated_at
  FROM legislation.bills
 WHERE id = $1`, string(id))
        var b BillSnapshot
        if err := row.Scan(&b.ID, &b.CountryID, &b.Title, &b.ShortTitle,
                &b.HouseID, &b.SponsorID, &b.CurrentStage,
                &b.IntroducedAt, &b.UpdatedAt); err != nil {
                if errors.Is(err, sql.ErrNoRows) {
                        return nil, contracts.ErrNotFound{Kind: "bill", ID: string(id)}
                }
                return nil, fmt.Errorf("scan bill: %w", err)
        }
        return b.ToDomain(), nil
}

// List implements domain.BillRepository.
func (r *BillRepository) List(ctx context.Context, filter domain.BillFilter, page contracts.PageRequest) (contracts.Page[*domain.Bill], error) {
        page = page.Normalize()
        args := []any{page.Size(), page.Offset()}
        where := "WHERE 1=1"
        if filter.CountryCode != "" {
                args = append(args, filter.CountryCode)
                where += fmt.Sprintf(" AND country_code = $%d", len(args))
        }
        if filter.HouseCode != "" {
                args = append(args, filter.HouseCode)
                where += fmt.Sprintf(" AND house_code = $%d", len(args))
        }
        if filter.Stage != "" {
                args = append(args, filter.Stage)
                where += fmt.Sprintf(" AND current_stage = $%d", len(args))
        }
        if filter.Search != "" {
                args = append(args, "%"+filter.Search+"%")
                where += fmt.Sprintf(" AND title ILIKE $%d", len(args))
        }

        var total int64
        if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM legislation.bills "+where, args[2:]...).Scan(&total); err != nil {
                return contracts.Page[*domain.Bill]{}, mapPgErr(err)
        }

        rows, err := r.db.QueryContext(ctx, `
SELECT id, country_id, title, short_title, house_id, sponsor_id,
       current_stage, introduced_at, updated_at
  FROM legislation.bills `+where+`
 ORDER BY updated_at DESC
 LIMIT $1 OFFSET $2`, args...)
        if err != nil {
                return contracts.Page[*domain.Bill]{}, mapPgErr(err)
        }
        defer rows.Close()

        out := make([]*domain.Bill, 0, page.Size)
        for rows.Next() {
                var b BillSnapshot
                if err := rows.Scan(&b.ID, &b.CountryID, &b.Title, &b.ShortTitle,
                        &b.HouseID, &b.SponsorID, &b.CurrentStage,
                        &b.IntroducedAt, &b.UpdatedAt); err != nil {
                        return contracts.Page[*domain.Bill]{}, fmt.Errorf("scan bill row: %w", err)
                }
                out = append(out, b.ToDomain())
        }
        return contracts.Page[*domain.Bill]{Items: out, Total: total, Page: page.Page, Size: page.Size}, nil
}

// GetByExternalID implements domain.BillRepository.
func (r *BillRepository) GetByExternalID(ctx context.Context, countryCode, externalID string) (*domain.Bill, error) {
        row := r.db.QueryRowContext(ctx, `
SELECT b.id, b.country_id, b.title, b.short_title, b.house_id, b.sponsor_id,
       b.current_stage, b.introduced_at, b.updated_at
  FROM legislation.bills b
  JOIN legislation.countries c ON c.id = b.country_id
 WHERE c.code = $1 AND b.external_id = $2`, countryCode, externalID)
        var b BillSnapshot
        if err := row.Scan(&b.ID, &b.CountryID, &b.Title, &b.ShortTitle,
                &b.HouseID, &b.SponsorID, &b.CurrentStage,
                &b.IntroducedAt, &b.UpdatedAt); err != nil {
                if errors.Is(err, sql.ErrNoRows) {
                        return nil, contracts.ErrNotFound{Kind: "bill", ID: externalID}
                }
                return nil, fmt.Errorf("scan bill by external id: %w", err)
        }
        return b.ToDomain(), nil
}

// BillSnapshot is the row representation; it deliberately mirrors the SQL
// columns so we can keep the SQL above honest.
type BillSnapshot struct {
        ID           string
        CountryID    string
        Title        string
        ShortTitle   string
        HouseID      string
        SponsorID    *string
        CurrentStage string
        IntroducedAt time.Time
        UpdatedAt    time.Time
}

// ToDomain projects the snapshot into a fresh Bill aggregate. The aggregate
// starts in its current stage; the caller can rehydrate history via the
// events repository if needed.
func (b BillSnapshot) ToDomain() *domain.Bill {
        var sponsor *domain.ID
        if b.SponsorID != nil {
                s := domain.ID(*b.SponsorID)
                sponsor = &s
        }
        return domain.NewBill(
                domain.ID(b.ID), domain.ID(b.CountryID), b.Title, b.ShortTitle,
                domain.ID(b.HouseID), sponsor, b.CurrentStage, b.IntroducedAt,
        )
}

// idToNullable converts an *ID to a SQL-friendly value (NULL or string).
func idToNullable(p *domain.ID) any {
        if p == nil {
                return nil
        }
        return string(*p)
}

// mapPgErr maps a sql/driver error into a typed contracts error. We keep
// this small because pgx returns rich errors that we'd otherwise need to
// import; the canonical mapping is by error code.
func mapPgErr(err error) error {
        if err == nil {
                return nil
        }
        if errors.Is(err, sql.ErrNoRows) {
                return contracts.ErrNotFound{Kind: "row", ID: ""}
        }
        return fmt.Errorf("postgres: %w", err)
}

// Compile-time assertion that BillRepository satisfies the domain interface.
var _ domain.BillRepository = (*BillRepository)(nil)
