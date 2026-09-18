// Package main — trust + provenance handlers (issue #165 — Provenance explorer).
//
// Implements an in-memory trust store that backs the provenance explorer API
// and the /trust page. The trust schema in migration 018_trust_schema.up.sql
// already owns the canonical tables (trust.sources, trust.claims, trust.evidence,
// trust.contradictions, trust.corrections, trust.audit_events,
// trust.review_queue). Once the api service has a real Postgres connection,
// this in-memory store will be replaced by a SQL-backed repository that reads
// from those tables; the HTTP contract (request/response shapes) is
// intentionally stable so the frontend does not change.
//
// Endpoints:
//   GET /api/v1/provenance/{entity_type}/{id} — returns the full evidence chain
//   GET /api/v1/evidence/{id}                — evidence detail
//   GET /api/v1/claims/{id}/evidence         — claims with their evidence
//   GET /api/v1/contradictions                — list active source conflicts
//   GET /api/v1/sources/{id}                 — source detail with health
package main

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

// --- Authority + verification state constants (mirror trust.claims enum) ---

const (
	AuthorityPrimaryOfficial    = "PRIMARY_OFFICIAL"
	AuthorityOfficialRepository = "OFFICIAL_REPOSITORY"
	AuthoritySecondaryVerified = "SECONDARY_VERIFIED"
	AuthorityUnverified         = "UNVERIFIED"

	ClaimTypeFact         = "FACT"
	ClaimTypeExplanation  = "EXPLANATION"
	ClaimTypeInference    = "INFERENCE"
	ClaimTypeUnknown      = "UNKNOWN"

	VerificationUnverified = "UNVERIFIED"
	VerificationDiscovered  = "DISCOVERED"
	VerificationExtracted   = "EXTRACTED"
	VerificationValidating = "VALIDATING"
	VerificationVerified    = "VERIFIED"
	VerificationConflicted  = "CONFLICTED"
	VerificationCorrected  = "CORRECTED"
	VerificationSuperseded  = "SUPERSEDED"
	VerificationRejected     = "REJECTED"

	ContradictionDetected     = "detected"
	ContradictionUnderReview  = "under_review"
	ContradictionResolved     = "resolved"
	ContradictionDismissed    = "dismissed"
)

// validAuthorityLevels is the allow-list for trust.sources.authority_level.
var validAuthorityLevels = map[string]bool{
	AuthorityPrimaryOfficial:     true,
	AuthorityOfficialRepository: true,
	AuthoritySecondaryVerified: true,
	AuthorityUnverified:         true,
}

// --- Records (mirror trust.* tables) ---

// TrustSource mirrors a row in trust.sources.
type TrustSource struct {
	ID                string    `json:"id"`
	InstitutionID     string    `json:"institution_id"`
	Country           string    `json:"country"`
	SourceType        string    `json:"source_type"`
	AuthorityLevel    string    `json:"authority_level"`
	OfficialURL       string    `json:"official_url"`
	Domain            string    `json:"domain"`
	Status            string    `json:"status"`
	VerificationMethod string   `json:"verification_method,omitempty"`
	LastVerifiedAt    *time.Time `json:"last_verified_at,omitempty"`
	HealthStatus      string    `json:"health_status"`
	CreatedAt         time.Time `json:"created_at"`
	// LastCheck is the most recent source_checks row (denormalised for the
	// /trust page so the frontend can show liveness without a second call).
	LastCheck         *SourceCheck `json:"last_check,omitempty"`
}

// SourceCheck mirrors a row in trust.source_checks.
type SourceCheck struct {
	ID          string    `json:"id"`
	SourceID    string    `json:"source_id"`
	CheckedAt   time.Time `json:"checked_at"`
	HTTPStatus  int       `json:"http_status"`
	LatencyMS   int       `json:"latency_ms"`
	TLSValid    bool      `json:"tls_valid"`
	ContentHash string    `json:"content_hash,omitempty"`
	Changed     bool      `json:"changed"`
}

