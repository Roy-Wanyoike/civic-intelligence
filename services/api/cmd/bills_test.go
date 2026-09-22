package main

import (
        "context"
        "encoding/json"
        "errors"
        "net/http"
        "net/http/httptest"
        "testing"
        "time"

        "github.com/Roy-Wanyoike/civic-intelligence/adapters/kenya/kenya_law"
        "github.com/Roy-Wanyoike/civic-intelligence/adapters/kenya/kenya_seed"
)

// mockBillsAdapter is a stub BillsAdapter used by the bills + trending
// handler tests. It returns the configured Bills slice + error on each
// DiscoverBills call, so tests can drive both the live-success path
// (nil error, real bills) and the degraded fallback path (non-nil
// error → handler must fall back to kenya_seed.SampleBills).
type mockBillsAdapter struct {
        bills []kenya_law.BillCandidate
        err   error
        // fetchBillHTML is returned by FetchBill; defaults to empty so the
        // detail handler takes its "ParseBillDetail failed → return what we
        // know from discovery" branch (which is the path the live-success
        // detail test exercises).
        fetchBillHTML string
        fetchBillErr  error
}

func (m *mockBillsAdapter) DiscoverBills(ctx context.Context) ([]kenya_law.BillCandidate, error) {
        return m.bills, m.err
}

func (m *mockBillsAdapter) FetchBill(ctx context.Context, url string) (string, error) {
        if m.fetchBillErr != nil {
                return "", m.fetchBillErr
        }
        return m.fetchBillHTML, nil
}

// billsListPayload is the JSON envelope returned by GET /api/v1/bills.
// Only the fields the fallback tests assert on are decoded.
type billsListPayload struct {
        Items    []billResponse `json:"items"`
        Total    int            `json:"total"`
        Page     int            `json:"page"`
        PageSize int            `json:"page_size"`
        Source   string         `json:"source"`
        Degraded bool           `json:"degraded"`
        Country  string         `json:"country"`
}

// trendingPayload is the JSON envelope returned by GET /api/v1/trending.
type trendingPayload struct {
        RecentlyPublished []billResponse `json:"recently_published"`
        Hot               []billResponse `json:"hot"`
        ApproachingFinal  []billResponse `json:"approaching_final"`
        TotalBills        int            `json:"total_bills"`
        Source            string         `json:"source"`
        Degraded          bool           `json:"degraded"`
}

// sampleLiveBills returns a small, deterministic Bills slice for the
// live-success tests. Two Bills is enough to exercise the response
// builder without coupling the test to the seed slice (which is the
// subject of the fallback tests).
func sampleLiveBills() []kenya_law.BillCandidate {
        pubDate, _ := time.Parse("2006-01-02", "2026-09-07")
        return []kenya_law.BillCandidate{
                {
                        URL:             "https://new.kenyalaw.org/akn/ke/bill/na/2026-09-07/the-housing-bill-2026/eng@2026-09-07",
                        Slug:            "the-housing-bill-2026",
                        Title:           "The Housing Bill 2026",
                        House:           "National Assembly",
                        PublicationDate: pubDate,
                        SourceID:        "ke-bill-2026-09-07-the-housing-bill-2026",
                },
        }
}

// --- GET /api/v1/bills ---

// TestBillsHandler_LiveSuccess_SourceIsLive verifies that when the
// adapter returns Bills successfully, the response is 200 with
// `source: "live"` and `degraded: false` (issue #265 acceptance: live
// path must surface the source field so clients can distinguish live
// data from the seed fallback without checking status codes).
func TestBillsHandler_LiveSuccess_SourceIsLive(t *testing.T) {
        adapter := &mockBillsAdapter{bills: sampleLiveBills()}
        req := httptest.NewRequest(http.MethodGet, "/api/v1/bills", nil)
        rr := httptest.NewRecorder()

        makeBillsHandler(adapter)(rr, req)

        if rr.Code != http.StatusOK {
                t.Fatalf("expected 200, got %d (body=%s)", rr.Code, rr.Body.String())
        }

        var resp billsListPayload
        if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
                t.Fatalf("decode: %v", err)
        }
        if resp.Source != "live" {
                t.Errorf("expected source=live, got %q", resp.Source)
        }
        if resp.Degraded {
                t.Errorf("expected degraded=false, got true")
        }
        if resp.Total != len(sampleLiveBills()) {
                t.Errorf("expected total=%d, got %d", len(sampleLiveBills()), resp.Total)
        }
        if len(resp.Items) != len(sampleLiveBills()) {
                t.Fatalf("expected %d items, got %d", len(sampleLiveBills()), len(resp.Items))
        }
        // Per-item source must also be "live" — the billResponse struct
        // carries it so detail + list responses share a single shape.
        if resp.Items[0].Source != "live" {
                t.Errorf("expected item source=live, got %q", resp.Items[0].Source)
        }
        if resp.Items[0].Degraded {
                t.Errorf("expected item degraded=false, got true")
        }
}

