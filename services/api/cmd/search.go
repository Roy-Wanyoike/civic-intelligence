// Package main — search handler (in-memory stopgap).
//
// Spec section 38 calls for keyword/semantic/hybrid/temporal/jurisdictional
// search across Bills, Acts, Constitution, Articles, Institutions, People,
// Governments, Legislatures, Loans, Debt, Documents, Evidence, Topics, and
// Research. The full surface is out of scope for this stopgap (GAP-49-1).
//
// What this file implements:
//   - GET /api/v1/search?q=<term> — in-memory substring search across:
//       * Bills         — derived from kenya_seed.KenyaActs (each Act carries
//                          the originating BillID + ActName + Description +
//                          SourceURL; we synthesise a bill search item per
//                          Act so the search endpoint works offline without
//                          calling the Kenya Law network adapter).
//       * Acts          — from the package-level actRepo (seeded from
//                          kenya_seed.KenyaActs via legislation.Wire()).
//       * Constitution  — kenya_seed.KenyaConstitutionArticles() (flat list
//                          of articles across every chapter).
//   - Ranking: title match (1000 pts, minus position) + body match
//     (100 pts, minus position/10). Sorted descending. Ties broken by
//     (type asc, ID asc) for stable ordering across runs.
//   - Result cap: 20 (maxSearchResults). The stopgap is intentionally
//     small-fan-out; Postgres FTS will lift this when wired.
//
// STOPGAP NOTE (issue #45): this in-memory search is a stopgap until
// Postgres full-text search (migration 015_search_projections) is wired
// end-to-end. The Postgres projection already materialises a search index
// for bills, acts, and constitution articles — the API layer just does not
// yet query it. When FTS lands, this handler should be replaced with a
// repository call; the response shape (searchResponse / searchItem) should
// remain stable so the frontend does not need to change.
//
// The handler is registered in main.go as
// `apiHandler.HandleFunc("/api/v1/search", handleSearch)`.
package main

import (
        "net/http"
        "sort"
        "strings"

        kenya_seed "github.com/Roy-Wanyoike/civic-intelligence/adapters/kenya/kenya_seed"
        "github.com/Roy-Wanyoike/civic-intelligence/services/api/internal/middleware"
        "github.com/Roy-Wanyoike/civic-intelligence/services/legislation"
)

// searchItem is a single search result returned by /api/v1/search. The shape
// is intentionally minimal so the frontend can render uniformly across types
// (bills, acts, constitution articles, and future types). The score field
// carries the internal relevance ranking and is NOT serialised.
type searchItem struct {
        Type    string `json:"type"`             // "bill" | "act" | "constitution_article"
        ID      string `json:"id"`
        Title   string `json:"title"`
        Snippet string `json:"snippet,omitempty"`
        URL     string `json:"url"`
        score   int    `json:"-"` // internal relevance ranking; not serialised
}

// maxSearchResults caps the number of items returned by /api/v1/search.
// Spec section 38 calls for keyword search across many entity types; the
// in-memory stopgap keeps the response small until Postgres FTS is wired.
const maxSearchResults = 20

// searchStopgapNote is attached to every search response so developers
// (and downstream consumers) can see that the result set is sourced from
// the in-memory stopgap, not the eventual Postgres FTS projection. The
// note is verbatim on every response; downstream UIs can choose to render
// it as a developer hint.
const searchStopgapNote = "Search — in-memory stopgap (issue #45). Postgres FTS pending."

