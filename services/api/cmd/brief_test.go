package main

import (
        "context"
        "encoding/json"
        "io"
        "net/http"
        "net/http/httptest"
        "strings"
        "testing"
        "time"

        "github.com/Roy-Wanyoike/civic-intelligence/adapters/kenya/kenya_law"
)

// billsListFixture returns a minimal Kenya Law bills-listing HTML that the
// kenya_law parser accepts. Three Bills: an Amendment, a Finance, and a
// plain Bill — enough to exercise every branch of the headline + classifier.
const billsListFixture = `<html><body>
<a href="/akn/ke/bill/na/2026-09-17/the-housing-amendment-bill-2026/eng@2026-09-17">The Housing (Amendment) Bill, 2026</a>
<a href="/akn/ke/bill/na/2026-09-16/the-finance-bill-2026/eng@2026-09-16">The Finance Bill, 2026</a>
<a href="/akn/ke/bill/senate/2026-09-15/the-public-health-bill-2026/eng@2026-09-15">The Public Health Bill, 2026</a>
</body></html>`

// parseTestBills is a thin helper that runs the kenya_law parser over the
// fixture HTML and returns the discovered Bills. Used by every test so we
// don't re-type the adapter setup.
func parseTestBills(t *testing.T) []kenya_law.BillCandidate {
        t.Helper()
        adapter := newTestAdapter(billsListFixture)
        bills, err := adapter.DiscoverBills(context.Background())
        if err != nil {
                t.Fatalf("discover bills: %v", err)
        }
        if len(bills) == 0 {
                t.Fatal("expected at least 1 bill from fixture, got 0")
        }
        return bills
}

// TestGenerateBrief_BasicShape verifies the brief generated from three
// discovered Bills carries the expected ID, headline, four sections, AI
// disclaimer, and a non-zero evidence count.
func TestGenerateBrief_BasicShape(t *testing.T) {
        bills := parseTestBills(t)
        now := time.Date(2026, 9, 18, 8, 0, 0, 0, time.UTC)

        b := generateBrief(now, briefRequest{Country: "KE"}, bills, "")

        if b.ID != "brief-2026-09-18" {
                t.Errorf("expected id=brief-2026-09-18, got %q", b.ID)
        }
        if b.Date != "2026-09-18" {
                t.Errorf("expected date=2026-09-18, got %q", b.Date)
        }
        if b.Country != "KE" {
                t.Errorf("expected country=KE, got %q", b.Country)
        }
        if b.Headline == "" {
                t.Error("expected non-empty headline")
        }
        if !strings.Contains(b.Headline, "Bill") {
                t.Errorf("expected headline to mention 'Bill', got %q", b.Headline)
        }
        if len(b.Sections) != 4 {
                t.Fatalf("expected 4 sections, got %d", len(b.Sections))
        }
        wantTitles := []string{
                "What Changed Today",
                "Your Followed Topics",
                "What to Watch",
                "Constitutional Context",
        }
        for i, want := range wantTitles {
                if b.Sections[i].Title != want {
                        t.Errorf("section[%d]: expected title=%q, got %q", i, want, b.Sections[i].Title)
                }
        }
        if b.AISummary == "" {
                t.Error("expected non-empty ai_summary")
        }
        if b.AISource != briefAISourceTemplate {
                t.Errorf("expected ai_source=%q, got %q", briefAISourceTemplate, b.AISource)
        }
        if b.AIDisclaimer != briefAISummaryDisclaimer {
                t.Errorf("expected ai_disclaimer=%q, got %q", briefAISummaryDisclaimer, b.AIDisclaimer)
        }
        if b.EvidenceCount == 0 {
                t.Error("expected non-zero evidence_count")
        }
        if b.GeneratedAt == "" {
                t.Error("expected non-empty generated_at")
        }
}

// TestGenerateBrief_EveryItemHasEvidenceURL verifies that EVERY item in
// EVERY section carries a non-empty evidence_url. This is the brief's core
// contract — no claim without a primary source.
func TestGenerateBrief_EveryItemHasEvidenceURL(t *testing.T) {
        bills := parseTestBills(t)
        now := time.Date(2026, 9, 18, 8, 0, 0, 0, time.UTC)

        b := generateBrief(now, briefRequest{
                Country:        "KE",
                FollowedTopics: []string{"health"},
        }, bills, "")

        total := 0
        for sIdx, s := range b.Sections {
                for iIdx, it := range s.Items {
                        total++
                        if strings.TrimSpace(it.EvidenceURL) == "" {
                                t.Errorf("section[%d] (%s) item[%d] (%s): empty evidence_url",
                                        sIdx, s.Title, iIdx, it.Title)
                        }
                }
        }
        if total == 0 {
                t.Fatal("expected at least 1 item across all sections, got 0")
        }
        if b.EvidenceCount != total {
                t.Errorf("evidence_count mismatch: brief says %d, actual items %d", b.EvidenceCount, total)
        }
}

