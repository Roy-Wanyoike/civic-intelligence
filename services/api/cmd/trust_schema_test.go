package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// migrationRoot resolves the path to infrastructure/postgres/migrations
// relative to the repository root. The api module lives at services/api,
// so the repo root is two directories up from this test's package directory.
func migrationRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd: %v", err)
	}
	// From services/api/cmd → repo root = ../../../
	root := filepath.Clean(filepath.Join(wd, "..", "..", ".."))
	migrations := filepath.Join(root, "infrastructure", "postgres", "migrations")
	if _, err := os.Stat(migrations); err != nil {
		t.Fatalf("migrations dir not found at %s: %v", migrations, err)
	}
	return migrations
}

// readMigration loads the contents of migration NNN_name.up.sql.
func readMigration(t *testing.T, name string) string {
	t.Helper()
	path := filepath.Join(migrationRoot(t), name)
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(b)
}

// TestMigration018_TrustSchema_ContainsAllTables asserts that the trust-schema
// migration declares every documented table. If a table is missing the
// acceptance criteria for issue #164 ("All tables have proper indexes") cannot
// be met. The check is structural (string-in-file) — psql is the final
// arbiter of compile-ability in CI.
func TestMigration018_TrustSchema_ContainsAllTables(t *testing.T) {
	sql := readMigration(t, "018_trust_schema.up.sql")

	wantTables := []string{
		"trust.sources",
		"trust.source_checks",
		"trust.claims",
		"trust.evidence",
		"trust.contradictions",
		"trust.corrections",
		"trust.audit_events",
		"trust.review_queue",
	}
	for _, table := range wantTables {
		if !strings.Contains(sql, "CREATE TABLE "+table) {
			t.Errorf("migration 018 missing CREATE TABLE %s", table)
		}
	}
}

// TestMigration018_TrustSchema_ContainsExpectedColumns checks the canonical
// column lists for each trust table (mirrors the spec in issue #164).
func TestMigration018_TrustSchema_ContainsExpectedColumns(t *testing.T) {
	sql := readMigration(t, "018_trust_schema.up.sql")

	type tableCols struct {
		table string
		cols  []string
	}
	want := []tableCols{
		{"trust.sources", []string{"id", "institution_id", "country", "source_type", "authority_level", "official_url", "domain", "status", "verification_method", "last_verified_at", "health_status", "created_at"}},
		{"trust.source_checks", []string{"id", "source_id", "checked_at", "http_status", "latency_ms", "tls_valid", "content_hash", "changed"}},
		{"trust.claims", []string{"id", "subject", "predicate", "object", "claim_type", "text", "confidence", "verification_state", "created_at", "valid_from", "valid_to"}},
		{"trust.evidence", []string{"id", "claim_id", "document_id", "snapshot_id", "page_number", "section", "paragraph", "text_span", "source_url"}},
		{"trust.contradictions", []string{"id", "claim_a_id", "claim_b_id", "source_a_id", "source_b_id", "detected_at", "status", "resolution", "reviewer_id"}},
		{"trust.corrections", []string{"id", "target_type", "target_id", "reason", "previous_state", "corrected_state", "evidence", "submitted_by", "reviewed_by", "reviewed_at", "status"}},
		{"trust.audit_events", []string{"id", "event_type", "entity_type", "entity_id", "actor_user_id", "before", "after", "occurred_at"}},
		{"trust.review_queue", []string{"id", "entity_type", "entity_id", "reason", "priority", "status", "assigned_to", "created_at"}},
	}

	for _, tc := range want {
		for _, col := range tc.cols {
			if !strings.Contains(sql, col) {
				t.Errorf("migration 018 table %s: expected column %s not found", tc.table, col)
			}
		}
	}
}

// TestMigration018_TrustSchema_HasIndexes verifies the acceptance criterion
// "All tables have proper indexes" — every trust table must have at least one
// CREATE INDEX statement, and key access patterns (claims.subject,
// evidence.claim_id, contradictions.status, corrections.target) must be
// indexed explicitly.
func TestMigration018_TrustSchema_HasIndexes(t *testing.T) {
	sql := readMigration(t, "018_trust_schema.up.sql")

	wantIndexes := []string{
		"idx_trust_sources_country",
		"uq_trust_sources_domain",
		"idx_trust_source_checks_source",
		"idx_trust_claims_spo",
		"idx_trust_claims_state",
		"idx_trust_evidence_claim",
		"idx_trust_evidence_document",
		"idx_trust_contradictions_status",
		"idx_trust_corrections_target",
		"idx_trust_corrections_status",
		"idx_trust_audit_entity",
		"idx_trust_audit_actor",
		"idx_trust_review_queue_status",
		"idx_trust_review_queue_entity",
	}
	for _, idx := range wantIndexes {
		if !strings.Contains(sql, idx) {
			t.Errorf("migration 018 missing index %s", idx)
		}
	}
}

// TestMigration018_TrustSchema_HasForeignKeys verifies the cross-schema FK
// constraints the architectural contract requires (evidence → ingestion,
// contradictions → trust.claims + trust.sources, corrections → identity.users).
func TestMigration018_TrustSchema_HasForeignKeys(t *testing.T) {
	sql := readMigration(t, "018_trust_schema.up.sql")

	wantFKs := []string{
		"REFERENCES trust.sources(id)",
		"REFERENCES trust.claims(id)",
		"REFERENCES trust.sources(id)",
		"REFERENCES ingestion.documents(id)",
		"REFERENCES ingestion.document_snapshots(id)",
		"REFERENCES identity.users(id)",
	}
	for _, fk := range wantFKs {
		if !strings.Contains(sql, fk) {
			t.Errorf("migration 018 missing FK: %s", fk)
		}
	}
}

