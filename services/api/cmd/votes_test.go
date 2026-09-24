package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// voteRecordJSON is the JSON-decoded form of a single VoteRecord item.
// Decoding into a struct (rather than map[string]any) lets the tests
// reference fields by name + type, which makes failures easier to
// diagnose when a refactor renames a field or flips an omitempty tag.
type voteRecordJSON struct {
	PersonID   string `json:"person_id"`
	PersonName string `json:"person_name"`
	Vote       string `json:"vote"`
	BillID     string `json:"bill_id"`
	BillTitle  string `json:"bill_title"`
	Division   string `json:"division"`
	Date       string `json:"date"`
	SourceURL  string `json:"source_url"`
}

// votesByPersonPayload is the JSON envelope returned by
// GET /api/v1/people/{id}/votes. Only the fields the tests assert on
// are decoded — the rest are ignored.
type votesByPersonPayload struct {
	PersonID     string           `json:"person_id"`
	Name         string           `json:"name"`
	Items        []voteRecordJSON `json:"items"`
	Total        int              `json:"total"`
	Source       string           `json:"source"`
	ScorecardURL string           `json:"scorecard_url"`
}

// voteCountsJSON is the JSON-decoded form of the Counts block returned
// by /bills/{id}/votes.
type voteCountsJSON struct {
	Aye     int `json:"aye"`
	Nay     int `json:"nay"`
	Abstain int `json:"abstain"`
	Absent  int `json:"absent"`
	Total   int `json:"total"`
}

// votesByBillPayload is the JSON envelope returned by
// GET /api/v1/bills/{id}/votes.
type votesByBillPayload struct {
	BillID    string           `json:"bill_id"`
	BillTitle string           `json:"bill_title"`
	Division  string           `json:"division"`
	Date      string           `json:"date"`
	SourceURL string           `json:"source_url"`
	Counts    voteCountsJSON   `json:"counts"`
	Items     []voteRecordJSON `json:"items"`
	Total     int              `json:"total"`
	Source    string           `json:"source"`
}

// validVoteKinds is the canonical set of VoteKind wire values. Used by
// the tests to assert every response item carries one of the 4 allowed
// kinds (no typos, no silently-introduced "present" / "no" / "yes"
// variants). Kept in the test file (rather than reading allVoteKinds
// from votes.go) so the test fails loudly if a new kind is added
// without a corresponding test update.
var validVoteKinds = map[string]bool{
	"aye":     true,
	"nay":     true,
	"abstain": true,
	"absent":  true,
}

// expectedVotesForPerson returns the expected seed-matrix vote records
// for the given MP, sorted most-recent-first by Bill date. The test
// uses this to cross-check the response items against the seed matrix
// (catches a refactor that loses the sort or drops a row).
func expectedVotesForPerson(t *testing.T, personID string) []voteRecordJSON {
	t.Helper()
	// Build the expected set directly from the seed slice
	// (seedVoteRecords) — this is the same source of truth the
	// handler uses, so a refactor that breaks the seed init will fail
	// this test loudly rather than producing a silent mismatch.
	want := make([]voteRecordJSON, 0, 6)
	for _, v := range seedVoteRecords {
		if v.PersonID != personID {
			continue
		}
		want = append(want, voteRecordJSON{
			PersonID:   v.PersonID,
			PersonName: v.PersonName,
			Vote:       string(v.Vote),
			BillID:     v.BillID,
			BillTitle:  v.BillTitle,
			Division:   v.Division,
			Date:       v.Date,
			SourceURL:  v.SourceURL,
		})
	}
	if len(want) == 0 {
		t.Fatalf("expected seed slice to attribute ≥1 vote to %s", personID)
	}
	return want
}

// expectedVotesForBill returns the expected seed-matrix vote records
// for the given Bill, in seed-matrix order (person-001 first,
// person-005 last). Used by the per-Bill test to cross-check the
// response items against the seed matrix.
func expectedVotesForBill(t *testing.T, billID string) []voteRecordJSON {
	t.Helper()
	want := make([]voteRecordJSON, 0, 5)
	for _, v := range seedVoteRecords {
		if v.BillID != billID {
			continue
		}
		want = append(want, voteRecordJSON{
			PersonID:   v.PersonID,
			PersonName: v.PersonName,
			Vote:       string(v.Vote),
			BillID:     v.BillID,
			BillTitle:  v.BillTitle,
			Division:   v.Division,
			Date:       v.Date,
			SourceURL:  v.SourceURL,
		})
	}
	if len(want) == 0 {
		t.Fatalf("expected seed slice to have ≥1 vote record for %s", billID)
	}
	return want
}