// TestGenerateBrief_AIDisclaimerPresent verifies the AI disclaimer is the
// exact documented string — the frontend renders it next to the ASSUMPTION
// badge so a reader knows the AI summary must be verified.
func TestGenerateBrief_AIDisclaimerPresent(t *testing.T) {
        bills := parseTestBills(t)
        now := time.Date(2026, 9, 18, 8, 0, 0, 0, time.UTC)

        b := generateBrief(now, briefRequest{Country: "KE"}, bills, "")
        if b.AIDisclaimer != "AI-generated summary. Verify against primary sources." {
                t.Errorf("expected disclaimer, got %q", b.AIDisclaimer)
        }
        // The ai_summary itself must NOT be empty (template fallback fires when
        // AI service is unreachable).
        if b.AISummary == "" {
                t.Error("expected non-empty ai_summary (template fallback should fire)")
        }
}

// TestGenerateBrief_TemplateFallbackNotLabelledAsAI verifies that when the
// AI service is unreachable (empty URL), the fallback summary is explicitly
// prefixed with "Auto-generated summary." — NEVER presented as an AI output.
// This is the no-fake-AI rule from docs/NO_FAKE_COMPLETION.md.
func TestGenerateBrief_TemplateFallbackNotLabelledAsAI(t *testing.T) {
        bills := parseTestBills(t)
        now := time.Date(2026, 9, 18, 8, 0, 0, 0, time.UTC)

        b := generateBrief(now, briefRequest{Country: "KE"}, bills, "")
        if b.AISource != briefAISourceTemplate {
                t.Errorf("expected ai_source=%q, got %q", briefAISourceTemplate, b.AISource)
        }
        if !strings.HasPrefix(b.AISummary, "Auto-generated summary.") {
                t.Errorf("expected template summary to start with 'Auto-generated summary.', got: %q", b.AISummary)
        }
}

// TestGenerateBrief_EmptyBills verifies that when the adapter returns zero
// Bills, the brief surfaces an honest "no activity" message and the AI
// summary falls back to the empty template (NOT a fabricated AI response).
func TestGenerateBrief_EmptyBills(t *testing.T) {
        now := time.Date(2026, 9, 18, 8, 0, 0, 0, time.UTC)
        b := generateBrief(now, briefRequest{Country: "KE"}, nil, "")

        if b.Headline == "" {
                t.Error("expected non-empty headline even for empty brief")
        }
        if !strings.Contains(strings.ToLower(b.Headline), "no new") && !strings.Contains(strings.ToLower(b.Headline), "no civic") {
                t.Errorf("expected headline to mention 'no new'/'no civic', got %q", b.Headline)
        }
        if b.AISource != briefAISourceTemplate {
                t.Errorf("expected ai_source=%q for empty brief, got %q", briefAISourceTemplate, b.AISource)
        }
        if !strings.Contains(b.AISummary, "Auto-generated summary.") {
                t.Errorf("expected empty brief AI summary to be templated, got %q", b.AISummary)
        }

        // Every section should still exist (with empty items where appropriate).
        if len(b.Sections) != 4 {
                t.Fatalf("expected 4 sections even for empty brief, got %d", len(b.Sections))
        }
        // Constitutional Context should always have at least the Article 10 default.
        constitutionSection := b.Sections[3]
        if constitutionSection.Title != "Constitutional Context" {
                t.Fatalf("expected section 3 to be Constitutional Context, got %q", constitutionSection.Title)
        }
        if len(constitutionSection.Items) == 0 {
                t.Error("expected Constitutional Context section to default to Article 10 even when no topics matched")
        }
}