// Claim mirrors a row in trust.claims.
type Claim struct {
	ID                string     `json:"id"`
	Subject           string     `json:"subject"`
	Predicate         string     `json:"predicate"`
	Object            string     `json:"object"`
	ClaimType         string     `json:"claim_type"`
	Text              string     `json:"text"`
	Confidence        float64    `json:"confidence"`
	VerificationState string     `json:"verification_state"`
	CreatedAt         time.Time  `json:"created_at"`
	ValidFrom         *time.Time `json:"valid_from,omitempty"`
	ValidTo           *time.Time `json:"valid_to,omitempty"`
}

// Evidence mirrors a row in trust.evidence.
type Evidence struct {
	ID           string    `json:"id"`
	ClaimID      string    `json:"claim_id"`
	DocumentID   string    `json:"document_id,omitempty"`
	SnapshotID   string    `json:"snapshot_id,omitempty"`
	PageNumber   int       `json:"page_number,omitempty"`
	Section      string    `json:"section,omitempty"`
	Paragraph    string    `json:"paragraph,omitempty"`
	TextSpan     string    `json:"text_span,omitempty"`
	SourceURL    string    `json:"source_url"`
	RetrievedAt  time.Time `json:"retrieved_at"`
	ContentHash  string    `json:"content_hash,omitempty"`
	// Source is the denormalised trust.sources row this evidence points at.
	// Populated by the provenance explorer for the /trust page.
	Source       *TrustSource `json:"source,omitempty"`
}

// Contradiction mirrors a row in trust.contradictions.
type Contradiction struct {
	ID          string     `json:"id"`
	ClaimAID    string     `json:"claim_a_id"`
	ClaimBID    string     `json:"claim_b_id"`
	SourceAID   string     `json:"source_a_id,omitempty"`
	SourceBID   string     `json:"source_b_id,omitempty"`
	DetectedAt  time.Time  `json:"detected_at"`
	Status      string     `json:"status"`
	Resolution  string     `json:"resolution,omitempty"`
	ReviewerID  string     `json:"reviewer_id,omitempty"`
	ResolvedAt  *time.Time `json:"resolved_at,omitempty"`
	// Denormalised for the API consumer.
	ClaimA *Claim       `json:"claim_a,omitempty"`
	ClaimB *Claim       `json:"claim_b,omitempty"`
	SourceA *TrustSource `json:"source_a,omitempty"`
	SourceB *TrustSource `json:"source_b,omitempty"`
}

// --- Store ---

// TrustStore is the in-memory backing store for the provenance API.
// All methods are safe for concurrent use.
//
// In production this is replaced by SQL queries against the trust schema
// (migration 018). The query shapes are intentionally simple so the swap is
// mechanical.
type TrustStore struct {
	mu              sync.RWMutex
	sources         map[string]TrustSource
	sourceChecks    map[string][]SourceCheck // source_id → checks (sorted desc by checked_at)
	claims          map[string]Claim
	evidence        map[string]Evidence     // evidence_id → row
	claimEvidence   map[string][]Evidence   // claim_id → evidence list
	contradictions  map[string]Contradiction
}

// NewTrustStore returns an empty in-memory trust store.
func NewTrustStore() *TrustStore {
	return &TrustStore{
		sources:        make(map[string]TrustSource),
		sourceChecks:   make(map[string][]SourceCheck),
		claims:         make(map[string]Claim),
		evidence:       make(map[string]Evidence),
		claimEvidence:  make(map[string][]Evidence),
		contradictions: make(map[string]Contradiction),
	}
}

// AddSource inserts a source (or no-ops if the ID already exists).
func (s *TrustStore) AddSource(src TrustSource) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if src.CreatedAt.IsZero() {
		src.CreatedAt = time.Now().UTC()
	}
	if _, ok := s.sources[src.ID]; !ok {
		s.sources[src.ID] = src
	}
}

// AddClaim inserts a claim.
func (s *TrustStore) AddClaim(c Claim) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if c.CreatedAt.IsZero() {
		c.CreatedAt = time.Now().UTC()
	}
	s.claims[c.ID] = c
}

// AddEvidence inserts an evidence row and indexes it by claim_id.
func (s *TrustStore) AddEvidence(e Evidence) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if e.RetrievedAt.IsZero() {
		e.RetrievedAt = time.Now().UTC()
	}
	s.evidence[e.ID] = e
	s.claimEvidence[e.ClaimID] = append(s.claimEvidence[e.ClaimID], e)
}

