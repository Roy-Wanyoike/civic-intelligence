// Package main is the entrypoint for the Civic Intelligence API / BFF.
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

	"github.com/Roy-Wanyoike/civic-intelligence/adapters/kenya/kenya_law"
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

	// Build the Kenya Law adapter for real Bill discovery.
	kenyaLaw := kenya_law.NewAdapter(nil, "CivicIntelligence/0.1 (+https://github.com/Roy-Wanyoike/civic-intelligence)")

	// Build the router with middleware chain.
	mux := http.NewServeMux()

	// === Public endpoints (no auth required) ===
	mux.HandleFunc("/api/v1/healthz", healthz)
	mux.HandleFunc("/api/v1/readyz", readyz)

	// Apply OptionalAuth to the main API routes.
	apiHandler := http.NewServeMux()

	// Bills — read is public; uses the Kenya Law adapter for real data.
	apiHandler.HandleFunc("/api/v1/bills", makeBillsHandler(kenyaLaw))
	apiHandler.HandleFunc("/api/v1/bills/", makeBillDetailHandler(kenyaLaw))

	// Search — public.
	apiHandler.HandleFunc("/api/v1/search", handleSearch)

	// Briefing — public.
	apiHandler.HandleFunc("/api/v1/briefing", handleBriefing)

	// People, committees, institutions — public read.
	apiHandler.HandleFunc("/api/v1/people/", handlePeople)
	apiHandler.HandleFunc("/api/v1/committees/", handleCommittees)
	apiHandler.HandleFunc("/api/v1/institutions/", handleInstitutions)

	// Loans + grants — public read.
	apiHandler.HandleFunc("/api/v1/loans", handleLoansList)
	apiHandler.HandleFunc("/api/v1/loans/", handleLoanDetail)
	apiHandler.HandleFunc("/api/v1/grants", handleGrantsList)
	apiHandler.HandleFunc("/api/v1/grants/", handleGrantDetail)

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
	apiHandler.Handle("/api/v1/follow", followHandler)

	// Apply OptionalAuth + rate limiting.
	rateLimited := middleware.RateLimit(300, time.Minute)(apiHandler)
	mux.Handle("/api/v1/", middleware.OptionalAuth(verifier)(rateLimited))

	// Build the server.
	srv := &http.Server{
		Addr:         cfg.HTTPAddr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 60 * time.Second, // longer for adapter calls
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
		"version": "0.2.0",
	})
}

func readyz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "ready",
		"service": "api",
	})
}

// --- Bills (real data from Kenya Law adapter) ---

// billResponse is the JSON shape returned by the bills endpoint.
type billResponse struct {
	ID          string   `json:"id"`
	Identifier  string   `json:"identifier"`
	Title       string   `json:"title"`
	House       string   `json:"house"`
	Year        int      `json:"year"`
	Status      string   `json:"status"`
	CurrentStage string  `json:"current_stage"`
	Purpose     string   `json:"purpose,omitempty"`
	Description string   `json:"description,omitempty"`
	Country     string   `json:"country"`
	SourceURL   string   `json:"source_url"`
	PublicationDate string `json:"publication_date"`
	Topics      []string `json:"topics"`
}

func makeBillsHandler(adapter *kenya_law.Adapter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()

		bills, err := adapter.DiscoverBills(ctx)
		if err != nil {
			log.Printf("bills handler: adapter error: %v", err)
			writeError(w, http.StatusServiceUnavailable, "adapter_error", "failed to discover bills from Kenya Law")
			return
		}

		// Convert BillCandidate → billResponse
		items := make([]billResponse, 0, len(bills))
		for _, b := range bills {
			year := 0
			if !b.PublicationDate.IsZero() {
				year = b.PublicationDate.Year()
			}
			items = append(items, billResponse{
				ID:              b.SourceID,
				Identifier:      b.Slug,
				Title:           b.Title,
				House:           b.House,
				Year:            year,
				Status:          "in_progress",
				CurrentStage:    "published",
				Country:         "KE",
				SourceURL:       b.URL,
				PublicationDate: b.PublicationDate.Format("2006-01-02"),
				Topics:          []string{},
			})
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"items":     items,
			"total":     len(items),
			"page":      1,
			"page_size": len(items),
			"source":    "new.kenyalaw.org",
		})
	}
}

