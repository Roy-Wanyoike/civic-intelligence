// Package main is the entrypoint for the Civic Intelligence API / BFF.
package main

import (
        "context"
        "encoding/json"
        "fmt"
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
        "github.com/Roy-Wanyoike/civic-intelligence/packages/observability"
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

        // Build the metric registry for observability.
        metrics := observability.NewMetricRegistry()

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
        apiHandler.HandleFunc("/api/v1/bills", makeBillsHandler(kenyaLaw))
        apiHandler.HandleFunc("/api/v1/bills/", makeBillDetailHandler(kenyaLaw))

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

        // Apply OptionalAuth + rate limiting + metrics to the API routes.
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
                                handleBillTimeline(w, r, adapter, billID)
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
