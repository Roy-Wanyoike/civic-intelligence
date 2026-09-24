package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Roy-Wanyoike/civic-intelligence/adapters/kenya/kenya_law"
	"github.com/Roy-Wanyoike/civic-intelligence/adapters/kenya/kenya_seed"
)

// amendmentsPayload is the JSON envelope returned by
// GET /api/v1/bills/{id}/amendments. Only the fields the handler tests
// assert on are decoded — the full Amendment shape is covered by the
// JSON-shape test below.
type amendmentsPayload struct {
	BillID     string      `json:"bill_id"`
	Amendments []Amendment `json:"amendments"`
	Total      int         `json:"total"`
	Source     string      `json:"source"`
	Note       string      `json:"note,omitempty"`
}

// TestAmendmentsByBill_ReturnsAmendments verifies the happy path:
// given a known seed Bill ID, the handler returns 200 with the seed
// amendments attached to that Bill. The mock adapter returns an error
// (simulating a dead upstream) so the test exercises the issue #265
// seed-fallback contract — a known Bill must always return 200, never
// 503, even when kenyalaw.org is unreachable.
//
// The test picks the first seed Bill (which has 4 amendments per
// sampleAmendmentSpecs) so the assertion is deterministic and
// decoupled from slug-truncation drift in the seed slice.
func TestAmendmentsByBill_ReturnsAmendments(t *testing.T) {
	if len(kenya_seed.SampleBills) == 0 {
		t.Fatal("seed slice must be non-empty for this test")
	}
	seedBill := kenya_seed.SampleBills[0]
	billID := seedBill.SourceID

	// Mock adapter returns an error so the handler takes its seed-fallback
	// branch (issue #265 contract — never 503 for a known Bill).
	adapter := &mockBillsAdapter{err: errors.New("simulated upstream 503")}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/bills/"+billID+"/amendments", nil)
	rr := httptest.NewRecorder()

	handleBillAmendments(rr, req, adapter, billID)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body=%s)", rr.Code, rr.Body.String())
	}

	var resp amendmentsPayload
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}

	// The response must echo the resolved Bill ID.
	if resp.BillID != billID {
		t.Errorf("expected bill_id=%s, got %q", billID, resp.BillID)
	}

	// Amendments are seed-only for FEAT-15 — the response must surface
	// `source: "seed"` so callers can distinguish seed data from future
	// live amendments (issue #19).
	if resp.Source != "seed" {
		t.Errorf("expected source=seed, got %q", resp.Source)
	}

	// The first seed Bill has exactly 4 amendments per sampleAmendmentSpecs.
	// Assert the count rather than the full slice so the test stays
	// robust to (intentional) summary-text edits.
	if resp.Total != 4 {
		t.Errorf("expected 4 amendments for the first seed Bill, got %d", resp.Total)
	}
	if len(resp.Amendments) != 4 {
		t.Fatalf("expected 4 amendments in the response, got %d", len(resp.Amendments))
	}

	// Every amendment in the response must carry the parent Bill's ID +
	// a non-empty title + a valid status. This guards against a
	// regression where sampleAmendments is built without resolving the
	// bill_id (e.g., because the billIndex is out of range).
	for i, a := range resp.Amendments {
		if a.BillID != billID {
			t.Errorf("amendment %d: expected bill_id=%s, got %q", i, billID, a.BillID)
		}
		if a.Title == "" {
			t.Errorf("amendment %d: expected non-empty title", i)
		}
		if a.ID == "" {
			t.Errorf("amendment %d: expected non-empty id", i)
		}
		if a.Status != AmendmentStatusProposed &&
			a.Status != AmendmentStatusAccepted &&
			a.Status != AmendmentStatusRejected {
			t.Errorf("amendment %d: invalid status %q", i, a.Status)
		}
	}

	// The seed slice must contain at least one amendment per status
	// (proposed + accepted + rejected) so the frontend's status-badge
	// rendering has all three states to display.
	seenStatuses := map[AmendmentStatus]bool{}
	for _, a := range resp.Amendments {
		seenStatuses[a.Status] = true
	}
	// Note: the first seed Bill's 4 amendments cover proposed + accepted +
	// rejected (per sampleAmendmentSpecs). Assert all three are present
	// so a future spec edit that drops one state is caught.
	for _, want := range []AmendmentStatus{
		AmendmentStatusProposed,
		AmendmentStatusAccepted,
		AmendmentStatusRejected,
	} {
		if !seenStatuses[want] {
			t.Errorf("expected at least one amendment with status %q in the response", want)
		}
	}
}

