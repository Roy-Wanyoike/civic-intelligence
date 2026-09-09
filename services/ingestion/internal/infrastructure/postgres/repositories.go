package postgres

import (
        "context"
        "database/sql"
        "errors"
        "fmt"
        "time"

        "github.com/Roy-Wanyoike/civic-intelligence/packages/contracts"
        "github.com/Roy-Wanyoike/civic-intelligence/services/ingestion/internal/domain"
)

type querier interface {
        ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
        QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
        QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

// SourceRepository is the Postgres implementation of domain.SourceRepository.
type SourceRepository struct {
        db querier
}

// NewSourceRepository constructs the repository.
func NewSourceRepository(db querier) *SourceRepository { return &SourceRepository{db: db} }

// Save implements domain.SourceRepository.
func (r *SourceRepository) Save(ctx context.Context, s domain.Source) error {
        _, err := r.db.ExecContext(ctx, `
INSERT INTO ingestion.sources
  (id, country_code, name, base_url, type, adapter_code, poll_period, enabled, created_at, updated_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
ON CONFLICT (id) DO UPDATE SET
   name = EXCLUDED.name,
   base_url = EXCLUDED.base_url,
   poll_period = EXCLUDED.poll_period,
   enabled = EXCLUDED.enabled,
   updated_at = EXCLUDED.updated_at
`, s.ID, s.CountryCode, s.Name, s.BaseURL, string(s.Type),
                s.AdapterCode, int64(s.PollPeriod.Seconds()), s.Enabled,
                s.CreatedAt, s.UpdatedAt,
        )
        return mapErr(err)
}

// Get implements domain.SourceRepository.
func (r *SourceRepository) Get(ctx context.Context, id string) (*domain.Source, error) {
        row := r.db.QueryRowContext(ctx, `
SELECT id, country_code, name, base_url, type, adapter_code,
       COALESCE(poll_period,0), enabled, created_at, updated_at
  FROM ingestion.sources
 WHERE id = $1`, id)
        var s domain.Source
        var pollSecs int64
        var kind string
        if err := row.Scan(&s.ID, &s.CountryCode, &s.Name, &s.BaseURL, &kind,
                &s.AdapterCode, &pollSecs, &s.Enabled, &s.CreatedAt, &s.UpdatedAt); err != nil {
                if errors.Is(err, sql.ErrNoRows) {
                        return nil, contracts.ErrNotFound{Kind: "source", ID: id}
                }
                return nil, fmt.Errorf("scan source: %w", err)
        }
        s.Type = domain.SourceType(kind)
        s.PollPeriod = durationFromSeconds(pollSecs)
        return &s, nil
}

// List implements domain.SourceRepository.
func (r *SourceRepository) List(ctx context.Context, country string) ([]domain.Source, error) {
        q := "SELECT id, country_code, name, base_url, type, adapter_code, COALESCE(poll_period,0), enabled, created_at, updated_at FROM ingestion.sources"
        args := []any{}
        if country != "" {
                q += " WHERE country_code = $1"
                args = append(args, country)
        }
        q += " ORDER BY name ASC"
        rows, err := r.db.QueryContext(ctx, q, args...)
        if err != nil {
                return nil, mapErr(err)
        }
        defer rows.Close()
        out := make([]domain.Source, 0)
        for rows.Next() {
                var s domain.Source
                var pollSecs int64
                var kind string
                if err := rows.Scan(&s.ID, &s.CountryCode, &s.Name, &s.BaseURL, &kind,
                        &s.AdapterCode, &pollSecs, &s.Enabled, &s.CreatedAt, &s.UpdatedAt); err != nil {
                        return nil, fmt.Errorf("scan source row: %w", err)
                }
                s.Type = domain.SourceType(kind)
                s.PollPeriod = durationFromSeconds(pollSecs)
                out = append(out, s)
        }
        return out, nil
}

// GetByBaseURL implements domain.SourceRepository.
func (r *SourceRepository) GetByBaseURL(ctx context.Context, baseURL string) (*domain.Source, error) {
        row := r.db.QueryRowContext(ctx, `
SELECT id, country_code, name, base_url, type, adapter_code,
       COALESCE(poll_period,0), enabled, created_at, updated_at
  FROM ingestion.sources
 WHERE base_url = $1`, baseURL)
        var s domain.Source
        var pollSecs int64
        var kind string
        if err := row.Scan(&s.ID, &s.CountryCode, &s.Name, &s.BaseURL, &kind,
                &s.AdapterCode, &pollSecs, &s.Enabled, &s.CreatedAt, &s.UpdatedAt); err != nil {
                if errors.Is(err, sql.ErrNoRows) {
                        return nil, contracts.ErrNotFound{Kind: "source", ID: baseURL}
                }
                return nil, fmt.Errorf("scan source: %w", err)
        }
        s.Type = domain.SourceType(kind)
        s.PollPeriod = durationFromSeconds(pollSecs)
        return &s, nil
}

// FetchJobRepository is the Postgres implementation of domain.FetchJobRepository.
type FetchJobRepository struct {
        db querier
}

// NewFetchJobRepository constructs the repository.
func NewFetchJobRepository(db querier) *FetchJobRepository { return &FetchJobRepository{db: db} }

// Save implements domain.FetchJobRepository.
func (r *FetchJobRepository) Save(ctx context.Context, j domain.FetchJob) error {
        _, err := r.db.ExecContext(ctx, `
INSERT INTO ingestion.fetch_jobs
  (id, source_id, endpoint_id, url, external_id, country_code, status, attempts, bytes,
   content_hash, mime_type, fetched_at, error)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
ON CONFLICT (id) DO UPDATE SET
   status = EXCLUDED.status,
   attempts = EXCLUDED.attempts,
   bytes = EXCLUDED.bytes,
   content_hash = EXCLUDED.content_hash,
   mime_type = EXCLUDED.mime_type,
   fetched_at = EXCLUDED.fetched_at,
   error = EXCLUDED.error
`, j.ID, j.SourceID, j.EndpointID, j.URL, j.ExternalID, j.CountryCode,
                string(j.Status), j.Attempts, j.Bytes, j.ContentHash, j.MimeType,
                j.FetchedAt, j.Error,
        )
        return mapErr(err)
}

// Get implements domain.FetchJobRepository.
func (r *FetchJobRepository) Get(ctx context.Context, id string) (*domain.FetchJob, error) {
        row := r.db.QueryRowContext(ctx, `
SELECT id, source_id, endpoint_id, url, external_id, country_code,
       status, attempts, bytes, content_hash, mime_type, fetched_at, error
  FROM ingestion.fetch_jobs WHERE id = $1`, id)
        var (
                j      domain.FetchJob
                status string
        )
        if err := row.Scan(&j.ID, &j.SourceID, &j.EndpointID, &j.URL, &j.ExternalID,
                &j.CountryCode, &status, &j.Attempts, &j.Bytes, &j.ContentHash,
                &j.MimeType, &j.FetchedAt, &j.Error); err != nil {
                if errors.Is(err, sql.ErrNoRows) {
                        return nil, contracts.ErrNotFound{Kind: "fetch_job", ID: id}
                }
                return nil, fmt.Errorf("scan fetch job: %w", err)
        }
        j.Status = domain.JobStatus(status)
        return &j, nil
}

// ListPending implements domain.FetchJobRepository.
func (r *FetchJobRepository) ListPending(ctx context.Context, limit int) ([]domain.FetchJob, error) {
        rows, err := r.db.QueryContext(ctx, `
SELECT id, source_id, endpoint_id, url, external_id, country_code,
       status, attempts, bytes, content_hash, mime_type, fetched_at, error
  FROM ingestion.fetch_jobs
 WHERE status = 'pending'
 ORDER BY created_at ASC
 LIMIT $1`, limit)
        if err != nil {
                return nil, mapErr(err)
        }
        defer rows.Close()
        out := make([]domain.FetchJob, 0)
        for rows.Next() {
                var (
                        j      domain.FetchJob
                        status string
                )
                if err := rows.Scan(&j.ID, &j.SourceID, &j.EndpointID, &j.URL, &j.ExternalID,
                        &j.CountryCode, &status, &j.Attempts, &j.Bytes, &j.ContentHash,
                        &j.MimeType, &j.FetchedAt, &j.Error); err != nil {
                        return nil, fmt.Errorf("scan fetch job row: %w", err)
                }
                j.Status = domain.JobStatus(status)
                out = append(out, j)
        }
        return out, nil
}

// MarkRunning implements domain.FetchJobRepository.
func (r *FetchJobRepository) MarkRunning(ctx context.Context, id string) error {
        _, err := r.db.ExecContext(ctx, `UPDATE ingestion.fetch_jobs SET status='running' WHERE id=$1`, id)
        return mapErr(err)
}

// MarkDone implements domain.FetchJobRepository.
func (r *FetchJobRepository) MarkDone(ctx context.Context, id string, hash string, bytes int64, mime string) error {
        _, err := r.db.ExecContext(ctx, `
UPDATE ingestion.fetch_jobs
   SET status='succeeded', content_hash=$2, bytes=$3, mime_type=$4, fetched_at=NOW(), error=NULL
 WHERE id=$1`, id, hash, bytes, mime)
        return mapErr(err)
}

// MarkFailed implements domain.FetchJobRepository.
func (r *FetchJobRepository) MarkFailed(ctx context.Context, id string, reason string) error {
        _, err := r.db.ExecContext(ctx, `
UPDATE ingestion.fetch_jobs
   SET status='failed', error=$2, attempts=attempts+1
 WHERE id=$1`, id, reason)
        return mapErr(err)
}

// DocumentRepository persists raw documents (bytes excluded; they live in
// object storage and are referenced by ContentHash).
type DocumentRepository struct {
        db querier
}

// NewDocumentRepository constructs the repository.
func NewDocumentRepository(db querier) *DocumentRepository { return &DocumentRepository{db: db} }

// Save implements domain.DocumentRepository. We do NOT store Bytes in the DB
// in production (object storage holds them); here we include them in the
// row for the development-friendly single-binary deployment.
func (r *DocumentRepository) Save(ctx context.Context, d domain.RawDocument) error {
        _, err := r.db.ExecContext(ctx, `
INSERT INTO ingestion.documents
  (id, source_id, fetch_job_id, external_id, country_code, url, mime_type,
   content_hash, bytes, fetched_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
ON CONFLICT (id) DO UPDATE SET
   mime_type = EXCLUDED.mime_type,
   bytes = EXCLUDED.bytes,
   content_hash = EXCLUDED.content_hash,
   url = EXCLUDED.url
`, d.ID, d.SourceID, d.FetchJobID, d.ExternalID, d.CountryCode, d.URL,
                d.MimeType, d.ContentHash, d.Bytes, d.FetchedAt)
        return mapErr(err)
}

// Get implements domain.DocumentRepository.
func (r *DocumentRepository) Get(ctx context.Context, id string) (*domain.RawDocument, error) {
        row := r.db.QueryRowContext(ctx, `
SELECT id, source_id, fetch_job_id, external_id, country_code, url, mime_type,
       content_hash, bytes, fetched_at
  FROM ingestion.documents WHERE id=$1`, id)
        var d domain.RawDocument
        if err := row.Scan(&d.ID, &d.SourceID, &d.FetchJobID, &d.ExternalID,
                &d.CountryCode, &d.URL, &d.MimeType, &d.ContentHash, &d.Bytes, &d.FetchedAt); err != nil {
                if errors.Is(err, sql.ErrNoRows) {
                        return nil, contracts.ErrNotFound{Kind: "document", ID: id}
                }
                return nil, fmt.Errorf("scan document: %w", err)
        }
        return &d, nil
}

// GetByContentHash implements domain.DocumentRepository.
func (r *DocumentRepository) GetByContentHash(ctx context.Context, hash string) (*domain.RawDocument, error) {
        row := r.db.QueryRowContext(ctx, `
SELECT id, source_id, fetch_job_id, external_id, country_code, url, mime_type,
       content_hash, bytes, fetched_at
  FROM ingestion.documents WHERE content_hash=$1`, hash)
        var d domain.RawDocument
        if err := row.Scan(&d.ID, &d.SourceID, &d.FetchJobID, &d.ExternalID,
                &d.CountryCode, &d.URL, &d.MimeType, &d.ContentHash, &d.Bytes, &d.FetchedAt); err != nil {
                if errors.Is(err, sql.ErrNoRows) {
                        return nil, contracts.ErrNotFound{Kind: "document", ID: hash}
                }
                return nil, fmt.Errorf("scan document: %w", err)
        }
        return &d, nil
}

// GetByExternalID implements domain.DocumentRepository.
func (r *DocumentRepository) GetByExternalID(ctx context.Context, countryCode, externalID string) (*domain.RawDocument, error) {
        row := r.db.QueryRowContext(ctx, `
SELECT id, source_id, fetch_job_id, external_id, country_code, url, mime_type,
       content_hash, bytes, fetched_at
  FROM ingestion.documents
 WHERE country_code=$1 AND external_id=$2
 ORDER BY fetched_at DESC
 LIMIT 1`, countryCode, externalID)
        var d domain.RawDocument
        if err := row.Scan(&d.ID, &d.SourceID, &d.FetchJobID, &d.ExternalID,
                &d.CountryCode, &d.URL, &d.MimeType, &d.ContentHash, &d.Bytes, &d.FetchedAt); err != nil {
                if errors.Is(err, sql.ErrNoRows) {
                        return nil, contracts.ErrNotFound{Kind: "document", ID: externalID}
                }
                return nil, fmt.Errorf("scan document: %w", err)
        }
        return &d, nil
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

// durationFromSeconds converts seconds to time.Duration without importing
// time in the SQL layer (kept here to make the import honest).
func durationFromSeconds(s int64) (d time.Duration) {
        if s <= 0 {
                return 0
        }
        d = time.Duration(s) * time.Second
        return
}