// TestVotesByPerson_ReturnsVotes verifies the new
// /api/v1/people/{id}/votes endpoint (issue #284) returns the MP's
// voting history — every record carrying all 8 required fields, every
// Vote value one of the 4 allowed kinds, and the items list sorted
// most-recent-first by division date.
//
// The test drives handlePeople (not the inner handler) so the dispatch
// logic in handlePeople is exercised — the same way
// TestBillsByPersonHandler_ReturnsSponsoredBills does.
func TestVotesByPerson_ReturnsVotes(t *testing.T) {
	const personID = "person-001"
	want := expectedVotesForPerson(t, personID)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/people/"+personID+"/votes", nil)
	rr := httptest.NewRecorder()
	handlePeople(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 for /api/v1/people/%s/votes, got %d (body=%s)", personID, rr.Code, rr.Body.String())
	}

	var resp votesByPersonPayload
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.PersonID != personID {
		t.Errorf("expected person_id=%q, got %q", personID, resp.PersonID)
	}
	// person-001 is Kimani Ichung'wah (sampleScorecards[0]).
	if resp.Name != "Kimani Ichung'wah" {
		t.Errorf("expected name=%q, got %q", "Kimani Ichung'wah", resp.Name)
	}
	if resp.Source != "seed" {
		t.Errorf("expected source=seed (issue #284 acceptance: seed data only), got %q", resp.Source)
	}
	wantScorecardURL := "/api/v1/people/" + personID + "/scorecard"
	if resp.ScorecardURL != wantScorecardURL {
		t.Errorf("expected scorecard_url=%q, got %q", wantScorecardURL, resp.ScorecardURL)
	}

	// Issue #284 acceptance: 30 vote records across 5 MPs × 6 Bills →
	// every MP has exactly 6 vote records.
	if resp.Total != len(want) {
		t.Errorf("expected total=%d (matches seed matrix), got %d", len(want), resp.Total)
	}
	if len(resp.Items) != len(want) {
		t.Fatalf("expected %d items, got %d", len(want), len(resp.Items))
	}

	// Assert every item carries all 8 required fields + every Vote
	// value is one of the 4 allowed kinds + items are sorted
	// most-recent-first.
	previousDate := ""
	for i, item := range resp.Items {
		if item.PersonID != personID {
			t.Errorf("item[%d]: expected person_id=%q, got %q (every item in this list MUST belong to the requested MP)", i, personID, item.PersonID)
		}
		if item.PersonName == "" {
			t.Errorf("item[%d]: expected person_name to be non-empty", i)
		}
		if !validVoteKinds[item.Vote] {
			t.Errorf("item[%d]: expected vote to be one of aye/nay/abstain/absent, got %q", i, item.Vote)
		}
		if item.BillID == "" {
			t.Errorf("item[%d]: expected bill_id to be non-empty", i)
		}
		if item.BillTitle == "" {
			t.Errorf("item[%d]: expected bill_title to be non-empty", i)
		}
		if item.Division == "" {
			t.Errorf("item[%d]: expected division to be non-empty", i)
		}
		if item.Date == "" {
			t.Errorf("item[%d]: expected date to be non-empty", i)
		}
		if item.SourceURL == "" {
			t.Errorf("item[%d]: expected source_url to be non-empty", i)
		}
		// Items sorted most-recent-first (date descending).
		if previousDate != "" && item.Date > previousDate {
			t.Errorf("item[%d]: dates not sorted most-recent-first (previous=%q, current=%q)", i, previousDate, item.Date)
		}
		previousDate = item.Date
	}

	// Cross-check the response items against the seed matrix (catches
	// a refactor that loses a row or scrambles the per-MP filter).
	wantByBill := map[string]voteRecordJSON{}
	for _, w := range want {
		wantByBill[w.BillID] = w
	}
	for _, got := range resp.Items {
		w, ok := wantByBill[got.BillID]
		if !ok {
			t.Errorf("response item with bill_id=%q not in seed matrix for %s", got.BillID, personID)
			continue
		}
		if got.Vote != w.Vote {
			t.Errorf("bill_id=%q: expected vote=%q, got %q", got.BillID, w.Vote, got.Vote)
		}
		if got.BillTitle != w.BillTitle {
			t.Errorf("bill_id=%q: expected bill_title=%q, got %q", got.BillID, w.BillTitle, got.BillTitle)
		}
		if got.Division != w.Division {
			t.Errorf("bill_id=%q: expected division=%q, got %q", got.BillID, w.Division, got.Division)
		}
		if got.Date != w.Date {
			t.Errorf("bill_id=%q: expected date=%q, got %q", got.BillID, w.Date, got.Date)
		}
		if got.SourceURL != w.SourceURL {
			t.Errorf("bill_id=%q: expected source_url=%q, got %q", got.BillID, w.SourceURL, got.SourceURL)
		}
	}
}

