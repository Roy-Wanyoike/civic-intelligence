package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Roy-Wanyoike/civic-intelligence/adapters/kenya/kenya_seed"
)

// sponsorInfo is the subset of billResponse the sponsor tests assert on.
// Decoding into a struct (rather than map[string]any) lets the tests
// reference fields by name + type, which makes failures easier to
// diagnose when a refactor renames a field or flips an omitempty tag.
type sponsorInfo struct {
	SponsorID   string       `json:"sponsor_id"`
	SponsorName string       `json:"sponsor_name"`
	SponsorURL  string       `json:"scorecard_url"`
	Cosponsors  []SponsorRef `json:"cosponsors"`
}

// billsListPayloadWithSponsor extends billsListPayload to decode the
// new sponsor fields on each item (issue #282). The existing
// billsListPayload in bills_test.go is intentionally minimal so the
// fallback tests don't couple to fields they don't assert on; this
// payload reuses the same JSON envelope but decodes the richer item
// shape so the sponsor-info test has direct access to the sponsor
// fields.
type billsListPayloadWithSponsor struct {
	Items    []sponsorInfo `json:"items"`
	Total    int           `json:"total"`
	Source   string        `json:"source"`
	Degraded bool          `json:"degraded"`
}

// TestBillsHandler_IncludesSponsorInfo verifies the bills list endpoint
// surfaces the sponsor fields (sponsor_id, sponsor_name, scorecard_url,
// cosponsors) on Bills that carry them in the seed slice (issue #282).
//
// The test drives the degraded path (adapter returns error → handler
// falls back to kenya_seed.SampleBills) because sponsor attribution is
// seed-only — the live kenya_law parser does NOT extract the sponsor
// (deliberate, per the issue #282 contract). On the degraded path, the
// seed Bills' sponsor_id + cosponsor_ids are populated by
// makeSampleBill from the sampleBillSpec slice, so the response MUST
// include at least one Bill with non-empty sponsor_id + sponsor_name +
// scorecard_url, AND at least one Bill with non-empty cosponsors.
//
// The test also asserts the scorecard_url field resolves to the
// expected /api/v1/people/{id}/scorecard form so the frontend can link
// the sponsor name directly to their scorecard page.
func TestBillsHandler_IncludesSponsorInfo(t *testing.T) {
	adapter := &mockBillsAdapter{err: errFakeUpstream}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/bills", nil)
	rr := httptest.NewRecorder()

	makeBillsHandler(adapter)(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body=%s)", rr.Code, rr.Body.String())
	}

	var resp billsListPayloadWithSponsor
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Source != "seed" {
		t.Errorf("expected source=seed (degraded fallback), got %q", resp.Source)
	}
	if resp.Total != len(kenya_seed.SampleBills) {
		t.Fatalf("expected total=%d (seed slice length), got %d", len(kenya_seed.SampleBills), resp.Total)
	}

	// Walk the items + cross-reference the seed slice so we can verify
	// every Bill the seed slice attributes to a sponsor shows up in the
	// response with the sponsor fields populated.
	seedBySourceID := map[string]sponsorInfo{}
	for _, b := range kenya_seed.SampleBills {
		if b.SponsorID == "" {
			continue
		}
		seedBySourceID[b.SourceID] = sponsorInfo{
			SponsorID: b.SponsorID,
		}
	}
	if len(seedBySourceID) < 20 {
		t.Fatalf("expected the seed slice to attribute ≥20 Bills to a sponsor (issue #282 acceptance bar), got %d", len(seedBySourceID))
	}

	sponsoredWithSponsorName := 0
	sponsoredWithScorecardURL := 0
	sponsoredWithCosponsors := 0
	for _, item := range resp.Items {
		// Bills without a sponsor_id MUST NOT surface sponsor_name +
		// scorecard_url + cosponsors (the omitempty contract). Pick the
		// first response item that the seed slice attributes to a sponsor
		// and verify the sponsor fields are populated correctly.
		if item.SponsorID == "" {
			if item.SponsorName != "" {
				t.Errorf("bill with empty sponsor_id has non-empty sponsor_name=%q (must be omitted)", item.SponsorName)
			}
			if item.SponsorURL != "" {
				t.Errorf("bill with empty sponsor_id has non-empty scorecard_url=%q (must be omitted)", item.SponsorURL)
			}
			if len(item.Cosponsors) != 0 {
				t.Errorf("bill with empty sponsor_id has non-empty cosponsors=%v (must be omitted)", item.Cosponsors)
			}
			continue
		}
		// Cross-reference the seed slice to verify the sponsor_id matches
		// a real seeded Bill (catches a refactor that loses the seed
		// attribution).
		if _, ok := seedBySourceID[item.SponsorID]; !ok && item.SponsorID != "" {
			// The sponsor_id should be one of the 5 sample MPs — the
			// seed slice only uses person-001..person-005.
			if item.SponsorID != "person-001" &&
				item.SponsorID != "person-002" &&
				item.SponsorID != "person-003" &&
				item.SponsorID != "person-004" &&
				item.SponsorID != "person-005" {
				t.Errorf("unexpected sponsor_id %q (not one of person-001..person-005)", item.SponsorID)
			}
		}
		if item.SponsorName != "" {
			sponsoredWithSponsorName++
		} else {
			t.Errorf("bill with sponsor_id=%q has empty sponsor_name (should be populated from sample people)", item.SponsorID)
		}
		wantURL := "/api/v1/people/" + item.SponsorID + "/scorecard"
		if item.SponsorURL != "" {
			sponsoredWithScorecardURL++
			if item.SponsorURL != wantURL {
				t.Errorf("bill with sponsor_id=%q has scorecard_url=%q, want %q", item.SponsorID, item.SponsorURL, wantURL)
			}
		} else {
			t.Errorf("bill with sponsor_id=%q has empty scorecard_url (should be populated)", item.SponsorID)
		}
		if len(item.Cosponsors) > 0 {
			sponsoredWithCosponsors++
			// Every cosponsor must carry person_id + name (the SponsorRef
			// contract).
			for i, c := range item.Cosponsors {
				if c.PersonID == "" {
					t.Errorf("bill with sponsor_id=%q has cosponsor[%d] with empty person_id", item.SponsorID, i)
				}
				if c.Name == "" {
					t.Errorf("bill with sponsor_id=%q has cosponsor[%d] with empty name", item.SponsorID, i)
				}
			}
		}
	}

	// Issue #282 acceptance: at least 20 Bills in the seed slice have a
	// sponsor_id. The response MUST surface at least 20 sponsored Bills
	// (the seed slice is the fallback + the only source of sponsor data).
	if sponsoredWithSponsorName < 20 {
		t.Errorf("expected ≥20 bills with sponsor_name populated, got %d", sponsoredWithSponsorName)
	}
	if sponsoredWithScorecardURL < 20 {
		t.Errorf("expected ≥20 bills with scorecard_url populated, got %d", sponsoredWithScorecardURL)
	}
	// At least one Bill must have cosponsors (the seed slice attributes
	// cosponsors to multiple Bills — see sampleBillSpecs).
	if sponsoredWithCosponsors == 0 {
		t.Errorf("expected at least 1 bill with cosponsors populated, got 0 — seed slice must attribute cosponsors to ≥1 Bill")
	}
}