// TestGenerateBrief_FollowedTopicsFilters verifies the "Your Followed
// Topics" section surfaces only Bills matching the user's followed topics
// (case-insensitive substring on title).
func TestGenerateBrief_FollowedTopicsFilters(t *testing.T) {
        bills := parseTestBills(t)
        now := time.Date(2026, 9, 18, 8, 0, 0, 0, time.UTC)

        b := generateBrief(now, briefRequest{
                Country:        "KE",
                FollowedTopics: []string{"health"},
        }, bills, "")

        followedSection := b.Sections[1]
        if followedSection.Title != "Your Followed Topics" {
                t.Fatalf("expected section 1 to be Your Followed Topics, got %q", followedSection.Title)
        }
        for _, it := range followedSection.Items {
                // Allow the "no_matches" templated item if no Bill matched — but here
                // we DO have a "Public Health Bill" so we should NOT see that fallback.
                if it.Type == "no_matches" {
                        t.Errorf("did not expect no_matches fallback when a Health Bill exists: %+v", it)
                }
        }
        if len(followedSection.Items) == 0 {
                t.Fatal("expected followed section to surface the Public Health Bill")
        }
        // At least one item should mention Health.
        foundHealth := false
        for _, it := range followedSection.Items {
                if strings.Contains(strings.ToLower(it.Title), "health") {
                        foundHealth = true
                        break
                }
        }
        if !foundHealth {
                t.Errorf("expected followed section to contain a Health-related Bill; items=%+v", followedSection.Items)
        }
}

// TestGenerateBrief_FollowedTopicsNoMatch verifies the "Your Followed
// Topics" section surfaces the templated "no matches" item when the user
// follows a topic that has no Bills today.
func TestGenerateBrief_FollowedTopicsNoMatch(t *testing.T) {
        bills := parseTestBills(t)
        now := time.Date(2026, 9, 18, 8, 0, 0, 0, time.UTC)

        b := generateBrief(now, briefRequest{
                Country:        "KE",
                FollowedTopics: []string{"space-exploration"},
        }, bills, "")

        followedSection := b.Sections[1]
        foundNoMatches := false
        for _, it := range followedSection.Items {
                if it.Type == "no_matches" {
                        foundNoMatches = true
                        if it.EvidenceURL == "" {
                                t.Errorf("no_matches item must still carry an evidence_url")
                        }
                }
        }
        if !foundNoMatches {
                t.Errorf("expected no_matches templated item when no Bill matches followed topics")
        }
}

// TestGenerateBrief_ConstitutionalContextForHealth verifies the
// Constitutional Context section surfaces Article 43 (Economic and Social
// Rights) when the user follows "health" — and that the article's
// evidence_url points to kenyalaw.org.
func TestGenerateBrief_ConstitutionalContextForHealth(t *testing.T) {
        bills := parseTestBills(t)
        now := time.Date(2026, 9, 18, 8, 0, 0, 0, time.UTC)

        b := generateBrief(now, briefRequest{
                Country:        "KE",
                FollowedTopics: []string{"health"},
        }, bills, "")

        constitutionSection := b.Sections[3]
        foundArticle43 := false
        for _, it := range constitutionSection.Items {
                if it.Article == "Article 43" {
                        foundArticle43 = true
                        if !strings.Contains(it.EvidenceURL, "kenyalaw.org") {
                                t.Errorf("expected constitution evidence_url to point to kenyalaw.org, got %q", it.EvidenceURL)
                        }
                        if it.Text == "" {
                                t.Error("expected constitution item to carry the article text")
                        }
                        if it.Connection == "" {
                                t.Error("expected constitution item to carry a plain-language connection")
                        }
                }
        }
        if !foundArticle43 {
                t.Errorf("expected Article 43 to surface for followed topic 'health'; items=%+v", constitutionSection.Items)
        }
}

// TestGenerateBrief_HeadlineCounts verifies the headline counts Bills into
// the documented buckets: published, amendments, finance.
func TestGenerateBrief_HeadlineCounts(t *testing.T) {
        bills := parseTestBills(t)
        now := time.Date(2026, 9, 18, 8, 0, 0, 0, time.UTC)

        b := generateBrief(now, briefRequest{Country: "KE"}, bills, "")

        // The fixture has 1 plain Bill (Public Health), 1 Amendment (Housing
        // Amendment), 1 Finance. Headline should mention all three.
        if !strings.Contains(b.Headline, "Bill published") {
                t.Errorf("expected headline to mention 'Bill published', got %q", b.Headline)
        }
        if !strings.Contains(b.Headline, "Amendment Bill") {
                t.Errorf("expected headline to mention 'Amendment Bill', got %q", b.Headline)
        }
        if !strings.Contains(b.Headline, "Finance Bill") {
                t.Errorf("expected headline to mention 'Finance Bill', got %q", b.Headline)
        }
}

