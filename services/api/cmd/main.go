// Package main is the entrypoint for the Civic Intelligence API / BFF.
//
// This service is the citizen-facing HTTP gateway. It:
//   - validates OIDC JWTs (via Keycloak JWKS in prod, DevVerifier in dev)
//   - enforces RBAC per endpoint (scope-based authorization)
//   - rate-limits per IP (60/min anonymous, 300/min authenticated)
//   - calls the domain services (legislation, search, intelligence, etc.)
//   - NEVER touches the database directly
package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/Roy-Wanyoike/civic-intelligence/packages/auth"
	"github.com/Roy-Wanyoike/civic-intelligence/packages/config"
	"github.com/Roy-Wanyoike/civic-intelligence/services/api/internal/middleware"
	"github.com/Roy-Wanyoike/civic-intelligence/services/api/internal/oidc"
)

// Config holds the API service's runtime configuration.
type Config struct {
	HTTPAddr        string        `env:"API_SERVICE_ADDR" default:":9000"`
	OIDCIssuer      string        `env:"OIDC_ISSUER" default:"http://localhost:8081/realms/civic"`
	OIDCAudience    string        `env:"OIDC_AUDIENCE" default:"civic-intelligence"`
	OIDCJWKSURL     string        `env:"OIDC_JWKS_URL" default:"http://localhost:8081/realms/civic/protocol/openid-connect/certs"`
	DevMode         bool          `env:"DEV_MODE" default:"true"`
	ShutdownTimeout time.Duration `env:"SHUTDOWN_TIMEOUT" default:"15s"`
}

func main() {
	var cfg Config
	if err := config.Load(&cfg); err != nil {
		log.Fatalf("config: %v", err)
	}

	// Build the OIDC verifier.
	var verifier auth.OIDCTokenVerifier
	if cfg.DevMode {
		log.Println("WARNING: running in DEV_MODE — JWT signatures are NOT verified")
		verifier = oidc.DevVerifier{}
	} else {
		verifier = oidc.NewKeycloakVerifier(cfg.OIDCIssuer, cfg.OIDCAudience, cfg.OIDCJWKSURL)
	}

	// Build the router with middleware chain.
	mux := http.NewServeMux()

	// === Public endpoints (no auth required) ===
	mux.HandleFunc("/api/v1/healthz", healthz)
	mux.HandleFunc("/api/v1/readyz", readyz)

	// Apply OptionalAuth to the main API routes — extracts principal if
	// present, but doesn't require it. Individual route handlers can then
	// check the principal to customize behavior.
	apiHandler := http.NewServeMux()

	// Bills — read is public (anonymous can browse); follow requires auth.
	apiHandler.HandleFunc("/api/v1/bills", handleBillsList)
	apiHandler.HandleFunc("/api/v1/bills/", handleBillDetail) // handles /bills/{id} and sub-routes

	// Search — public.
	apiHandler.HandleFunc("/api/v1/search", handleSearch)

	// Briefing — public.
	apiHandler.HandleFunc("/api/v1/briefing", handleBriefing)

	// People, committees, institutions — public read.
	apiHandler.HandleFunc("/api/v1/people/", handlePeople)
	apiHandler.HandleFunc("/api/v1/committees/", handleCommittees)
	apiHandler.HandleFunc("/api/v1/institutions/", handleInstitutions)

	// Questions (AI Q&A) — requires auth + scope.
	questionsHandler := middleware.RequireToken(verifier)(
		middleware.RequireScope(auth.ScopeAIAsk)(http.HandlerFunc(handleQuestions)),
	)
	apiHandler.Handle("/api/v1/questions", questionsHandler)
	apiHandler.Handle("/api/v1/questions/stream", questionsHandler)

	// Follow — requires auth.
	followHandler := middleware.RequireToken(verifier)(
		middleware.RequireScope(auth.ScopeNotificationWrite)(http.HandlerFunc(handleFollow)),
	)
	apiHandler.Handle("/api/v1/bills/", followHandler) // mounted under /bills/{id}/follow

	// Apply OptionalAuth + rate limiting to the API routes.
	rateLimited := middleware.RateLimit(300, time.Minute)(apiHandler)
	mux.Handle("/api/v1/", middleware.OptionalAuth(verifier)(rateLimited))

	// Build the server.
	srv := &http.Server{
		Addr:         cfg.HTTPAddr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		log.Printf("civic-api listening on %s (dev_mode=%v)", cfg.HTTPAddr, cfg.DevMode)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	log.Println("civic-api shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("shutdown error: %v", err)
	}
}

// --- Health ---

func healthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "ok",
		"service": "api",
		"version": "0.1.0",
	})
}