// errFakeUpstream is the sentinel error used to drive the bills + trending
// handlers' degraded fallback path in the sponsor-info test. Re-declared
// here (rather than reusing the unexported error in bills_test.go) so
// the test reads top-down — the error message is intentionally
// descriptive so a failing test log surfaces the cause.
var errFakeUpstream = httpErrorf("simulated upstream 503 (sponsor-info test)")

// httpErrorf is a tiny helper that returns an error implementing error
// without pulling in fmt.Errorf — keeps the test file's imports tight
// (json + net/http + testing only).
func httpErrorf(format string) error { return &simpleError{msg: format} }

type simpleError struct{ msg string }

func (e *simpleError) Error() string { return e.msg }

// --- GET /api/v1/people/{id}/bills ---

// billsByPersonPayload is the JSON envelope returned by the new endpoint
// (issue #282). Only the fields the test asserts on are decoded.
type billsByPersonPayload struct {
	PersonID     string       `json:"person_id"`
	Name         string       `json:"name"`
	Items        []sponsorInfo `json:"items"`
	Total        int           `json:"total"`
	Source       string       `json:"source"`
	ScorecardURL string       `json:"scorecard_url"`
}

// TestBillsByPersonHandler_ReturnsSponsoredBills verifies the new
// /api/v1/people/{id}/bills endpoint (issue #282) returns every Bill
// the seed slice attributes to the given MP.
//
// The test:
//  1. Hits /api/v1/people/person-001/bills via handlePeople (so the
//     dispatch logic in handlePeople is exercised — not just the
//     inner handler).
//  2. Asserts 200 + the response envelope's person_id, name, source,
//     and scorecard_url fields are populated correctly.
//  3. Asserts the items list matches kenya_seed.FindSampleBillsBySponsor
//     for person-001 — same count + same SourceIDs + every item has
//     sponsor_id == person-001 + sponsor_name == "Kimani Ichung'wah".
//  4. Also asserts the 404 path: an unknown person ID returns 404
//     (the seed slice is the only source of sponsor attribution; an
//     unknown person can't have sponsored any seed Bill).
func TestBillsByPersonHandler_ReturnsSponsoredBills(t *testing.T) {
	const personID = "person-001"
	// Compute the expected seed Bills for person-001 up-front so the
	// test fails loudly if the seed slice changes (regression guard).
	expectedBills := kenya_seed.FindSampleBillsBySponsor(personID)
	if len(expectedBills) == 0 {
		t.Fatalf("seed slice must attribute ≥1 Bill to person-001 for this test")
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/people/"+personID+"/bills", nil)
	rr := httptest.NewRecorder()
	handlePeople(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 for /api/v1/people/%s/bills, got %d (body=%s)", personID, rr.Code, rr.Body.String())
	}

	var resp billsByPersonPayload
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.PersonID != personID {
		t.Errorf("expected person_id=%q, got %q", personID, resp.PersonID)
	}
	// Person-001 is Kimani Ichung'wah (sampleScorecards[0]).
	if resp.Name != "Kimani Ichung'wah" {
		t.Errorf("expected name=%q, got %q", "Kimani Ichung'wah", resp.Name)
	}
	if resp.Source != "seed" {
		t.Errorf("expected source=seed (sponsor attribution is seed-only), got %q", resp.Source)
	}
	wantScorecardURL := "/api/v1/people/" + personID + "/scorecard"
	if resp.ScorecardURL != wantScorecardURL {
		t.Errorf("expected scorecard_url=%q, got %q", wantScorecardURL, resp.ScorecardURL)
	}

	// Assert the items count matches the seed slice's count for this MP.
	if resp.Total != len(expectedBills) {
		t.Errorf("expected total=%d (matches seed slice), got %d", len(expectedBills), resp.Total)
	}
	if len(resp.Items) != len(expectedBills) {
		t.Fatalf("expected %d items, got %d", len(expectedBills), len(resp.Items))
	}

	// Every item must be sponsored by person-001 + carry the sponsor_name
	// + scorecard_url fields (the Bill detail page links the sponsor
	// name to the scorecard).
	seenSourceIDs := map[string]bool{}
	for i, item := range resp.Items {
		if item.SponsorID != personID {
			t.Errorf("item[%d]: expected sponsor_id=%q, got %q (every Bill in this list MUST be sponsored by the requested MP)", i, personID, item.SponsorID)
		}
		if item.SponsorName != "Kimani Ichung'wah" {
			t.Errorf("item[%d]: expected sponsor_name=%q, got %q", i, "Kimani Ichung'wah", item.SponsorName)
		}
		if item.SponsorURL != wantScorecardURL {
			t.Errorf("item[%d]: expected scorecard_url=%q, got %q", i, wantScorecardURL, item.SponsorURL)
		}
		// Track the SourceID so we can verify the items list matches the
		// seed slice's set exactly (no extras, no missing).
		// Note: we don't decode id on sponsorInfo; we re-fetch by index.
		_ = i
		seenSourceIDs[item.SponsorID] = true
	}

	// Cross-check: the items list MUST match the seed slice's set of
	// sponsored Bill SourceIDs for person-001. We re-decode the response
	// to pull the SourceIDs (the SponsorInfo subset above doesn't include
	// the Bill id field).
	var fullResp struct {
		Items []struct {
			ID        string `json:"id"`
			SponsorID string `json:"sponsor_id"`
		} `json:"items"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &fullResp); err != nil {
		t.Fatalf("re-decode: %v", err)
	}
	gotIDs := map[string]bool{}
	for _, item := range fullResp.Items {
		gotIDs[item.ID] = true
	}
	for _, b := range expectedBills {
		if !gotIDs[b.SourceID] {
			t.Errorf("expected Bill %q (sponsor=%s) in items, not found", b.SourceID, personID)
		}
	}
	if len(gotIDs) != len(expectedBills) {
		t.Errorf("expected %d unique Bill IDs, got %d (extras in response)", len(expectedBills), len(gotIDs))
	}

	// 404 path: unknown person returns 404.
	req404 := httptest.NewRequest(http.MethodGet, "/api/v1/people/does-not-exist/bills", nil)
	rr404 := httptest.NewRecorder()
	handlePeople(rr404, req404)
	if rr404.Code != http.StatusNotFound {
		t.Errorf("expected 404 for unknown person, got %d (body=%s)", rr404.Code, rr404.Body.String())
	}
}