// handleSearch implements GET /api/v1/search?q=<term>. The handler is a
// method-less top-level function so it can be wired directly into the
// ServeMux from main.go without any closure allocation.
//
// Empty query: returns 200 with an empty items array (not 400). The
// frontend search page renders before the user types anything; an empty
// 200 response lets it render "no results yet" without an error state.
// This is also more aligned with modern search APIs (Google Custom Search,
// Algolia, Meilisearch all return 200 + empty for empty queries).
//
// Non-empty query: substring-matches the lower-cased query against the
// title + body of every bill, act, and constitution article in the seed
// dataset. Results are ranked, capped at 20, and returned with a stable
// shape.
func handleSearch(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodGet {
                writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
                return
        }

        q := strings.TrimSpace(r.URL.Query().Get("q"))
        if q == "" {
                // Empty query: 200 + empty items. See handler doc comment.
                writeJSON(w, http.StatusOK, map[string]any{
                        "q":     "",
                        "items": []any{},
                        "total": 0,
                        "note":  searchStopgapNote,
                })
                return
        }

        needle := strings.ToLower(q)
        items := make([]searchItem, 0, 32)

        // Bills — derived from kenya_seed.KenyaActs. Each seed Act carries
        // the originating BillID + ActName + Description + SourceURL. We
        // synthesise a bill search item per Act (the platform does not yet
        // have a separate bills seed dataset; the Kenya Law adapter
        // discovers bills at runtime via DiscoverBills, which is a network
        // call we deliberately avoid here so search is always available
        // offline).
        for _, a := range kenya_seed.KenyaActs {
                title := a.ActName
                body := a.Description
                pos, score := scoreMatch(needle, title, body)
                if score == 0 {
                        continue
                }
                items = append(items, searchItem{
                        Type:    "bill",
                        ID:      a.BillID,
                        Title:   title,
                        Snippet: truncateSnippet(body, pos),
                        URL:     a.SourceURL,
                        score:   score,
                })
        }

        // Acts — from the in-memory actRepo (seeded from kenya_seed.KenyaActs
        // via legislation.Wire()). The repo is the authoritative source for
        // Acts; the API layer queries it directly via ListActs.
        if acts, err := actRepo.ListActs(r.Context(), legislation.ActFilter{}); err == nil {
                for _, a := range acts {
                        title := a.ActName
                        body := a.Description
                        pos, score := scoreMatch(needle, title, body)
                        if score == 0 {
                                continue
                        }
                        items = append(items, searchItem{
                                Type:    "act",
                                ID:      string(a.ID),
                                Title:   title,
                                Snippet: truncateSnippet(body, pos),
                                URL:     a.SourceURL,
                                score:   score,
                        })
                }
        }

        // Constitution articles — flat list from kenya_seed. The Number
        // field (e.g. "Article 19") is prepended to the title so users can
        // search by article number too.
        for _, art := range kenya_seed.KenyaConstitutionArticles() {
                title := art.Number + " — " + art.Title
                body := art.Text
                pos, score := scoreMatch(needle, title, body)
                if score == 0 {
                        continue
                }
                items = append(items, searchItem{
                        Type:    "constitution_article",
                        ID:      string(art.ID),
                        Title:   title,
                        Snippet: truncateSnippet(body, pos),
                        URL:     art.SourceURL,
                        score:   score,
                })
        }

        // Sort by score descending; ties broken by (type asc, ID asc) so
        // the order is stable across runs (no map-iteration nondeterminism).
        sort.Slice(items, func(i, j int) bool {
                if items[i].score != items[j].score {
                        return items[i].score > items[j].score
                }
                if items[i].Type != items[j].Type {
                        return items[i].Type < items[j].Type
                }
                return items[i].ID < items[j].ID
        })

        // Cap at maxSearchResults. The slice may be longer than the cap;
        // truncate. (We do not paginate in the stopgap; the spec calls
        // for cursor pagination once FTS is wired.)
        if len(items) > maxSearchResults {
                items = items[:maxSearchResults]
        }

        // ENG-J1: filter by the country from the request context. The seed
        // data today is Kenya-only (kenya_seed.KenyaActs + KenyaConstitution
        // articles + the ActRepository seeded from kenya_seed), so a non-KE
        // country yields an empty list (which is the correct behaviour —
        // there is no Uganda seed yet). When other country adapters ship
        // their seed data, the search handler should be extended to consult
        // each adapter's seed (the per-adapter seeds already implement the
        // same shape).
        //
        // "ALL" (GlobalCountry) returns results from every country's seed
        // data — the dashboard view.
        country := middleware.CountryFromContext(r.Context())
        if country != "" && country != middleware.GlobalCountry {
                filtered := make([]searchItem, 0, len(items))
                for _, it := range items {
                        // kenya_seed rows all start with "ke-" IDs; we use that
                        // prefix as a quick country filter. When a real multi-country
                        // search index lands, this is replaced by an explicit
                        // country field on each searchItem.
                        if matchesCountry(it.ID, country) {
                                filtered = append(filtered, it)
                        }
                }
                items = filtered
        }

        writeJSON(w, http.StatusOK, map[string]any{
                "q":       q,
                "items":   items,
                "total":   len(items),
                "country": country,
                "note":    searchStopgapNote,
        })
}