func makeBillDetailHandler(adapter *kenya_law.Adapter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()

		// Extract bill ID from path: /api/v1/bills/{id}
		path := strings.TrimPrefix(r.URL.Path, "/api/v1/bills/")
		parts := strings.SplitN(path, "/", 2)
		billID := parts[0]
		if billID == "" {
			writeError(w, http.StatusBadRequest, "bad_request", "bill ID required")
			return
		}

		// Sub-routes: timeline, versions, etc.
		if len(parts) == 2 && parts[1] != "" {
			sub := parts[1]
			switch {
			case strings.HasPrefix(sub, "timeline"):
				writeJSON(w, http.StatusOK, map[string]any{"bill_id": billID, "events": []any{}})
			case strings.HasPrefix(sub, "versions"):
				writeJSON(w, http.StatusOK, map[string]any{"bill_id": billID, "versions": []any{}})
			case strings.HasPrefix(sub, "documents"):
				writeJSON(w, http.StatusOK, map[string]any{"bill_id": billID, "documents": []any{}})
			case strings.HasPrefix(sub, "follow"):
				handleFollow(w, r)
			default:
				writeError(w, http.StatusNotFound, "not_found", "unknown sub-route: "+sub)
			}
			return
		}

		// GET /api/v1/bills/{id} — fetch + parse the bill detail
		// The billID is the source ID from DiscoverBills. We need to find
		// the bill URL by re-discovering (in production, this would be a DB lookup).
		bills, err := adapter.DiscoverBills(ctx)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "adapter_error", "failed to discover bills")
			return
		}

		var found *kenya_law.BillCandidate
		for i := range bills {
			if bills[i].SourceID == billID || strings.Contains(bills[i].SourceID, billID) {
				found = &bills[i]
				break
			}
		}
		if found == nil {
			writeError(w, http.StatusNotFound, "not_found", "bill not found: "+billID)
			return
		}

		// Fetch + parse the bill detail page
		html, err := adapter.FetchBill(ctx, found.URL)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "fetch_error", "failed to fetch bill detail")
			return
		}

		records, err := kenya_law.ParseBillDetail(html, found.URL)
		if err != nil || len(records) == 0 {
			// Return what we know from discovery
			year := 0
			if !found.PublicationDate.IsZero() {
				year = found.PublicationDate.Year()
			}
			writeJSON(w, http.StatusOK, billResponse{
				ID:              found.SourceID,
				Identifier:      found.Slug,
				Title:           found.Title,
				House:           found.House,
				Year:            year,
				Status:          "in_progress",
				CurrentStage:    "published",
				Country:         "KE",
				SourceURL:       found.URL,
				PublicationDate: found.PublicationDate.Format("2006-01-02"),
				Topics:          []string{},
			})
			return
		}

		rec := records[0]
		year := 0
		if !rec.PublishedAt.IsZero() {
			year = rec.PublishedAt.Year()
		}
		writeJSON(w, http.StatusOK, billResponse{
			ID:              found.SourceID,
			Identifier:      found.Slug,
			Title:           rec.Title,
			House:           rec.House,
			Year:            year,
			Status:          "in_progress",
			CurrentStage:    "published",
			Country:         "KE",
			SourceURL:       found.URL,
			PublicationDate: rec.PublishedAt.Format("2006-01-02"),
			Topics:          []string{},
		})
	}
}

// --- Search ---

func handleSearch(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	if q == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "query parameter 'q' is required")
		return
	}
	// TODO: call search service. For now, search through discovered bills.
	writeJSON(w, http.StatusOK, map[string]any{
		"q":     q,
		"items": []any{},
		"total": 0,
		"note":  "Search — full-text search pending (issue #45)",
	})
}

// --- Briefing ---

func handleBriefing(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"country": "KE",
		"date":    time.Now().UTC().Format(time.RFC3339),
		"items":   []any{},
		"note":    "Briefing — pending issue #40",
	})
}

// --- People / Committees / Institutions ---

func handlePeople(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/people/")
	writeJSON(w, http.StatusOK, map[string]any{"id": id, "note": "People — pending (issue #19)"})
}

func handleCommittees(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/committees/")
	writeJSON(w, http.StatusOK, map[string]any{"id": id, "note": "Committees — pending (issue #28)"})
}

func handleInstitutions(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/institutions/")
	writeJSON(w, http.StatusOK, map[string]any{"id": id, "note": "Institutions — pending (issue #19)"})
}

// --- Loans + Grants ---

func handleLoansList(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"items": []any{},
		"total": 0,
		"note":  "Loans — pending database connection (issue #93)",
	})
}

func handleLoanDetail(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/loans/")
	writeJSON(w, http.StatusOK, map[string]any{"id": id, "note": "Loan detail — pending (issue #93)"})
}

func handleGrantsList(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"items": []any{},
		"total": 0,
		"note":  "Grants — pending database connection (issue #93)",
	})
}

func handleGrantDetail(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/grants/")
	writeJSON(w, http.StatusOK, map[string]any{"id": id, "note": "Grant detail — pending (issue #93)"})
}

// --- Questions (AI Q&A) — requires auth + scope ---

func handleQuestions(w http.ResponseWriter, r *http.Request) {
	p := middleware.PrincipalFromRequest(r)
	writeJSON(w, http.StatusOK, map[string]any{
		"answer":    "Q&A proxy — AI service connection pending (issue #19)",
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
		"note":      "Follow — notifications service pending (issue #39)",
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