// AddContradiction inserts a contradiction row.
func (s *TrustStore) AddContradiction(c Contradiction) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if c.DetectedAt.IsZero() {
		c.DetectedAt = time.Now().UTC()
	}
	s.contradictions[c.ID] = c
}

// AddSourceCheck appends a liveness probe to a source's check history.
func (s *TrustStore) AddSourceCheck(check SourceCheck) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if check.CheckedAt.IsZero() {
		check.CheckedAt = time.Now().UTC()
	}
	s.sourceChecks[check.SourceID] = append(s.sourceChecks[check.SourceID], check)

	// Denormalise the latest check onto the source row.
	if src, ok := s.sources[check.SourceID]; ok {
		c := check
		src.LastCheck = &c
		src.LastVerifiedAt = &check.CheckedAt
		if check.HTTPStatus >= 200 && check.HTTPStatus < 300 && check.TLSValid {
			src.HealthStatus = "healthy"
		} else if check.HTTPStatus >= 400 || !check.TLSValid {
			src.HealthStatus = "degraded"
		}
		s.sources[check.SourceID] = src
	}
}

// GetSource returns a single source by ID (with denormalised last check).
func (s *TrustStore) GetSource(id string) (TrustSource, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	src, ok := s.sources[id]
	return src, ok
}

// ListSources returns every source, optionally filtered by authority level
// or country. Results are sorted by authority level (PRIMARY_OFFICIAL first)
// then domain (alphabetical) so the /trust page renders a stable order.
func (s *TrustStore) ListSources(authority, country string) []TrustSource {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]TrustSource, 0, len(s.sources))
	for _, src := range s.sources {
		if authority != "" && src.AuthorityLevel != authority {
			continue
		}
		if country != "" && src.Country != country {
			continue
		}
		out = append(out, src)
	}

	// Sort: PRIMARY_OFFICIAL → OFFICIAL_REPOSITORY → SECONDARY_VERIFIED → UNVERIFIED,
	// then domain ascending.
	authorityRank := map[string]int{
		AuthorityPrimaryOfficial:     0,
		AuthorityOfficialRepository: 1,
		AuthoritySecondaryVerified: 2,
		AuthorityUnverified:         3,
	}
	sort.SliceStable(out, func(i, j int) bool {
		ri, rj := authorityRank[out[i].AuthorityLevel], authorityRank[out[j].AuthorityLevel]
		if ri != rj {
			return ri < rj
		}
		return out[i].Domain < out[j].Domain
	})
	return out
}

// GetClaim returns a single claim by ID.
func (s *TrustStore) GetClaim(id string) (Claim, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.claims[id]
	return c, ok
}

// EvidenceForClaim returns every evidence row attached to the given claim,
// newest-first. Each evidence row's Source field is populated if the
// document_url maps to a known trust source.
func (s *TrustStore) EvidenceForClaim(claimID string) []Evidence {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rows := s.claimEvidence[claimID]
	out := make([]Evidence, len(rows))
	copy(out, rows)
	// Sort newest-first by RetrievedAt.
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].RetrievedAt.After(out[j].RetrievedAt)
	})
	// Hydrate source where the evidence's source_url matches a known source.
	for i := range out {
		for _, src := range s.sources {
			if src.OfficialURL == out[i].SourceURL {
				s := src
				out[i].Source = &s
				break
			}
		}
	}
	return out
}

// GetEvidence returns a single evidence row by ID, with the Source hydrated.
func (s *TrustStore) GetEvidence(id string) (Evidence, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	e, ok := s.evidence[id]
	if !ok {
		return Evidence{}, false
	}
	for _, src := range s.sources {
		if src.OfficialURL == e.SourceURL {
			s := src
			e.Source = &s
			break
		}
	}
	return e, true
}

