package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/Roy-Wanyoike/civic-intelligence/packages/contracts"
	"github.com/Roy-Wanyoike/civic-intelligence/services/documents/internal/domain"
)

type querier interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

// DocumentRepository is the Postgres implementation.
type DocumentRepository struct {
	db querier
}

// NewDocumentRepository constructs the repository.
func NewDocumentRepository(db querier) *DocumentRepository { return &DocumentRepository{db: db} }

// Save implements domain.DocumentRepository.
func (r *DocumentRepository) Save(ctx context.Context, d domain.Document) error {
	_, err := r.db.ExecContext(ctx, `
INSERT INTO documents.documents
  (id, source_id, country_code, title, mime_type, content_hash, source_document_id, parsed_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
ON CONFLICT (id) DO UPDATE SET
   title = EXCLUDED.title,
   mime_type = EXCLUDED.mime_type,
   content_hash = EXCLUDED.content_hash,
   parsed_at = EXCLUDED.parsed_at
`, d.ID, d.SourceID, d.CountryCode, d.Title, d.MimeType, d.ContentHash,
		d.SourceDocumentID, d.ParsedAt)
	if err != nil {
		return mapErr(err)
	}
	for _, p := range d.Pages {
		if _, err := r.db.ExecContext(ctx, `
INSERT INTO documents.pages (id, document_id, page_number, text)
VALUES ($1,$2,$3,$4)
ON CONFLICT (id) DO UPDATE SET text = EXCLUDED.text
`, p.ID, p.DocumentID, p.PageNumber, p.Text); err != nil {
			return mapErr(err)
		}
	}
	for _, c := range d.Chunks {
		if _, err := r.db.ExecContext(ctx, `
INSERT INTO documents.chunks
  (id, document_id, page_number, section_id, "offset", text, token_estimate)
VALUES ($1,$2,$3,$4,$5,$6,$7)
ON CONFLICT (id) DO UPDATE SET text = EXCLUDED.text
`, c.ID, c.DocumentID, c.PageNumber, c.SectionID, c.Offset, c.Text, c.TokenEstimate); err != nil {
			return mapErr(err)
		}
	}
	return nil
}

// Get implements domain.DocumentRepository.
func (r *DocumentRepository) Get(ctx context.Context, id string) (*domain.Document, error) {
	row := r.db.QueryRowContext(ctx, `
SELECT id, source_id, country_code, title, mime_type, content_hash,
       COALESCE(source_document_id, ''), parsed_at
  FROM documents.documents WHERE id=$1`, id)
	var d domain.Document
	if err := row.Scan(&d.ID, &d.SourceID, &d.CountryCode, &d.Title, &d.MimeType,
		&d.ContentHash, &d.SourceDocumentID, &d.ParsedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, contracts.ErrNotFound{Kind: "document", ID: id}
		}
		return nil, fmt.Errorf("scan document: %w", err)
	}
	// Load pages + chunks for the document.
	pages, err := r.loadPages(ctx, id)
	if err != nil {
		return nil, err
	}
	chunks, err := r.loadChunks(ctx, id)
	if err != nil {
		return nil, err
	}
	d.Pages = pages
	d.Chunks = chunks
	return &d, nil
}

// loadPages loads all pages for a document, ordered by page number.
func (r *DocumentRepository) loadPages(ctx context.Context, id string) ([]domain.DocumentPage, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT id, document_id, page_number, text FROM documents.pages
 WHERE document_id=$1 ORDER BY page_number ASC`, id)
	if err != nil {
		return nil, mapErr(err)
	}
	defer rows.Close()
	out := make([]domain.DocumentPage, 0)
	for rows.Next() {
		var p domain.DocumentPage
		if err := rows.Scan(&p.ID, &p.DocumentID, &p.PageNumber, &p.Text); err != nil {
			return nil, fmt.Errorf("scan page: %w", err)
		}
		out = append(out, p)
	}
	return out, nil
}

// loadChunks loads all chunks for a document.
func (r *DocumentRepository) loadChunks(ctx context.Context, id string) ([]domain.DocumentChunk, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT id, document_id, page_number, COALESCE(section_id,''), "offset", text, token_estimate
  FROM documents.chunks WHERE document_id=$1 ORDER BY page_number ASC, "offset" ASC`, id)
	if err != nil {
		return nil, mapErr(err)
	}
	defer rows.Close()
	out := make([]domain.DocumentChunk, 0)
	for rows.Next() {
		var c domain.DocumentChunk
		if err := rows.Scan(&c.ID, &c.DocumentID, &c.PageNumber, &c.SectionID,
			&c.Offset, &c.Text, &c.TokenEstimate); err != nil {
			return nil, fmt.Errorf("scan chunk: %w", err)
		}
		out = append(out, c)
	}
	return out, nil
}

// ListBySource implements domain.DocumentRepository.
func (r *DocumentRepository) ListBySource(ctx context.Context, sourceID string) ([]domain.Document, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT id, source_id, country_code, title, mime_type, content_hash,
       COALESCE(source_document_id, ''), parsed_at
  FROM documents.documents WHERE source_id=$1 ORDER BY parsed_at DESC`, sourceID)
	if err != nil {
		return nil, mapErr(err)
	}
	defer rows.Close()
	out := make([]domain.Document, 0)
	for rows.Next() {
		var d domain.Document
		if err := rows.Scan(&d.ID, &d.SourceID, &d.CountryCode, &d.Title, &d.MimeType,
			&d.ContentHash, &d.SourceDocumentID, &d.ParsedAt); err != nil {
			return nil, fmt.Errorf("scan document row: %w", err)
		}
		out = append(out, d)
	}
	return out, nil
}

// mapErr maps a database error into a typed contracts error.
func mapErr(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return contracts.ErrNotFound{Kind: "row", ID: ""}
	}
	return fmt.Errorf("postgres: %w", err)
}
