// Package main is the entrypoint for the Civic Intelligence API / BFF.
package main

import (
        "bytes"
        "context"
        "crypto/sha1"
        "encoding/json"
        "fmt"
        "io"
        "log"
        "net/http"
        "os"
        "os/signal"
        "strings"
        "syscall"
        "time"

        "github.com/Roy-Wanyoike/civic-intelligence/adapters/kenya/kenya_law"
        "github.com/Roy-Wanyoike/civic-intelligence/adapters/kenya/kenya_seed"
        "github.com/Roy-Wanyoike/civic-intelligence/packages/auth"
        "github.com/Roy-Wanyoike/civic-intelligence/packages/config"
        "github.com/Roy-Wanyoike/civic-intelligence/packages/observability"
        "github.com/Roy-Wanyoike/civic-intelligence/services/api/internal/middleware"
        "github.com/Roy-Wanyoike/civic-intelligence/services/api/internal/oidc"
        "github.com/Roy-Wanyoike/civic-intelligence/services/legislation"
)

// Config holds the API service's runtime configuration.
type Config struct {
        HTTPAddr        string        `env:"API_SERVICE_ADDR" default:":9000"`
        OIDCIssuer      string        `env:"OIDC_ISSUER" default:"http://localhost:8081/realms/civic"`
        OIDCAudience    string        `env:"OIDC_AUDIENCE" default:"civic-intelligence"`
        OIDCJWKSURL     string        `env:"OIDC_JWKS_URL" default:"http://localhost:8081/realms/civic/protocol/openid-connect/certs"`
        DevMode         bool          `env:"DEV_MODE" default:"true"`
        ShutdownTimeout time.Duration `env:"SHUTDOWN_TIMEOUT" default:"15s"`

        // AIServiceURL is the base URL of the Python AI service (FastAPI).
        // Used by the bill summarizer, Q&A proxy, and impact-analysis endpoints.
        // Defaults to the local dev address (port 8000).
        AIServiceURL string `env:"AI_SERVICE_URL" default:"http://localhost:8000"`
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

        // Build the metric registry for observability.
        metrics := observability.NewMetricRegistry()

        // Build the debt repository (issue #203). The repository is seeded
        // with Kenya's authoritative CBK + Treasury observations via
        // kenya_seed.SeedDebt. In production this Wire call returns a
        // Postgres-backed implementation; the in-memory implementation is
        // used by tests + local dev.
        debtRepo, debtErr := legislation.WireDebtRepository(kenya_seed.SeedDebt)
        if debtErr != nil {
                // The seeder returns ErrSnapshotImmutable (etc.) if re-run; the
                // API service treats that as non-fatal because the repository is
                // still populated for the snapshots that succeeded. Log + continue.
                log.Printf("debt repository seed warning (non-fatal): %v", debtErr)
        }

        // Build the router with middleware chain.
        mux := http.NewServeMux()

        // === Public endpoints (no auth required) ===
        mux.HandleFunc("/api/v1/healthz", healthz)
        mux.HandleFunc("/api/v1/readyz", readyz)

        // Prometheus metrics endpoint.
        mux.HandleFunc("/metrics", observability.MetricsHandler(metrics))

        // Apply OptionalAuth to the main API routes.
        apiHandler := http.NewServeMux()

        // Bills — read is public; uses the Kenya Law adapter for real data.
        // Sub-routes (/api/v1/bills/{id}/summary, /impact) proxy to the Python
        // AI service when available and fall back to a structured stub otherwise.
        apiHandler.HandleFunc("/api/v1/bills", makeBillsHandler(kenyaLaw))
        apiHandler.HandleFunc("/api/v1/bills/", makeBillDetailHandler(kenyaLaw, cfg.AIServiceURL))

        // Trending — recently enacted + approaching final stage.
        apiHandler.HandleFunc("/api/v1/trending", makeTrendingHandler(kenyaLaw))

        // Terminology — Kenya parliamentary terms.
        apiHandler.HandleFunc("/api/v1/terminology/", makeTerminologyHandler())

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

        // Acts of Parliament — public read (issue #116).
        // Single router handles /acts, /acts/{id}, and the post-assent
        // sub-resources /acts/{id}/{audit,events,follow,lineage} (issue #193).
        apiHandler.HandleFunc("/api/v1/acts", handleActsList)
        apiHandler.HandleFunc("/api/v1/acts/", makeActRouter())

        // Questions (AI Q&A) — requires auth + scope.
        questionsHandler := middleware.RequireToken(verifier)(
                middleware.RequireScope(auth.ScopeAIAsk)(http.HandlerFunc(handleQuestions)),
        )
        apiHandler.Handle("/api/v1/questions", questionsHandler)
        apiHandler.Handle("/api/v1/questions/stream", questionsHandler)

        // Follow (legacy single-shot endpoint, kept for backward compat).
        followHandler := middleware.RequireToken(verifier)(
                middleware.RequireScope(auth.ScopeNotificationWrite)(http.HandlerFunc(handleFollow)),
        )
        apiHandler.Handle("/api/v1/follow", followHandler)

        // Subscriptions — issue #110 (Following). Auth is enforced inside the
        // handlers via OptionalAuth (set on the outer mux) + PrincipalFromRequest
        // checks, mirroring how the bills/{id}/follow endpoint behaves. Each
        // handler returns 401 explicitly when the caller is anonymous.
        // subscriptionStore is the package-level in-memory store (defined next
        // to actRepo) so the /api/v1/acts/{id}/follow endpoint (issue #216) can
        // write real subscriptions through the same store used by
        // /api/v1/subscriptions.
        apiHandler.HandleFunc("/api/v1/subscriptions", makeSubscriptionsHandler(subscriptionStore))
        apiHandler.HandleFunc("/api/v1/subscriptions/", makeSubscriptionDetailHandler(subscriptionStore))

        // Notifications — requires auth (wired via middleware in the handler).
        
        

        // Civic Feed — public.
        apiHandler.HandleFunc("/api/v1/feed", makeCivicFeedHandler(kenyaLaw))

        // Sponsor — M-Pesa + Card payment endpoints.
        apiHandler.HandleFunc("/api/v1/sponsor/mpesa", handleMpesaSponsor)
        apiHandler.HandleFunc("/api/v1/sponsor/card", handleCardSponsor)
        apiHandler.HandleFunc("/api/v1/sponsor/mpesa/callback", handleMpesaCallback)
        apiHandler.HandleFunc("/api/v1/sponsor/card/webhook", handleStripeWebhook)

        // Data refresh — triggers adapter re-discovery (called by cron)
        apiHandler.HandleFunc("/api/v1/refresh", makeRefreshHandler(kenyaLaw))

        // What Changed — proactive change detection feed
        apiHandler.HandleFunc("/api/v1/what-changed", makeWhatChangedHandler(kenyaLaw))

        // Policies — public.
        apiHandler.HandleFunc("/api/v1/policies", handlePoliciesList)

        // Apply OptionalAuth + rate limiting + metrics to the API routes.
        // Notifications — in-memory store for now.
        notifStore := NewNotificationStore()
        apiHandler.Handle("/api/v1/notifications", makeNotificationsHandler(notifStore))
        apiHandler.Handle("/api/v1/notifications/", makeNotificationDetailHandler(notifStore))

        // Trust + Provenance (issue #165) — in-memory trust store seeded
        // with sample Kenyan sources, claims, evidence, and one active
        // contradiction so the /trust page has something concrete to render.
        trustStore := NewTrustStore()
        trustStore.SeedSampleTrustData()
        apiHandler.HandleFunc("/api/v1/provenance/", makeProvenanceHandler(trustStore))
        apiHandler.HandleFunc("/api/v1/evidence/", makeEvidenceHandler(trustStore))
        apiHandler.HandleFunc("/api/v1/claims/", makeClaimEvidenceHandler(trustStore))
        apiHandler.HandleFunc("/api/v1/contradictions", makeContradictionsHandler(trustStore))
        apiHandler.HandleFunc("/api/v1/sources", makeTrustSourcesListHandler(trustStore))
        apiHandler.HandleFunc("/api/v1/sources/", makeTrustSourceHandler(trustStore))

        // Corrections (issue #166) — public submit, admin-only list + detail.
        // Auth is enforced inside the handlers via OptionalAuth + scope checks.
        correctionStore := NewCorrectionStore()
        apiHandler.HandleFunc("/api/v1/corrections", makeCorrectionsHandler(correctionStore))
        apiHandler.HandleFunc("/api/v1/corrections/", makeCorrectionDetailHandler(correctionStore))

        // Scenarios — Phase 18 simulation infrastructure. Every response carries
        // an explicit HYPOTHETICAL / SIMULATED reality-layer tag.
        apiHandler.HandleFunc("/api/v1/scenarios", makeScenariosHandler())
        apiHandler.HandleFunc("/api/v1/scenarios/compare", makeScenarioCompareHandler())
        apiHandler.HandleFunc("/api/v1/scenarios/", makeScenarioDetailHandler())

        // Constitution + Government (issue #192). The Constitution is
        // authoritative source material — the platform never reinterprets it.
        // Constitution chapters + articles (issue #212) are seeded from the
        // kenya_seed package and exposed via the /articles sub-resources.
        apiHandler.HandleFunc("/api/v1/governments", makeGovernmentsListHandler())
        apiHandler.HandleFunc("/api/v1/governments/", makeGovernmentDetailHandler())
        apiHandler.HandleFunc("/api/v1/constitution", makeConstitutionHandler())
        apiHandler.HandleFunc("/api/v1/constitution/articles", makeConstitutionArticlesHandler())
        apiHandler.HandleFunc("/api/v1/constitution/articles/", makeConstitutionArticleDetailHandler())
        apiHandler.HandleFunc("/api/v1/transitions", makeTransitionsHandler())

        // Public Debt & Borrowing Intelligence (issue #195, #203). The platform
        // NEVER attributes sovereign borrowing personally to a president.
        // makeDebtRouter consumes the legislation.DebtRepository constructed
        // above (seeded with Kenya's CBK + Treasury observations).
        apiHandler.HandleFunc("/api/v1/debt", makeDebtRouter(debtRepo))
        apiHandler.HandleFunc("/api/v1/debt/", makeDebtRouter(debtRepo))

        rateLimited := middleware.RateLimit(300, time.Minute)(apiHandler)
        metered := observability.MetricsMiddleware(metrics, rateLimited)
        mux.Handle("/api/v1/", middleware.OptionalAuth(verifier)(metered))

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

                // Country subdomain support: read X-Civic-Country header
                // set by Next.js middleware (ke.civicintelligence.com → KE)
                country := r.Header.Get("X-Civic-Country")
                if country == "" {
                        country = "KE" // default
                }

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
                        "country":   country,
                })
        }
}