// ClaimsForEntity returns every claim whose Subject matches the given
// (entity_type, entity_id). The subject is encoded as "{entity_type}:{entity_id}"
// (e.g. "bill:00000000-...-001"). This mirrors how the evidence pipeline
// stamps provenance; canonical entities live in legislation/ingestion.
func (s *TrustStore) ClaimsForEntity(entityType, entityID string) []Claim {
	s.mu.RLock()
	defer s.mu.RUnlock()
	subject := entityType + ":" + entityID
	out := make([]Claim, 0)
	for _, c := range s.claims {
		if c.Subject == subject {
			out = append(out, c)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].CreatedAt.After(out[j].CreatedAt)
	})
	return out
}

// ListContradictions returns contradictions filtered by status. Empty status
// returns all rows. Sorted by detected_at descending (newest first).
func (s *TrustStore) ListContradictions(status string) []Contradiction {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Contradiction, 0, len(s.contradictions))
	for _, c := range s.contradictions {
		if status != "" && c.Status != status {
			continue
		}
		// Hydrate claim + source lookups for the API response.
		if a, ok := s.claims[c.ClaimAID]; ok {
			c.ClaimA = &a
		}
		if b, ok := s.claims[c.ClaimBID]; ok {
			c.ClaimB = &b
		}
		if c.SourceAID != "" {
			if src, ok := s.sources[c.SourceAID]; ok {
				c.SourceA = &src
			}
		}
		if c.SourceBID != "" {
			if src, ok := s.sources[c.SourceBID]; ok {
				c.SourceB = &src
			}
		}
		out = append(out, c)
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].DetectedAt.After(out[j].DetectedAt)
	})
	return out
}

// --- ID generators ---

func newTrustID(prefix string) string {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		return prefix + time.Now().UTC().Format("20060102150405")
	}
	return prefix + "_" + hex.EncodeToString(b)
}

func newClaimID() string    { return newTrustID("clm") }
func newEvidenceID() string { return newTrustID("evd") }
func newContradictionID() string { return newTrustID("ctr") }
func newSourceCheckID() string  { return newTrustID("chk") }

// --- HTTP handlers ---

// makeProvenanceHandler handles GET /api/v1/provenance/{entity_type}/{id}.
//
// Returns the full evidence chain for the given entity: every claim whose
// subject is "{entity_type}:{entity_id}", each with its evidence list and
// source. The response shape is intentionally a tree so the /trust page can
// render the chain without a second round-trip.
func makeProvenanceHandler(store *TrustStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", "GET")
			writeError(w, http.StatusMethodNotAllowed, "method_not_allowed",
				"only GET is supported on /api/v1/provenance/{entity_type}/{id}")
			return
		}

		// Extract path: /api/v1/provenance/{entity_type}/{id}
		path := strings.TrimPrefix(r.URL.Path, "/api/v1/provenance/")
		parts := strings.SplitN(path, "/", 2)
		if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
			writeError(w, http.StatusBadRequest, "bad_request",
				"path must be /api/v1/provenance/{entity_type}/{id}")
			return
		}
		entityType := strings.ToLower(strings.TrimSpace(parts[0]))
		entityID := strings.TrimSpace(parts[1])
		if !validProvenanceEntityTypes[entityType] {
			writeError(w, http.StatusBadRequest, "bad_request",
				"unsupported entity_type: "+entityType+
					" (allowed: bill, act, regulation, policy, claim, source, person, committee)")
			return
		}

		// For "claim" entity_type we return the single claim + its evidence.
		if entityType == "claim" {
			c, ok := store.GetClaim(entityID)
			if !ok {
				writeError(w, http.StatusNotFound, "not_found", "claim not found: "+entityID)
				return
			}
			ev := store.EvidenceForClaim(entityID)
			writeJSON(w, http.StatusOK, provenanceResponse{
				EntityType: entityType,
				EntityID:   entityID,
				Claims:     []claimWithEvidence{{Claim: c, Evidence: ev}},
				Total:      1,
			})
			return
		}

		// For "source" entity_type we return the source itself.
		if entityType == "source" {
			src, ok := store.GetSource(entityID)
			if !ok {
				writeError(w, http.StatusNotFound, "not_found", "source not found: "+entityID)
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{
				"entity_type": entityType,
				"entity_id":   entityID,
				"source":      src,
				"total":       1,
			})
			return
		}

		claims := store.ClaimsForEntity(entityType, entityID)
		out := make([]claimWithEvidence, 0, len(claims))
		for _, c := range claims {
			out = append(out, claimWithEvidence{
				Claim:    c,
				Evidence: store.EvidenceForClaim(c.ID),
			})
		}

		writeJSON(w, http.StatusOK, provenanceResponse{
			EntityType: entityType,
			EntityID:   entityID,
			Claims:     out,
			Total:      len(out),
		})
	}
}

