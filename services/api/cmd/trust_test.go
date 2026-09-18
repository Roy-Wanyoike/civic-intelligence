package main

import (
        "bytes"
        "encoding/json"
        "net/http"
        "net/http/httptest"
        "strings"
        "testing"
        "time"

        "github.com/Roy-Wanyoike/civic-intelligence/packages/auth"
        "github.com/Roy-Wanyoike/civic-intelligence/services/api/internal/middleware"
)

// trustDispatch runs a request through an OptionalAuth chain populated with the
// given principal, mirroring how the production router wires trust endpoints.
func trustDispatch(handler http.Handler, method, url string, body []byte, p auth.Principal) *httptest.ResponseRecorder {
        var br *bytes.Reader
        if body != nil {
                br = bytes.NewReader(body)
        } else {
                br = bytes.NewReader(nil)
        }
        req := httptest.NewRequest(method, url, br)
        if p.IsAuthenticated() {
                req.Header.Set("Authorization", "Bearer test-token")
        }
        v := auth.StaticVerifier{Principal: p}
        wrapped := middleware.OptionalAuth(v)(handler)
        rr := httptest.NewRecorder()
        wrapped.ServeHTTP(rr, req)
        return rr
}

// seededTrustStore returns a fresh store populated with the sample dataset.
func seededTrustStore(t *testing.T) *TrustStore {
        t.Helper()
        s := NewTrustStore()
        s.SeedSampleTrustData()
        return s
}

// TestProvenance_ByEntityReturnsClaimsWithEvidence verifies the provenance
// explorer returns every claim attached to an entity, each with its evidence
// list, sorted by claim.created_at descending.
func TestProvenance_ByEntityReturnsClaimsWithEvidence(t *testing.T) {
        store := seededTrustStore(t)
        handler := makeProvenanceHandler(store)

        rr := trustDispatch(handler, http.MethodGet, "/api/v1/provenance/bill/ke-bill-housing-2024", nil, auth.Anonymous())
        if rr.Code != http.StatusOK {
                t.Fatalf("expected 200, got %d (body=%s)", rr.Code, rr.Body.String())
        }

        var resp struct {
                EntityType string              `json:"entity_type"`
                EntityID   string              `json:"entity_id"`
                Claims     []claimWithEvidence `json:"claims"`
                Total      int                 `json:"total"`
        }
        if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
                t.Fatalf("invalid JSON: %v", err)
        }
        if resp.Total < 2 {
                t.Fatalf("expected >=2 claims for the sample bill, got %d", resp.Total)
        }
        // Both claims must carry evidence.
        for _, c := range resp.Claims {
                if len(c.Evidence) == 0 {
                        t.Errorf("claim %s has no evidence", c.Claim.ID)
                }
        }
}

// TestProvenance_ForClaimEntityReturnsSingleClaim verifies the special-case
// where entity_type=claim returns the claim + its evidence.
func TestProvenance_ForClaimEntityReturnsSingleClaim(t *testing.T) {
        store := seededTrustStore(t)
        handler := makeProvenanceHandler(store)

        // Find the sample claim id.
        claims := store.ClaimsForEntity("bill", "ke-bill-housing-2024")
        if len(claims) == 0 {
                t.Fatalf("seed data missing claims")
        }
        claimID := claims[0].ID

        rr := trustDispatch(handler, http.MethodGet, "/api/v1/provenance/claim/"+claimID, nil, auth.Anonymous())
        if rr.Code != http.StatusOK {
                t.Fatalf("expected 200, got %d", rr.Code)
        }
        var resp struct {
                EntityType string              `json:"entity_type"`
                Claims     []claimWithEvidence `json:"claims"`
                Total      int                 `json:"total"`
        }
        _ = json.Unmarshal(rr.Body.Bytes(), &resp)
        if resp.Total != 1 {
                t.Errorf("expected total=1 for /provenance/claim/{id}, got %d", resp.Total)
        }
}

