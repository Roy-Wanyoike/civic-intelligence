// Package main — corrections handlers (issue #166 — Correction system).
//
// Implements an in-memory correction store keyed by correction ID. The
// canonical trust.corrections table (migration 018) already owns the schema;
// once the api service has a real Postgres connection this in-memory store
// will be replaced by a SQL-backed repository. The HTTP contract
// (request/response shapes) is intentionally stable so the frontend does not
// change.
//
// Endpoints:
//   POST /api/v1/corrections — submit a correction request (public; anonymous
//                              submitters must provide an email)
//   GET  /api/v1/corrections — list corrections (admin scope: user:admin or
//                              evidence:write)
//
// Architectural contract (ADR-0011 immutable bill versions; ADR-0005 AI
// cannot mutate truth): corrections are NEVER applied directly to canonical
// state. Each submission becomes a row with status pending → in_review →
// accepted | rejected | superseded. When accepted, a *new* immutable version
// is created on the target entity and the previous_state JSONB preserves the
// before-snapshot for the audit trail (trust.audit_events).
package main

import (
        "crypto/rand"
        "encoding/hex"
        "encoding/json"
        "errors"
        "net/http"
        "net/url"
        "strings"
        "sync"
        "time"

        "github.com/Roy-Wanyoike/civic-intelligence/packages/auth"
        "github.com/Roy-Wanyoike/civic-intelligence/services/api/internal/middleware"
)

// --- Correction category constants (mirror trust.corrections.category enum) ---

const (
        CorrectionWrongFact              = "wrong_fact"
        CorrectionWrongSource            = "wrong_source"
        CorrectionWrongCitation          = "wrong_citation"
        CorrectionOutdated               = "outdated"
        CorrectionIncorrectInterpretation = "incorrect_interpretation"
        CorrectionMissingInformation     = "missing_information"
        CorrectionConflictingSources     = "conflicting_sources"
        CorrectionBrokenDocument         = "broken_document"
)

// validCorrectionCategories is the allow-list of correction categories.
var validCorrectionCategories = map[string]bool{
        CorrectionWrongFact:               true,
        CorrectionWrongSource:             true,
        CorrectionWrongCitation:           true,
        CorrectionOutdated:                true,
        CorrectionIncorrectInterpretation: true,
        CorrectionMissingInformation:      true,
        CorrectionConflictingSources:      true,
        CorrectionBrokenDocument:          true,
}

// validCorrectionStatuses mirrors the trust.corrections.status CHECK constraint.
var validCorrectionStatuses = map[string]bool{
        "pending":     true,
        "in_review":   true,
        "accepted":    true,
        "rejected":    true,
        "superseded":  true,
}

// validCorrectionTargetTypes mirrors trust.corrections.target_type CHECK.
var validCorrectionTargetTypes = map[string]bool{
        "bill":        true,
        "act":         true,
        "claim":       true,
        "evidence":    true,
        "source":      true,
        "regulation":  true,
        "policy":      true,
        "person":      true,
        "committee":   true,
}

// CorrectionRecord mirrors a row in trust.corrections.
type CorrectionRecord struct {
        ID              string                 `json:"id"`
        TargetType      string                 `json:"target_type"`
        TargetID        string                 `json:"target_id"`
        Reason          string                 `json:"reason"`
        Category        string                 `json:"category"`
        PageURL         string                 `json:"page_url,omitempty"`
        Evidence        string                 `json:"evidence,omitempty"`
        PreviousState   map[string]any         `json:"previous_state,omitempty"`
        CorrectedState  map[string]any         `json:"corrected_state,omitempty"`
        SubmittedBy     string                 `json:"submitted_by,omitempty"`
        SubmittedEmail  string                 `json:"submitted_email,omitempty"`
        ReviewedBy      string                 `json:"reviewed_by,omitempty"`
        ReviewedAt      *time.Time             `json:"reviewed_at,omitempty"`
        Status          string                 `json:"status"`
        CreatedAt       time.Time              `json:"created_at"`
        UpdatedAt       time.Time              `json:"updated_at"`
}

// correctionRequest is the JSON body for POST /api/v1/corrections.
// Anonymous submitters must provide submitted_email; authenticated submitters
// are tracked via submitted_by (set from the principal).
type correctionRequest struct {
        TargetType     string         `json:"target_type"`
        TargetID       string         `json:"target_id"`
        Category       string         `json:"category"`
        Reason         string         `json:"reason"`
        PageURL        string         `json:"page_url"`
        Evidence       string         `json:"evidence"`
        SubmittedEmail string         `json:"submitted_email"`
        CorrectedState map[string]any `json:"corrected_state,omitempty"`
}

// CorrectionStore is the in-memory backing store for the corrections API.
// All methods are safe for concurrent use.
//
// In production this is replaced by SQL queries against trust.corrections
// (migration 018). The query shapes are intentionally simple.
type CorrectionStore struct {
        mu          sync.RWMutex
        corrections map[string]CorrectionRecord
}