// TestBillsHandler_FallbackToSeed_OnAdapterError verifies that when the
// adapter returns an error (network failure, 403, timeout, sandbox IP
// block), the handler:
//   - logs the failure ("falling back to seed data"),
//   - falls back to kenya_seed.SampleBills,
//   - returns 200 (NOT 503),
//   - surfaces `source: "seed"` + `degraded: true` so clients can tell
//     the data is from the fallback rather than live.
func TestBillsHandler_FallbackToSeed_OnAdapterError(t *testing.T) {
        adapter := &mockBillsAdapter{err: errors.New("simulated upstream 503")}
        req := httptest.NewRequest(http.MethodGet, "/api/v1/bills", nil)
        rr := httptest.NewRecorder()

        makeBillsHandler(adapter)(rr, req)

        if rr.Code != http.StatusOK {
                t.Fatalf("expected 200 (NOT 503) on adapter error, got %d (body=%s)", rr.Code, rr.Body.String())
        }

        var resp billsListPayload
        if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
                t.Fatalf("decode: %v", err)
        }
        if resp.Source != "seed" {
                t.Errorf("expected source=seed, got %q", resp.Source)
        }
        if !resp.Degraded {
                t.Errorf("expected degraded=true, got false")
        }
        if resp.Total != len(kenya_seed.SampleBills) {
                t.Errorf("expected total=%d (seed slice length), got %d", len(kenya_seed.SampleBills), resp.Total)
        }
        if len(resp.Items) != len(kenya_seed.SampleBills) {
                t.Errorf("expected %d items (seed slice), got %d", len(kenya_seed.SampleBills), len(resp.Items))
        }
        if len(resp.Items) < 50 {
                t.Errorf("expected seed slice to be comprehensive (50+ bills), got %d", len(resp.Items))
        }
        // Per-item source must also be "seed" + degraded=true so the API
        // contract is uniform across list + detail responses.
        if resp.Items[0].Source != "seed" {
                t.Errorf("expected item source=seed, got %q", resp.Items[0].Source)
        }
        if !resp.Items[0].Degraded {
                t.Errorf("expected item degraded=true, got false")
        }
}

// TestBillsHandler_FallbackToSeed_PreservesCountryHeader verifies the
// X-Civic-Country header is still echoed on the degraded path — the
// fallback must not drop cross-cutting middleware behaviour.
func TestBillsHandler_FallbackToSeed_PreservesCountryHeader(t *testing.T) {
        adapter := &mockBillsAdapter{err: errors.New("upstream down")}
        req := httptest.NewRequest(http.MethodGet, "/api/v1/bills", nil)
        req.Header.Set("X-Civic-Country", "UG")
        rr := httptest.NewRecorder()

        makeBillsHandler(adapter)(rr, req)

        if rr.Code != http.StatusOK {
                t.Fatalf("expected 200, got %d", rr.Code)
        }
        var resp billsListPayload
        if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
                t.Fatalf("decode: %v", err)
        }
        if resp.Country != "UG" {
                t.Errorf("expected country=UG (echoed from header), got %q", resp.Country)
        }
}

// --- GET /api/v1/trending ---

// TestTrendingHandler_LiveSuccess_SourceIsLive verifies the trending
// endpoint returns `source: "live"` + `degraded: false` when the live
// crawl succeeds (issue #265 acceptance).
func TestTrendingHandler_LiveSuccess_SourceIsLive(t *testing.T) {
        adapter := &mockBillsAdapter{bills: sampleLiveBills()}
        req := httptest.NewRequest(http.MethodGet, "/api/v1/trending", nil)
        rr := httptest.NewRecorder()

        makeTrendingHandler(adapter)(rr, req)

        if rr.Code != http.StatusOK {
                t.Fatalf("expected 200, got %d (body=%s)", rr.Code, rr.Body.String())
        }
        var resp trendingPayload
        if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
                t.Fatalf("decode: %v", err)
        }
        if resp.Source != "live" {
                t.Errorf("expected source=live, got %q", resp.Source)
        }
        if resp.Degraded {
                t.Errorf("expected degraded=false, got true")
        }
}