// TestAmendmentsByBill_UnknownBill_Returns404 verifies the unknown-Bill
// path: when the live crawl fails AND the requested ID is not in the
// seed slice, the handler returns 404 (not 503) — the seed fallback
// only satisfies known IDs. Mirrors the
// TestBillDetailHandler_FallbackToSeed_NotFoundInSeed contract in
// bills_test.go.
func TestAmendmentsByBill_UnknownBill_Returns404(t *testing.T) {
	adapter := &mockBillsAdapter{err: errors.New("simulated timeout")}
	billID := "this-id-does-not-exist-anywhere-2026"
	req := httptest.NewRequest(http.MethodGet, "/api/v1/bills/"+billID+"/amendments", nil)
	rr := httptest.NewRecorder()

	handleBillAmendments(rr, req, adapter, billID)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404 (unknown Bill), got %d (body=%s)", rr.Code, rr.Body.String())
	}
}

// TestAmendmentsByBill_EmptyID_Returns400 verifies the handler returns
// 400 (bad_request) when the Bill ID is empty — guards against the
// /api/v1/bills//amendments path silently matching the seed-fallback
// branch via the substring helper.
func TestAmendmentsByBill_EmptyID_Returns400(t *testing.T) {
	adapter := &mockBillsAdapter{err: errors.New("simulated 503")}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/bills//amendments", nil)
	rr := httptest.NewRecorder()

	handleBillAmendments(rr, req, adapter, "")

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 (empty Bill ID), got %d (body=%s)", rr.Code, rr.Body.String())
	}
}

// TestAmendmentsByBill_KnownBillButNoAmendments verifies the handler
// returns 200 with an empty amendments array (NOT 404) when the Bill
// exists in the seed slice but has no recorded amendments. This guards
// against a regression where the handler treats "no amendments" as
// "Bill unknown" — an empty amendments array is a valid response shape.
func TestAmendmentsByBill_KnownBillButNoAmendments(t *testing.T) {
	if len(kenya_seed.SampleBills) < 2 {
		t.Fatal("seed slice must have ≥2 entries for this test")
	}
	// Pick a seed Bill that has NO amendments in sampleAmendmentSpecs.
	// Bills 0, 1, 3 carry amendments; every other seed Bill is amendment-free.
	// Bill 2 is the County Governments Retirement Scheme Bill 2026 — safe
	// pick (it is NOT referenced in sampleAmendmentSpecs).
	amendmentFreeBill := kenya_seed.SampleBills[2]
	billID := amendmentFreeBill.SourceID

	adapter := &mockBillsAdapter{err: errors.New("simulated 503")}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/bills/"+billID+"/amendments", nil)
	rr := httptest.NewRecorder()

	handleBillAmendments(rr, req, adapter, billID)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 (known Bill, no amendments), got %d (body=%s)", rr.Code, rr.Body.String())
	}
	var resp amendmentsPayload
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Total != 0 {
		t.Errorf("expected 0 amendments for amendment-free Bill, got %d", resp.Total)
	}
	if resp.Amendments == nil {
		t.Errorf("expected non-nil amendments slice (empty) for amendment-free Bill, got nil")
	}
	if resp.BillID != billID {
		t.Errorf("expected bill_id=%s, got %q", billID, resp.BillID)
	}
}

// TestAmendmentsByBill_LiveDiscovery_StillReturnsSeedAmendments verifies
// that when the live crawl succeeds and the Bill is found, the handler
// still returns the seed amendments (issue #293 contract: amendments
// are seed-only for FEAT-15 — the `source` field is always "seed"
// even when the Bill itself was discovered live). Catches a regression
// where the handler returns 404 or an empty array because it looked
// for amendments keyed on the live-discovered Bill's ID without
// falling back to the seed-keyed amendments.
func TestAmendmentsByBill_LiveDiscovery_StillReturnsSeedAmendments(t *testing.T) {
	if len(kenya_seed.SampleBills) == 0 {
		t.Fatal("seed slice must be non-empty for this test")
	}
	seedBill := kenya_seed.SampleBills[0]
	billID := seedBill.SourceID

	// Mock adapter returns the seed Bill via live discovery (the live
	// path returns the Bill with the same SourceID, simulating a
	// successful live crawl that happens to match a seeded Bill).
	adapter := &mockBillsAdapter{bills: []kenya_law.BillCandidate{seedBill}}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/bills/"+billID+"/amendments", nil)
	rr := httptest.NewRecorder()

	handleBillAmendments(rr, req, adapter, billID)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body=%s)", rr.Code, rr.Body.String())
	}
	var resp amendmentsPayload
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	// Amendments are seed-only for FEAT-15 — the response source must
	// be "seed" even when the Bill itself was discovered live. This is
	// the contract that lets the frontend's amendments tab render
	// consistently while issue #19 is in flight.
	if resp.Source != "seed" {
		t.Errorf("expected source=seed (amendments are seed-only), got %q", resp.Source)
	}
	if resp.Total != 4 {
		t.Errorf("expected 4 seed amendments for the first seed Bill, got %d", resp.Total)
	}
}