// NewCorrectionStore returns an empty in-memory correction store.
func NewCorrectionStore() *CorrectionStore {
        return &CorrectionStore{corrections: make(map[string]CorrectionRecord)}
}

// Submit creates a new correction record. The submitter may be anonymous
// (submittedBy = "") as long as submittedEmail is provided. Returns the
// created record (status = pending).
func (s *CorrectionStore) Submit(req correctionRequest, submittedBy string) (CorrectionRecord, error) {
        req.TargetType = strings.ToLower(strings.TrimSpace(req.TargetType))
        req.TargetID = strings.TrimSpace(req.TargetID)
        req.Category = strings.ToLower(strings.TrimSpace(req.Category))
        req.Reason = strings.TrimSpace(req.Reason)
        req.PageURL = strings.TrimSpace(req.PageURL)
        req.Evidence = strings.TrimSpace(req.Evidence)
        req.SubmittedEmail = strings.TrimSpace(req.SubmittedEmail)

        if !validCorrectionTargetTypes[req.TargetType] {
                return CorrectionRecord{}, errors.New("invalid target_type")
        }
        if req.TargetID == "" {
                return CorrectionRecord{}, errors.New("target_id required")
        }
        if !validCorrectionCategories[req.Category] {
                return CorrectionRecord{}, errors.New("invalid category")
        }
        if req.Reason == "" {
                return CorrectionRecord{}, errors.New("reason required")
        }
        if req.PageURL == "" {
                return CorrectionRecord{}, errors.New("page_url required")
        }
        if !isValidURL(req.PageURL) {
                return CorrectionRecord{}, errors.New("page_url must be a valid http(s) URL")
        }
        if req.Evidence != "" && !isValidURL(req.Evidence) {
                return CorrectionRecord{}, errors.New("evidence must be a valid http(s) URL")
        }
        if submittedBy == "" && req.SubmittedEmail == "" {
                return CorrectionRecord{}, errors.New("submitted_email required for anonymous submissions")
        }

        now := time.Now().UTC()
        rec := CorrectionRecord{
                ID:             newCorrectionID(),
                TargetType:     req.TargetType,
                TargetID:       req.TargetID,
                Reason:         req.Reason,
                Category:       req.Category,
                PageURL:        req.PageURL,
                Evidence:       req.Evidence,
                CorrectedState: req.CorrectedState,
                SubmittedBy:    submittedBy,
                SubmittedEmail: req.SubmittedEmail,
                Status:         "pending",
                CreatedAt:      now,
                UpdatedAt:      now,
        }

        s.mu.Lock()
        defer s.mu.Unlock()
        s.corrections[rec.ID] = rec
        return rec, nil
}

// List returns all corrections, optionally filtered by status. Sorted by
// CreatedAt descending (newest first).
func (s *CorrectionStore) List(status string) []CorrectionRecord {
        s.mu.RLock()
        defer s.mu.RUnlock()
        out := make([]CorrectionRecord, 0, len(s.corrections))
        for _, c := range s.corrections {
                if status != "" && c.Status != status {
                        continue
                }
                out = append(out, c)
        }
        // Sort by CreatedAt descending.
        for i := 1; i < len(out); i++ {
                for j := i; j > 0 && out[j].CreatedAt.After(out[j-1].CreatedAt); j-- {
                        out[j], out[j-1] = out[j-1], out[j]
                }
        }
        return out
}

// Get returns a single correction by ID.
func (s *CorrectionStore) Get(id string) (CorrectionRecord, bool) {
        s.mu.RLock()
        defer s.mu.RUnlock()
        rec, ok := s.corrections[id]
        return rec, ok
}

// Transition mutates a correction's status. Used by the (future) admin
// review endpoint. Returns the updated record or an error if the
// transition is invalid.
func (s *CorrectionStore) Transition(id, newStatus, reviewerID string) (CorrectionRecord, error) {
        if !validCorrectionStatuses[newStatus] {
                return CorrectionRecord{}, errors.New("invalid status")
        }
        s.mu.Lock()
        defer s.mu.Unlock()
        rec, ok := s.corrections[id]
        if !ok {
                return CorrectionRecord{}, errors.New("correction not found")
        }
        now := time.Now().UTC()
        rec.Status = newStatus
        rec.ReviewedBy = reviewerID
        rec.ReviewedAt = &now
        rec.UpdatedAt = now
        s.corrections[id] = rec
        return rec, nil
}

// newCorrectionID generates a 12-byte hex ID with the `crt_` prefix so
// correction IDs are easy to spot in logs.
func newCorrectionID() string {
        b := make([]byte, 12)
        if _, err := rand.Read(b); err != nil {
                return "crt_" + time.Now().UTC().Format("20060102150405")
        }
        return "crt_" + hex.EncodeToString(b)
}

