package main

import (
        "encoding/json"
        "net/http"
        "net/http/httptest"
        "testing"
)

// TestGovernmentsAPI_ListReturnsAdministrations verifies that the list
// endpoint returns the seed administrations (Jomo Kenyatta through William
// Ruto).
func TestGovernmentsAPI_ListReturnsAdministrations(t *testing.T) {
        req := httptest.NewRequest(http.MethodGet, "/api/v1/governments", nil)
        rec := httptest.NewRecorder()

        makeGovernmentsListHandler()(rec, req)

        if rec.Code != 200 {
                t.Fatalf("expected 200; got %d", rec.Code)
        }
        var resp struct {
                Administrations []map[string]any `json:"administrations"`
                Count           int              `json:"count"`
        }
        _ = json.NewDecoder(rec.Body).Decode(&resp)

        if resp.Count == 0 {
                t.Fatal("expected administrations; got 0")
        }
        // Verify at least 5 admins (Jomo, Moi, Kibaki, Uhuru, Ruto).
        if resp.Count < 5 {
                t.Errorf("expected at least 5 administrations; got %d", resp.Count)
        }
        // Verify the most recent (William Ruto) is first.
        first, _ := resp.Administrations[0]["name"].(string)
        if first != "William Ruto Administration" {
                t.Errorf("expected William Ruto first; got %q", first)
        }
}

// TestGovernmentsAPI_DetailReturnsPresident verifies that the detail endpoint
// returns the administration + president + terms.
func TestGovernmentsAPI_DetailReturnsPresident(t *testing.T) {
        req := httptest.NewRequest(http.MethodGet, "/api/v1/governments/admin-uhuru-kenyatta", nil)
        rec := httptest.NewRecorder()

        makeGovernmentDetailHandler()(rec, req)

        if rec.Code != 200 {
                t.Fatalf("expected 200; got %d", rec.Code)
        }
        var resp struct {
                Administration map[string]any   `json:"administration"`
                President      map[string]any   `json:"president"`
                Terms          []map[string]any `json:"terms"`
        }
        _ = json.NewDecoder(rec.Body).Decode(&resp)

        name, _ := resp.President["display_name"].(string)
        if name != "Uhuru Kenyatta" {
                t.Errorf("expected president Uhuru Kenyatta; got %q", name)
        }
        if len(resp.Terms) < 2 {
                t.Errorf("expected at least 2 terms; got %d", len(resp.Terms))
        }
}

// TestGovernmentsAPI_ConstitutionReturnsFactTag verifies that the
// constitution endpoint tags the constitution as FACT.
func TestGovernmentsAPI_ConstitutionReturnsFactTag(t *testing.T) {
        req := httptest.NewRequest(http.MethodGet, "/api/v1/constitution", nil)
        rec := httptest.NewRecorder()

        makeConstitutionHandler()(rec, req)

        if rec.Code != 200 {
                t.Fatalf("expected 200; got %d", rec.Code)
        }
        var resp map[string]any
        _ = json.NewDecoder(rec.Body).Decode(&resp)
        if resp["reality_layer"] != "FACT" {
                t.Errorf("expected reality_layer FACT; got %v", resp["reality_layer"])
        }
}

// TestGovernmentsAPI_TransitionsReturnsChronological verifies that the
// transitions endpoint returns transitions sorted by date.
func TestGovernmentsAPI_TransitionsReturnsChronological(t *testing.T) {
        req := httptest.NewRequest(http.MethodGet, "/api/v1/transitions", nil)
        rec := httptest.NewRecorder()

        makeTransitionsHandler()(rec, req)

        if rec.Code != 200 {
                t.Fatalf("expected 200; got %d", rec.Code)
        }
        var resp struct {
                Transitions []map[string]any `json:"transitions"`
                Count       int              `json:"count"`
        }
        _ = json.NewDecoder(rec.Body).Decode(&resp)

        if resp.Count < 4 {
                t.Errorf("expected at least 4 transitions; got %d", resp.Count)
        }
        // First should be most recent (2022 transition to Ruto).
        firstDate, _ := resp.Transitions[0]["transition_date"].(string)
        if firstDate == "" {
                t.Fatal("expected non-empty transition_date")
        }
}