// --- Handler tests ---

// TestBriefGenerateHandler_Post verifies the POST /api/v1/brief/generate
// handler returns 200 + a populated brief + persists to the store.
func TestBriefGenerateHandler_Post(t *testing.T) {
        adapter := newTestAdapter(billsListFixture)
        store := newBriefStore()

        body := `{"user_id":"u-1","followed_topics":["health"],"country":"KE"}`
        req := httptest.NewRequest(http.MethodPost, "/api/v1/brief/generate", strings.NewReader(body))
        req.Header.Set("Content-Type", "application/json")
        rr := httptest.NewRecorder()

        makeBriefGenerateHandler(adapter, "", store).ServeHTTP(rr, req)

        if rr.Code != http.StatusOK {
                t.Fatalf("expected 200, got %d (body=%s)", rr.Code, rr.Body.String())
        }
        var b brief
        if err := json.Unmarshal(rr.Body.Bytes(), &b); err != nil {
                t.Fatalf("decode: %v", err)
        }
        if b.UserID != "u-1" {
                t.Errorf("expected user_id=u-1, got %q", b.UserID)
        }
        if b.EvidenceCount == 0 {
                t.Error("expected non-zero evidence_count")
        }
        // Verify the brief was persisted.
        if _, ok := store.get(b.ID); !ok {
                t.Errorf("expected brief %q to be persisted in the store", b.ID)
        }
}

// TestBriefGenerateHandler_GetRejectsMethod verifies GET is not allowed on
// /api/v1/brief/generate (POST-only).
func TestBriefGenerateHandler_GetRejectsMethod(t *testing.T) {
        adapter := newTestAdapter(billsListFixture)
        store := newBriefStore()
        req := httptest.NewRequest(http.MethodGet, "/api/v1/brief/generate", nil)
        rr := httptest.NewRecorder()
        makeBriefGenerateHandler(adapter, "", store).ServeHTTP(rr, req)
        if rr.Code != http.StatusMethodNotAllowed {
                t.Errorf("expected 405, got %d", rr.Code)
        }
}

// TestBriefTodayHandler_OnDemand verifies that GET /api/v1/brief/today
// generates a brief on-demand when none exists yet, and that the second
// call returns the cached brief (not regenerated).
func TestBriefTodayHandler_OnDemand(t *testing.T) {
        adapter := newTestAdapter(billsListFixture)
        store := newBriefStore()

        req := httptest.NewRequest(http.MethodGet, "/api/v1/brief/today", nil)
        rr := httptest.NewRecorder()
        makeBriefTodayHandler(adapter, "", store).ServeHTTP(rr, req)

        if rr.Code != http.StatusOK {
                t.Fatalf("expected 200, got %d (body=%s)", rr.Code, rr.Body.String())
        }
        var b1 brief
        if err := json.Unmarshal(rr.Body.Bytes(), &b1); err != nil {
                t.Fatalf("decode: %v", err)
        }

        // Second call should return the same brief (cached).
        rr2 := httptest.NewRecorder()
        makeBriefTodayHandler(adapter, "", store).ServeHTTP(rr2, req)
        if rr2.Code != http.StatusOK {
                t.Fatalf("expected 200 on second call, got %d", rr2.Code)
        }
        var b2 brief
        if err := json.Unmarshal(rr2.Body.Bytes(), &b2); err != nil {
                t.Fatalf("decode 2: %v", err)
        }
        if b1.GeneratedAt != b2.GeneratedAt {
                t.Errorf("expected cached brief to have the same generated_at; got %q vs %q", b1.GeneratedAt, b2.GeneratedAt)
        }
}

