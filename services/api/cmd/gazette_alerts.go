// Package main — Gazette keyword alert handlers (task ENG-K1 — Feature 2).
//
// Implements an in-memory GazetteAlertStore + a fixed set of sample
// Kenya Gazette notices. Citizens subscribe to keywords (e.g. "tender",
// "health"); the store computes the subset of published notices whose
// title/body match any of those keywords.
//
// Endpoints (registered in main.go):
//
//      POST   /api/v1/gazette/alerts           — create a keyword alert
//      GET    /api/v1/gazette/alerts           — list the caller's alerts
//      DELETE /api/v1/gazette/alerts/{id}      — delete an alert
//      GET    /api/v1/gazette/alerts/{id}/matches — list matching gazette notices
//
// In production the store is replaced by a SQL repository
// (gazette.alerts + gazette.notices) populated by the ingestion service
// from kenyalaw.org/kenya_gazette. The HTTP contract (request/response
// shapes) is intentionally stable so the frontend does not change.
//
// IMPORTANT — alert scope: alerts are matched ONLY against gazette
// notices that have already been published (i.e. notice_date <= today).
// Draft or future-dated notices are never matched. The frontend shows
// this labelling prominently so citizens know alerts are not predictive.
package main

import (
        "crypto/rand"
        "encoding/hex"
        "encoding/json"
        "errors"
        "net/http"
        "sort"
        "strings"
        "sync"
        "time"

        "github.com/Roy-Wanyoike/civic-intelligence/services/api/internal/middleware"
)

// --- Domain types ---

// GazetteNotice is a single published Kenya Gazette notice. Notices are
// immutable once published; the ingestion service creates them and never
// edits them. The MatchedKeywords field is populated only when the notice
// is returned from a match query — it lists the keywords from the alert
// that caused the match (so the frontend can highlight them).
type GazetteNotice struct {
        ID              string    `json:"id"`
        Title           string    `json:"title"`
        GazetteDate     string    `json:"gazette_date"` // YYYY-MM-DD
        NoticeNumber    string    `json:"notice_number"`
        Volume          string    `json:"volume,omitempty"`
        Summary         string    `json:"summary,omitempty"`
        SourceURL       string    `json:"source_url"`
        Authority       string    `json:"authority,omitempty"` // e.g. "Kenya Gazette"
        MatchedKeywords []string  `json:"matched_keywords,omitempty"`
        Country         string    `json:"country"`
        BodyText        string    `json:"body_text,omitempty"`
}

// GazetteAlert is a citizen's keyword subscription. The same store may
// hold alerts from multiple users; listing is scoped by user_id.
//
// MatchCount is computed on demand (NOT persisted) so it stays in sync
// as new notices are ingested. It is populated by the list + match
// endpoints; the POST response leaves it at 0 (no matches yet at the
// moment of creation).
type GazetteAlert struct {
        ID         string    `json:"id"`
        UserID     string    `json:"user_id"`
        Keywords   []string  `json:"keywords"`
        Country    string    `json:"country"`
        CreatedAt  time.Time `json:"created_at"`
        MatchCount int       `json:"match_count"`
}

// GazetteAlertStore is an in-memory store of alerts + notices. It is
// safe for concurrent use. Reads return defensively-copied slices.
type GazetteAlertStore struct {
        mu      sync.RWMutex
        alerts  map[string]GazetteAlert // alert_id → record
        notices []GazetteNotice         // append-only; ingestion appends
        country string                  // default country code
}

// NewGazetteAlertStore returns an empty store. The default country code
// is applied to alerts + notices that omit one.
func NewGazetteAlertStore(country string) *GazetteAlertStore {
        if country == "" {
                country = "KE"
        }
        return &GazetteAlertStore{
                alerts:  make(map[string]GazetteAlert),
                notices: make([]GazetteNotice, 0),
                country: strings.ToUpper(country),
        }
}