// TestAmendment_JSONShape verifies the JSON shape of the Amendment
// struct — every field must serialize with the documented JSON key so
// the frontend + OpenAPI spec + integration tests can rely on a
// stable contract.
func TestAmendment_JSONShape(t *testing.T) {
	a := Amendment{
		ID:         "amend-001",
		BillID:     "ke-bill-2026-09-07-sample",
		Title:      "Clause 14(2) — increase contribution rate to 8%",
		ProposedBy: "person-001",
		ProposedAt: "2026-09-15T10:30:00Z",
		Status:     AmendmentStatusProposed,
		Summary:    "Raises the member contribution rate from 6% to 8%.",
		SourceURL:  "https://new.kenyalaw.org/akn/ke/bill/na/2026-09-07/sample/eng@2026-09-07",
	}
	b, err := json.Marshal(a)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	s := string(b)
	required := []string{
		`"id":"amend-001"`,
		`"bill_id":"ke-bill-2026-09-07-sample"`,
		`"title":"Clause 14(2) — increase contribution rate to 8%"`,
		`"proposed_by":"person-001"`,
		`"proposed_at":"2026-09-15T10:30:00Z"`,
		`"status":"proposed"`,
		`"summary":"Raises the member contribution rate from 6% to 8%."`,
		`"source_url":"https://new.kenyalaw.org/akn/ke/bill/na/2026-09-07/sample/eng@2026-09-07"`,
	}
	for _, key := range required {
		if !strings.Contains(s, key) {
			t.Errorf("expected JSON to contain %q, got %s", key, s)
		}
	}
}

// TestSampleAmendments_DistributedAcrossThreeBills verifies the seed
// slice's structural contract: 10 amendments distributed across 3
// sample Bills. Catches a regression where someone trims the seed
// slice (or collapses it onto one Bill) without realising it powers
// the amendments tab's realistic mix of in-flight + disposed
// amendments.
func TestSampleAmendments_DistributedAcrossThreeBills(t *testing.T) {
	if len(sampleAmendments) != 10 {
		t.Errorf("expected 10 seed amendments, got %d", len(sampleAmendments))
	}
	bills := map[string]int{}
	for _, a := range sampleAmendments {
		if a.BillID == "" {
			t.Errorf("amendment %q has empty bill_id (billIndex out of range?)", a.ID)
		}
		bills[a.BillID]++
	}
	if len(bills) != 3 {
		t.Errorf("expected amendments distributed across 3 Bills, got %d", len(bills))
	}
	// Every status must be represented (proposed + accepted + rejected)
	// so the frontend's status-badge rendering has all three states.
	statuses := map[AmendmentStatus]bool{}
	for _, a := range sampleAmendments {
		statuses[a.Status] = true
	}
	for _, want := range []AmendmentStatus{
		AmendmentStatusProposed,
		AmendmentStatusAccepted,
		AmendmentStatusRejected,
	} {
		if !statuses[want] {
			t.Errorf("expected at least one amendment with status %q in the seed slice", want)
		}
	}
}

// TestFindAmendmentsByBillID verifies the lookup helper. Returns the
// amendments whose BillID exactly matches (no substring matching —
// amendments are tied to a specific Bill version per ADR-0011).
func TestFindAmendmentsByBillID(t *testing.T) {
	if len(sampleAmendments) == 0 {
		t.Fatal("sampleAmendments must be non-empty for this test")
	}
	first := sampleAmendments[0]
	got := findAmendmentsByBillID(first.BillID)
	if len(got) == 0 {
		t.Errorf("expected ≥1 amendment for Bill %q, got 0", first.BillID)
	}
	// Empty ID returns an empty (non-nil) slice — no panic.
	if empty := findAmendmentsByBillID(""); empty == nil {
		t.Errorf("expected non-nil slice for empty bill_id, got nil")
	}
	// Non-matching ID returns an empty (non-nil) slice.
	if empty := findAmendmentsByBillID("nonexistent-bill-id-xyz"); len(empty) != 0 {
		t.Errorf("expected 0 amendments for unknown bill_id, got %d", len(empty))
	}
}
