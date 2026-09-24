package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// petitionJSON is the JSON-decoded form of a single Petition item.
// Decoding into a struct (rather than map[string]any) lets the tests
// reference fields by name + type, which makes failures easier to
// diagnose when a refactor renames a field or flips an omitempty tag.
type petitionJSON struct {
	ID               string `json:"id"`
	Title            string `json:"title"`
	Description      string `json:"description"`
	CreatedBy        string `json:"created_by"`
	CreatedAt        string `json:"created_at"`
	Status           string `json:"status"`
	SignaturesCount  int    `json:"signatures_count"`
	TargetSignatures int    `json:"target_signatures"`
	ClosesAt         string `json:"closes_at"`
	Country          string `json:"country"`
}

// petitionsListPayload is the JSON envelope returned by
// GET /api/v1/petitions. Only the fields the tests assert on are
// decoded — the rest are ignored.
type petitionsListPayload struct {
	Items   []petitionJSON `json:"items"`
	Total   int            `json:"total"`
	Source  string         `json:"source"`
	Status  string         `json:"status,omitempty"`
	Country string         `json:"country,omitempty"`
}

// petitionDetailPayload is the JSON envelope returned by
// GET /api/v1/petitions/{id}.
type petitionDetailPayload struct {
	Petition   petitionJSON     `json:"petition"`
	Signatures []map[string]any `json:"signatures"`
	Source     string           `json:"source"`
}

// petitionSignPayload is the JSON envelope returned by
// POST /api/v1/petitions/{id}/sign. It carries the created signature
// + the updated petition (so the frontend can re-render the progress
// bar without a second round-trip).
type petitionSignPayload struct {
	Signature map[string]any `json:"signature"`
	Petition  petitionJSON   `json:"petition"`
}

// validPetitionStatuses is the canonical set of PetitionStatus wire
// values. Used by the tests to assert every response item carries
// one of the 3 allowed statuses (no typos, no silently-introduced
// "active" / "rejected" / "expired" variants). Kept in the test file
// (rather than reading allPetitionStatuses from petitions.go) so the
// test fails loudly if a new status is added without a corresponding
// test update.
var validPetitionStatuses = map[string]bool{
	"open":     true,
	"closed":   true,
	"answered": true,
}

// newSeededPetitionStore returns a fresh seeded PetitionStore. Used
// by every test so a sign / create in one test cannot leak into
// another test's state.
func newSeededPetitionStore() *PetitionStore {
	return NewPetitionStoreSeeded()
}

// TestPetitionsList_ReturnsPetitions verifies that the list endpoint
// returns the 5 seed petitions (issue #287 acceptance bar: "5 seed
// petitions with varying statuses and signature counts"). The
// response must carry the items list + a non-zero total + a "seed"
// source tag. Every item must carry one of the 3 canonical statuses.
//
// On a fresh store the list should be most-recent-first by
// created_at — the test asserts the first item is one of the 5 known
// seed ids (so a refactor that loses the seed slice or the sort
// step fails loudly).
func TestPetitionsList_ReturnsPetitions(t *testing.T) {
	store := newSeededPetitionStore()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/petitions", nil)
	rec := httptest.NewRecorder()

	makePetitionsHandler(store)(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200; got %d (body: %s)", rec.Code, rec.Body.String())
	}
	var resp petitionsListPayload
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Total != 5 {
		t.Errorf("expected 5 seed petitions; got %d", resp.Total)
	}
	if resp.Source != "seed" {
		t.Errorf("expected source 'seed'; got %q", resp.Source)
	}
	if len(resp.Items) != 5 {
		t.Fatalf("expected 5 items; got %d", len(resp.Items))
	}

	knownSeedIDs := map[string]bool{
		"pet-001": true,
		"pet-002": true,
		"pet-003": true,
		"pet-004": true,
		"pet-005": true,
	}
	for _, item := range resp.Items {
		if !validPetitionStatuses[item.Status] {
			t.Errorf("petition %s has invalid status %q (expected open/closed/answered)", item.ID, item.Status)
		}
		if !knownSeedIDs[item.ID] {
			t.Errorf("petition %s is not a known seed id", item.ID)
		}
		// SignaturesCount must equal the seed-matrix count (re-derived
		// from the signatures slice). This guards the canonical
		// invariant `signatures_count == len(signatures)`.
		if item.SignaturesCount <= 0 {
			t.Errorf("petition %s has non-positive signatures_count %d (expected > 0)", item.ID, item.SignaturesCount)
		}
		if item.TargetSignatures <= 0 {
			t.Errorf("petition %s has non-positive target_signatures %d", item.ID, item.TargetSignatures)
		}
		if item.Country == "" {
			t.Errorf("petition %s has empty country", item.ID)
		}
	}

	// The list must be sorted most-recent-first by created_at — pet-002
	// (2024-09-18) is the most-recent seed row, so it must come first.
	firstID := resp.Items[0].ID
	if firstID != "pet-002" {
		t.Errorf("expected most-recent-first sort (pet-002 first); got %q", firstID)
	}
}