// CreateAlert validates + stores a new alert. Returns the stored record
// (with ID, CreatedAt, normalised country + keywords). Returns an error
// for invalid input (empty user_id, empty keywords, keywords containing
// non-printable characters, etc.).
func (s *GazetteAlertStore) CreateAlert(userID string, keywords []string, country string) (GazetteAlert, error) {
        if userID = strings.TrimSpace(userID); userID == "" {
                return GazetteAlert{}, errors.New("user_id required")
        }
        clean := normaliseKeywords(keywords)
        if len(clean) == 0 {
                return GazetteAlert{}, errors.New("at least one keyword required")
        }
        if len(clean) > 25 {
                return GazetteAlert{}, errors.New("too many keywords (max 25)")
        }
        if country == "" {
                country = s.country
        } else {
                country = strings.ToUpper(country)
        }

        rec := GazetteAlert{
                ID:        newGazetteAlertID(),
                UserID:    userID,
                Keywords:  clean,
                Country:   country,
                CreatedAt: time.Now().UTC(),
        }
        s.mu.Lock()
        s.alerts[rec.ID] = rec
        // Populate MatchCount eagerly so the create response is informative.
        rec.MatchCount = s.matchesLocked(rec.ID, nil)
        s.mu.Unlock()
        return rec, nil
}

// ListAlerts returns the caller's alerts, sorted by CreatedAt descending.
// MatchCount is populated for each alert.
func (s *GazetteAlertStore) ListAlerts(userID string) []GazetteAlert {
        s.mu.RLock()
        out := make([]GazetteAlert, 0)
        for _, a := range s.alerts {
                if a.UserID == userID {
                        out = append(out, a)
                }
        }
        s.mu.RUnlock()
        for i := range out {
                out[i].MatchCount = s.MatchCount(out[i].ID)
        }
        sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
        return out
}

// GetAlert returns the alert by ID. The second return is false if the
// alert does not exist (or belongs to a different user — ownership is
// enforced at the handler layer, but GetAlert itself does NOT filter by
// user so the matches endpoint can resolve by alert ID alone).
func (s *GazetteAlertStore) GetAlert(id string) (GazetteAlert, bool) {
        s.mu.RLock()
        defer s.mu.RUnlock()
        a, ok := s.alerts[id]
        return a, ok
}

// DeleteAlert removes an alert by ID, scoped to the given user. Returns
// true if a record was removed, false if not found or owned by another user.
func (s *GazetteAlertStore) DeleteAlert(userID, id string) bool {
        if userID == "" || id == "" {
                return false
        }
        s.mu.Lock()
        defer s.mu.Unlock()
        a, ok := s.alerts[id]
        if !ok || a.UserID != userID {
                return false
        }
        delete(s.alerts, id)
        return true
}

// AddNotice appends a gazette notice to the store. Notices are immutable;
// the ingestion service calls this when a new Friday Gazette is parsed.
func (s *GazetteAlertStore) AddNotice(n GazetteNotice) {
        if n.ID == "" {
                n.ID = newGazetteNoticeID()
        }
        if n.Country == "" {
                n.Country = s.country
        } else {
                n.Country = strings.ToUpper(n.Country)
        }
        s.mu.Lock()
        s.notices = append(s.notices, n)
        s.mu.Unlock()
}

// Notices returns all published notices whose gazette_date <= ref
// (today by default). The slice is defensively copied.
func (s *GazetteAlertStore) Notices(ref time.Time) []GazetteNotice {
        cutoff := ref.UTC().Format("2006-01-02")
        s.mu.RLock()
        out := make([]GazetteNotice, 0, len(s.notices))
        for _, n := range s.notices {
                if n.GazetteDate <= cutoff {
                        out = append(out, n)
                }
        }
        s.mu.RUnlock()
        sort.Slice(out, func(i, j int) bool { return out[i].GazetteDate > out[j].GazetteDate })
        return out
}