// TestBriefTodayHandler_AdapterError verifies the handler returns 200 (not
// 5xx) even when the adapter fails — the brief surfaces the failure
// honestly with an empty Bill list + the template fallback summary.
//
// Note: the brief is NOT entirely empty even on adapter failure — the
// Constitutional Context section defaults to Article 10 (Kenya's national
// values), which carries an evidence_url to kenyalaw.org. So evidence_count
// is at least 1.
func TestBriefTodayHandler_AdapterError(t *testing.T) {
        // errorKenyaLawClient always returns HTTP 500.
        client := &errorKenyaLawClient{}
        adapter := kenya_law.NewAdapter(client, "CivicIntelligence-test/0.1")
        store := newBriefStore()

        req := httptest.NewRequest(http.MethodGet, "/api/v1/brief/today", nil)
        rr := httptest.NewRecorder()
        makeBriefTodayHandler(adapter, "", store).ServeHTTP(rr, req)

        if rr.Code != http.StatusOK {
                t.Fatalf("expected 200 even on adapter error, got %d (body=%s)", rr.Code, rr.Body.String())
        }
        var b brief
        if err := json.Unmarshal(rr.Body.Bytes(), &b); err != nil {
                t.Fatalf("decode: %v", err)
        }
        // What Changed / Followed / Watch sections should be empty — no Bills
        // discovered. Constitutional Context defaults to Article 10, so total
        // evidence_count is exactly 1 (the Article 10 source URL).
        if b.EvidenceCount != 1 {
                t.Errorf("expected exactly 1 evidence_count (Constitutional Context Article 10 default) when adapter fails, got %d", b.EvidenceCount)
        }
        if !strings.Contains(b.AISummary, "Auto-generated summary.") {
                t.Errorf("expected template fallback summary, got %q", b.AISummary)
        }
        // What Changed Today section summary should explicitly mention no Bills.
        if !strings.Contains(strings.ToLower(b.Sections[0].Summary), "no new bills") {
                t.Errorf("expected 'What Changed Today' summary to mention no Bills, got %q", b.Sections[0].Summary)
        }
}

// TestBriefArchiveHandler_ListsStoredBriefs verifies GET /api/v1/brief/archive
// returns briefs in the order they were stored (newest-first by date).
func TestBriefArchiveHandler_ListsStoredBriefs(t *testing.T) {
        store := newBriefStore()
        // Populate two briefs with different dates.
        now1 := time.Date(2026, 9, 17, 8, 0, 0, 0, time.UTC)
        now2 := time.Date(2026, 9, 18, 8, 0, 0, 0, time.UTC)
        store.put(generateBrief(now1, briefRequest{Country: "KE"}, nil, ""))
        store.put(generateBrief(now2, briefRequest{Country: "KE"}, nil, ""))

        req := httptest.NewRequest(http.MethodGet, "/api/v1/brief/archive", nil)
        rr := httptest.NewRecorder()
        makeBriefArchiveHandler(store).ServeHTTP(rr, req)

        if rr.Code != http.StatusOK {
                t.Fatalf("expected 200, got %d", rr.Code)
        }
        var resp struct {
                Items []briefArchiveEntry `json:"items"`
                Total int                 `json:"total"`
        }
        if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
                t.Fatalf("decode: %v", err)
        }
        if resp.Total != 2 {
                t.Fatalf("expected 2 archive entries, got %d", resp.Total)
        }
        if resp.Items[0].Date != "2026-09-18" {
                t.Errorf("expected newest-first ordering; first entry date=%q", resp.Items[0].Date)
        }
}

// TestBriefDetailHandler_Found verifies GET /api/v1/brief/{id} returns the
// full brief when the ID exists in the store.
func TestBriefDetailHandler_Found(t *testing.T) {
        store := newBriefStore()
        now := time.Date(2026, 9, 18, 8, 0, 0, 0, time.UTC)
        b := generateBrief(now, briefRequest{Country: "KE"}, nil, "")
        store.put(b)

        req := httptest.NewRequest(http.MethodGet, "/api/v1/brief/"+b.ID, nil)
        rr := httptest.NewRecorder()
        makeBriefDetailHandler(store).ServeHTTP(rr, req)

        if rr.Code != http.StatusOK {
                t.Fatalf("expected 200, got %d (body=%s)", rr.Code, rr.Body.String())
        }
        var got brief
        if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
                t.Fatalf("decode: %v", err)
        }
        if got.ID != b.ID {
                t.Errorf("expected id=%q, got %q", b.ID, got.ID)
        }
        if len(got.Sections) != 4 {
                t.Errorf("expected 4 sections, got %d", len(got.Sections))
        }
}

