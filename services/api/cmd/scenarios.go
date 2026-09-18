// Package main provides scenario API endpoints that proxy to the simulation
// service. Phase 18 section 33.
//
// Endpoints:
//
//      POST /api/v1/scenarios          -- create a scenario
//      GET  /api/v1/scenarios          -- list scenarios
//      GET  /api/v1/scenarios/{id}     -- get a scenario
//      POST /api/v1/scenarios/{id}?action=validate  -- validate
//      POST /api/v1/scenarios/{id}?action=run       -- run
//      POST /api/v1/scenarios/{id}?action=replay    -- replay (run ID)
//      GET  /api/v1/scenarios/{id}/assumptions
//      GET  /api/v1/scenarios/{id}/evidence
//      GET  /api/v1/scenarios/{id}/results
//      GET  /api/v1/scenarios/{id}/timeline
//      GET  /api/v1/scenarios/{id}/methodology
//      POST /api/v1/scenarios/compare
package main

import (
        "encoding/json"
        "io"
        "log"
        "net/http"
        "strings"
        "time"

        sim "github.com/Roy-Wanyoike/civic-intelligence/services/simulation"
)

// scenarioSvc is the in-memory scenario service, seeded with the golden
// dataset (Phase 18 section 30). In production this is replaced by a
// Postgres-backed repository.
var (
        scenarioSvc *sim.ScenarioService
        runSvc      *sim.RunService
)

func init() {
        scenarioSvc, runSvc = sim.Wire()
}

// tenantFromRequest returns the tenant ID. In production this comes from the
// OIDC token. For now we use a default tenant.
func tenantFromRequest(r *http.Request) sim.TenantID {
        if t := r.Header.Get("X-Tenant-ID"); t != "" {
                return sim.TenantID(t)
        }
        return "tenant-default"
}

// makeScenariosHandler handles POST (create) + GET (list) on /api/v1/scenarios.
// Special case: POST /api/v1/scenarios/compare goes to makeScenarioCompareHandler.
func makeScenariosHandler() http.HandlerFunc {
        return func(w http.ResponseWriter, r *http.Request) {
                w.Header().Set("Content-Type", "application/json")
                switch r.Method {
                case http.MethodGet:
                        handleScenariosList(w, r)
                case http.MethodPost:
                        handleScenarioCreate(w, r)
                default:
                        writeMethodNotAllowed(w)
                }
        }
}

func handleScenariosList(w http.ResponseWriter, r *http.Request) {
        tenant := tenantFromRequest(r)
        scenarios, err := scenarioSvc.ListScenarios(r.Context(), tenant, sim.ScenarioFilter{})
        if err != nil {
                writeSimError(w, 500, "list failed: "+err.Error())
                return
        }
        writeJSON(w, 200, map[string]any{
                "scenarios":  scenarios,
                "count":      len(scenarios),
                "disclaimer": "All scenarios are HYPOTHETICAL. They are NOT observed civic facts.",
        })
}

func handleScenarioCreate(w http.ResponseWriter, r *http.Request) {
        tenant := tenantFromRequest(r)
        var s sim.Scenario
        if err := decodeSimBody(r, &s); err != nil {
                writeSimError(w, 400, "invalid body: "+err.Error())
                return
        }
        s.TenantID = tenant
        if s.ID == "" {
                s.ID = sim.ID("scn-" + randID(8))
        }
        if err := scenarioSvc.CreateScenario(r.Context(), &s); err != nil {
                writeSimError(w, 400, "create failed: "+err.Error())
                return
        }
        writeJSON(w, 201, s)
}

// makeScenarioDetailHandler handles GET /scenarios/{id} and sub-resources.
func makeScenarioDetailHandler() http.HandlerFunc {
        return func(w http.ResponseWriter, r *http.Request) {
                w.Header().Set("Content-Type", "application/json")
                tenant := tenantFromRequest(r)
                path := strings.TrimPrefix(r.URL.Path, "/api/v1/scenarios/")
                // path is now: {id} or {id}/assumptions etc.
                parts := strings.SplitN(path, "/", 2)
                id := sim.ID(parts[0])
                sub := ""
                if len(parts) > 1 {
                        sub = strings.TrimSuffix(parts[1], "/")
                }

                switch {
                case sub == "" && r.Method == http.MethodGet:
                        handleScenarioGet(w, r, tenant, id)
                case sub == "" && r.Method == http.MethodPost:
                        q := r.URL.Query().Get("action")
                        switch q {
                        case "validate":
                                handleScenarioValidate(w, r, tenant, id)
                        case "run":
                                handleScenarioRun(w, r, tenant, id)
                        case "replay":
                                handleScenarioReplay(w, r, tenant, id)
                        default:
                                writeSimError(w, 400, "unknown action")
                        }
                case sub == "assumptions" && r.Method == http.MethodGet:
                        handleScenarioAssumptions(w, r, tenant, id)
                case sub == "evidence" && r.Method == http.MethodGet:
                        handleScenarioEvidence(w, r, tenant, id)
                case sub == "results" && r.Method == http.MethodGet:
                        handleScenarioResults(w, r, tenant, id)
                case sub == "timeline" && r.Method == http.MethodGet:
                        handleScenarioTimeline(w, r, tenant, id)
                case sub == "methodology" && r.Method == http.MethodGet:
                        handleScenarioMethodology(w, r, tenant, id)
                default:
                        writeMethodNotAllowed(w)
                }
        }
}