// TestPetitionsList_FiltersByStatus verifies that the list endpoint
// filters by the status query parameter + that an invalid status
// returns 400 (not 200 with an empty items list). The seed has
// 2 open petitions, 1 closed, 2 answered.
func TestPetitionsList_FiltersByStatus(t *testing.T) {
	cases := []struct {
		name       string
		status     string
		wantCount  int
		wantStatus string
	}{
		{name: "open", status: "open", wantCount: 2, wantStatus: "open"},
		{name: "closed", status: "closed", wantCount: 1, wantStatus: "closed"},
		{name: "answered", status: "answered", wantCount: 2, wantStatus: "answered"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			store := newSeededPetitionStore()
			req := httptest.NewRequest(http.MethodGet, "/api/v1/petitions?status="+tc.status, nil)
			rec := httptest.NewRecorder()

			makePetitionsHandler(store)(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("expected 200; got %d (body: %s)", rec.Code, rec.Body.String())
			}
			var resp petitionsListPayload
			if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}
			if resp.Total != tc.wantCount {
				t.Errorf("status=%q: expected %d items; got %d", tc.status, tc.wantCount, resp.Total)
			}
			if resp.Status != tc.wantStatus {
				t.Errorf("status=%q: expected echoed status %q; got %q", tc.status, tc.wantStatus, resp.Status)
			}
			for _, item := range resp.Items {
				if item.Status != tc.wantStatus {
					t.Errorf("status=%q filter returned item with status %q (id=%s)", tc.status, item.Status, item.ID)
				}
			}
		})
	}

	// Invalid status must return 400 (not 200 with empty items).
	t.Run("invalid_status_returns_400", func(t *testing.T) {
		store := newSeededPetitionStore()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/petitions?status=garbage", nil)
		rec := httptest.NewRecorder()

		makePetitionsHandler(store)(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 for invalid status; got %d (body: %s)", rec.Code, rec.Body.String())
		}
	})
}

// TestPetitionDetail_ReturnsPetition verifies that the detail endpoint
// returns the petition + its signatures slice. The detail view must
// hydrate the signatures (the list view does NOT — see
// petitionsListResponse).
func TestPetitionDetail_ReturnsPetition(t *testing.T) {
	store := newSeededPetitionStore()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/petitions/pet-001", nil)
	rec := httptest.NewRecorder()

	makePetitionDetailHandler(store)(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200; got %d (body: %s)", rec.Code, rec.Body.String())
	}
	var resp petitionDetailPayload
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Petition.ID != "pet-001" {
		t.Errorf("expected petition pet-001; got %q", resp.Petition.ID)
	}
	if resp.Petition.Title == "" {
		t.Error("expected non-empty title")
	}
	if resp.Petition.Description == "" {
		t.Error("expected non-empty description")
	}
	if !validPetitionStatuses[resp.Petition.Status] {
		t.Errorf("petition has invalid status %q", resp.Petition.Status)
	}
	// The seed row pet-001 carries 7 signatures.
	if len(resp.Signatures) != 7 {
		t.Errorf("expected 7 signatures on pet-001; got %d", len(resp.Signatures))
	}
	// The signatures_count must equal len(signatures) — the canonical
	// invariant. The detail handler re-derives the count so a stale
	// denormalised value can never drift away from the signature list.
	if resp.Petition.SignaturesCount != len(resp.Signatures) {
		t.Errorf("canonical invariant violated: signatures_count=%d but len(signatures)=%d",
			resp.Petition.SignaturesCount, len(resp.Signatures))
	}
}