// validProvenanceEntityTypes is the allow-list of entity types whose
// provenance can be queried.
var validProvenanceEntityTypes = map[string]bool{
	"bill":        true,
	"act":         true,
	"regulation":  true,
	"policy":      true,
	"claim":       true,
	"source":      true,
	"person":      true,
	"committee":   true,
	"institution": true,
}

// provenanceResponse is the JSON shape returned by the provenance endpoint.
type provenanceResponse struct {
	EntityType string              `json:"entity_type"`
	EntityID   string              `json:"entity_id"`
	Claims     []claimWithEvidence `json:"claims"`
	Total      int                 `json:"total"`
}

type claimWithEvidence struct {
	Claim    Claim     `json:"claim"`
	Evidence []Evidence `json:"evidence"`
}

// makeEvidenceHandler handles GET /api/v1/evidence/{id}.
func makeEvidenceHandler(store *TrustStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", "GET")
			writeError(w, http.StatusMethodNotAllowed, "method_not_allowed",
				"only GET is supported on /api/v1/evidence/{id}")
			return
		}
		id := strings.TrimPrefix(r.URL.Path, "/api/v1/evidence/")
		id = strings.Trim(id, "/")
		if id == "" {
			writeError(w, http.StatusBadRequest, "bad_request", "evidence id required")
			return
		}
		e, ok := store.GetEvidence(id)
		if !ok {
			writeError(w, http.StatusNotFound, "not_found", "evidence not found: "+id)
			return
		}
		writeJSON(w, http.StatusOK, e)
	}
}

// makeClaimEvidenceHandler handles GET /api/v1/claims/{id}/evidence.
func makeClaimEvidenceHandler(store *TrustStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", "GET")
			writeError(w, http.StatusMethodNotAllowed, "method_not_allowed",
				"only GET is supported on /api/v1/claims/{id}/evidence")
			return
		}
		path := strings.TrimPrefix(r.URL.Path, "/api/v1/claims/")
		parts := strings.SplitN(path, "/", 2)
		if len(parts) == 0 || parts[0] == "" {
			writeError(w, http.StatusBadRequest, "bad_request", "claim id required")
			return
		}
		claimID := parts[0]
		if len(parts) < 2 || parts[1] != "evidence" {
			writeError(w, http.StatusNotFound, "not_found",
				"unknown sub-route; only /evidence is supported")
			return
		}
		c, ok := store.GetClaim(claimID)
		if !ok {
			writeError(w, http.StatusNotFound, "not_found", "claim not found: "+claimID)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"claim":    c,
			"evidence": store.EvidenceForClaim(claimID),
			"total":    len(store.EvidenceForClaim(claimID)),
		})
	}
}

// makeContradictionsHandler handles GET /api/v1/contradictions.
// Optional query: ?status=detected|under_review|resolved|dismissed
func makeContradictionsHandler(store *TrustStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", "GET")
			writeError(w, http.StatusMethodNotAllowed, "method_not_allowed",
				"only GET is supported on /api/v1/contradictions")
			return
		}
		status := strings.TrimSpace(strings.ToLower(r.URL.Query().Get("status")))
		if status != "" {
			if !validContradictionStatuses[status] {
				writeError(w, http.StatusBadRequest, "bad_request",
					"invalid status; must be one of: detected, under_review, resolved, dismissed")
				return
			}
		}
		items := store.ListContradictions(status)
		writeJSON(w, http.StatusOK, map[string]any{
			"items": items,
			"total": len(items),
		})
	}
}

var validContradictionStatuses = map[string]bool{
	ContradictionDetected:    true,
	ContradictionUnderReview: true,
	ContradictionResolved:    true,
	ContradictionDismissed:   true,
}