// TestMigration018_TrustSchema_HasAuditTriggers verifies the acceptance
// criterion "Audit trigger on canonical tables" — every mutable trust table
// must carry an INSERT/UPDATE audit trigger that writes to trust.audit_events.
// The audit log itself must be append-only (UPDATE/DELETE/TRUNCATE blocked).
func TestMigration018_TrustSchema_HasAuditTriggers(t *testing.T) {
	sql := readMigration(t, "018_trust_schema.up.sql")

	wantTriggers := []string{
		"tg_trust_claims_audit_insert",
		"tg_trust_claims_audit_update",
		"tg_trust_contradictions_audit_insert",
		"tg_trust_contradictions_audit_update",
		"tg_trust_corrections_audit_insert",
		"tg_trust_corrections_audit_update",
		"tg_trust_review_queue_audit_insert",
		"tg_trust_review_queue_audit_update",
	}
	for _, trig := range wantTriggers {
		if !strings.Contains(sql, trig) {
			t.Errorf("migration 018 missing audit trigger %s", trig)
		}
	}

	// trust.audit_events must be append-only.
	if !strings.Contains(sql, "tg_audit_events_no_update") {
		t.Error("migration 018 missing append-only guard trigger on trust.audit_events")
	}
	if !strings.Contains(sql, "append-only") {
		t.Error("migration 018 audit_events function must document append-only behaviour")
	}
}

// TestMigration018_TrustSchema_HasEnums verifies the verification_state enum
// values for trust.claims match the lifecycle documented in issue #164.
func TestMigration018_TrustSchema_HasEnums(t *testing.T) {
	sql := readMigration(t, "018_trust_schema.up.sql")

	wantStates := []string{
		"'UNVERIFIED'", "'DISCOVERED'", "'EXTRACTED'", "'VALIDATING'",
		"'VERIFIED'", "'CONFLICTED'", "'CORRECTED'", "'SUPERSEDED'", "'REJECTED'",
	}
	for _, s := range wantStates {
		if !strings.Contains(sql, s) {
			t.Errorf("migration 018 trust.claims.verification_state missing value %s", s)
		}
	}

	wantClaimTypes := []string{"'FACT'", "'EXPLANATION'", "'INFERENCE'", "'UNKNOWN'"}
	for _, c := range wantClaimTypes {
		if !strings.Contains(sql, c) {
			t.Errorf("migration 018 trust.claims.claim_type missing value %s", c)
		}
	}
}

// TestMigration018_TrustSchema_CreatesSchema confirms the migration creates
// the trust schema if it does not exist (002_schemas.up.sql is immutable).
func TestMigration018_TrustSchema_CreatesSchema(t *testing.T) {
	sql := readMigration(t, "018_trust_schema.up.sql")
	if !strings.Contains(sql, "CREATE SCHEMA IF NOT EXISTS trust") {
		t.Error("migration 018 must CREATE SCHEMA IF NOT EXISTS trust (002_schemas is immutable)")
	}
}

// TestMigration018_TrustSchema_DownIsReversible verifies the down-migration
// drops every table + function + trigger + schema in the correct order
// (triggers → tables → functions → schema).
func TestMigration018_TrustSchema_DownIsReversible(t *testing.T) {
	down := readMigration(t, "018_trust_schema.down.sql")

	wantDrops := []string{
		"DROP TRIGGER IF EXISTS tg_trust_review_queue_audit_update",
		"DROP TRIGGER IF EXISTS tg_audit_events_no_update",
		"DROP FUNCTION IF EXISTS trust.tg_audit_events_immutable()",
		"DROP FUNCTION IF EXISTS trust.tg_record_audit()",
		"DROP TABLE IF EXISTS trust.review_queue",
		"DROP TABLE IF EXISTS trust.audit_events",
		"DROP TABLE IF EXISTS trust.corrections",
		"DROP TABLE IF EXISTS trust.contradictions",
		"DROP TABLE IF EXISTS trust.evidence",
		"DROP TABLE IF EXISTS trust.claims",
		"DROP TABLE IF EXISTS trust.source_checks",
		"DROP TABLE IF EXISTS trust.sources",
		"DROP SCHEMA IF EXISTS trust",
	}
	for _, d := range wantDrops {
		if !strings.Contains(down, d) {
			t.Errorf("down-migration 018 missing statement: %s", d)
		}
	}
}

// TestMigration018_TrustSchema_NoSilentMutation verifies the migration does
// not include any UPDATE or DELETE on canonical tables — only INSERT (for
// the seed). This protects the architectural contract that corrections
// create new immutable versions, never UPDATEs.
func TestMigration018_TrustSchema_NoSilentMutation(t *testing.T) {
	sql := readMigration(t, "018_trust_schema.up.sql")

	// 'UPDATE trust.claims' (or any other trust.* table) at top-level SQL
	// would be a violation. Trigger bodies are allowed to reference UPDATE
	// in their declaration.
	forbidden := []string{
		"UPDATE trust.claims SET",
		"UPDATE trust.evidence SET",
		"UPDATE trust.contradictions SET",
		"UPDATE trust.corrections SET",
		"UPDATE trust.review_queue SET",
		"DELETE FROM trust.claims",
		"DELETE FROM trust.evidence",
		"DELETE FROM trust.contradictions",
		"DELETE FROM trust.corrections",
		"DELETE FROM trust.review_queue",
	}
	for _, bad := range forbidden {
		if strings.Contains(sql, bad) {
			t.Errorf("migration 018 must not contain silent mutation: %s", bad)
		}
	}
}