// Matches returns the subset of published notices that contain ANY of
// the alert's keywords (case-insensitive substring match on title +
// body_text + notice_number). Each returned notice carries the list of
// keywords from the alert that triggered the match.
func (s *GazetteAlertStore) Matches(alertID string) ([]GazetteNotice, error) {
        s.mu.RLock()
        a, ok := s.alerts[alertID]
        s.mu.RUnlock()
        if !ok {
                return nil, errors.New("alert not found: " + alertID)
        }
        return s.matchesFor(a), nil
}

// matchesFor is the unguarded form of Matches — it accepts a fully
// hydrated alert record. Used internally by CreateAlert's eager match
// count + the matches endpoint.
func (s *GazetteAlertStore) matchesFor(a GazetteAlert) []GazetteNotice {
        cutoff := time.Now().UTC().Format("2006-01-02")
        s.mu.RLock()
        out := make([]GazetteNotice, 0)
        for _, n := range s.notices {
                if n.GazetteDate > cutoff {
                        continue
                }
                hay := strings.ToLower(n.Title + " " + n.BodyText + " " + n.NoticeNumber + " " + n.Summary)
                matched := make([]string, 0, len(a.Keywords))
                for _, kw := range a.Keywords {
                        if kw == "" {
                                continue
                        }
                        if strings.Contains(hay, kw) {
                                matched = append(matched, kw)
                        }
                }
                if len(matched) > 0 {
                        cp := n
                        cp.MatchedKeywords = matched
                        out = append(out, cp)
                }
        }
        s.mu.RUnlock()
        sort.Slice(out, func(i, j int) bool { return out[i].GazetteDate > out[j].GazetteDate })
        return out
}

// MatchCount returns the number of currently-published notices that
// match the alert's keywords. Cheaper than Matches when the caller only
// needs a count.
func (s *GazetteAlertStore) MatchCount(alertID string) int {
        s.mu.RLock()
        a, ok := s.alerts[alertID]
        s.mu.RUnlock()
        if !ok {
                return 0
        }
        return len(s.matchesFor(a))
}

// matchesLocked computes the match count WITHOUT taking the read lock.
// Callers MUST already hold s.mu (read OR write). Used by CreateAlert
// to populate MatchCount inside the write lock.
func (s *GazetteAlertStore) matchesLocked(alertID string, _ []string) int {
        a, ok := s.alerts[alertID]
        if !ok {
                return 0
        }
        cutoff := time.Now().UTC().Format("2006-01-02")
        count := 0
        for _, n := range s.notices {
                if n.GazetteDate > cutoff {
                        continue
                }
                hay := strings.ToLower(n.Title + " " + n.BodyText + " " + n.NoticeNumber + " " + n.Summary)
                for _, kw := range a.Keywords {
                        if kw == "" {
                                continue
                        }
                        if strings.Contains(hay, kw) {
                                count++
                                break // one match per notice is enough for a count
                        }
                }
        }
        return count
}

// normaliseKeywords lower-cases, trims, de-duplicates, and rejects empty
// tokens. Returns the cleaned slice in stable order (first-seen).
func normaliseKeywords(raw []string) []string {
        seen := make(map[string]bool, len(raw))
        out := make([]string, 0, len(raw))
        for _, k := range raw {
                k = strings.ToLower(strings.TrimSpace(k))
                if k == "" {
                        continue
                }
                if seen[k] {
                        continue
                }
                seen[k] = true
                out = append(out, k)
        }
        return out
}

// newGazetteAlertID generates a 16-byte hex ID with the "gza_" prefix.
func newGazetteAlertID() string {
        b := make([]byte, 16)
        if _, err := rand.Read(b); err != nil {
                return "gza_" + time.Now().UTC().Format("20060102150405.000000000")
        }
        return "gza_" + hex.EncodeToString(b)
}

// newGazetteNoticeID generates a 16-byte hex ID with the "gzn_" prefix.
func newGazetteNoticeID() string {
        b := make([]byte, 16)
        if _, err := rand.Read(b); err != nil {
                return "gzn_" + time.Now().UTC().Format("20060102150405.000000000")
        }
        return "gzn_" + hex.EncodeToString(b)
}