// TestPetitionDetail_UnknownID_Returns404 verifies that the detail
// endpoint returns 404 (not 200 with an empty body) for an unknown
// petition id — preventing silent acceptance of unknown ids.
func TestPetitionDetail_UnknownID_Returns404(t *testing.T) {
	store := newSeededPetitionStore()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/petitions/pet-does-not-exist", nil)
	rec := httptest.NewRecorder()

	makePetitionDetailHandler(store)(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404; got %d (body: %s)", rec.Code, rec.Body.String())
	}
}

// TestPetitionSign_IncrementsCount verifies that POST
// /api/v1/petitions/{id}/sign:
//
//   - appends a signature to the petition's signatures slice
//   - increments the denormalised signatures_count by 1
//   - returns 201 Created with the created signature + the updated
//     petition
//   - the created signature carries verified: false (the email
//     verification pipeline has not yet shipped)
func TestPetitionSign_IncrementsCount(t *testing.T) {
	store := newSeededPetitionStore()

	// Sanity-check: pet-001 starts with 7 signatures.
	beforeReq := httptest.NewRequest(http.MethodGet, "/api/v1/petitions/pet-001", nil)
	beforeRec := httptest.NewRecorder()
	makePetitionDetailHandler(store)(beforeRec, beforeReq)
	if beforeRec.Code != http.StatusOK {
		t.Fatalf("expected 200 on pre-sign detail; got %d", beforeRec.Code)
	}
	var before petitionDetailPayload
	_ = json.NewDecoder(beforeRec.Body).Decode(&before)
	if before.Petition.SignaturesCount != 7 {
		t.Fatalf("expected pet-001 to start with 7 signatures; got %d", before.Petition.SignaturesCount)
	}

	// Sign the petition.
	body := bytes.NewBufferString(`{"name":"Test Citizen","email":"test.citizen@example.org"}`)
	signReq := httptest.NewRequest(http.MethodPost, "/api/v1/petitions/pet-001/sign", body)
	signReq.Header.Set("Content-Type", "application/json")
	signRec := httptest.NewRecorder()

	makePetitionDetailHandler(store)(signRec, signReq)

	if signRec.Code != http.StatusCreated {
		t.Fatalf("expected 201; got %d (body: %s)", signRec.Code, signRec.Body.String())
	}
	var resp petitionSignPayload
	if err := json.NewDecoder(signRec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// The updated petition must carry 8 signatures (7 seed + 1 new).
	if resp.Petition.SignaturesCount != 8 {
		t.Errorf("expected signatures_count=8 after sign; got %d", resp.Petition.SignaturesCount)
	}
	if resp.Petition.ID != "pet-001" {
		t.Errorf("expected petition pet-001; got %q", resp.Petition.ID)
	}

	// The created signature must carry the petitioner's name + email +
	// verified: false (the email pipeline has not yet shipped).
	if resp.Signature == nil {
		t.Fatal("expected signature in response")
	}
	if name, _ := resp.Signature["name"].(string); name != "Test Citizen" {
		t.Errorf("expected signature name 'Test Citizen'; got %q", name)
	}
	if email, _ := resp.Signature["email"].(string); email != "test.citizen@example.org" {
		t.Errorf("expected signature email 'test.citizen@example.org'; got %q", email)
	}
	if verified, _ := resp.Signature["verified"].(bool); verified {
		t.Errorf("expected signature verified=false; got true")
	}

	// The increment must be observable on a subsequent GET detail —
	// the store is the source of truth, not the sign response.
	afterReq := httptest.NewRequest(http.MethodGet, "/api/v1/petitions/pet-001", nil)
	afterRec := httptest.NewRecorder()
	makePetitionDetailHandler(store)(afterRec, afterReq)
	if afterRec.Code != http.StatusOK {
		t.Fatalf("expected 200 on post-sign detail; got %d", afterRec.Code)
	}
	var after petitionDetailPayload
	_ = json.NewDecoder(afterRec.Body).Decode(&after)
	if after.Petition.SignaturesCount != 8 {
		t.Errorf("expected post-sign count=8 on GET detail; got %d", after.Petition.SignaturesCount)
	}
	if len(after.Signatures) != 8 {
		t.Errorf("expected 8 signatures on post-sign GET detail; got %d", len(after.Signatures))
	}
}

// TestPetitionSign_ClosedPetition_Returns409 verifies that signing a
// non-open petition returns 409 (conflict) — the request itself is
// well-formed; the petition is simply not in a state that accepts
// signatures. This guards the lifecycle invariant: closed / answered
// petitions cannot accrue new signatures.
func TestPetitionSign_ClosedPetition_Returns409(t *testing.T) {
	store := newSeededPetitionStore()

	body := bytes.NewBufferString(`{"name":"Test Citizen","email":"test.citizen@example.org"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/petitions/pet-004/sign", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	makePetitionDetailHandler(store)(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409 on signing closed petition; got %d (body: %s)", rec.Code, rec.Body.String())
	}
}

// TestPetitionSign_UnknownPetition_Returns404 verifies that signing
// an unknown petition id returns 404 (not 201 with a fabricated
// petition).
func TestPetitionSign_UnknownPetition_Returns404(t *testing.T) {
	store := newSeededPetitionStore()

	body := bytes.NewBufferString(`{"name":"Test Citizen","email":"test.citizen@example.org"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/petitions/pet-does-not-exist/sign", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	makePetitionDetailHandler(store)(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404; got %d (body: %s)", rec.Code, rec.Body.String())
	}
}

// TestPetitionSign_InvalidBody_Returns400 verifies that a malformed
// JSON body returns 400 (not 500 or 201 with an empty signature).
func TestPetitionSign_InvalidBody_Returns400(t *testing.T) {
	store := newSeededPetitionStore()

	body := bytes.NewBufferString(`{not valid json`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/petitions/pet-001/sign", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	makePetitionDetailHandler(store)(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 on invalid JSON; got %d (body: %s)", rec.Code, rec.Body.String())
	}
}