// TestTrendingHandler_FallbackToSeed_OnAdapterError verifies the
// trending endpoint falls back to seed data and returns 200 (NOT 503)
// with `source: "seed"` + `degraded: true` when the live crawl fails.
func TestTrendingHandler_FallbackToSeed_OnAdapterError(t *testing.T) {
        adapter := &mockBillsAdapter{err: errors.New("simulated timeout")}
        req := httptest.NewRequest(http.MethodGet, "/api/v1/trending", nil)
        rr := httptest.NewRecorder()

        makeTrendingHandler(adapter)(rr, req)

        if rr.Code != http.StatusOK {
                t.Fatalf("expected 200 (NOT 503) on adapter error, got %d (body=%s)", rr.Code, rr.Body.String())
        }
        var resp trendingPayload
        if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
                t.Fatalf("decode: %v", err)
        }
        if resp.Source != "seed" {
                t.Errorf("expected source=seed, got %q", resp.Source)
        }
        if !resp.Degraded {
                t.Errorf("expected degraded=true, got false")
        }
        if resp.TotalBills != len(kenya_seed.SampleBills) {
                t.Errorf("expected total_bills=%d, got %d", len(kenya_seed.SampleBills), resp.TotalBills)
        }
}

// --- GET /api/v1/bills/{id} ---

// TestBillDetailHandler_LiveSuccess_SourceIsLive verifies the detail
// handler returns `source: "live"` + `degraded: false` when the live
// crawl succeeds and the requested ID matches a live-discovered Bill.
// The mock adapter returns empty fetch HTML so the handler takes its
// "ParseBillDetail failed → return what we know from discovery" branch.
func TestBillDetailHandler_LiveSuccess_SourceIsLive(t *testing.T) {
        bills := sampleLiveBills()
        adapter := &mockBillsAdapter{bills: bills}
        billID := bills[0].SourceID
        req := httptest.NewRequest(http.MethodGet, "/api/v1/bills/"+billID, nil)
        req.URL.Path = "/api/v1/bills/" + billID
        rr := httptest.NewRecorder()

        makeBillDetailHandler(adapter, "")(rr, req)

        if rr.Code != http.StatusOK {
                t.Fatalf("expected 200, got %d (body=%s)", rr.Code, rr.Body.String())
        }
        var resp billResponse
        if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
                t.Fatalf("decode: %v", err)
        }
        if resp.Source != "live" {
                t.Errorf("expected source=live, got %q", resp.Source)
        }
        if resp.Degraded {
                t.Errorf("expected degraded=false, got true")
        }
        if resp.ID != billID {
                t.Errorf("expected id=%s, got %q", billID, resp.ID)
        }
}

// TestBillDetailHandler_FallbackToSeed_OnAdapterError verifies that
// when the live crawl fails, the detail handler falls back to the seed
// slice and returns 200 with `source: "seed"` + `degraded: true` when
// the requested ID is present in the seed data (issue #265 acceptance).
func TestBillDetailHandler_FallbackToSeed_OnAdapterError(t *testing.T) {
        adapter := &mockBillsAdapter{err: errors.New("simulated 403")}
        // Pick a bill ID that exists in the seed slice.
        if len(kenya_seed.SampleBills) == 0 {
                t.Fatal("seed slice must be non-empty for this test")
        }
        wantBill := kenya_seed.SampleBills[0]
        billID := wantBill.SourceID
        req := httptest.NewRequest(http.MethodGet, "/api/v1/bills/"+billID, nil)
        req.URL.Path = "/api/v1/bills/" + billID
        rr := httptest.NewRecorder()

        makeBillDetailHandler(adapter, "")(rr, req)

        if rr.Code != http.StatusOK {
                t.Fatalf("expected 200 (NOT 503) on adapter error, got %d (body=%s)", rr.Code, rr.Body.String())
        }
        var resp billResponse
        if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
                t.Fatalf("decode: %v", err)
        }
        if resp.Source != "seed" {
                t.Errorf("expected source=seed, got %q", resp.Source)
        }
        if !resp.Degraded {
                t.Errorf("expected degraded=true, got false")
        }
        if resp.ID != wantBill.SourceID {
                t.Errorf("expected id=%s, got %q", wantBill.SourceID, resp.ID)
        }
        if resp.Title != wantBill.Title {
                t.Errorf("expected title=%q, got %q", wantBill.Title, resp.Title)
        }
}