func makeBillDetailHandler(adapter *kenya_law.Adapter, aiServiceURL string) http.HandlerFunc {
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

                // Sub-routes: timeline, versions, summary, etc.
                if len(parts) == 2 && parts[1] != "" {
                        sub := parts[1]
                        switch {
                        case strings.HasPrefix(sub, "timeline"):
                                handleBillTimeline(w, r, adapter, billID)
                        case strings.HasPrefix(sub, "changes"):
                                handleBillChanges(w, r, adapter, billID)
                        case strings.HasPrefix(sub, "versions"):
                                writeJSON(w, http.StatusOK, map[string]any{"bill_id": billID, "versions": []any{}})
                        case strings.HasPrefix(sub, "documents"):
                                writeJSON(w, http.StatusOK, map[string]any{"bill_id": billID, "documents": []any{}})
                        case strings.HasPrefix(sub, "summary"):
                                handleBillSummary(w, r, adapter, billID, aiServiceURL)
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

// --- Bill Timeline (#101) ---

// billEventResponse is the JSON shape returned by /api/v1/bills/{id}/timeline.
// Every field except `note` is non-empty when sourced from a verified primary
// source. `note` carries any caveat (e.g., an inferred date).
type billEventResponse struct {
        ID                string  `json:"id"`
        BillID            string  `json:"bill_id"`
        EventType         string  `json:"event_type"`
        Date              string  `json:"date"`
        DateIsApproximate bool    `json:"date_is_approximate"`
        House             string  `json:"house"`
        Description       string  `json:"description"`
        SourceURL         string  `json:"source_url"`
        Confidence        string  `json:"confidence"`
        Note              *string `json:"note"`
}

// findBillByID locates a single Bill by its source ID. In production this would
// be a Postgres lookup; for now we re-discover from the Kenya Law adapter and
// match by SourceID (with a substring fallback because callers sometimes use
// truncated IDs from URLs).
func findBillByID(ctx context.Context, adapter *kenya_law.Adapter, billID string) (*kenya_law.BillCandidate, error) {
        if billID == "" {
                return nil, errEmptyBillID
        }
        bills, err := adapter.DiscoverBills(ctx)
        if err != nil {
                return nil, err
        }
        for i := range bills {
                if bills[i].SourceID == billID || strings.Contains(bills[i].SourceID, billID) {
                        return &bills[i], nil
                }
        }
        return nil, nil
}

// errEmptyBillID is returned by findBillByID when no ID was supplied.
var errEmptyBillID = &billLookupError{message: "bill ID required"}

// billLookupError is a simple error type used by findBillByID.
type billLookupError struct{ message string }

func (e *billLookupError) Error() string { return e.message }

// buildTimelineForBill constructs the verified-event timeline for a single
// Bill. Until the events table is wired (issue #19), the only verified event
// is the publication of the Bill on Kenya Law Reports — every Bill carries a
// source URL + publication date extracted from the Akoma Ntoso URL itself, so
// this event is always evidence-backed. The function never invents events:
// if the publication date is missing, an empty timeline is returned.
func buildTimelineForBill(bill *kenya_law.BillCandidate) []billEventResponse {
        if bill == nil {
                return []billEventResponse{}
        }
        if bill.PublicationDate.IsZero() {
                return []billEventResponse{}
        }
        note := "Inferred from Akoma Ntoso URL date component (publication date)."
        return []billEventResponse{
                {
                        ID:                fmt.Sprintf("%s-publication", bill.SourceID),
                        BillID:            bill.SourceID,
                        EventType:         "publication",
                        Date:              bill.PublicationDate.Format(time.RFC3339),
                        DateIsApproximate: false,
                        House:             bill.House,
                        Description:       fmt.Sprintf("Bill published on Kenya Law Reports (%s).", bill.House),
                        SourceURL:         bill.URL,
                        Confidence:        "high",
                        Note:              &note,
                },
        }
}

// handleBillTimeline returns the verified timeline for a Bill.
//
// Route: GET /api/v1/bills/{id}/timeline
//
// Behavior:
//   - If events exist in the DB (future), they are returned.
//   - Otherwise, the Bill's publication date is returned as the first event,
//     sourced from the Akoma Ntoso URL on kenyalaw.org.
//   - If the Bill cannot be found, returns 404.
//   - If the adapter is unreachable, returns 503.
func handleBillTimeline(w http.ResponseWriter, r *http.Request, adapter *kenya_law.Adapter, billID string) {
        ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
        defer cancel()

        bill, err := findBillByID(ctx, adapter, billID)
        if err != nil {
                if err == errEmptyBillID {
                        writeError(w, http.StatusBadRequest, "bad_request", err.Error())
                        return
                }
                log.Printf("timeline handler: adapter error: %v", err)
                writeError(w, http.StatusServiceUnavailable, "adapter_error", "failed to discover bills from Kenya Law")
                return
        }
        if bill == nil {
                writeError(w, http.StatusNotFound, "not_found", "bill not found: "+billID)
                return
        }

        events := buildTimelineForBill(bill)
        writeJSON(w, http.StatusOK, map[string]any{
                "bill_id": bill.SourceID,
                "events":  events,
                "total":   len(events),
                "source":  "new.kenyalaw.org",
        })
}

// --- Bill Version Comparison (#102) ---

// billChangeResponse is the JSON shape returned by /api/v1/bills/{id}/changes.
// It represents a single structural difference between two Bill versions.
// Until the documents service stores multiple Bill versions (issue #19 + ADR-0011),
// the `changes` array is empty and `note` explains why.
type billChangeResponse struct {
        Kind         string `json:"kind"`                   // "addition" | "removal" | "modification"
        Section      string `json:"section,omitempty"`      // e.g., "Clause 14(2)"
        Description  string `json:"description,omitempty"`   // plain-language description
        Before       string `json:"before,omitempty"`        // prior text (for modification/removal)
        After        string `json:"after,omitempty"`         // new text (for modification/addition)
        SourceURL    string `json:"source_url,omitempty"`    // link to the version that introduced the change
        Confidence   string `json:"confidence,omitempty"`   // "high" | "medium" | "low"
}

// handleBillChanges returns a structural diff between Bill versions.
//
// Route: GET /api/v1/bills/{id}/changes
//
// Behavior:
//   - If the Bill cannot be found, returns 404.
//   - If the adapter is unreachable, returns 503.
//   - If only one version exists in the DB (the current state), returns an
//     empty `changes` array with a note explaining that version comparison
//     requires multiple Bill versions in the database.
//   - Never invents changes. When versions are eventually available, the AI
//     document_comparator capability will produce the diff, gated by the
//     anti-hallucination validator.
func handleBillChanges(w http.ResponseWriter, r *http.Request, adapter *kenya_law.Adapter, billID string) {
        ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
        defer cancel()

        bill, err := findBillByID(ctx, adapter, billID)
        if err != nil {
                if err == errEmptyBillID {
                        writeError(w, http.StatusBadRequest, "bad_request", err.Error())
                        return
                }
                log.Printf("changes handler: adapter error: %v", err)
                writeError(w, http.StatusServiceUnavailable, "adapter_error", "failed to discover bills from Kenya Law")
                return
        }
        if bill == nil {
                writeError(w, http.StatusNotFound, "not_found", "bill not found: "+billID)
                return
        }

        // Until the documents service stores multiple immutable Bill versions
        // (per ADR-0011 + issue #19), there is nothing to diff. Return an
        // empty array with a transparent note explaining the gap.
        writeJSON(w, http.StatusOK, map[string]any{
                "bill_id": bill.SourceID,
                "changes": []billChangeResponse{},
                "total":   0,
                "note":    "Version comparison requires multiple Bill versions in the database",
                "source":  "new.kenyalaw.org",
        })
}

// --- Bill Summary (#100) ---

// billSummaryResponse is the JSON shape returned by /api/v1/bills/{id}/summary.
// Every field maps directly to the Python AI service's BillSummary model so the
// frontend's existing BillSummary type works unchanged. The fallback path
// (when the AI service is unavailable) populates the same shape but flags every
// claim as "could not be verified" — the architectural rule is that no claim
// is ever surfaced without either a citation or an explicit unverifiable flag.
type billSummaryResponse struct {
        BillID                   string           `json:"bill_id"`
        ShortTitle               string           `json:"short_title"`
        OneLineSummary           string           `json:"one_line_summary"`
        PlainLanguageExplanation string           `json:"plain_language_explanation"`
        CurrentStageExplained    string           `json:"current_stage_explained"`
        WhatItWouldDo            []string         `json:"what_it_would_do"`
        WhoItAffects             []string         `json:"who_it_affects"`
        WhatHappensNext          []string         `json:"what_happens_next"`
        Citations                []map[string]any `json:"citations"`
        Confidence               string           `json:"confidence"`
        Validated                bool             `json:"validated"`
        ValidationFailures       []string         `json:"validation_failures"`
        Source                   string           `json:"source"`
}

// handleBillSummary returns an evidence-grounded plain-language summary of a
// single Bill. It proxies to the Python AI service's
// POST /v1/bills/{id}/summarize endpoint when available. If the AI service is
// unreachable, it returns a structured stub derived from the Bill's metadata —
// every claim is either backed by the source URL (publication metadata from
// kenyalaw.org) or explicitly flagged as "could not be verified".
//
// Route: GET /api/v1/bills/{id}/summary
func handleBillSummary(w http.ResponseWriter, r *http.Request, adapter *kenya_law.Adapter, billID, aiServiceURL string) {
        if billID == "" {
                writeError(w, http.StatusBadRequest, "bad_request", "bill ID required")
                return
        }

        ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
        defer cancel()

        bill, err := findBillByID(ctx, adapter, billID)
        if err != nil {
                log.Printf("summary handler: adapter error: %v", err)
                writeError(w, http.StatusServiceUnavailable, "adapter_error", "failed to discover bills from Kenya Law")
                return
        }
        if bill == nil {
                writeError(w, http.StatusNotFound, "not_found", "bill not found: "+billID)
                return
        }

        if aiServiceURL != "" {
                if summary, ok := callAIBillSummary(ctx, aiServiceURL, bill); ok {
                        summary.Source = "ai-service"
                        writeJSON(w, http.StatusOK, summary)
                        return
                }
                log.Printf("summary handler: AI service unavailable, falling back to structured stub for bill %s", billID)
        }

        // Fallback: structured stub. Every claim is evidence-backed by the
        // kenyalaw.org source URL or explicitly flagged as unverifiable.
        year := 0
        if !bill.PublicationDate.IsZero() {
                year = bill.PublicationDate.Year()
        }
        pubDateStr := ""
        if !bill.PublicationDate.IsZero() {
                pubDateStr = bill.PublicationDate.Format("2 January 2006")
        }

        oneLine := fmt.Sprintf("%s (House: %s, Year: %d)", bill.Title, bill.House, year)
        explanation := fmt.Sprintf("This is a %s Bill titled %q.", bill.House, bill.Title)
        if pubDateStr != "" {
                explanation += fmt.Sprintf(" It was published on Kenya Law Reports on %s.", pubDateStr)
        }
        explanation += " The detailed plain-language explanation could not be generated because the AI summarization service is currently unavailable."

        citation := map[string]any{
                "document_id":  bill.SourceID,
                "source_url":   bill.URL,
                "source_type":  "kenya_law",
                "snippet":      bill.Title,
                "retrieved_at": time.Now().UTC().Format(time.RFC3339),
        }

        resp := billSummaryResponse{
                BillID:                   bill.SourceID,
                ShortTitle:               bill.Title,
                OneLineSummary:           oneLine,
                PlainLanguageExplanation: explanation,
                CurrentStageExplained:    "Published on Kenya Law Reports. The next stage (First Reading) could not be verified — the parliamentary stage tracker was not consulted.",
                WhatItWouldDo: []string{
                        "Could not be verified — the AI summarization service is unavailable. Consult the source document for the Bill's substantive provisions.",
                },
                WhoItAffects: []string{
                        "Could not be verified — the AI summarization service is unavailable.",
                },
                WhatHappensNext: []string{
                        "Could not be verified — pending parliamentary stage data (issue #19).",
                },
                Citations:          []map[string]any{citation},
                Confidence:         "low",
                Validated:          false,
                ValidationFailures: []string{"ai_service_unavailable", "no_full_text_extracted"},
                Source:             "fallback",
        }
        writeJSON(w, http.StatusOK, resp)
}

// callAIBillSummary POSTs to the Python AI service's
// /v1/bills/{id}/summarize endpoint and returns the parsed summary.
// Returns (summary, false) on any error — callers fall back to the stub.
func callAIBillSummary(ctx context.Context, aiServiceURL string, bill *kenya_law.BillCandidate) (billSummaryResponse, bool) {
        year := 0
        if !bill.PublicationDate.IsZero() {
                year = bill.PublicationDate.Year()
        }
        // The AI service validates bill_id as UUID, but only uses it as a
        // passthrough identifier — never for lookup. We generate a deterministic
        // UUID v5 from the bill's source ID so the same Bill always maps to the
        // same UUID across calls (useful for log correlation).
        billUUID := billIDToUUID(bill.SourceID)

        reqBody := map[string]any{
                "title":         bill.Title,
                "identifier":    bill.Slug,
                "year":          year,
                "sponsor":       nil,
                "house":         bill.House,
                "current_stage": "published",
                "plain_text":    fmt.Sprintf("Bill: %s\nHouse: %s\nPublished: %s\nSource: %s\n", bill.Title, bill.House, bill.PublicationDate.Format("2006-01-02"), bill.URL),
                "citations":     []any{},
        }
        bodyBytes, err := json.Marshal(reqBody)
        if err != nil {
                return billSummaryResponse{}, false
        }

        url := strings.TrimRight(aiServiceURL, "/") + "/v1/bills/" + billUUID + "/summarize"
        req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
        if err != nil {
                return billSummaryResponse{}, false
        }
        req.Header.Set("Content-Type", "application/json")
        req.Header.Set("Accept", "application/json")

        client := &http.Client{Timeout: 25 * time.Second}
        resp, err := client.Do(req)
        if err != nil {
                return billSummaryResponse{}, false
        }
        defer resp.Body.Close()
        if resp.StatusCode != http.StatusOK {
                return billSummaryResponse{}, false
        }

        raw, err := io.ReadAll(resp.Body)
        if err != nil {
                return billSummaryResponse{}, false
        }
        var summary billSummaryResponse
        if err := json.Unmarshal(raw, &summary); err != nil {
                return billSummaryResponse{}, false
        }
        return summary, true
}

// billIDToUUID derives a deterministic UUID v5 (SHA-1 namespace) string from a
// bill source ID. The Python AI service's path parameter is typed as UUID.
func billIDToUUID(billID string) string {
        var ns [16]byte
        copy(ns[:], []byte("civic-intelligen"))
        h := sha1.New()
        h.Write(ns[:])
        h.Write([]byte(billID))
        sum := h.Sum(nil)
        var b [16]byte
        copy(b[:], sum[:16])
        b[6] = (b[6] & 0x0f) | 0x50
        b[8] = (b[8] & 0x3f) | 0x80
        return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// --- Search ---
//
// The search handler used to live inline in this file (returning an empty
// result with a "pending issue #45" note). It has been promoted to its own
// file, search.go, with an in-memory stopgap implementation that actually
// returns results across Bills, Acts, and Constitution articles. The route
// registration `apiHandler.HandleFunc("/api/v1/search", handleSearch)` below
// is unchanged; only the function's body has moved.
//
// See search.go for the implementation and the stopgap note about the
// pending Postgres FTS wiring.

// --- Briefing ---

func handleBriefing(w http.ResponseWriter, r *http.Request) {
        writeJSON(w, http.StatusOK, map[string]any{
                "country": "KE",
                "date":    time.Now().UTC().Format(time.RFC3339),
                "items":   []any{},
                "note":    "Briefing — pending issue #40",
        })
}

// --- Trending Bills (#107) ---

func makeTrendingHandler(adapter *kenya_law.Adapter) http.HandlerFunc {
        return func(w http.ResponseWriter, r *http.Request) {
                ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
                defer cancel()

                bills, err := adapter.DiscoverBills(ctx)
                if err != nil {
                        writeError(w, http.StatusServiceUnavailable, "adapter_error", "failed to discover bills")
                        return
                }

                // Categorize bills:
                // - recently_published: published in last 30 days
                // - trending: most recent (hot)
                // - approaching_final: bills whose titles suggest late-stage (Amendment, etc.)
                // In production, this would use actual stage data from the Parliament adapter.

                now := time.Now()
                var recentlyPublished []billResponse
                var hot []billResponse
                var approaching []billResponse

                for _, b := range bills {
                        year := 0
                        if !b.PublicationDate.IsZero() {
                                year = b.PublicationDate.Year()
                        }
                        resp := billResponse{
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
                        }

                        // Recently published (last 30 days)
                        if !b.PublicationDate.IsZero() && now.Sub(b.PublicationDate) < 30*24*time.Hour {
                                recentlyPublished = append(recentlyPublished, resp)
                        }

                        // Hot = most recent 5
                        if len(hot) < 5 {
                                hot = append(hot, resp)
                        }

                        // Approaching final stage — bills with "Amendment" in the title
                        // (in production, use actual stage data from Parliament adapter)
                        titleLower := strings.ToLower(b.Title)
                        if strings.Contains(titleLower, "amendment") ||
                           strings.Contains(titleLower, "finance") ||
                           strings.Contains(titleLower, "appropriation") {
                                approaching = append(approaching, resp)
                        }
                }

                // Limit recently_published to 10
                if len(recentlyPublished) > 10 {
                        recentlyPublished = recentlyPublished[:10]
                }
                // Limit approaching to 5
                if len(approaching) > 5 {
                        approaching = approaching[:5]
                }

                writeJSON(w, http.StatusOK, map[string]any{
                        "recently_published": recentlyPublished,
                        "hot":                hot,
                        "approaching_final":  approaching,
                        "total_bills":        len(bills),
                        "source":             "new.kenyalaw.org",
                })
        }
}

// --- Terminology (#103) ---

func makeTerminologyHandler() http.HandlerFunc {
        // Kenya parliamentary terminology — sourced from adapters/kenya/internal/terminology.go
        // In production, this would call the Kenya adapter's GetTerminology() method.
        terms := map[string]map[string]string{
                "second_reading": {
                        "term":               "Second Reading",
                        "simple_explanation": "MPs debate the principles and policy of the Bill. A vote is taken on whether the Bill should proceed.",
                        "official_definition": "Per Standing Order 95.",
                        "stage_code":         "SECOND_READING",
                        "country":            "KE",
                        "source":             "https://parliament.go.ke/standing-orders",
                },
                "first_reading": {
                        "term":               "First Reading",
                        "simple_explanation": "The Bill is read for the first time in the House. No debate on the substance yet — the Bill is simply introduced.",
                        "official_definition": "Per Standing Order 91.",
                        "stage_code":         "FIRST_READING",
                        "country":            "KE",
                        "source":             "https://parliament.go.ke/standing-orders",
                },
                "committee_stage": {
                        "term":               "Committee Stage",
                        "simple_explanation": "A committee examines the Bill clause-by-clause and may propose amendments.",
                        "official_definition": "Per Standing Order 117.",
                        "stage_code":         "COMMITTEE_STAGE",
                        "country":            "KE",
                        "source":             "https://parliament.go.ke/standing-orders",
                },
                "presidential_assent": {
                        "term":               "Presidential Assent",
                        "simple_explanation": "The President signs the Bill into law. Within 14 days of receipt. The President may refer it back once.",
                        "official_definition": "Article 115, Constitution of Kenya, 2010.",
                        "stage_code":         "PRESIDENTIAL_ASSENT",
                        "country":            "KE",
                        "source":             "https://www.kenyalaw.org/kl/index.php?id=398",
                },
                "commencement": {
                        "term":               "Commencement",
                        "simple_explanation": "The Act comes into force. Either on the date of assent, on a date specified in the Act, or by a separate commencement notice in the Kenya Gazette.",
                        "official_definition": "Article 116, Constitution of Kenya, 2010.",
                        "stage_code":         "COMMENCEMENT",
                        "country":            "KE",
                        "source":             "https://www.kenyalaw.org/kl/index.php?id=398",
                },
                "hansard": {
                        "term":               "Hansard",
                        "simple_explanation": "The official verbatim record of parliamentary debates.",
                        "country":            "KE",
                        "source":             "https://parliament.go.ke/hansard",
                },
                "order_paper": {
                        "term":               "Order Paper",
                        "simple_explanation": "The official daily agenda of the House — what will be discussed, in what order.",
                        "country":            "KE",
                        "source":             "https://parliament.go.ke/order-papers",
                },
                "gazette_notice": {
                        "term":               "Gazette Notice",
                        "simple_explanation": "An official publication in the Kenya Gazette — the official record of government notices.",
                        "country":            "KE",
                        "source":             "https://www.kenyalaw.org/kl/index.php?id=589",
                },
                "money_bill": {
                        "term":               "Money Bill",
                        "simple_explanation": "A Bill that concerns taxation, public debt, or public expenditure. Originates only in the National Assembly.",
                        "official_definition": "Article 114, Constitution of Kenya, 2010.",
                        "country":            "KE",
                        "source":             "https://www.kenyalaw.org/kl/index.php?id=398",
                },
                "public_participation": {
                        "term":               "Public Participation",
                        "simple_explanation": "Constitutionally required process where the public is invited to submit views on proposed legislation.",
                        "official_definition": "Article 118, Constitution of Kenya, 2010.",
                        "country":            "KE",
                        "source":             "https://www.kenyalaw.org/kl/index.php?id=398",
                },
        }

        return func(w http.ResponseWriter, r *http.Request) {
                // GET /api/v1/terminology/ → list all terms
                // GET /api/v1/terminology/{term} → single term
                path := strings.TrimPrefix(r.URL.Path, "/api/v1/terminology/")
                if path == "" {
                        // List all
                        items := make([]map[string]string, 0, len(terms))
                        for _, t := range terms {
                                items = append(items, t)
                        }
                        writeJSON(w, http.StatusOK, map[string]any{
                                "items": items,
                                "total": len(items),
                        })
                        return
                }

                // Single term lookup
                term, ok := terms[path]
                if !ok {
                        // Try with spaces replaced by underscores
                        term, ok = terms[strings.ReplaceAll(path, " ", "_")]
                }
                if !ok {
                        writeError(w, http.StatusNotFound, "not_found", "terminology not found: "+path)
                        return
                }
                writeJSON(w, http.StatusOK, term)
        }
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

// --- Acts of Parliament (issue #116) ---

// actResponse is the JSON shape returned by the acts endpoint.
type actResponse struct {
        ID               string `json:"id"`
        Title            string `json:"title"`
        Citation         string `json:"citation"`
        AssentDate       string `json:"assent_date,omitempty"`
        CommencementDate string `json:"commencement_date,omitempty"`
        SourceURL        string `json:"source_url"`
        Status           string `json:"status"`
        Country          string `json:"country"`
        Summary          string `json:"summary,omitempty"`
}

// actRepo is the in-memory ActRepository, seeded with verified Kenyan
// Acts of Parliament from adapters/kenya/kenya_seed (issue #202). The
// legislation service's Wire() constructs the repository and seeds it;
// handlers query it directly instead of relying on hardcoded arrays.
var actRepo = buildActRepo()

// subscriptionStore is the in-memory SubscriptionStore (issue #110). It is
// promoted to a package-level var so that BOTH /api/v1/subscriptions (issue
// #110) AND /api/v1/acts/{id}/follow (issue #216 — flagship Follow-a-Law
// experience) can write real subscription records through the same store.
// In production this is replaced by notifications.follows in Postgres
// (migration 014_notifications.up.sql).
var subscriptionStore = NewSubscriptionStore()

// buildActRepo constructs the ActRepository from the Kenya seed data.
// The seed DTOs are mapped into the legislation service's domain.Act /
// PostAssentEvent types via kenya_seed.MapActs / MapPostAssentEvents,
// then handed to legislation.Wire() which builds the in-memory repo and
// populates it.
func buildActRepo() legislation.ActRepository {
        now := time.Now().UTC()
        acts := kenya_seed.MapActs(kenya_seed.KenyaActs, now)
        events := kenya_seed.MapPostAssentEvents(kenya_seed.KenyaPostAssentEvents, now)
        return legislation.Wire(acts, events)
}

// sampleActs is a cached slice built from actRepo at startup. It is kept
// for backward compatibility with the existing API contract test
// (TestActsList_HasThreeToFive) and as a convenient in-process lookup.
// The authoritative source is actRepo — handlers query it directly.
var sampleActs = buildSampleActs()

func buildSampleActs() []actResponse {
        acts, err := actRepo.ListActs(context.Background(), legislation.ActFilter{})
        if err != nil {
                return []actResponse{}
        }
        out := make([]actResponse, 0, len(acts))
        for _, a := range acts {
                out = append(out, toActResponse(a))
        }
        return out
}

// toActResponse converts a domain Act into the JSON response shape. Field
// mapping:
//   - actResponse.Title    ← Act.ActName
//   - actResponse.Citation  ← Act.ActNumber
//   - actResponse.Status    ← string(Act.Status)        // "in_force", "amended", ...
//   - actResponse.Country   ← string(Act.CountryID)      // "KE"
//   - actResponse.Summary   ← Act.Description
func toActResponse(a legislation.Act) actResponse {
        resp := actResponse{
                ID:        string(a.ID),
                Title:     a.ActName,
                Citation:  a.ActNumber,
                SourceURL: a.SourceURL,
                Status:    string(a.Status),
                Country:   string(a.CountryID),
                Summary:   a.Description,
        }
        if !a.AssentedAt.IsZero() {
                resp.AssentDate = a.AssentedAt.Format("2006-01-02")
        }
        if a.CommencementDate != nil {
                resp.CommencementDate = a.CommencementDate.Format("2006-01-02")
        }
        return resp
}

func handleActsList(w http.ResponseWriter, r *http.Request) {
        q := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("q")))
        status := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("status")))

        // Query the repository for all acts. Search + status filtering is
        // applied in-memory to preserve the existing API contract.
        acts, err := actRepo.ListActs(r.Context(), legislation.ActFilter{})
        if err != nil {
                writeError(w, http.StatusInternalServerError, "internal_error", "failed to list acts")
                return
        }

        items := make([]actResponse, 0, len(acts))
        for _, a := range acts {
                resp := toActResponse(a)
                if status != "" && resp.Status != status {
                        continue
                }
                if q != "" {
                        haystack := strings.ToLower(resp.Title + " " + resp.Citation + " " + resp.Summary)
                        if !strings.Contains(haystack, q) {
                                continue
                        }
                }
                items = append(items, resp)
        }

        writeJSON(w, http.StatusOK, map[string]any{
                "items":  items,
                "total":  len(items),
                "source": "kenyalaw.org",
                "note":   "Sample data — full ingestion pending (issue #19). Every entry links to a verified Kenya Law source.",
        })
}

func handleActDetail(w http.ResponseWriter, r *http.Request) {
        id := strings.TrimPrefix(r.URL.Path, "/api/v1/acts/")
        if id == "" {
                writeError(w, http.StatusBadRequest, "bad_request", "act ID required")
                return
        }
        a, err := actRepo.GetAct(r.Context(), legislation.ID(id))
        if err != nil {
                writeError(w, http.StatusNotFound, "not_found", "act not found: "+id)
                return
        }
        writeJSON(w, http.StatusOK, toActResponse(*a))
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


// --- Civic Feed (#113) ---

func makeCivicFeedHandler(adapter *kenya_law.Adapter) http.HandlerFunc {
        return func(w http.ResponseWriter, r *http.Request) {
        // For now, return the most recent bills as feed items.
        // In production, this would aggregate events from all civic domains.
        ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
        defer cancel()

        bills, err := adapter.DiscoverBills(ctx)
        if err != nil {
                writeJSON(w, http.StatusOK, map[string]any{"items": []any{}, "note": "feed unavailable"})
                return
        }

        items := make([]map[string]any, 0, len(bills))
        for i, b := range bills {
                if i >= 20 {
                        break
                }
                items = append(items, map[string]any{
                        "kind":         "bill_published",
                        "title":        b.Title,
                        "house":        b.House,
                        "date":         b.PublicationDate.Format("2006-01-02"),
                        "source_url":   b.URL,
                        "description":  "New Bill published on Kenya Law",
                        "significance": "medium",
                })
        }

                writeJSON(w, http.StatusOK, map[string]any{
                        "items": items,
                        "total": len(items),
                        "source": "new.kenyalaw.org",
                })
        }
}

// --- Policies (#118) ---

func handlePoliciesList(w http.ResponseWriter, r *http.Request) {
        policies := []map[string]any{
                {
                        "id":             "ke-policy-digital-economy",
                        "title":          "Digital Economy Strategy",
                        "institution":    "Ministry of Information, Communication and Digital Economy",
                        "published_date": "2023-03-15",
                        "status":         "active",
                        "source_url":     "https://www.ict.go.ke",
                        "summary":        "Strategy for Kenya's digital transformation including digital identity, e-government, and digital economy growth.",
                },
                {
                        "id":             "ke-policy-affordable-housing",
                        "title":          "Affordable Housing Programme",
                        "institution":    "Ministry of Lands, Public Works, Housing and Urban Development",
                        "published_date": "2023-01-20",
                        "status":         "active",
                        "source_url":     "https://www.housing.go.ke",
                        "summary":        "Government programme to deliver 500,000 affordable housing units.",
                },
                {
                        "id":             "ke-policy-uhc",
                        "title":          "Universal Health Coverage",
                        "institution":    "Ministry of Health",
                        "published_date": "2023-06-01",
                        "status":         "active",
                        "source_url":     "https://www.health.go.ke",
                        "summary":        "Programme to provide affordable healthcare to all Kenyan citizens.",
                },
        }
        writeJSON(w, http.StatusOK, map[string]any{"items": policies, "total": len(policies)})
}


// --- Sponsor (#137) ---

// handleMpesaSponsor initiates an M-Pesa STK Push via the Safaricom Daraja API.
// POST /api/v1/sponsor/mpesa
// Body: {"phone": "0712345678", "amount": 500}
// Response: {"status": "pending", "message": "Check your phone for the M-Pesa prompt"}
func handleMpesaSponsor(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodPost {
                writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use POST")
                return
        }

        var req struct {
                Phone  string `json:"phone"`
                Amount int    `json:"amount"`
        }
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
                writeError(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
                return
        }

        if req.Phone == "" || req.Amount < 1 {
                writeError(w, http.StatusBadRequest, "bad_request", "phone and amount (min 1) required")
                return
        }

        // In production, this would call the Safaricom Daraja API:
        // 1. Get OAuth token from Safaricom
        // 2. Send STK Push request to the phone number
        // 3. Return the checkout request ID
        //
        // For now, return a success response indicating the STK push was initiated.
        // The actual Daraja API integration requires:
        //   - MPESA_CONSUMER_KEY env var
        //   - MPESA_CONSUMER_SECRET env var
        //   - MPESA_SHORTCODE env var (paybill/till number)
        //   - MPESA_PASSKEY env var
        //   - MPESA_CALLBACK_URL env var

        writeJSON(w, http.StatusOK, map[string]any{
                "status":  "pending",
                "message": "M-Pesa STK Push initiated. Check your phone to confirm the payment of KES " + fmt.Sprintf("%d", req.Amount) + ".",
                "phone":   req.Phone,
                "amount":  req.Amount,
                "note":    "In production, this calls the Safaricom Daraja STK Push API. Configure MPESA_CONSUMER_KEY, MPESA_CONSUMER_SECRET, MPESA_SHORTCODE, MPESA_PASSKEY env vars.",
        })
}

// handleMpesaCallback handles the M-Pesa STK Push callback from Safaricom.
// POST /api/v1/sponsor/mpesa/callback
// This endpoint is called by Safaricom's servers after the user confirms/denies the payment.
func handleMpesaCallback(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodPost {
                writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use POST")
                return
        }

        // Parse the callback from Safaricom Daraja API
        // In production, this would:
        // 1. Parse the callback JSON
        // 2. Verify the payment was successful
        // 3. Record the sponsorship in the database
        // 4. Send a receipt email
        // 5. Return 200 OK to Safaricom

        writeJSON(w, http.StatusOK, map[string]any{
                "status": "received",
        })
}

// handleCardSponsor creates a Stripe Checkout session for card payments.
// POST /api/v1/sponsor/card
// Body: {"amount": 500, "email": "you@example.com"}
// Response: {"checkout_url": "https://checkout.stripe.com/..."}
func handleCardSponsor(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodPost {
                writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use POST")
                return
        }

        var req struct {
                Amount int    `json:"amount"`
                Email  string `json:"email"`
        }
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
                writeError(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
                return
        }

        if req.Amount < 1 {
                writeError(w, http.StatusBadRequest, "bad_request", "amount (min 1) required")
                return
        }

        // In production, this would create a Stripe Checkout Session:
        // 1. Initialize Stripe client with STRIPE_SECRET_KEY env var
        // 2. Create a checkout session with:
        //    - amount (converted to KES cents)
        //    - currency: kes
        //    - success_url: https://civic-intelligence.vercel.app/sponsor?status=success
        //    - cancel_url: https://civic-intelligence.vercel.app/sponsor?status=cancelled
        //    - customer_email (if provided)
        // 3. Return the checkout URL

        // For now, return a placeholder checkout URL
        writeJSON(w, http.StatusOK, map[string]any{
                "checkout_url": "https://checkout.stripe.com/c/pay/cs_test_placeholder_" + fmt.Sprintf("%d", req.Amount),
                "amount":       req.Amount,
                "currency":     "KES",
                "note":         "In production, this creates a real Stripe Checkout session. Configure STRIPE_SECRET_KEY env var.",
        })
}

// handleStripeWebhook handles Stripe webhook events for card payment confirmation.
// POST /api/v1/sponsor/card/webhook
func handleStripeWebhook(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodPost {
                writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use POST")
                return
        }

        // In production, this would:
        // 1. Verify the Stripe webhook signature
        // 2. Parse the event (payment_intent.succeeded, payment_intent.failed)
        // 3. Record the sponsorship in the database
        // 4. Send a receipt email

        writeJSON(w, http.StatusOK, map[string]any{
                "status": "received",
        })
}


// --- Data Refresh (#181) ---

// makeRefreshHandler triggers re-discovery of Bills from kenyalaw.org.
// This endpoint is called by a cron job (hourly) to keep data fresh.
// POST /api/v1/refresh
func makeRefreshHandler(adapter *kenya_law.Adapter) http.HandlerFunc {
        return func(w http.ResponseWriter, r *http.Request) {
                if r.Method != http.MethodPost {
                        writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use POST")
                        return
                }

                ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
                defer cancel()

                bills, err := adapter.DiscoverBills(ctx)
                if err != nil {
                        log.Printf("refresh: adapter error: %v", err)
                        writeError(w, http.StatusServiceUnavailable, "adapter_error", "failed to refresh bills")
                        return
                }

                log.Printf("refresh: discovered %d bills from kenyalaw.org", len(bills))
                writeJSON(w, http.StatusOK, map[string]any{
                        "status":     "refreshed",
                        "bills_found": len(bills),
                        "source":     "new.kenyalaw.org",
                        "refreshed_at": time.Now().UTC().Format(time.RFC3339),
                })
        }
}


// --- What Changed Engine (#186 Phase 14) ---

// ChangeItem represents a single verified civic change.
type ChangeItem struct {
        Kind          string `json:"kind"`           // bill_published, stage_changed, assent, regulation, policy
        Title         string `json:"title"`
        Description   string `json:"description"`
        House         string `json:"house"`
        Date          string `json:"date"`
        SourceURL     string `json:"source_url"`
        Significance  string `json:"significance"`   // INFORMATIONAL, MINOR, PROCEDURAL, SUBSTANTIVE, HIGH_IMPACT, CRITICAL
        Verification  string `json:"verification"`   // VERIFIED, PENDING, CONFLICTED
        EvidenceURL   string `json:"evidence_url"`
        Country       string `json:"country"`
}

// handleWhatChanged returns a feed of recent verified civic changes.
// GET /api/v1/what-changed
// This is the proactive intelligence feed — it shows what changed recently,
// not what a citizen asked for. Every item links to evidence.
func makeWhatChangedHandler(kenyaLaw *kenya_law.Adapter) http.HandlerFunc {
        return func(w http.ResponseWriter, r *http.Request) {
        // In production, this would query the trust.claims + trust.audit_events tables
        // for recently verified changes. For now, return the most recently published
        // Bills as "what changed" items.
        ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
        defer cancel()

        country := r.Header.Get("X-Civic-Country")
        if country == "" {
                country = "KE"
        }

        bills, err := kenyaLaw.DiscoverBills(ctx)
        if err != nil {
                writeJSON(w, http.StatusOK, map[string]any{
                        "items":   []any{},
                        "message": "No recent changes detected. Check back later.",
                })
                return
        }

        // Transform bills into change items
        items := make([]ChangeItem, 0, len(bills))
        for i, b := range bills {
                if i >= 20 { // limit to 20 most recent
                        break
                }

                significance := "INFORMATIONAL"
                if strings.Contains(strings.ToLower(b.Title), "amendment") {
                        significance = "SUBSTANTIVE"
                } else if strings.Contains(strings.ToLower(b.Title), "finance") || strings.Contains(strings.ToLower(b.Title), "appropriation") {
                        significance = "HIGH_IMPACT"
                }

                items = append(items, ChangeItem{
                        Kind:         "bill_published",
                        Title:        b.Title,
                        Description:  "New Bill published on Kenya Law.",
                        House:        b.House,
                        Date:         b.PublicationDate.Format("2006-01-02"),
                        SourceURL:    b.URL,
                        Significance: significance,
                        Verification: "VERIFIED",
                        EvidenceURL:  b.URL,
                        Country:      country,
                })
        }

        writeJSON(w, http.StatusOK, map[string]any{
                "items":    items,
                "total":    len(items),
                "country":  country,
                "source":   "new.kenyalaw.org",
                "generated_at": time.Now().UTC().Format(time.RFC3339),
        })
}

}

// handleWhatChangedDetail returns the full change explanation for a specific change.
// GET /api/v1/what-changed/{id}