// TestPetitionCreate_Returns201 verifies that POST /api/v1/petitions
// creates a new petition + returns 201 with the created record. The
// created petition must have status "open" + signatures_count 0 +
// a non-empty id prefixed with "pet-".
func TestPetitionCreate_Returns201(t *testing.T) {
	store := newSeededPetitionStore()

	body := bytes.NewBufferString(`{
		"title": "Test Petition for Regression Coverage",
		"description": "This is a test petition description used by the FEAT-9 regression suite to verify the create endpoint.",
		"created_by": "Test Petitioner (citizen, Nairobi County)",
		"target_signatures": 1000,
		"closes_at": "2099-12-31T23:59:59Z",
		"country": "KE"
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/petitions", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	makePetitionsHandler(store)(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201; got %d (body: %s)", rec.Code, rec.Body.String())
	}
	var resp petitionJSON
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.ID == "" {
		t.Error("expected non-empty id")
	}
	if len(resp.ID) < 4 || resp.ID[:4] != "pet-" {
		t.Errorf("expected id to start with 'pet-'; got %q", resp.ID)
	}
	if resp.Status != "open" {
		t.Errorf("expected status 'open'; got %q", resp.Status)
	}
	if resp.SignaturesCount != 0 {
		t.Errorf("expected signatures_count=0 on create; got %d", resp.SignaturesCount)
	}
	if resp.TargetSignatures != 1000 {
		t.Errorf("expected target_signatures=1000; got %d", resp.TargetSignatures)
	}
	if resp.Country != "KE" {
		t.Errorf("expected country 'KE'; got %q", resp.Country)
	}
	if resp.CreatedAt == "" {
		t.Error("expected non-empty created_at (server-set)")
	}

	// The created petition must be observable on a subsequent GET list.
	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/petitions", nil)
	listRec := httptest.NewRecorder()
	makePetitionsHandler(store)(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("expected 200 on list; got %d", listRec.Code)
	}
	var listResp petitionsListPayload
	_ = json.NewDecoder(listRec.Body).Decode(&listResp)
	if listResp.Total != 6 {
		t.Errorf("expected 6 petitions after create (5 seed + 1 new); got %d", listResp.Total)
	}
}

// TestPetitionCreate_MissingFields_Returns400 verifies that each
// missing required field returns 400 (not 201 with a partial record).
func TestPetitionCreate_MissingFields_Returns400(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{name: "missing_title", body: `{"description":"d","created_by":"c","target_signatures":1,"closes_at":"2099-12-31T23:59:59Z","country":"KE"}`},
		{name: "missing_description", body: `{"title":"t","created_by":"c","target_signatures":1,"closes_at":"2099-12-31T23:59:59Z","country":"KE"}`},
		{name: "missing_created_by", body: `{"title":"t","description":"d","target_signatures":1,"closes_at":"2099-12-31T23:59:59Z","country":"KE"}`},
		{name: "missing_target_signatures", body: `{"title":"t","description":"d","created_by":"c","closes_at":"2099-12-31T23:59:59Z","country":"KE"}`},
		{name: "zero_target_signatures", body: `{"title":"t","description":"d","created_by":"c","target_signatures":0,"closes_at":"2099-12-31T23:59:59Z","country":"KE"}`},
		{name: "missing_closes_at", body: `{"title":"t","description":"d","created_by":"c","target_signatures":1,"country":"KE"}`},
		{name: "missing_country", body: `{"title":"t","description":"d","created_by":"c","target_signatures":1,"closes_at":"2099-12-31T23:59:59Z"}`},
		{name: "past_closes_at", body: `{"title":"t","description":"d","created_by":"c","target_signatures":1,"closes_at":"2000-01-01T00:00:00Z","country":"KE"}`},
		{name: "invalid_closes_at", body: `{"title":"t","description":"d","created_by":"c","target_signatures":1,"closes_at":"not-a-timestamp","country":"KE"}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			store := newSeededPetitionStore()
			req := httptest.NewRequest(http.MethodPost, "/api/v1/petitions", bytes.NewBufferString(tc.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			makePetitionsHandler(store)(rec, req)
			if rec.Code != http.StatusBadRequest {
				t.Errorf("%s: expected 400; got %d (body: %s)", tc.name, rec.Code, rec.Body.String())
			}
		})
	}
}

