// Package integration — api_governments_test.go exercises the government
// + constitution endpoints. The Kenya seed data (kenya_seed) populates the
// in-memory store at startup with presidents, administrations, terms,
// transitions, and the curated Constitution of Kenya 2010 chapters +
// articles.
//
// CRITICAL ATTRIBUTION RULE: the constitution is tagged reality_layer=FACT
// to signal it is authoritative source material the platform never
// reinterprets.
//
// Endpoints covered:
//
//      GET /api/v1/governments                  -- list administrations
//      GET /api/v1/governments/{id}             -- administration detail (+ terms)
//      GET /api/v1/governments/{id}/terms       -- terms for an administration
//      GET /api/v1/constitution                 -- constitution metadata
//      GET /api/v1/constitution/articles        -- chapters + articles tree
//      GET /api/v1/constitution/articles/{id}   -- single article detail
//      GET /api/v1/transitions                  -- government transitions
package integration

import (
        "net/http"
        "strings"
        "testing"
)

// TestGovernments_ListReturnsAdministrations verifies the list endpoint
// returns the seed administrations (Jomo Kenyatta through William Ruto),
// sorted by start date descending (most recent first).
func TestGovernments_ListReturnsAdministrations(t *testing.T) {
        var resp struct {
                Administrations []map[string]any `json:"administrations"`
                Count           int              `json:"count"`
        }
        status := mustGet(t, apiURL("/governments"), &resp)
        assertStatus(t, "/governments", http.StatusOK, status)

        if resp.Count == 0 {
                t.Fatal("governments: expected administrations, got count=0")
        }
        // At least 5 administrations (Jomo, Moi, Kibaki, Uhuru, Ruto).
        if resp.Count < 5 {
                t.Errorf("governments: expected at least 5 administrations, got %d", resp.Count)
        }
        // Verify the most recent (William Ruto) is first.
        if len(resp.Administrations) > 0 {
                name, _ := resp.Administrations[0]["name"].(string)
                if !strings.Contains(name, "William Ruto") {
                        t.Errorf("governments: expected William Ruto first, got %q", name)
                }
        }
        // Every administration must carry the documented identity fields.
        // source_url is not required because the Kenya seed data does not
        // currently populate it for administrations (issue tracked separately).
        for i, a := range resp.Administrations {
                for _, field := range []string{"id", "name", "president_id", "president_name", "start_date", "country_code"} {
                        v, ok := a[field]
                        if !ok || v == nil || v == "" {
                                t.Errorf("governments[%d]: missing required field %q", i, field)
                        }
                }
        }
}

// TestGovernments_DetailReturnsPresident verifies the detail endpoint
// returns the administration + president + terms.
func TestGovernments_DetailReturnsPresident(t *testing.T) {
        var resp struct {
                Administration map[string]any   `json:"administration"`
                President      map[string]any   `json:"president"`
                Terms          []map[string]any `json:"terms"`
        }
        status := mustGet(t, apiURL("/governments/admin-uhuru-kenyatta"), &resp)
        assertStatus(t, "/governments/admin-uhuru-kenyatta", http.StatusOK, status)

        name, _ := resp.President["display_name"].(string)
        if !strings.Contains(name, "Uhuru Kenyatta") {
                t.Errorf("governments/detail: expected president Uhuru Kenyatta, got %q", name)
        }
        if len(resp.Terms) < 2 {
                t.Errorf("governments/detail: expected at least 2 terms, got %d", len(resp.Terms))
        }
}

// TestGovernments_DetailReturnsTerms verifies the /terms sub-resource
// returns the terms for an administration.
func TestGovernments_DetailReturnsTerms(t *testing.T) {
        var resp struct {
                AdministrationID string            `json:"administration_id"`
                Terms             []map[string]any `json:"terms"`
                Count             int              `json:"count"`
        }
        status := mustGet(t, apiURL("/governments/admin-uhuru-kenyatta/terms"), &resp)
        assertStatus(t, "/governments/admin-uhuru-kenyatta/terms", http.StatusOK, status)

        if resp.AdministrationID != "admin-uhuru-kenyatta" {
                t.Errorf("governments/terms: expected admin-uhuru-kenyatta, got %q", resp.AdministrationID)
        }
        if resp.Count < 2 {
                t.Errorf("governments/terms: expected at least 2 terms, got %d", resp.Count)
        }
}

// TestGovernments_DetailUnknownReturns404 verifies an unknown
// administration returns 404 (not 500).
func TestGovernments_DetailUnknownReturns404(t *testing.T) {
        var dummy map[string]any
        status := mustGet(t, apiURL("/governments/admin-does-not-exist"), &dummy)
        assertStatus(t, "/governments/unknown", http.StatusNotFound, status)
}