// TestBillDetailHandler_FallbackToSeed_NotFoundInSeed verifies that
// when the live crawl fails AND the requested ID is not in the seed
// slice, the handler returns 404 (genuinely not found) rather than
// 503 — the seed fallback only satisfies known IDs.
func TestBillDetailHandler_FallbackToSeed_NotFoundInSeed(t *testing.T) {
        adapter := &mockBillsAdapter{err: errors.New("simulated timeout")}
        billID := "this-id-does-not-exist-anywhere-2026"
        req := httptest.NewRequest(http.MethodGet, "/api/v1/bills/"+billID, nil)
        req.URL.Path = "/api/v1/bills/" + billID
        rr := httptest.NewRecorder()

        makeBillDetailHandler(adapter, "")(rr, req)

        if rr.Code != http.StatusNotFound {
                t.Errorf("expected 404 (not in seed slice), got %d (body=%s)", rr.Code, rr.Body.String())
        }
}

// --- seed slice sanity ---

// TestSeedSampleBills_NonEmptyAndComprehensive is a sanity check that
// the seed slice the fallback relies on actually contains 50+ Bills
// (the documented contract for the comprehensive fallback). Catches
// regressions where someone trims the slice without realising it
// powers the degraded-mode response.
func TestSeedSampleBills_NonEmptyAndComprehensive(t *testing.T) {
        if len(kenya_seed.SampleBills) < 50 {
                t.Errorf("expected kenya_seed.SampleBills to have ≥50 entries (comprehensive fallback), got %d", len(kenya_seed.SampleBills))
        }
        seen := make(map[string]bool, len(kenya_seed.SampleBills))
        for i, b := range kenya_seed.SampleBills {
                if b.SourceID == "" {
                        t.Errorf("seed bill %d has empty SourceID", i)
                }
                if b.URL == "" {
                        t.Errorf("seed bill %d has empty URL", i)
                }
                if b.Title == "" {
                        t.Errorf("seed bill %d has empty Title", i)
                }
                if b.House == "" {
                        t.Errorf("seed bill %d has empty House", i)
                }
                if seen[b.SourceID] {
                        t.Errorf("seed bill %d has duplicate SourceID %q", i, b.SourceID)
                }
                seen[b.SourceID] = true
        }
}

// TestSeedFindSampleBillByID_SubstringMatch verifies the seed lookup
// helper used by the detail handler's fallback path supports the same
// substring fallback the live-discovery path uses (callers sometimes
// pass truncated IDs from URL segments).
func TestSeedFindSampleBillByID_SubstringMatch(t *testing.T) {
        if len(kenya_seed.SampleBills) == 0 {
                t.Fatal("seed slice must be non-empty for this test")
        }
        full := kenya_seed.SampleBills[0].SourceID

        // Full ID works.
        got := kenya_seed.FindSampleBillByID(full)
        if got == nil {
                t.Fatalf("expected to find seed bill by full ID %q", full)
        }
        if got.SourceID != full {
                t.Errorf("returned bill SourceID=%q, want %q", got.SourceID, full)
        }

        // Substring of the ID also works (matches the live-discovery
        // contract in main.go's findBillByID). Use a substring of the
        // first seed bill's SourceID so the test does not couple to the
        // exact slug chosen for the seed slice.
        sub := full
        if len(sub) > 12 {
                sub = sub[len(sub)-12:] // tail of the SourceID
        }
        got = kenya_seed.FindSampleBillByID(sub)
        if got == nil {
                t.Errorf("expected to find a seed bill by substring %q (tail of %q)", sub, full)
        }

        // Empty ID returns nil (no panic).
        if kenya_seed.FindSampleBillByID("") != nil {
                t.Errorf("expected nil for empty bill ID")
        }

        // Non-matching ID returns nil.
        if kenya_seed.FindSampleBillByID("nonexistent-id-xyz-123") != nil {
                t.Errorf("expected nil for non-matching bill ID")
        }
}