func handleScenarioGet(w http.ResponseWriter, r *http.Request, tenant sim.TenantID, id sim.ID) {
        s, err := scenarioSvc.GetScenario(r.Context(), tenant, id)
        if err != nil {
                writeSimError(w, 404, "not found")
                return
        }
        writeJSON(w, 200, map[string]any{
                "scenario":      s,
                "reality_layer": s.RealityLayer,
                "disclaimer":    "This is a HYPOTHETICAL SCENARIO. It is NOT an observed civic fact.",
        })
}

func handleScenarioValidate(w http.ResponseWriter, r *http.Request, tenant sim.TenantID, id sim.ID) {
        writeJSON(w, 200, map[string]any{
                "valid":      true,
                "disclaimer": "Scenario validated. It remains HYPOTHETICAL.",
        })
}

func handleScenarioRun(w http.ResponseWriter, r *http.Request, tenant sim.TenantID, id sim.ID) {
        userID := sim.ID(r.Header.Get("X-User-ID"))
        if userID == "" {
                userID = "user-anon"
        }
        seed := int64(42)
        if s := r.URL.Query().Get("seed"); s != "" {
                // best-effort numeric parse
                var n int64
                for _, c := range s {
                        if c < '0' || c > '9' {
                                break
                        }
                        n = n*10 + int64(c-'0')
                }
                if n > 0 {
                        seed = n
                }
        }
        run, err := runSvc.Run(r.Context(), tenant, id, userID, seed)
        if err != nil {
                writeSimError(w, 500, "run failed: "+err.Error())
                return
        }
        writeJSON(w, 200, map[string]any{
                "run":           run,
                "reality_layer": "MODELED",
                "disclaimer":    "These results are SIMULATED. They are NOT observed civic facts.",
        })
}

func handleScenarioReplay(w http.ResponseWriter, r *http.Request, tenant sim.TenantID, id sim.ID) {
        run, err := runSvc.Replay(r.Context(), tenant, id)
        if err != nil {
                writeSimError(w, 500, "replay failed: "+err.Error())
                return
        }
        writeJSON(w, 200, map[string]any{
                "run":       run,
                "disclaimer": "Replayed run. Outputs are SIMULATED.",
        })
}

func handleScenarioAssumptions(w http.ResponseWriter, r *http.Request, tenant sim.TenantID, id sim.ID) {
        s, err := scenarioSvc.GetScenario(r.Context(), tenant, id)
        if err != nil {
                writeSimError(w, 404, "not found")
                return
        }
        writeJSON(w, 200, map[string]any{
                "assumptions": s.Assumptions,
                "count":       len(s.Assumptions),
        })
}

func handleScenarioEvidence(w http.ResponseWriter, r *http.Request, tenant sim.TenantID, id sim.ID) {
        s, err := scenarioSvc.GetScenario(r.Context(), tenant, id)
        if err != nil {
                writeSimError(w, 404, "not found")
                return
        }
        writeJSON(w, 200, map[string]any{
                "source_evidence":           s.SourceEvidence,
                "baseline_evidence":         s.Baseline.SourceRefs,
                "assumption_evidence_count": countAssumptionEvidence(s.Assumptions),
        })
}

func handleScenarioResults(w http.ResponseWriter, r *http.Request, tenant sim.TenantID, id sim.ID) {
        runs, err := runSvc.Runs.List(r.Context(), tenant, &id)
        if err != nil {
                writeSimError(w, 500, "list runs failed: "+err.Error())
                return
        }
        out := []map[string]any{}
        for _, run := range runs {
                if run.ResultSummary == nil {
                        continue
                }
                out = append(out, map[string]any{
                        "run_id":       run.ID,
                        "completed_at": run.CompletedAt,
                        "duration":     run.Duration.String(),
                        "summary":      run.ResultSummary,
                })
        }
        writeJSON(w, 200, map[string]any{
                "results":       out,
                "count":         len(out),
                "reality_layer": "MODELED",
                "disclaimer":    "These results are SIMULATED. They are NOT observed civic facts.",
        })
}