// TestProvenance_InvalidEntityType verifies the validator rejects unknown
// entity types (e.g. "spaceship") with a 400.
func TestProvenance_InvalidEntityType(t *testing.T) {
        store := seededTrustStore(t)
        handler := makeProvenanceHandler(store)

        rr := trustDispatch(handler, http.MethodGet, "/api/v1/provenance/spaceship/123", nil, auth.Anonymous())
        if rr.Code != http.StatusBadRequest {
                t.Errorf("expected 400 for unknown entity_type, got %d", rr.Code)
        }
}

// TestProvenance_NotFound verifies a query for an entity with no claims
// returns 200 with an empty list (not 404 — "no evidence" is a valid result).
func TestProvenance_NotFoundReturnsEmpty(t *testing.T) {
        store := seededTrustStore(t)
        handler := makeProvenanceHandler(store)

        rr := trustDispatch(handler, http.MethodGet, "/api/v1/provenance/bill/no-such-bill", nil, auth.Anonymous())
        if rr.Code != http.StatusOK {
                t.Fatalf("expected 200 (empty list), got %d", rr.Code)
        }
        var resp provenanceResponse
        _ = json.Unmarshal(rr.Body.Bytes(), &resp)
        if resp.Total != 0 || len(resp.Claims) != 0 {
                t.Errorf("expected empty list, got total=%d", resp.Total)
        }
}

// TestProvenance_MethodNotAllowed verifies the Allow header is set on 405.
func TestProvenance_MethodNotAllowed(t *testing.T) {
        store := seededTrustStore(t)
        handler := makeProvenanceHandler(store)

        rr := trustDispatch(handler, http.MethodPost, "/api/v1/provenance/bill/x", []byte("{}"), auth.Anonymous())
        if rr.Code != http.StatusMethodNotAllowed {
                t.Errorf("expected 405, got %d", rr.Code)
        }
        if rr.Header().Get("Allow") != "GET" {
                t.Errorf("expected Allow=GET, got %q", rr.Header().Get("Allow"))
        }
}

// TestEvidenceDetail_Found verifies GET /api/v1/evidence/{id} returns the
// evidence row with the source hydrated.
func TestEvidenceDetail_Found(t *testing.T) {
        store := seededTrustStore(t)
        // Pull an evidence ID out of the seed via the claims route.
        claims := store.ClaimsForEntity("bill", "ke-bill-housing-2024")
        if len(claims) == 0 {
                t.Fatalf("seed missing claims")
        }
        evidence := store.EvidenceForClaim(claims[0].ID)
        if len(evidence) == 0 {
                t.Fatalf("seed missing evidence for claim %s", claims[0].ID)
        }
        evID := evidence[0].ID

        handler := makeEvidenceHandler(store)
        rr := trustDispatch(handler, http.MethodGet, "/api/v1/evidence/"+evID, nil, auth.Anonymous())
        if rr.Code != http.StatusOK {
                t.Fatalf("expected 200, got %d (body=%s)", rr.Code, rr.Body.String())
        }
        var ev Evidence
        _ = json.Unmarshal(rr.Body.Bytes(), &ev)
        if ev.ID != evID {
                t.Errorf("expected evidence %s, got %s", evID, ev.ID)
        }
        if ev.ClaimID == "" {
                t.Error("evidence response missing claim_id")
        }
        if ev.SourceURL == "" {
                t.Error("evidence response missing source_url")
        }
        // The source SHOULD be hydrated because the seed includes a trust source
        // whose OfficialURL matches the evidence's SourceURL.
        if ev.Source == nil {
                t.Error("expected source to be hydrated on evidence detail")
        }
}

// TestEvidenceDetail_NotFound verifies GET /api/v1/evidence/{id} 404s on
// an unknown ID.
func TestEvidenceDetail_NotFound(t *testing.T) {
        store := seededTrustStore(t)
        handler := makeEvidenceHandler(store)
        rr := trustDispatch(handler, http.MethodGet, "/api/v1/evidence/evd_does_not_exist", nil, auth.Anonymous())
        if rr.Code != http.StatusNotFound {
                t.Errorf("expected 404, got %d", rr.Code)
        }
}