func readyz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status":     "ready",
		"service":    "api",
		"downstream": "unchecked",
	})
}

// --- Bills ---

func handleBillsList(w http.ResponseWriter, r *http.Request) {
	// TODO(issue #19): call legislation service to list Bills.
	// For now, return a placeholder indicating the endpoint is wired but
	// the backend service is not yet connected.
	p := middleware.PrincipalFromRequest(r)
	writeJSON(w, http.StatusOK, map[string]any{
		"items":     []any{},
		"total":     0,
		"page":      1,
		"page_size": 20,
		"note":      "Bill listing — backend service connection pending (issue #19)",
		"principal": p.UserID,
	})
}

func handleBillDetail(w http.ResponseWriter, r *http.Request) {
	// Route: /api/v1/bills/{id} or /api/v1/bills/{id}/{sub}
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/bills/")
	parts := strings.SplitN(path, "/", 2)
	billID := parts[0]
	if billID == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "bill ID required")
		return
	}
	if len(parts) == 1 || parts[1] == "" {
		// GET /api/v1/bills/{id}
		// TODO(issue #19): call legislation service to get Bill.
		writeJSON(w, http.StatusOK, map[string]any{
			"id":   billID,
			"note": "Bill detail — backend service connection pending (issue #19)",
		})
		return
	}
	// Sub-routes: timeline, versions, compare, documents, chat, follow
	sub := parts[1]
	switch {
	case strings.HasPrefix(sub, "timeline"):
		writeJSON(w, http.StatusOK, map[string]any{"bill_id": billID, "events": []any{}})
	case strings.HasPrefix(sub, "versions"):
		writeJSON(w, http.StatusOK, map[string]any{"bill_id": billID, "versions": []any{}})
	case strings.HasPrefix(sub, "follow"):
		// Follow is handled by the auth-protected handler below.
		handleFollow(w, r)
	default:
		writeError(w, http.StatusNotFound, "not_found", "unknown sub-route: "+sub)
	}
}

// --- Search ---

func handleSearch(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	if q == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "query parameter 'q' is required")
		return
	}
	// TODO(issue #19): call search service.
	writeJSON(w, http.StatusOK, map[string]any{
		"q":      q,
		"items":  []any{},
		"total":  0,
		"note":   "Search — backend service connection pending (issue #19)",
	})
}

// --- Briefing ---

func handleBriefing(w http.ResponseWriter, r *http.Request) {
	// TODO(issue #19): call AI service / briefing endpoint.
	writeJSON(w, http.StatusOK, map[string]any{
		"country": "KE",
		"date":    time.Now().UTC().Format(time.RFC3339),
		"items":   []any{},
		"note":    "Briefing — backend service connection pending (issue #19)",
	})
}

// --- People / Committees / Institutions ---

func handlePeople(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/people/")
	writeJSON(w, http.StatusOK, map[string]any{"id": id, "note": "People — pending (issue #19)"})
}

func handleCommittees(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/committees/")
	writeJSON(w, http.StatusOK, map[string]any{"id": id, "note": "Committees — pending (issue #19)"})
}

func handleInstitutions(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/institutions/")
	writeJSON(w, http.StatusOK, map[string]any{"id": id, "note": "Institutions — pending (issue #19)"})
}

// --- Questions (AI Q&A) — requires auth + scope ---

func handleQuestions(w http.ResponseWriter, r *http.Request) {
	p := middleware.PrincipalFromRequest(r)
	// TODO(issue #19): proxy to AI service with the principal's context.
	writeJSON(w, http.StatusOK, map[string]any{
		"answer":    "Q&A proxy — backend connection pending (issue #19)",
		"principal": p.UserID,
		"validated": false,
	})
}

// --- Follow — requires auth ---

func handleFollow(w http.ResponseWriter, r *http.Request) {
	p := middleware.PrincipalFromRequest(r)
	writeJSON(w, http.StatusOK, map[string]any{
		"followed":  true,
		"principal": p.UserID,
		"note":      "Follow — backend connection pending (issue #19)",
	})
}

// --- Helpers ---

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": code, "message": message})
}