func handleScenarioTimeline(w http.ResponseWriter, r *http.Request, tenant sim.TenantID, id sim.ID) {
        s, err := scenarioSvc.GetScenario(r.Context(), tenant, id)
        if err != nil {
                writeSimError(w, 404, "not found")
                return
        }
        timeline := []map[string]any{
                {
                        "timestamp":   s.TimeHorizon.Start,
                        "label":       "OBSERVED",
                        "title":       "Baseline",
                        "description": s.Baseline.Description,
                },
        }
        for _, a := range s.Assumptions {
                timeline = append(timeline, map[string]any{
                        "timestamp":   s.TimeHorizon.Start,
                        "label":       "ASSUMED",
                        "title":       a.Statement,
                        "description": "Assumption: " + a.Statement,
                })
        }
        timeline = append(timeline, map[string]any{
                "timestamp":   s.TimeHorizon.End,
                "label":       "MODELED",
                "title":       "Modeled endpoint",
                "description": "Simulated outcome under the stated assumptions.",
        })
        writeJSON(w, 200, map[string]any{
                "timeline":   timeline,
                "disclaimer": "Timeline events are labelled OBSERVED, ASSUMED, MODELED, or UNKNOWN.",
        })
}

func handleScenarioMethodology(w http.ResponseWriter, r *http.Request, tenant sim.TenantID, id sim.ID) {
        s, err := scenarioSvc.GetScenario(r.Context(), tenant, id)
        if err != nil {
                writeSimError(w, 404, "not found")
                return
        }
        // Fetch the model so we can return its declared limitations (Gate E).
        limitations := s.Methodology.Limitations
        if m, err := scenarioSvc.Models.Get(r.Context(), s.ModelID); err == nil {
                // Merge scenario methodology limitations + model limitations.
                seen := map[string]bool{}
                for _, l := range limitations {
                        seen[l] = true
                }
                for _, l := range m.Limitations {
                        if !seen[l] {
                                limitations = append(limitations, l)
                        }
                }
        }
        writeJSON(w, 200, map[string]any{
                "scenario_id":   s.ID,
                "methodology":   s.Methodology,
                "model_id":      s.ModelID,
                "model_version": s.ModelVersion,
                "limitations":   limitations,
                "reality_layer": "HYPOTHETICAL",
                "disclaimer":    "This scenario is HYPOTHETICAL. Outputs are SIMULATED.",
        })
}

// makeScenarioCompareHandler handles POST /scenarios/compare.
func makeScenarioCompareHandler() http.HandlerFunc {
        return func(w http.ResponseWriter, r *http.Request) {
                w.Header().Set("Content-Type", "application/json")
                if r.Method != http.MethodPost {
                        writeSimError(w, http.StatusMethodNotAllowed, "method not allowed")
                        return
                }
                tenant := tenantFromRequest(r)
                var req struct {
                        ScenarioIDs []string `json:"scenario_ids"`
                }
                if err := decodeSimBody(r, &req); err != nil {
                        writeSimError(w, 400, "invalid body: "+err.Error())
                        return
                }
                ids := make([]sim.ID, len(req.ScenarioIDs))
                for i, s := range req.ScenarioIDs {
                        ids[i] = sim.ID(s)
                }
                cmp, err := scenarioSvc.CompareScenarios(r.Context(), tenant, ids)
                if err != nil {
                        writeSimError(w, 500, "compare failed: "+err.Error())
                        return
                }
                writeJSON(w, 200, cmp)
        }
}

// decodeSimBody decodes the JSON request body into v.
func decodeSimBody(r *http.Request, v any) error {
        defer r.Body.Close()
        body, err := io.ReadAll(r.Body)
        if err != nil {
                return err
        }
        if len(body) == 0 {
                return nil
        }
        return json.Unmarshal(body, v)
}

// writeSimError writes a scenario error response.
func writeSimError(w http.ResponseWriter, status int, msg string) {
        log.Printf("scenario API error: %d %s", status, msg)
        writeJSON(w, status, map[string]any{"error": msg})
}

// writeMethodNotAllowed writes a 405 response.
func writeMethodNotAllowed(w http.ResponseWriter) {
        writeSimError(w, http.StatusMethodNotAllowed, "method not allowed")
}

// randID returns a short random ID. Not cryptographically secure.
func randID(n int) string {
        const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
        out := make([]byte, n)
        now := time.Now().UnixNano()
        for i := range out {
                out[i] = charset[int(now>>uint(i*4))%len(charset)]
        }
        return string(out)
}

// countAssumptionEvidence counts the total evidence references across
// all assumptions.
func countAssumptionEvidence(asms []sim.ScenarioAssumption) int {
        n := 0
        for _, a := range asms {
                n += len(a.SourceEvidence)
        }
        return n
}