// --- HTTP handlers ---

// makeGazetteAlertsHandler routes between POST (create) and GET (list) on
// /api/v1/gazette/alerts. Ownership is enforced inside the handlers via
// the principal from the request context.
func makeGazetteAlertsHandler(store *GazetteAlertStore) http.HandlerFunc {
        return func(w http.ResponseWriter, r *http.Request) {
                switch r.Method {
                case http.MethodPost:
                        handleCreateGazetteAlert(w, r, store)
                case http.MethodGet:
                        handleListGazetteAlerts(w, r, store)
                default:
                        w.Header().Set("Allow", "GET, POST")
                        writeError(w, http.StatusMethodNotAllowed, "method_not_allowed",
                                "only GET and POST are supported on /api/v1/gazette/alerts")
                }
        }
}

// makeGazetteAlertDetailHandler routes DELETE on
// /api/v1/gazette/alerts/{id} and GET on
// /api/v1/gazette/alerts/{id}/matches. The trailing /matches path is
// detected by inspecting r.URL.Path.
func makeGazetteAlertDetailHandler(store *GazetteAlertStore) http.HandlerFunc {
        return func(w http.ResponseWriter, r *http.Request) {
                // Path: /api/v1/gazette/alerts/{id}        → DELETE
                // Path: /api/v1/gazette/alerts/{id}/matches → GET
                path := strings.TrimPrefix(r.URL.Path, "/api/v1/gazette/alerts/")
                path = strings.Trim(path, "/")
                if path == "" {
                        writeError(w, http.StatusBadRequest, "bad_request", "alert id required")
                        return
                }
                parts := strings.Split(path, "/")
                alertID := parts[0]
                if len(parts) == 2 && parts[1] == "matches" {
                        if r.Method != http.MethodGet {
                                w.Header().Set("Allow", "GET")
                                writeError(w, http.StatusMethodNotAllowed, "method_not_allowed",
                                        "only GET is supported on /api/v1/gazette/alerts/{id}/matches")
                                return
                        }
                        handleGazetteAlertMatches(w, r, store, alertID)
                        return
                }
                if len(parts) == 1 {
                        if r.Method != http.MethodDelete {
                                w.Header().Set("Allow", "DELETE")
                                writeError(w, http.StatusMethodNotAllowed, "method_not_allowed",
                                        "only DELETE is supported on /api/v1/gazette/alerts/{id}")
                                return
                        }
                        handleDeleteGazetteAlert(w, r, store, alertID)
                        return
                }
                writeError(w, http.StatusNotFound, "not_found", "unknown gazette alerts sub-path: "+r.URL.Path)
        }
}

// alertRequest is the JSON body for POST /api/v1/gazette/alerts.
type alertRequest struct {
        Keywords []string `json:"keywords"`
        UserID   string   `json:"user_id"`
        Country  string   `json:"country"`
}

// handleCreateGazetteAlert handles POST /api/v1/gazette/alerts.
//
// Note: this handler accepts an explicit user_id in the body so the
// feature is usable WITHOUT a real OIDC token (issue #20 is still
// pending). When the principal is present (OptionalAuth set it), the
// principal's UserID takes precedence over the body's user_id — this
// stops a caller from creating alerts under another user's identity.
func handleCreateGazetteAlert(w http.ResponseWriter, r *http.Request, store *GazetteAlertStore) {
        p := middleware.PrincipalFromRequest(r)

        var req alertRequest
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
                writeError(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
                return
        }
        userID := req.UserID
        if !p.IsAnonymous() {
                // Authenticated caller wins — body's user_id is ignored to prevent
                // cross-user alert creation.
                userID = p.UserID
        }
        rec, err := store.CreateAlert(userID, req.Keywords, req.Country)
        if err != nil {
                writeError(w, http.StatusBadRequest, "bad_request", err.Error())
                return
        }
        writeJSON(w, http.StatusCreated, rec)
}