// TestVotesByPerson_UnknownPerson_Returns404 verifies the per-MP
// endpoint returns 404 (NOT an empty 200 items list) when the person
// ID does not match a sample MP. The MP must exist in
// sampleScorecards before any vote records are surfaced — this guards
// the "every datum is sourced" invariant (a typo in a person ID must
// 404, not silently return an empty items list that looks like a
// legitimate "no votes" response).
func TestVotesByPerson_UnknownPerson_Returns404(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/people/person-does-not-exist/votes", nil)
	rr := httptest.NewRecorder()
	handlePeople(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for unknown person, got %d (body=%s)", rr.Code, rr.Body.String())
	}

	// Assert the canonical error envelope shape (error + message).
	var errResp struct {
		Error   string `json:"error"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("decode error envelope: %v", err)
	}
	if errResp.Error != "not_found" {
		t.Errorf("expected error=not_found, got %q", errResp.Error)
	}
	if errResp.Message == "" {
		t.Errorf("expected non-empty message, got empty")
	}
}

// TestVotesByBill_ReturnsDivision verifies the new
// /api/v1/bills/{id}/votes endpoint (issue #284) returns the division
// summary — the per-division tally (aye / nay / abstain / absent /
// total) + the per-MP roll-call items list.
//
// The test drives makeBillDetailHandler (not the inner handler) so the
// sub-route dispatch is exercised — the same pattern as the bills
// detail handler tests.
func TestVotesByBill_ReturnsDivision(t *testing.T) {
	const billID = "bill-vote-001"
	want := expectedVotesForBill(t, billID)

	adapter := &mockBillsAdapter{}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/bills/"+billID+"/votes", nil)
	req.URL.Path = "/api/v1/bills/" + billID + "/votes"
	rr := httptest.NewRecorder()

	makeBillDetailHandler(adapter, "")(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 for /api/v1/bills/%s/votes, got %d (body=%s)", billID, rr.Code, rr.Body.String())
	}

	var resp votesByBillPayload
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.BillID != billID {
		t.Errorf("expected bill_id=%q, got %q", billID, resp.BillID)
	}
	// bill-vote-001 is the Affordable Housing Bill, Third Reading.
	if resp.BillTitle == "" {
		t.Errorf("expected bill_title to be non-empty")
	}
	if resp.Division == "" {
		t.Errorf("expected division to be non-empty")
	}
	if resp.Date == "" {
		t.Errorf("expected date to be non-empty")
	}
	if resp.SourceURL == "" {
		t.Errorf("expected source_url to be non-empty")
	}
	if resp.Source != "seed" {
		t.Errorf("expected source=seed (issue #284 acceptance: seed data only), got %q", resp.Source)
	}

	// The Counts block MUST match the seed matrix's own tally (i.e.
	// re-counting the want slice should produce the same numbers as
	// the response).
	wantCounts := voteCountsJSON{}
	for _, w := range want {
		switch w.Vote {
		case "aye":
			wantCounts.Aye++
		case "nay":
			wantCounts.Nay++
		case "abstain":
			wantCounts.Abstain++
		case "absent":
			wantCounts.Absent++
		}
	}
	wantCounts.Total = len(want)
	if resp.Counts != wantCounts {
		t.Errorf("expected counts=%+v (re-counted from seed matrix), got %+v", wantCounts, resp.Counts)
	}
	// aye + nay + abstain + absent MUST sum to total (the invariant
	// the frontend relies on to render the division bar).
	sum := resp.Counts.Aye + resp.Counts.Nay + resp.Counts.Abstain + resp.Counts.Absent
	if sum != resp.Counts.Total {
		t.Errorf("expected aye+nay+abstain+absent=%d to equal total=%d", sum, resp.Counts.Total)
	}
	// Issue #284 acceptance: every Bill MUST surface all 4 vote kinds
	// (aye / nay / abstain / absent) across the 5 sample MPs so the
	// /bills/{id}/votes summary surfaces non-zero counts in every
	// cell. (This is a property of the seed matrix; if the matrix
	// ever drops a kind, this assertion catches it.)
	if resp.Counts.Aye == 0 {
		t.Errorf("expected ≥1 aye vote on %s (seed matrix property)", billID)
	}
	if resp.Counts.Nay == 0 {
		t.Errorf("expected ≥1 nay vote on %s (seed matrix property)", billID)
	}
	if resp.Counts.Abstain == 0 {
		t.Errorf("expected ≥1 abstain vote on %s (seed matrix property)", billID)
	}
	if resp.Counts.Absent == 0 {
		t.Errorf("expected ≥1 absent vote on %s (seed matrix property)", billID)
	}

	if resp.Total != len(want) {
		t.Errorf("expected total=%d (matches seed matrix), got %d", len(want), resp.Total)
	}
	if len(resp.Items) != len(want) {
		t.Fatalf("expected %d items, got %d", len(want), len(resp.Items))
	}

	// Every MP appears exactly once in the items list (5 MPs × 1
	// vote per Bill = 5 items).
	seenPersons := map[string]bool{}
	for i, item := range resp.Items {
		if item.BillID != billID {
			t.Errorf("item[%d]: expected bill_id=%q, got %q (every item in this list MUST belong to the requested Bill)", i, billID, item.BillID)
		}
		if item.PersonID == "" {
			t.Errorf("item[%d]: expected person_id to be non-empty", i)
		}
		if !validVoteKinds[item.Vote] {
			t.Errorf("item[%d]: expected vote to be one of aye/nay/abstain/absent, got %q", i, item.Vote)
		}
		if seenPersons[item.PersonID] {
			t.Errorf("item[%d]: person_id=%q appears more than once in the items list", i, item.PersonID)
		}
		seenPersons[item.PersonID] = true
	}

	// Cross-check the response items against the seed matrix (catches
	// a refactor that loses an MP or scrambles the per-Bill filter).
	wantByPerson := map[string]voteRecordJSON{}
	for _, w := range want {
		wantByPerson[w.PersonID] = w
	}
	for _, got := range resp.Items {
		w, ok := wantByPerson[got.PersonID]
		if !ok {
			t.Errorf("response item with person_id=%q not in seed matrix for %s", got.PersonID, billID)
			continue
		}
		if got.Vote != w.Vote {
			t.Errorf("person_id=%q: expected vote=%q, got %q", got.PersonID, w.Vote, got.Vote)
		}
		if got.PersonName != w.PersonName {
			t.Errorf("person_id=%q: expected person_name=%q, got %q", got.PersonID, w.PersonName, got.PersonName)
		}
	}
}

// TestVotesByBill_UnknownBill_Returns404 verifies the per-Bill
// endpoint returns 404 when the Bill ID does not match a sample
// vote Bill. Same "every datum is sourced" invariant as the per-MP
// handler — a typo in a Bill ID must 404, not silently return an
// empty items list.
func TestVotesByBill_UnknownBill_Returns404(t *testing.T) {
	adapter := &mockBillsAdapter{}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/bills/bill-vote-does-not-exist/votes", nil)
	req.URL.Path = "/api/v1/bills/bill-vote-does-not-exist/votes"
	rr := httptest.NewRecorder()

	makeBillDetailHandler(adapter, "")(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for unknown bill, got %d (body=%s)", rr.Code, rr.Body.String())
	}

	// Assert the canonical error envelope shape (error + message).
	var errResp struct {
		Error   string `json:"error"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("decode error envelope: %v", err)
	}
	if errResp.Error != "not_found" {
		t.Errorf("expected error=not_found, got %q", errResp.Error)
	}
	if errResp.Message == "" {
		t.Errorf("expected non-empty message, got empty")
	}
}