// TestBriefDetailHandler_NotFound verifies GET /api/v1/brief/{id} returns
// 404 when the ID is not in the store.
func TestBriefDetailHandler_NotFound(t *testing.T) {
        store := newBriefStore()
        req := httptest.NewRequest(http.MethodGet, "/api/v1/brief/brief-1999-01-01", nil)
        rr := httptest.NewRecorder()
        makeBriefDetailHandler(store).ServeHTTP(rr, req)
        if rr.Code != http.StatusNotFound {
                t.Errorf("expected 404, got %d", rr.Code)
        }
}

// TestBriefDetailHandler_RejectsReservedNames verifies the detail handler
// does not serve briefs whose ID collides with the other reserved brief
// routes (today, archive, generate). This is defensive — the exact-match
// routes registered in main.go should win, but if routing changes the
// detail handler still rejects them rather than returning a confusing 404.
func TestBriefDetailHandler_RejectsReservedNames(t *testing.T) {
        store := newBriefStore()
        for _, name := range []string{"today", "archive", "generate", ""} {
                req := httptest.NewRequest(http.MethodGet, "/api/v1/brief/"+name, nil)
                rr := httptest.NewRecorder()
                makeBriefDetailHandler(store).ServeHTTP(rr, req)
                if rr.Code != http.StatusNotFound {
                        t.Errorf("reserved name %q: expected 404, got %d (body=%s)", name, rr.Code, rr.Body.String())
                }
        }
}

// --- AI integration tests ---

// TestCallAIBriefingGenerator_Reachable verifies that when the Python AI
// service responds 200 with a DailyBriefing body, the brief's ai_summary
// is composed from the response and the ai_source is "ai-service".
func TestCallAIBriefingGenerator_Reachable(t *testing.T) {
        // Spin up a stub AI service that returns a minimal DailyBriefing.
        mux := http.NewServeMux()
        mux.HandleFunc("/v1/briefing/generate", func(w http.ResponseWriter, r *http.Request) {
                if r.Method != http.MethodPost {
                        http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
                        return
                }
                writeJSON(w, http.StatusOK, map[string]any{
                        "country":  "KE",
                        "date":     "2026-09-18T08:00:00Z",
                        "headline": "AI-generated headline (stub)",
                        "items": []map[string]any{
                                {"kind": "new_bill", "title": "Stub Bill", "description": "AI says: stub.", "significance": "test", "citations": []any{}},
                        },
                })
        })
        srv := httptest.NewServer(mux)
        defer srv.Close()

        bills := parseTestBills(t)
        summary, ok := callAIBriefingGenerator(context.Background(), srv.URL, "KE", bills)
        if !ok {
                t.Fatal("expected callAIBriefingGenerator to return ok=true when the AI service responds 200")
        }
        if summary == "" {
                t.Error("expected non-empty ai_summary")
        }
        // The composed summary references today's verified Bills — NOT the AI's
        // stub headline — because the brief composes its own summary from the
        // BFF's observed data.
        if !strings.Contains(summary, "tracked") && !strings.Contains(summary, "development") {
                t.Errorf("expected composed summary to reference today's developments; got %q", summary)
        }
}

// TestCallAIBriefingGenerator_Unreachable verifies that when the Python AI
// service is unreachable (connection refused / 500), the call returns
// (false) and the brief falls back to the template summary.
func TestCallAIBriefingGenerator_Unreachable(t *testing.T) {
        // Use a port that's almost certainly not listening.
        summary, ok := callAIBriefingGenerator(context.Background(), "http://127.0.0.1:1", "KE", parseTestBills(t))
        if ok {
                t.Fatal("expected callAIBriefingGenerator to return ok=false when AI service is unreachable")
        }
        if summary != "" {
                t.Errorf("expected empty summary on failure, got %q", summary)
        }
}

// TestGenerateAISummary_FallsBackToTemplate verifies the public entry point
// returns the template summary when the AI service is unreachable.
func TestGenerateAISummary_FallsBackToTemplate(t *testing.T) {
        bills := parseTestBills(t)
        summary, source := generateAISummary(context.Background(), "", "KE", bills)
        if source != briefAISourceTemplate {
                t.Errorf("expected source=%q, got %q", briefAISourceTemplate, source)
        }
        if !strings.HasPrefix(summary, "Auto-generated summary.") {
                t.Errorf("expected template summary prefix, got %q", summary)
        }
}

// --- JSON shape tests ---