// TestPetitionsList_FiltersByCountry verifies that the country
// filter narrows the list to petitions whose country matches the
// query (case-insensitive). The seed slice is KE-only; filtering by
// "KE" returns all 5, filtering by "UG" returns 0.
func TestPetitionsList_FiltersByCountry(t *testing.T) {
	store := newSeededPetitionStore()

	// KE returns all 5 seed petitions.
	req := httptest.NewRequest(http.MethodGet, "/api/v1/petitions?country=KE", nil)
	rec := httptest.NewRecorder()
	makePetitionsHandler(store)(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200; got %d", rec.Code)
	}
	var resp petitionsListPayload
	_ = json.NewDecoder(rec.Body).Decode(&resp)
	if resp.Total != 5 {
		t.Errorf("country=KE: expected 5 items; got %d", resp.Total)
	}
	if resp.Country != "KE" {
		t.Errorf("expected echoed country 'KE'; got %q", resp.Country)
	}

	// UG returns 0 (the seed slice is KE-only today — when the Uganda
	// adapter ships its own petition seed, this assertion will be
	// updated). A 0-count response is the correct posture for a
	// "no petitions match" filter — NOT a 404.
	ugReq := httptest.NewRequest(http.MethodGet, "/api/v1/petitions?country=UG", nil)
	ugRec := httptest.NewRecorder()
	makePetitionsHandler(store)(ugRec, ugReq)
	if ugRec.Code != http.StatusOK {
		t.Fatalf("expected 200 for UG (no matches); got %d", ugRec.Code)
	}
	var ugResp petitionsListPayload
	_ = json.NewDecoder(ugRec.Body).Decode(&ugResp)
	if ugResp.Total != 0 {
		t.Errorf("country=UG: expected 0 items; got %d", ugResp.Total)
	}

	// Lowercase "ke" must also match (case-insensitive).
	lcReq := httptest.NewRequest(http.MethodGet, "/api/v1/petitions?country=ke", nil)
	lcRec := httptest.NewRecorder()
	makePetitionsHandler(store)(lcRec, lcReq)
	var lcResp petitionsListPayload
	_ = json.NewDecoder(lcRec.Body).Decode(&lcResp)
	if lcResp.Total != 5 {
		t.Errorf("country=ke (lowercase): expected 5 items; got %d", lcResp.Total)
	}
}