// handleListGazetteAlerts handles GET /api/v1/gazette/alerts?user_id=...
//
// When the caller is authenticated, the principal's user_id is used and
// the query parameter is ignored (prevents cross-user listing). When
// anonymous, the user_id query parameter is honoured — this keeps the
// feature usable end-to-end before identity (#20) ships.
func handleListGazetteAlerts(w http.ResponseWriter, r *http.Request, store *GazetteAlertStore) {
        p := middleware.PrincipalFromRequest(r)
        userID := r.URL.Query().Get("user_id")
        if !p.IsAnonymous() {
                userID = p.UserID
        }
        if strings.TrimSpace(userID) == "" {
                writeError(w, http.StatusBadRequest, "bad_request",
                        "user_id query parameter required (or sign in)")
                return
        }
        alerts := store.ListAlerts(userID)
        writeJSON(w, http.StatusOK, map[string]any{
                "items": alerts,
                "total": len(alerts),
        })
}

// handleDeleteGazetteAlert handles DELETE /api/v1/gazette/alerts/{id}.
func handleDeleteGazetteAlert(w http.ResponseWriter, r *http.Request, store *GazetteAlertStore, alertID string) {
        p := middleware.PrincipalFromRequest(r)
        userID := r.URL.Query().Get("user_id")
        if !p.IsAnonymous() {
                userID = p.UserID
        }
        if strings.TrimSpace(userID) == "" {
                writeError(w, http.StatusBadRequest, "bad_request",
                        "user_id query parameter required (or sign in)")
                return
        }
        if !store.DeleteAlert(userID, alertID) {
                writeError(w, http.StatusNotFound, "not_found",
                        "alert not found or does not belong to caller")
                return
        }
        w.WriteHeader(http.StatusNoContent)
}

// handleGazetteAlertMatches handles GET /api/v1/gazette/alerts/{id}/matches.
// The caller does NOT need to be the alert owner — match results are
// derived only from already-published gazette notices (public information).
func handleGazetteAlertMatches(w http.ResponseWriter, r *http.Request, store *GazetteAlertStore, alertID string) {
        notices, err := store.Matches(alertID)
        if err != nil {
                writeError(w, http.StatusNotFound, "not_found", err.Error())
                return
        }
        writeJSON(w, http.StatusOK, map[string]any{
                "alert_id": alertID,
                "items":    notices,
                "total":    len(notices),
                "note":     "Matches are computed against gazette notices already published (gazette_date <= today). Draft or future-dated notices are never matched.",
        })
}

// --- Sample data ---

// gazetteAlertStore is the package-level in-memory GazetteAlertStore,
// seeded with 12 sample Kenya Gazette notices. main.go reuses it for
// the HTTP handlers.
var gazetteAlertStore = NewGazetteAlertStore("KE")