// TestBrief_JSONShape verifies the brief struct serialises with the
// documented JSON keys. The frontend (BriefReader) and the OpenAPI contract
// depend on these keys.
func TestBrief_JSONShape(t *testing.T) {
        b := brief{
                ID:       "brief-2026-09-18",
                Date:     "2026-09-18",
                Headline: "Test headline",
                Sections: []briefSection{
                        {
                                Title:   "What Changed Today",
                                Summary: "summary",
                                Items: []briefSectionItem{
                                        {
                                                Type:        "bill_published",
                                                Title:       "Bill A",
                                                Bill:        "ke-bill-1",
                                                EvidenceURL: "https://new.kenyalaw.org/akn/ke/bill/na/2026-09-17/test/eng@2026-09-17",
                                        },
                                },
                        },
                },
                AISummary:    "AI summary.",
                AIDisclaimer:  briefAISummaryDisclaimer,
                AISource:      briefAISourceAI,
                EvidenceCount: 1,
                GeneratedAt:   "2026-09-18T08:00:00Z",
                Country:       "KE",
                UserID:        "u-1",
        }
        out, err := json.Marshal(b)
        if err != nil {
                t.Fatalf("marshal: %v", err)
        }
        s := string(out)
        required := []string{
                `"id":"brief-2026-09-18"`,
                `"date":"2026-09-18"`,
                `"headline":"Test headline"`,
                `"sections":[{`,
                `"title":"What Changed Today"`,
                `"summary":"summary"`,
                `"items":[{`,
                `"type":"bill_published"`,
                `"bill":"ke-bill-1"`,
                `"evidence_url":"https://new.kenyalaw.org/akn/ke/bill/na/2026-09-17/test/eng@2026-09-17"`,
                `"ai_summary":"AI summary."`,
                `"ai_disclaimer":"AI-generated summary. Verify against primary sources."`,
                `"ai_source":"ai-service"`,
                `"evidence_count":1`,
                `"generated_at":"2026-09-18T08:00:00Z"`,
                `"country":"KE"`,
                `"user_id":"u-1"`,
        }
        for _, key := range required {
                if !strings.Contains(s, key) {
                        t.Errorf("expected JSON to contain %q; got %s", key, s)
                }
        }
}

// TestBriefStore_PutGet verifies the in-memory store round-trips a brief.
func TestBriefStore_PutGet(t *testing.T) {
        store := newBriefStore()
        now := time.Date(2026, 9, 18, 8, 0, 0, 0, time.UTC)
        b := generateBrief(now, briefRequest{Country: "KE"}, nil, "")
        if _, ok := store.get(b.ID); ok {
                t.Fatal("expected miss before put")
        }
        store.put(b)
        got, ok := store.get(b.ID)
        if !ok {
                t.Fatal("expected hit after put")
        }
        if got.ID != b.ID {
                t.Errorf("expected id=%q, got %q", b.ID, got.ID)
        }
}

// TestBriefArchiveEntry_JSONShape verifies the archive entry omits the
// heavy section bodies (only id, date, headline, evidence_count,
// generated_at) so the archive list stays small.
func TestBriefArchiveEntry_JSONShape(t *testing.T) {
        now := time.Date(2026, 9, 18, 8, 0, 0, 0, time.UTC)
        e := briefArchiveEntry{
                ID:            "brief-2026-09-18",
                Date:          "2026-09-18",
                Headline:      "headline",
                EvidenceCount: 5,
                GeneratedAt:   now,
        }
        out, err := json.Marshal(e)
        if err != nil {
                t.Fatalf("marshal: %v", err)
        }
        s := string(out)
        required := []string{
                `"id":"brief-2026-09-18"`,
                `"date":"2026-09-18"`,
                `"headline":"headline"`,
                `"evidence_count":5`,
                `"generated_at":"2026-09-18T08:00:00Z"`,
        }
        for _, key := range required {
                if !strings.Contains(s, key) {
                        t.Errorf("expected JSON to contain %q; got %s", key, s)
                }
        }
        if strings.Contains(s, "sections") {
                t.Errorf("archive entry should NOT include sections; got %s", s)
        }
}

// --- HTTP-level integration ---