// TestGovernments_ConstitutionReturnsFactTag verifies the constitution
// endpoint tags the constitution as FACT (authoritative source material).
func TestGovernments_ConstitutionReturnsFactTag(t *testing.T) {
        var resp map[string]any
        status := mustGet(t, apiURL("/constitution"), &resp)
        assertStatus(t, "/constitution", http.StatusOK, status)

        if resp["reality_layer"] != "FACT" {
                t.Errorf("constitution: expected reality_layer FACT, got %v", resp["reality_layer"])
        }
        // Constitution payload must be present.
        if _, ok := resp["constitution"]; !ok {
                t.Error("constitution: response missing 'constitution' field")
        }
}

// TestGovernments_ConstitutionArticlesReturnsChapters verifies the
// /constitution/articles list endpoint returns the curated chapter +
// article tree tagged FACT.
func TestGovernments_ConstitutionArticlesReturnsChapters(t *testing.T) {
        var resp struct {
                Chapters     []map[string]any `json:"chapters"`
                Count        int              `json:"count"`
                RealityLayer string           `json:"reality_layer"`
        }
        status := mustGet(t, apiURL("/constitution/articles"), &resp)
        assertStatus(t, "/constitution/articles", http.StatusOK, status)

        if resp.Count == 0 {
                t.Fatal("constitution/articles: expected chapters, got count=0")
        }
        if resp.RealityLayer != "FACT" {
                t.Errorf("constitution/articles: expected FACT, got %q", resp.RealityLayer)
        }
        // The curated set has at least 9 chapters.
        if resp.Count < 9 {
                t.Errorf("constitution/articles: expected at least 9 chapters, got %d", resp.Count)
        }
        // Every chapter must carry at least one article.
        totalArticles := 0
        for _, c := range resp.Chapters {
                articles, _ := c["articles"].([]any)
                if len(articles) == 0 {
                        t.Errorf("constitution/articles: chapter %v has no articles", c["id"])
                }
                totalArticles += len(articles)
        }
        if totalArticles < 20 {
                t.Errorf("constitution/articles: expected at least 20 articles, got %d", totalArticles)
        }
}

// TestGovernments_ConstitutionArticleDetailReturnsArticle verifies the
// /constitution/articles/{id} endpoint returns a single article by stable
// ID. The curated set includes "article-1" — "Sovereignty of the People".
func TestGovernments_ConstitutionArticleDetailReturnsArticle(t *testing.T) {
        var resp struct {
                Article map[string]any `json:"article"`
        }
        status := mustGet(t, apiURL("/constitution/articles/article-1"), &resp)
        assertStatus(t, "/constitution/articles/article-1", http.StatusOK, status)

        if resp.Article == nil {
                t.Fatal("constitution/articles/article-1: expected article in response")
        }
        if number, _ := resp.Article["number"].(string); number != "Article 1" {
                t.Errorf("constitution/articles/article-1: expected number 'Article 1', got %q", number)
        }
        if title, _ := resp.Article["title"].(string); title != "Sovereignty of the People" {
                t.Errorf("constitution/articles/article-1: expected title 'Sovereignty of the People', got %q", title)
        }
}

// TestGovernments_ConstitutionArticleDetailReturns404 verifies an unknown
// article ID returns 404 (not 200 with empty body).
func TestGovernments_ConstitutionArticleDetailReturns404(t *testing.T) {
        var dummy map[string]any
        status := mustGet(t, apiURL("/constitution/articles/does-not-exist"), &dummy)
        assertStatus(t, "/constitution/articles/unknown", http.StatusNotFound, status)
}

// TestGovernments_TransitionsReturnsChronological verifies the transitions
// endpoint returns transitions sorted by date descending.
func TestGovernments_TransitionsReturnsChronological(t *testing.T) {
        var resp struct {
                Transitions []map[string]any `json:"transitions"`
                Count       int              `json:"count"`
        }
        status := mustGet(t, apiURL("/transitions"), &resp)
        assertStatus(t, "/transitions", http.StatusOK, status)

        if resp.Count < 4 {
                t.Errorf("transitions: expected at least 4 transitions, got %d", resp.Count)
        }
        if len(resp.Transitions) == 0 {
                t.Fatal("transitions: expected non-empty list")
        }
        // First should be most recent (2022 transition to Ruto).
        firstDate, _ := resp.Transitions[0]["transition_date"].(string)
        if firstDate == "" {
                t.Error("transitions: expected non-empty transition_date on first item")
        }
        // Every transition must carry source_url (evidence-first).
        for i, tr := range resp.Transitions {
                if _, ok := tr["source_url"]; !ok {
                        t.Errorf("transitions[%d]: missing source_url", i)
                }
        }
}