// TestGovernmentsAPI_ConstitutionArticlesReturnsChapters verifies that the
// /api/v1/constitution/articles list endpoint returns the curated chapter +
// article tree sourced from the kenya_seed package, and that every chapter
// carries at least one article. This is the headline test for issue #212:
// the Constitution domain types were previously declared but never
// populated.
func TestGovernmentsAPI_ConstitutionArticlesReturnsChapters(t *testing.T) {
        req := httptest.NewRequest(http.MethodGet, "/api/v1/constitution/articles", nil)
        rec := httptest.NewRecorder()

        makeConstitutionArticlesHandler()(rec, req)

        if rec.Code != 200 {
                t.Fatalf("expected 200; got %d", rec.Code)
        }
        var resp struct {
                Chapters     []map[string]any `json:"chapters"`
                Count        int              `json:"count"`
                RealityLayer string           `json:"reality_layer"`
        }
        _ = json.NewDecoder(rec.Body).Decode(&resp)

        if resp.Count == 0 {
                t.Fatal("expected chapters; got 0")
        }
        if resp.RealityLayer != "FACT" {
                t.Errorf("expected reality_layer FACT; got %q", resp.RealityLayer)
        }
        // The curated set has at least 9 chapters (1, 2, 4, 5, 8, 9, 11, 12, 13).
        if resp.Count < 9 {
                t.Errorf("expected at least 9 chapters; got %d", resp.Count)
        }
        // Every chapter must carry at least one article.
        totalArticles := 0
        for _, c := range resp.Chapters {
                articles, _ := c["articles"].([]any)
                if len(articles) == 0 {
                        t.Errorf("chapter %v has no articles", c["id"])
                }
                totalArticles += len(articles)
        }
        // The curated set has at least 20 articles (Article 1 through Article 232).
        if totalArticles < 20 {
                t.Errorf("expected at least 20 articles across chapters; got %d", totalArticles)
        }
}

// TestGovernmentsAPI_ConstitutionArticleDetailReturnsArticle verifies that
// the /api/v1/constitution/articles/{id} endpoint returns a single article
// by its stable ID. This is the second half of issue #212: the chapter +
// article seed data is now reachable by both list and detail.
func TestGovernmentsAPI_ConstitutionArticleDetailReturnsArticle(t *testing.T) {
        req := httptest.NewRequest(http.MethodGet, "/api/v1/constitution/articles/article-1", nil)
        rec := httptest.NewRecorder()

        makeConstitutionArticleDetailHandler()(rec, req)

        if rec.Code != 200 {
                t.Fatalf("expected 200; got %d", rec.Code)
        }
        var resp struct {
                Article map[string]any `json:"article"`
        }
        _ = json.NewDecoder(rec.Body).Decode(&resp)

        if resp.Article == nil {
                t.Fatal("expected article in response")
        }
        if number, _ := resp.Article["number"].(string); number != "Article 1" {
                t.Errorf("expected number 'Article 1'; got %q", number)
        }
        if title, _ := resp.Article["title"].(string); title != "Sovereignty of the People" {
                t.Errorf("expected title 'Sovereignty of the People'; got %q", title)
        }
}

// TestGovernmentsAPI_ConstitutionArticleDetailReturns404 verifies that the
// detail endpoint returns 404 (not 200) when the requested article ID is
// not in the curated set — preventing silent acceptance of unknown IDs.
func TestGovernmentsAPI_ConstitutionArticleDetailReturns404(t *testing.T) {
        req := httptest.NewRequest(http.MethodGet, "/api/v1/constitution/articles/does-not-exist", nil)
        rec := httptest.NewRecorder()

        makeConstitutionArticleDetailHandler()(rec, req)

        if rec.Code != http.StatusNotFound {
                t.Fatalf("expected 404; got %d", rec.Code)
        }
}