// TestClaimEvidence_Success verifies GET /api/v1/claims/{id}/evidence returns
// the claim plus its evidence list.
func TestClaimEvidence_Success(t *testing.T) {
        store := seededTrustStore(t)
        claims := store.ClaimsForEntity("bill", "ke-bill-housing-2024")
        claimID := claims[0].ID

        handler := makeClaimEvidenceHandler(store)
        rr := trustDispatch(handler, http.MethodGet, "/api/v1/claims/"+claimID+"/evidence", nil, auth.Anonymous())
        if rr.Code != http.StatusOK {
                t.Fatalf("expected 200, got %d (body=%s)", rr.Code, rr.Body.String())
        }
        var resp struct {
                Claim    Claim     `json:"claim"`
                Evidence []Evidence `json:"evidence"`
                Total    int       `json:"total"`
        }
        _ = json.Unmarshal(rr.Body.Bytes(), &resp)
        if resp.Claim.ID != claimID {
                t.Errorf("expected claim %s, got %s", claimID, resp.Claim.ID)
        }
        if resp.Total < 1 {
                t.Errorf("expected >=1 evidence row, got %d", resp.Total)
        }
}

// TestClaimEvidence_UnknownSubRoute verifies that /api/v1/claims/{id}/versions
// (not supported by this handler) returns 404 rather than 500.
func TestClaimEvidence_UnknownSubRoute(t *testing.T) {
        store := seededTrustStore(t)
        claims := store.ClaimsForEntity("bill", "ke-bill-housing-2024")
        claimID := claims[0].ID

        handler := makeClaimEvidenceHandler(store)
        rr := trustDispatch(handler, http.MethodGet, "/api/v1/claims/"+claimID+"/versions", nil, auth.Anonymous())
        if rr.Code != http.StatusNotFound {
                t.Errorf("expected 404 for unknown sub-route, got %d", rr.Code)
        }
}

// TestContradictions_List verifies GET /api/v1/contradictions returns the
// seeded active conflict with both claims + sources hydrated.
func TestContradictions_List(t *testing.T) {
        store := seededTrustStore(t)
        handler := makeContradictionsHandler(store)

        rr := trustDispatch(handler, http.MethodGet, "/api/v1/contradictions", nil, auth.Anonymous())
        if rr.Code != http.StatusOK {
                t.Fatalf("expected 200, got %d (body=%s)", rr.Code, rr.Body.String())
        }
        var resp struct {
                Items []Contradiction `json:"items"`
                Total int             `json:"total"`
        }
        _ = json.Unmarshal(rr.Body.Bytes(), &resp)
        if resp.Total < 1 {
                t.Fatalf("expected >=1 contradiction, got %d", resp.Total)
        }
        c := resp.Items[0]
        if c.ClaimA == nil || c.ClaimB == nil {
                t.Error("expected contradiction to hydrate both claims")
        }
        if c.SourceA == nil || c.SourceB == nil {
                t.Error("expected contradiction to hydrate both sources")
        }
        if c.ClaimAID == c.ClaimBID {
                t.Error("contradiction claim_a and claim_b must differ")
        }
}

// TestContradictions_FilterByStatus verifies the ?status=detected filter.
func TestContradictions_FilterByStatus(t *testing.T) {
        store := seededTrustStore(t)
        handler := makeContradictionsHandler(store)

        rr := trustDispatch(handler, http.MethodGet, "/api/v1/contradictions?status=detected", nil, auth.Anonymous())
        if rr.Code != http.StatusOK {
                t.Fatalf("expected 200, got %d", rr.Code)
        }
        var resp struct {
                Items []Contradiction `json:"items"`
        }
        _ = json.Unmarshal(rr.Body.Bytes(), &resp)
        for _, c := range resp.Items {
                if c.Status != ContradictionDetected {
                        t.Errorf("expected all items 'detected', got %q", c.Status)
                }
        }
}