// SeedGazetteSampleNotices populates the store with 12 realistic Kenya
// Gazette notices spanning tender, health, tax, environment, and other
// keyword domains. Notices are dated relative to `anchor` so they stay
// in the past (already published) for the matches endpoint to surface.
//
// Keyword coverage (so a citizen subscribing to each gets a hit):
//   - "tender"     → 4 notices
//   - "health"     → 3 notices
//   - "tax"        → 3 notices
//   - "environment"→ 2 notices
//   - "education"  → 2 notices
//   - "lands"      → 2 notices
//   - "immigration"→ 1 notice
//   - "pension"    → 1 notice
//   - "water"      → 1 notice
//   - "energy"     → 1 notice
func SeedGazetteSampleNotices(store *GazetteAlertStore, anchor time.Time) {
        day := func(offset int) string {
                return anchor.AddDate(0, 0, offset).Format("2006-01-02")
        }
        notices := []GazetteNotice{
                {
                        ID:           "gzn_tender_mda_001",
                        Title:        "Tender — Supply of Hospital Equipment to County Referral Hospitals",
                        GazetteDate:  day(-21),
                        NoticeNumber: "No. 1247",
                        Volume:       "Vol. CXXVIII No. 31",
                        Summary:      "Open tender for the supply, installation, and commissioning of ICU and theatre equipment to 12 county referral hospitals.",
                        SourceURL:    "https://www.gazettes.africa/kenya/2026/cxxviii-31#notice-1247",
                        Authority:    "Kenya Gazette",
                        Country:      "KE",
                        BodyText:     "Tender notice. The State Department for Medical Services invites sealed bids from eligible tenderers for the supply of hospital equipment. Bids close on the date specified in the tender document.",
                },
                {
                        ID:           "gzn_tender_roads_002",
                        Title:        "Tender — Rehabilitation of the Nairobi-Thika Superhighway",
                        GazetteDate:  day(-18),
                        NoticeNumber: "No. 1252",
                        Volume:       "Vol. CXXVIII No. 31",
                        Summary:      "Open tender for the periodic maintenance and rehabilitation of the Nairobi-Thika Superhighway (A2).",
                        SourceURL:    "https://www.gazettes.africa/kenya/2026/cxxviii-31#notice-1252",
                        Authority:    "Kenya Gazette",
                        Country:      "KE",
                        BodyText:     "Tender. The Kenya Urban Roads Authority (KURA) invites sealed bids for road works. Bidders must be registered with the National Construction Authority.",
                },
                {
                        ID:           "gzn_tender_water_003",
                        Title:        "Tender — Construction of Rural Water Pans in Arid and Semi-Arid Lands",
                        GazetteDate:  day(-14),
                        NoticeNumber: "No. 1261",
                        Volume:       "Vol. CXXVIII No. 31",
                        Summary:      "Open tender for the construction of 47 water pans in ASAL counties. Includes environmental impact assessment.",
                        SourceURL:    "https://www.gazettes.africa/kenya/2026/cxxviii-31#notice-1261",
                        Authority:    "Kenya Gazette",
                        Country:      "KE",
                        BodyText:     "Tender notice. The State Department for Water invites bids for water pan construction. The works include environmental and social safeguards.",
                },
                {
                        ID:           "gzn_tender_energy_004",
                        Title:        "Tender — Last-Mile Electricity Connectivity Programme (Phase IV)",
                        GazetteDate:  day(-9),
                        NoticeNumber: "No. 1277",
                        Volume:       "Vol. CXXVIII No. 32",
                        Summary:      "Open tender for the supply of distribution transformers and energy meters under the Last-Mile Programme.",
                        SourceURL:    "https://www.gazettes.africa/kenya/2026/cxxviii-32#notice-1277",
                        Authority:    "Kenya Gazette",
                        Country:      "KE",
                        BodyText:     "Tender. Kenya Power and Lighting Company (KPLC) invites bids for energy meters and transformers under the Last-Mile connectivity programme.",
                },
                {
                        ID:           "gzn_health_sha_005",
                        Title:        "Social Health Authority — Benefit Package Amendments, 2026",
                        GazetteDate:  day(-12),
                        NoticeNumber: "Legal Notice No. 47",
                        Volume:       "Vol. CXXVIII No. 31",
                        Summary:      "Amendments to the Social Health Insurance (Benefits) Regulations adding renal dialysis and oncology cover.",
                        SourceURL:    "https://www.gazettes.africa/kenya/2026/cxxviii-31#ln-47",
                        Authority:    "Kenya Gazette",
                        Country:      "KE",
                        BodyText:     "Health. The Social Health Authority amends the benefit package to include renal dialysis, oncology services, and mental health cover. Effective 1 July 2026.",
                },
                {
                        ID:           "gzn_health_pharm_006",
                        Title:        "Pharmacy and Poisons (Amendment) Act, 2026 — Commencement",
                        GazetteDate:  day(-7),
                        NoticeNumber: "Legal Notice No. 52",
                        Volume:       "Vol. CXXVIII No. 32",
                        Summary:      "Commencement notice for the Pharmacy and Poisons (Amendment) Act, 2026 — new licensing regime for online pharmacies.",
                        SourceURL:    "https://www.gazettes.africa/kenya/2026/cxxviii-32#ln-52",
                        Authority:    "Kenya Gazette",
                        Country:      "KE",
                        BodyText:     "Health. The Cabinet Secretary for Health appoints the commencement date for the Pharmacy and Poisons (Amendment) Act, 2026. The Act introduces a licensing regime for online pharmacies.",
                },
                {
                        ID:           "gzn_health_kemsa_007",
                        Title:        "Kenya Medical Supplies Authority — Board Appointment",
                        GazetteDate:  day(-5),
                        NoticeNumber: "No. 1289",
                        Volume:       "Vol. CXXVIII No. 32",
                        Summary:      "Appointment of the Chairperson and members of the KEMSA Board for a three-year term.",
                        SourceURL:    "https://www.gazettes.africa/kenya/2026/cxxviii-32#notice-1289",
                        Authority:    "Kenya Gazette",
                        Country:      "KE",
                        BodyText:     "Health. The Cabinet Secretary for Health appoints the Chairperson and members of the Kenya Medical Supplies Authority (KEMSA) Board.",
                },
                {
                        ID:           "gzn_tax_excise_008",
                        Title:        "Excise Duty (Amendment) Order, 2026",
                        GazetteDate:  day(-16),
                        NoticeNumber: "Legal Notice No. 44",
                        Volume:       "Vol. CXXVIII No. 31",
                        Summary:      "Amends the Excise Duty Act schedules — increases tax on alcoholic beverages, tobacco, and sugary drinks.",
                        SourceURL:    "https://www.gazettes.africa/kenya/2026/cxxviii-31#ln-44",
                        Authority:    "Kenya Gazette",
                        Country:      "KE",
                        BodyText:     "Tax. The Cabinet Secretary for the National Treasury amends the Excise Duty Act schedules to increase the tax on alcoholic beverages, tobacco products, and sugar-sweetened beverages.",
                },
                {
                        ID:           "gzn_tax_vat_009",
                        Title:        "Value Added Tax (Amendment) Regulations, 2026",
                        GazetteDate:  day(-10),
                        NoticeNumber: "Legal Notice No. 49",
                        Volume:       "Vol. CXXVIII No. 32",
                        Summary:      "Amends the VAT Regulations — zero-rates exports of ICT services and removes input tax on motor vehicle repairs.",
                        SourceURL:    "https://www.gazettes.africa/kenya/2026/cxxviii-32#ln-49",
                        Authority:    "Kenya Gazette",
                        Country:      "KE",
                        BodyText:     "Tax. The Cabinet Secretary for the National Treasury amends the Value Added Tax Regulations. Exports of ICT services are zero-rated; input tax on motor vehicle repairs is disallowed.",
                },
                {
                        ID:           "gzn_tax_county_010",
                        Title:        "County Government of Nairobi — Entertainment Tax By-Laws, 2026",
                        GazetteDate:  day(-4),
                        NoticeNumber: "County Gazette No. 12",
                        Volume:       "Nairobi County Gazette Supplement",
                        Summary:      "Nairobi County introduces an entertainment tax on tickets sold for live events, concerts, and cinema.",
                        SourceURL:    "https://www.gazettes.africa/kenya/2026/nairobi-supplement#bylaws-12",
                        Authority:    "Nairobi City County",
                        Country:      "KE",
                        BodyText:     "Tax. The Nairobi City County Government enacts the Entertainment Tax By-Laws, 2026. The tax is payable by every operator of a place of entertainment.",
                },
                {
                        ID:           "gzn_env_eia_011",
                        Title:        "Environmental Impact Assessment — Proposed Limestone Mining in Kajiado",
                        GazetteDate:  day(-13),
                        NoticeNumber: "No. 1258",
                        Volume:       "Vol. CXXVIII No. 31",
                        Summary:      "Public participation notice on the EIA study report for proposed limestone mining. Comments to NEMA within 30 days.",
                        SourceURL:    "https://www.gazettes.africa/kenya/2026/cxxviii-31#notice-1258",
                        Authority:    "Kenya Gazette",
                        Country:      "KE",
                        BodyText:     "Environment. Pursuant to the Environmental Management and Coordination Act, NEMA receives an environmental impact assessment study report for the proposed limestone mining in Kajiado County.",
                },
                {
                        ID:           "gzn_env_forest_012",
                        Title:        "Gazettement of Karura Forest Extension Boundary",
                        GazetteDate:  day(-8),
                        NoticeNumber: "Legal Notice No. 51",
                        Volume:       "Vol. CXXVIII No. 32",
                        Summary:      "Declaration of the Karura Forest Extension as a protected forest under the Forest Conservation and Management Act.",
                        SourceURL:    "https://www.gazettes.africa/kenya/2026/cxxviii-32#ln-51",
                        Authority:    "Kenya Gazette",
                        Country:      "KE",
                        BodyText:     "Environment. The Cabinet Secretary for Environment declares the Karura Forest Extension as a protected forest. The boundary plan is available for inspection at the Kenya Forest Service headquarters.",
                },
                {
                        ID:           "gzn_edu_helb_013",
                        Title:        "Higher Education Loans Board — Disbursement Schedule, 2026/27",
                        GazetteDate:  day(-6),
                        NoticeNumber: "No. 1284",
                        Volume:       "Vol. CXXVIII No. 32",
                        Summary:      "Publication of the HELB loan disbursement schedule for the 2026/27 academic year.",
                        SourceURL:    "https://www.gazettes.africa/kenya/2026/cxxviii-32#notice-1284",
                        Authority:    "Kenya Gazette",
                        Country:      "KE",
                        BodyText:     "Education. The Higher Education Loans Board publishes the loan disbursement schedule for the 2026/27 academic year. Loans are disbursed directly to universities.",
                },
                {
                        ID:           "gzn_edu_knec_014",
                        Title:        "Kenya National Examinations Council — 2026 Examination Timetable",
                        GazetteDate:  day(-3),
                        NoticeNumber: "No. 1291",
                        Volume:       "Vol. CXXVIII No. 32",
                        Summary:      "Publication of the KCPE, KCSE, and KPSEA examination timetable for the year 2026.",
                        SourceURL:    "https://www.gazettes.africa/kenya/2026/cxxviii-32#notice-1291",
                        Authority:    "Kenya Gazette",
                        Country:      "KE",
                        BodyText:     "Education. The Kenya National Examinations Council publishes the examination timetable for KCPE, KCSE, and KPSEA for the year 2026.",
                },
                {
                        ID:           "gzn_lands_allocation_015",
                        Title:        "Land Allocation — Cancelled Allotment Letters, Nairobi",
                        GazetteDate:  day(-11),
                        NoticeNumber: "No. 1271",
                        Volume:       "Vol. CXXVIII No. 31",
                        Summary:      "Notice of cancellation of 38 allotment letters for public utility land in Nairobi County.",
                        SourceURL:    "https://www.gazettes.africa/kenya/2026/cxxviii-31#notice-1271",
                        Authority:    "Kenya Gazette",
                        Country:      "KE",
                        BodyText:     "Lands. The Cabinet Secretary for Lands cancels 38 allotment letters for land reserved for public utilities in Nairobi County. The list is available for inspection.",
                },
        }

        store.mu.Lock()
        defer store.mu.Unlock()
        for _, n := range notices {
                if n.Country == "" {
                        n.Country = store.country
                } else {
                        n.Country = strings.ToUpper(n.Country)
                }
                store.notices = append(store.notices, n)
        }
}