// TestBriefEndpoints_RegisteredOnMux verifies that all four brief routes
// are reachable through a minimal mux wired exactly the way main.go wires
// them. This catches routing mistakes (e.g. /api/v1/brief/ shadowing
// /api/v1/brief/today) that the per-handler tests above would miss.
func TestBriefEndpoints_RegisteredOnMux(t *testing.T) {
        adapter := newTestAdapter(billsListFixture)
        store := newBriefStore()
        mux := http.NewServeMux()
        mux.HandleFunc("/api/v1/brief/generate", makeBriefGenerateHandler(adapter, "", store))
        mux.HandleFunc("/api/v1/brief/today", makeBriefTodayHandler(adapter, "", store))
        mux.HandleFunc("/api/v1/brief/archive", makeBriefArchiveHandler(store))
        mux.HandleFunc("/api/v1/brief/", makeBriefDetailHandler(store))

        // 1. POST /generate → 200 + brief
        req1 := httptest.NewRequest(http.MethodPost, "/api/v1/brief/generate", strings.NewReader(`{"country":"KE"}`))
        req1.Header.Set("Content-Type", "application/json")
        rr1 := httptest.NewRecorder()
        mux.ServeHTTP(rr1, req1)
        if rr1.Code != http.StatusOK {
                t.Errorf("POST /brief/generate: expected 200, got %d (body=%s)", rr1.Code, rr1.Body.String())
        }

        // 2. GET /today → 200 (cached from step 1 or on-demand)
        req2 := httptest.NewRequest(http.MethodGet, "/api/v1/brief/today", nil)
        rr2 := httptest.NewRecorder()
        mux.ServeHTTP(rr2, req2)
        if rr2.Code != http.StatusOK {
                t.Errorf("GET /brief/today: expected 200, got %d", rr2.Code)
        }

        // 3. GET /archive → 200 + items
        req3 := httptest.NewRequest(http.MethodGet, "/api/v1/brief/archive", nil)
        rr3 := httptest.NewRecorder()
        mux.ServeHTTP(rr3, req3)
        if rr3.Code != http.StatusOK {
                t.Errorf("GET /brief/archive: expected 200, got %d", rr3.Code)
        }

        // 4. GET /brief/{id} → 200
        var cached brief
        _ = json.Unmarshal(rr2.Body.Bytes(), &cached)
        if cached.ID == "" {
                t.Fatal("expected cached brief to have an ID")
        }
        req4 := httptest.NewRequest(http.MethodGet, "/api/v1/brief/"+cached.ID, nil)
        rr4 := httptest.NewRecorder()
        mux.ServeHTTP(rr4, req4)
        if rr4.Code != http.StatusOK {
                t.Errorf("GET /brief/{id}: expected 200, got %d (body=%s)", rr4.Code, rr4.Body.String())
        }
}

// --- AI disclaimer assertion ---

// TestBriefAISummaryDisclaimer_Constant verifies the disclaimer constant
// matches the documented exact string. The frontend renders this verbatim
// next to the ASSUMPTION reality badge, so a typo here would break the
// no-fake-AI gate.
func TestBriefAISummaryDisclaimer_Constant(t *testing.T) {
        want := "AI-generated summary. Verify against primary sources."
        if briefAISummaryDisclaimer != want {
                t.Errorf("disclaimer constant drift: expected %q, got %q", want, briefAISummaryDisclaimer)
        }
}

// TestComposeAISummaryFromResponse_NeverEchoesAIHeadline verifies the brief
// composes its OWN summary from the BFF's observed Bills — it does NOT
// echo the AI service's headline verbatim. This is the single-source-of-
// truth rule from docs/NO_FAKE_COMPLETION.md: the AI cannot mutate truth.
func TestComposeAISummaryFromResponse_NeverEchoesAIHeadline(t *testing.T) {
        bills := parseTestBills(t)
        resp := aiBriefingResponse{
                Headline: "AI-only headline the brief must never show",
                Items:    []aiBriefItem{{Kind: "new_bill", Title: "AI-only Bill"}},
        }
        got := composeAISummaryFromResponse(resp, bills)
        if strings.Contains(got, "AI-only headline") {
                t.Errorf("brief must not echo the AI service's headline verbatim; got %q", got)
        }
        if strings.Contains(got, "AI-only Bill") {
                t.Errorf("brief must not echo the AI service's item title verbatim; got %q", got)
        }
        // Must still mention the BFF's observed Bills.
        if !strings.Contains(got, "tracked") && !strings.Contains(got, "development") {
                t.Errorf("expected composed summary to reference observed Bills; got %q", got)
        }
}

// errEmptyReader is a helper to satisfy io.Reader when needed by tests.
var _ io.Reader = strings.NewReader("")