// TestContradictions_InvalidStatus verifies the status filter rejects unknown
// values.
func TestContradictions_InvalidStatus(t *testing.T) {
        store := seededTrustStore(t)
        handler := makeContradictionsHandler(store)
        rr := trustDispatch(handler, http.MethodGet, "/api/v1/contradictions?status=banana", nil, auth.Anonymous())
        if rr.Code != http.StatusBadRequest {
                t.Errorf("expected 400 for invalid status, got %d", rr.Code)
        }
}

// TestTrustSourceDetail_Found verifies GET /api/v1/sources/{id} returns the
// source with health + last check denormalised.
func TestTrustSourceDetail_Found(t *testing.T) {
        store := seededTrustStore(t)
        handler := makeTrustSourceHandler(store)

        rr := trustDispatch(handler, http.MethodGet, "/api/v1/sources/src_ke_parliament", nil, auth.Anonymous())
        if rr.Code != http.StatusOK {
                t.Fatalf("expected 200, got %d (body=%s)", rr.Code, rr.Body.String())
        }
        var src TrustSource
        _ = json.Unmarshal(rr.Body.Bytes(), &src)
        if src.ID != "src_ke_parliament" {
                t.Errorf("expected src_ke_parliament, got %s", src.ID)
        }
        if src.AuthorityLevel != AuthorityPrimaryOfficial {
                t.Errorf("expected authority PRIMARY_OFFICIAL, got %s", src.AuthorityLevel)
        }
        if src.LastCheck == nil {
                t.Error("expected last_check to be denormalised on source detail")
        }
}

// TestTrustSourceDetail_NotFound verifies GET /api/v1/sources/{id} 404s.
func TestTrustSourceDetail_NotFound(t *testing.T) {
        store := seededTrustStore(t)
        handler := makeTrustSourceHandler(store)
        rr := trustDispatch(handler, http.MethodGet, "/api/v1/sources/src_does_not_exist", nil, auth.Anonymous())
        if rr.Code != http.StatusNotFound {
                t.Errorf("expected 404, got %d", rr.Code)
        }
}

// TestTrustSourcesList_Default verifies GET /api/v1/sources lists all sources
// ordered by authority rank (PRIMARY_OFFICIAL first).
func TestTrustSourcesList_Default(t *testing.T) {
        store := seededTrustStore(t)
        handler := makeTrustSourcesListHandler(store)
        rr := trustDispatch(handler, http.MethodGet, "/api/v1/sources", nil, auth.Anonymous())
        if rr.Code != http.StatusOK {
                t.Fatalf("expected 200, got %d", rr.Code)
        }
        var resp struct {
                Items []TrustSource `json:"items"`
                Total int           `json:"total"`
        }
        _ = json.Unmarshal(rr.Body.Bytes(), &resp)
        if resp.Total < 3 {
                t.Fatalf("expected >=3 seeded sources, got %d", resp.Total)
        }
        // First item should be a PRIMARY_OFFICIAL (authority rank 0).
        first := resp.Items[0]
        if first.AuthorityLevel != AuthorityPrimaryOfficial {
                t.Errorf("expected first item to be PRIMARY_OFFICIAL, got %s", first.AuthorityLevel)
        }
}

// TestTrustSourcesList_FilterByAuthority verifies the ?authority= filter.
func TestTrustSourcesList_FilterByAuthority(t *testing.T) {
        store := seededTrustStore(t)
        handler := makeTrustSourcesListHandler(store)
        rr := trustDispatch(handler, http.MethodGet, "/api/v1/sources?authority=PRIMARY_OFFICIAL", nil, auth.Anonymous())
        if rr.Code != http.StatusOK {
                t.Fatalf("expected 200, got %d", rr.Code)
        }
        var resp struct {
                Items []TrustSource `json:"items"`
        }
        _ = json.Unmarshal(rr.Body.Bytes(), &resp)
        if len(resp.Items) == 0 {
                t.Fatal("expected at least 1 PRIMARY_OFFICIAL source")
        }
        for _, s := range resp.Items {
                if s.AuthorityLevel != AuthorityPrimaryOfficial {
                        t.Errorf("filter leaked: found %s", s.AuthorityLevel)
                }
        }
}