// matchesCountry reports whether the supplied item ID belongs to the
// supplied country. The platform's seed IDs all carry a country prefix
// (e.g. "ke-act-data-protection-2019", "ug-bill-…", "za-act-…"). When
// the search index moves to a real Postgres FTS projection, this
// helper is replaced by a per-row country_code column.
func matchesCountry(id, country string) bool {
        if id == "" || country == "" {
                return false
        }
        c := strings.ToLower(country)
        return strings.HasPrefix(strings.ToLower(id), c+"-") || strings.HasPrefix(strings.ToLower(id), c+"_")
}

// scoreMatch scores a search hit against (title, body). Returns the
// absolute position of the first match (used for snippet trimming) and a
// numeric relevance score.
//
// Score components:
//   - Title match: 1000 base points, minus the character offset of the
//     match (earlier = better).
//   - Body match: 100 base points, minus the character offset / 10
//     (earlier = better, but the offset is damped so it doesn't dominate
//     the title score).
//
// Returns (pos=-1, score=0) when neither title nor body contains the
// needle. The position is the smaller of (title match, body match) so the
// snippet can be anchored at the earliest hit; -1 when no match.
//
// The scoring is intentionally simple (substring match + position). It is
// NOT a relevance model — there is no TF-IDF, no BM25, no field weighting
// beyond the title/body bonus. The stopgap exists to un-break search until
// Postgres FTS (with proper ranking) is wired.
func scoreMatch(needle, title, body string) (pos int, score int) {
        pos = -1
        score = 0
        if needle == "" {
                return pos, score
        }

        titleL := strings.ToLower(title)
        if i := strings.Index(titleL, needle); i >= 0 {
                score += 1000 - i
                pos = i
        }

        bodyL := strings.ToLower(body)
        if j := strings.Index(bodyL, needle); j >= 0 {
                score += 100 - j/10
                if pos < 0 {
                        pos = j
                }
        }

        return pos, score
}

// truncateSnippet returns up to ~160 characters of body, anchored at the
// first match position when possible. When the body is shorter than the
// window, the entire body is returned.
//
// The window is centred on the match position so the user sees context
// around the hit. An ellipsis is prepended/appended when the snippet is
// trimmed on either side, so the UI can render it inline without further
// transformation.
func truncateSnippet(body string, matchPos int) string {
        if body == "" {
                return ""
        }
        const window = 160
        if len(body) <= window {
                return body
        }
        start := 0
        if matchPos > 0 {
                // Centre the window around the match position.
                start = matchPos - window/3
                if start < 0 {
                        start = 0
                }
        }
        end := start + window
        if end > len(body) {
                end = len(body)
                start = end - window
                if start < 0 {
                        start = 0
                }
        }
        snippet := body[start:end]
        if start > 0 {
                snippet = "…" + snippet
        }
        if end < len(body) {
                snippet = snippet + "…"
        }
        return snippet
}