// isValidURL returns true if the string parses as an http(s) URL with a
// non-empty host.
func isValidURL(s string) bool {
        if s == "" {
                return false
        }
        u, err := url.Parse(s)
        if err != nil {
                return false
        }
        if u.Scheme != "http" && u.Scheme != "https" {
                return false
        }
        return u.Host != ""
}

// --- HTTP handlers ---

// makeCorrectionsHandler routes between POST (submit) and GET (list) on
// /api/v1/corrections.
func makeCorrectionsHandler(store *CorrectionStore) http.HandlerFunc {
        return func(w http.ResponseWriter, r *http.Request) {
                switch r.Method {
                case http.MethodPost:
                        handleCorrectionSubmit(w, r, store)
                case http.MethodGet:
                        handleCorrectionsList(w, r, store)
                default:
                        w.Header().Set("Allow", "GET, POST")
                        writeError(w, http.StatusMethodNotAllowed, "method_not_allowed",
                                "only GET and POST are supported on /api/v1/corrections")
                }
        }
}

// handleCorrectionSubmit handles POST /api/v1/corrections.
//
// Public: anonymous submitters must provide submitted_email. Authenticated
// submitters (any scope) are tracked by submitted_by. We never apply the
// correction directly to canonical state — accepted corrections go through
// the review queue (issue #166).
func handleCorrectionSubmit(w http.ResponseWriter, r *http.Request, store *CorrectionStore) {
        var req correctionRequest
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
                writeError(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
                return
        }

        p := middleware.PrincipalFromRequest(r)
        submittedBy := ""
        if p.IsAuthenticated() {
                submittedBy = p.UserID
        }

        rec, err := store.Submit(req, submittedBy)
        if err != nil {
                // Distinguish validation errors from server errors.
                switch {
                case strings.Contains(err.Error(), "required") ||
                        strings.Contains(err.Error(), "invalid") ||
                        strings.Contains(err.Error(), "must be"):
                        writeError(w, http.StatusBadRequest, "bad_request", err.Error())
                default:
                        writeError(w, http.StatusInternalServerError, "submit_failed", err.Error())
                }
                return
        }

        writeJSON(w, http.StatusCreated, rec)
}

// handleCorrectionsList handles GET /api/v1/corrections (admin only).
//
// Required scope: user:admin OR evidence:write. The check is enforced via
// the principal's scopes (set by OptionalAuth upstream).
func handleCorrectionsList(w http.ResponseWriter, r *http.Request, store *CorrectionStore) {
        p := middleware.PrincipalFromRequest(r)
        if p.IsAnonymous() {
                writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
                return
        }
        // Admin scope check.
        if !p.Can(auth.ScopeUserAdmin, auth.ScopeEvidenceWrite) {
                writeError(w, http.StatusForbidden, "forbidden",
                        "admin scope (user:admin or evidence:write) required to list corrections")
                return
        }

        status := strings.TrimSpace(strings.ToLower(r.URL.Query().Get("status")))
        if status != "" && !validCorrectionStatuses[status] {
                writeError(w, http.StatusBadRequest, "bad_request",
                        "invalid status; must be one of: pending, in_review, accepted, rejected, superseded")
                return
        }

        items := store.List(status)
        writeJSON(w, http.StatusOK, map[string]any{
                "items": items,
                "total": len(items),
        })
}

// makeCorrectionDetailHandler handles GET /api/v1/corrections/{id} (admin
// only). Returns the full record including previous_state + corrected_state.
func makeCorrectionDetailHandler(store *CorrectionStore) http.HandlerFunc {
        return func(w http.ResponseWriter, r *http.Request) {
                if r.Method != http.MethodGet {
                        w.Header().Set("Allow", "GET")
                        writeError(w, http.StatusMethodNotAllowed, "method_not_allowed",
                                "only GET is supported on /api/v1/corrections/{id}")
                        return
                }
                p := middleware.PrincipalFromRequest(r)
                if p.IsAnonymous() {
                        writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
                        return
                }
                if !p.Can(auth.ScopeUserAdmin, auth.ScopeEvidenceWrite) {
                        writeError(w, http.StatusForbidden, "forbidden",
                                "admin scope (user:admin or evidence:write) required to view a correction")
                        return
                }

                id := strings.TrimPrefix(r.URL.Path, "/api/v1/corrections/")
                id = strings.Trim(id, "/")
                if id == "" {
                        writeError(w, http.StatusBadRequest, "bad_request", "correction id required")
                        return
                }
                rec, ok := store.Get(id)
                if !ok {
                        writeError(w, http.StatusNotFound, "not_found", "correction not found: "+id)
                        return
                }
                writeJSON(w, http.StatusOK, rec)
        }
}