// TestTrustSourcesList_InvalidAuthority verifies a bad ?authority= value 400s.
func TestTrustSourcesList_InvalidAuthority(t *testing.T) {
        store := seededTrustStore(t)
        handler := makeTrustSourcesListHandler(store)
        rr := trustDispatch(handler, http.MethodGet, "/api/v1/sources?authority=GODMODE", nil, auth.Anonymous())
        if rr.Code != http.StatusBadRequest {
                t.Errorf("expected 400 for invalid authority, got %d", rr.Code)
        }
}

// TestTrustStore_SeedIsIdempotent verifies re-seeding does not duplicate.
func TestTrustStore_SeedIsIdempotent(t *testing.T) {
        s := NewTrustStore()
        s.SeedSampleTrustData()
        firstCount := len(s.ListSources("", ""))
        s.SeedSampleTrustData()
        secondCount := len(s.ListSources("", ""))
        if firstCount != secondCount {
                t.Errorf("seed not idempotent: first=%d, second=%d", firstCount, secondCount)
        }
}

// TestTrustStore_AddSourceCheckUpdatesHealth verifies a source check correctly
// denormalises the latest health_status back onto the source row.
func TestTrustStore_AddSourceCheckUpdatesHealth(t *testing.T) {
        s := NewTrustStore()
        s.AddSource(TrustSource{
                ID:             "src_x",
                InstitutionID:  "ke-x",
                Country:        "KE",
                SourceType:     "parliament",
                AuthorityLevel: AuthorityPrimaryOfficial,
                OfficialURL:    "https://x.go.ke/",
                Domain:         "x.go.ke",
                Status:         "active",
                HealthStatus:   "unknown",
        })
        healthy := time.Now().UTC()
        s.AddSourceCheck(SourceCheck{
                ID:         newSourceCheckID(),
                SourceID:   "src_x",
                CheckedAt:  healthy,
                HTTPStatus: 200,
                LatencyMS:  100,
                TLSValid:   true,
        })
        src, _ := s.GetSource("src_x")
        if src.HealthStatus != "healthy" {
                t.Errorf("expected healthy after 200+TLS check, got %s", src.HealthStatus)
        }
        if src.LastCheck == nil {
                t.Error("expected LastCheck to be denormalised")
        }

        degraded := healthy.Add(1 * time.Hour)
        s.AddSourceCheck(SourceCheck{
                ID:         newSourceCheckID(),
                SourceID:   "src_x",
                CheckedAt:  degraded,
                HTTPStatus: 500,
                LatencyMS:  100,
                TLSValid:   true,
        })
        src, _ = s.GetSource("src_x")
        if src.HealthStatus != "degraded" {
                t.Errorf("expected degraded after 500, got %s", src.HealthStatus)
        }
}

// TestProvenance_MissingPathSegments verifies a malformed path returns 400.
func TestProvenance_MissingPathSegments(t *testing.T) {
        store := seededTrustStore(t)
        handler := makeProvenanceHandler(store)
        rr := trustDispatch(handler, http.MethodGet, "/api/v1/provenance/", nil, auth.Anonymous())
        if rr.Code != http.StatusBadRequest {
                t.Errorf("expected 400 for empty path, got %d", rr.Code)
        }
        if !strings.Contains(rr.Body.String(), "entity_type") && !strings.Contains(rr.Body.String(), "path") {
                t.Errorf("expected error to mention path/entity_type, got %s", rr.Body.String())
        }
}