// makeTrustSourceHandler handles GET /api/v1/sources/{id} — source detail
// with health + last check.
func makeTrustSourceHandler(store *TrustStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", "GET")
			writeError(w, http.StatusMethodNotAllowed, "method_not_allowed",
				"only GET is supported on /api/v1/sources/{id}")
			return
		}
		id := strings.TrimPrefix(r.URL.Path, "/api/v1/sources/")
		id = strings.Trim(id, "/")
		if id == "" {
			writeError(w, http.StatusBadRequest, "bad_request", "source id required")
			return
		}
		src, ok := store.GetSource(id)
		if !ok {
			writeError(w, http.StatusNotFound, "not_found", "source not found: "+id)
			return
		}
		writeJSON(w, http.StatusOK, src)
	}
}

// makeTrustSourcesListHandler handles GET /api/v1/sources — list sources
// with optional ?authority= and ?country= filters.
func makeTrustSourcesListHandler(store *TrustStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", "GET")
			writeError(w, http.StatusMethodNotAllowed, "method_not_allowed",
				"only GET is supported on /api/v1/sources")
			return
		}
		authority := strings.TrimSpace(strings.ToUpper(r.URL.Query().Get("authority")))
		if authority != "" && !validAuthorityLevels[authority] {
			writeError(w, http.StatusBadRequest, "bad_request",
				"invalid authority; must be one of: PRIMARY_OFFICIAL, OFFICIAL_REPOSITORY, SECONDARY_VERIFIED, UNVERIFIED")
			return
		}
		country := strings.TrimSpace(strings.ToUpper(r.URL.Query().Get("country")))
		items := store.ListSources(authority, country)
		writeJSON(w, http.StatusOK, map[string]any{
			"items": items,
			"total": len(items),
		})
	}
}

// errTrustSeedMissing is returned if a caller attempts to use the trust store
// before seeding it. Used internally to give a clearer error in dev.
var errTrustSeedMissing = errors.New("trust store not seeded; call SeedSampleTrustData first")

// SeedSampleTrustData populates the store with a small, hand-curated set of
// sources, claims, evidence, and one active contradiction. The data mirrors
// the shape of rows the ingestion + evidence pipeline will produce once fully
// wired, and gives the /trust page + provenance API something concrete to
// render in QA + the demo.
//
// Idempotent: re-seeding the same store is a no-op (sources keyed by ID).
func (s *TrustStore) SeedSampleTrustData() {
	// === Sources ===
	sources := []TrustSource{
		{
			ID:                "src_ke_parliament",
			InstitutionID:     "ke-parliament",
			Country:           "KE",
			SourceType:        "parliament",
			AuthorityLevel:    AuthorityPrimaryOfficial,
			OfficialURL:       "https://www.parliament.go.ke/",
			Domain:            "www.parliament.go.ke",
			Status:            "active",
			VerificationMethod: "manual",
			HealthStatus:      "healthy",
		},
		{
			ID:                "src_ke_kenyalaw",
			InstitutionID:     "ke-kenyalaw",
			Country:           "KE",
			SourceType:        "repository",
			AuthorityLevel:    AuthorityOfficialRepository,
			OfficialURL:       "https://www.kenyalaw.org/",
			Domain:            "www.kenyalaw.org",
			Status:            "active",
			VerificationMethod: "content_hash",
			HealthStatus:      "healthy",
		},
		{
			ID:                "src_ke_senate",
			InstitutionID:     "ke-senate",
			Country:           "KE",
			SourceType:        "parliament",
			AuthorityLevel:    AuthorityPrimaryOfficial,
			OfficialURL:       "https://www.senate.go.ke/",
			Domain:            "www.senate.go.ke",
			Status:            "active",
			VerificationMethod: "manual",
			HealthStatus:      "healthy",
		},
	}
	for _, src := range sources {
		s.AddSource(src)
	}

	// === Source checks (liveness probes) ===
	now := time.Now().UTC()
	s.AddSourceCheck(SourceCheck{
		ID:          newSourceCheckID(),
		SourceID:    "src_ke_parliament",
		CheckedAt:   now.Add(-30 * time.Minute),
		HTTPStatus:  200,
		LatencyMS:   412,
		TLSValid:    true,
		ContentHash: "sha256:9f2c3e1a4b8d7c6e5a4f3b2c1d0e9f8a7b6c5d4e3f2a1b0c9d8e7f6a5b4c3d2e",
		Changed:     false,
	})
	s.AddSourceCheck(SourceCheck{
		ID:          newSourceCheckID(),
		SourceID:    "src_ke_kenyalaw",
		CheckedAt:   now.Add(-1 * time.Hour),
		HTTPStatus:  200,
		LatencyMS:   287,
		TLSValid:    true,
		ContentHash: "sha256:1a2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d3e4f5a6b7c8d9e0f1a2b",
		Changed:     true, // a new bill was published
	})

	// === Claims ===
	// Claim 1: The Housing Bill, 2024 is in Committee Stage (sourced from
	// parliament.go.ke).
	validFrom := now.Add(-48 * time.Hour)
	claim1 := Claim{
		ID:                "clm_housing_bill_stage",
		Subject:           "bill:ke-bill-housing-2024",
		Predicate:         "stage",
		Object:            "committee_stage",
		ClaimType:         ClaimTypeFact,
		Text:              "The Housing Bill, 2024 is currently at the Committee Stage in the Departmental Committee on Lands.",
		Confidence:        0.95,
		VerificationState: VerificationVerified,
		ValidFrom:         &validFrom,
	}
	s.AddClaim(claim1)

	// Claim 2: A second source says the bill is still at First Reading
	// (the contradiction the explorer surfaces).
	validFrom2 := now.Add(-24 * time.Hour)
	claim2 := Claim{
		ID:                "clm_housing_bill_stage_alt",
		Subject:           "bill:ke-bill-housing-2024",
		Predicate:         "stage",
		Object:            "first_reading",
		ClaimType:         ClaimTypeFact,
		Text:              "The Housing Bill, 2024 is at First Reading.",
		Confidence:        0.62,
		VerificationState: VerificationConflicted,
		ValidFrom:         &validFrom2,
	}
	s.AddClaim(claim2)

	// === Evidence ===
	s.AddEvidence(Evidence{
		ID:        newEvidenceID(),
		ClaimID:   claim1.ID,
		DocumentID: "doc_housing_bill_na_hansard_2024_03_15",
		SnapshotID: "snap_001",
		PageNumber: 4,
		Section:    "Order Paper",
		Paragraph:  "12",
		TextSpan:   "The Housing Bill, 2024 stands committed to the Departmental Committee on Lands.",
		SourceURL:  "https://www.parliament.go.ke/",
		ContentHash: "sha256:9f2c3e1a4b8d7c6e5a4f3b2c1d0e9f8a7b6c5d4e3f2a1b0c9d8e7f6a5b4c3d2e",
	})
	s.AddEvidence(Evidence{
		ID:        newEvidenceID(),
		ClaimID:   claim1.ID,
		DocumentID: "doc_housing_bill_kenyalaw_text",
		SnapshotID: "snap_002",
		PageNumber: 1,
		Section:    "Long Title",
		Paragraph:  "1",
		TextSpan:   "A Bill for An Act of Parliament to provide for the regulation of housing...",
		SourceURL:  "https://www.kenyalaw.org/",
		ContentHash: "sha256:1a2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d3e4f5a6b7c8d9e0f1a2b",
	})
	s.AddEvidence(Evidence{
		ID:        newEvidenceID(),
		ClaimID:   claim2.ID,
		DocumentID: "doc_housing_bill_senate_notice",
		SnapshotID: "snap_003",
		PageNumber: 2,
		Section:    "Senate Notices",
		Paragraph:  "3",
		TextSpan:   "The Housing Bill, 2024 was published on 7 March 2024.",
		SourceURL:  "https://www.senate.go.ke/",
		ContentHash: "sha256:7e8f9a0b1c2d3e4f5a6b7c8d9e0f1a2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f",
	})

	// === Contradiction ===
	s.AddContradiction(Contradiction{
		ID:         newContradictionID(),
		ClaimAID:   claim1.ID,
		ClaimBID:   claim2.ID,
		SourceAID:  "src_ke_parliament",
		SourceBID:  "src_ke_senate",
		DetectedAt: now.Add(-12 * time.Hour),
		Status:     ContradictionDetected,
	})
}
