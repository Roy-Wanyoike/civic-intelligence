// Package contract — openapi_contract_test.go verifies every route
// registered in services/api/cmd/main.go has a corresponding entry in
// docs/api/openapi.yaml. This catches two classes of regression:
//
//  1. A new route is added to main.go but the developer forgets to
//     document it in the OpenAPI spec.
//  2. A route is removed from main.go but the documentation is left
//     dangling (this test does not catch the latter — openapi entries
//     with no implementation are surfaced by the e2e tests instead).
//
// The test reads main.go + openapi.yaml as text (no AST parsing, no YAML
// library dependency) so it can run in this minimal Go module that only
// depends on packages/contracts + packages/events. The route extraction
// uses a regex over the registered HandleFunc calls in main.go.
package contract

import (
        "os"
        "path/filepath"
        "regexp"
        "runtime"
        "strings"
        "testing"
)

// workspaceRoot returns the absolute path to the monorepo root (the
// directory containing services/api/cmd/main.go).
func workspaceRoot(t *testing.T) string {
        t.Helper()
        _, thisFile, _, ok := runtime.Caller(0)
        if !ok {
                t.Fatal("runtime.Caller failed")
        }
        dir := filepath.Dir(thisFile)
        for i := 0; i < 8; i++ {
                if _, err := os.Stat(filepath.Join(dir, "services", "api", "cmd", "main.go")); err == nil {
                        return dir
                }
                dir = filepath.Dir(dir)
        }
        t.Fatal("workspace root not found (services/api/cmd/main.go missing)")
        return ""
}

// routePattern matches HandleFunc / Handle calls inside main.go. It captures
// the first quoted string argument (the route path). Examples:
//
//      mux.HandleFunc("/api/v1/healthz", healthz)
//      apiHandler.HandleFunc("/api/v1/bills", makeBillsHandler(kenyaLaw))
//      apiHandler.Handle("/api/v1/questions", questionsHandler)
//      mux.Handle("/api/v1/", middleware.RequestID(nil)(authed))
var routePattern = regexp.MustCompile(`(?:HandleFunc|Handle)\("((?:/api/v1/|/metrics)[^"]*)"`)

// pathPattern matches top-level path keys in openapi.yaml. OpenAPI 3.x
// indents path keys exactly two spaces under the `paths:` mapping.
var pathPattern = regexp.MustCompile(`(?m)^  (/[^:]+):\s*$`)

// stripLineComments removes Go single-line comments (// to end of line) so
// the route regex doesn't pick up routes mentioned in doc comments. This
// is intentionally simple — it does not handle block comments (/* */) or
// strings containing "//" (those are vanishingly rare in route paths).
func stripLineComments(src string) string {
        var b strings.Builder
        for _, line := range strings.Split(src, "\n") {
                // Find the first "//" not inside a string literal. For route
                // registration code we never have "//" inside string literals, so
                // a simple strings.Index is sufficient.
                if idx := strings.Index(line, "//"); idx >= 0 {
                        line = line[:idx]
                }
                b.WriteString(line)
                b.WriteByte('\n')
        }
        return b.String()
}

// mainGoRoutes extracts every route registered in main.go. Returns the
// raw routes (with /api/v1 prefix and any trailing slash).
func mainGoRoutes(t *testing.T) []string {
        t.Helper()
        root := workspaceRoot(t)
        data, err := os.ReadFile(filepath.Join(root, "services", "api", "cmd", "main.go"))
        if err != nil {
                t.Fatalf("read main.go: %v", err)
        }
        src := stripLineComments(string(data))
        matches := routePattern.FindAllStringSubmatch(src, -1)
        routes := make([]string, 0, len(matches))
        seen := map[string]bool{}
        for _, m := range matches {
                if len(m) < 2 {
                        continue
                }
                r := m[1]
                if seen[r] {
                        continue
                }
                seen[r] = true
                routes = append(routes, r)
        }
        if len(routes) == 0 {
                t.Fatal("mainGoRoutes: extracted 0 routes from main.go — regex is wrong")
        }
        return routes
}

// openAPIPaths extracts every top-level path key from openapi.yaml.
func openAPIPaths(t *testing.T) map[string]bool {
        t.Helper()
        root := workspaceRoot(t)
        data, err := os.ReadFile(filepath.Join(root, "docs", "api", "openapi.yaml"))
        if err != nil {
                t.Fatalf("read openapi.yaml: %v", err)
        }
        paths := map[string]bool{}
        for _, m := range pathPattern.FindAllStringSubmatch(string(data), -1) {
                if len(m) >= 2 {
                        paths[m[1]] = true
                }
        }
        if len(paths) == 0 {
                t.Fatal("openAPIPaths: extracted 0 paths from openapi.yaml — regex is wrong")
        }
        return paths
}

// normalizeRoute converts a main.go route into the corresponding openapi
// path. Examples:
//
//      /api/v1/healthz            -> /healthz
//      /api/v1/bills              -> /bills
//      /api/v1/bills/             -> /bills/{id}        (sub-router)
//      /api/v1/scenarios/compare  -> /scenarios/compare
//      /api/v1/debt/governments/{id} (not present in main.go — openapi-style)
//
// Trailing-slash routes (sub-routers) are normalized to the /{id} form
// because the openapi spec uses path parameters for sub-resources.
func normalizeRoute(route string) string {
        // Strip /api/v1 prefix.
        r := strings.TrimPrefix(route, "/api/v1")
        if r == "" {
                r = "/"
        }
        // Trailing slash → /{id} (the sub-router pattern).
        if strings.HasSuffix(r, "/") && r != "/" {
                base := strings.TrimSuffix(r, "/")
                // Special-case routes that should NOT be normalized to /{id}
                // because they have explicit sub-resource patterns in openapi.
                // These sub-routers serve multiple paths; the contract is that
                // AT LEAST ONE path under the base exists in openapi.
                switch base {
                case "/bills", "/acts", "/scenarios", "/governments",
                        "/subscriptions", "/notifications", "/sources", "/corrections",
                        "/debt", "/terminology", "/people", "/committees", "/institutions",
                        "/loans", "/grants", "/provenance", "/evidence", "/claims",
                        "/constitution/articles", "/graph":
                        // The sub-router serves {id}-style sub-resources. Return the
                        // base so the caller can verify "at least one path starting
                        // with base/" exists in openapi.
                        return base + "/"
                }
                return base + "/{id}"
        }
        return r
}

// routeIsDocumented returns true if the normalized route has a matching
// entry in openapiPaths. For trailing-slash sub-routers, "matching" means
// at least one openapi path starts with the base + "/".
func routeIsDocumented(normalized string, openapiPaths map[string]bool) bool {
        if openapiPaths[normalized] {
                return true
        }
        // Sub-router: check that at least one path starts with the base.
        if strings.HasSuffix(normalized, "/") {
                base := strings.TrimSuffix(normalized, "/")
                for p := range openapiPaths {
                        if strings.HasPrefix(p, base+"/") {
                                return true
                        }
                }
        }
        return false
}

// intentionallyUndocumented is the small allowlist of routes that are
// registered in main.go but intentionally NOT in the public OpenAPI spec.
// These are operational / infrastructural endpoints (Prometheus metrics,
// health/readiness probes are documented; the rest are infrastructural).
var intentionallyUndocumented = map[string]string{
        "/metrics": "Prometheus metrics endpoint — operational, not part of the public API contract",
}

// TestOpenAPI_AllMainGoRoutesDocumented verifies every route registered in
// main.go has a corresponding entry in openapi.yaml. The test fails on
// the first undocumented route, listing the missing path so the developer
// knows exactly what to add.
func TestOpenAPI_AllMainGoRoutesDocumented(t *testing.T) {
        routes := mainGoRoutes(t)
        paths := openAPIPaths(t)

        missing := []string{}
        for _, r := range routes {
                if reason, ok := intentionallyUndocumented[r]; ok {
                        t.Logf("route %q intentionally undocumented: %s", r, reason)
                        continue
                }
                normalized := normalizeRoute(r)
                if !routeIsDocumented(normalized, paths) {
                        missing = append(missing, r+" (normalized: "+normalized+")")
                }
        }
        if len(missing) > 0 {
                t.Errorf("the following routes are registered in main.go but missing from "+
                        "docs/api/openapi.yaml (%d routes checked, %d missing):\n  %s",
                        len(routes), len(missing), strings.Join(missing, "\n  "))
        }
}

// TestOpenAPI_MainGoHasExpectedRouteCount is a sanity check: main.go
// should register at least 40 routes (we currently register ~50). If this
// number drops significantly, the routePattern regex may have drifted
// (e.g. main.go was refactored to use a different mux).
func TestOpenAPI_MainGoHasExpectedRouteCount(t *testing.T) {
        routes := mainGoRoutes(t)
        if len(routes) < 40 {
                t.Errorf("main.go route count dropped to %d — expected at least 40. "+
                        "routes extracted:\n  %s", len(routes), strings.Join(routes, "\n  "))
        }
        t.Logf("main.go route count: %d", len(routes))
}

// TestOpenAPI_OpenAPIHasExpectedPathCount is a sanity check: openapi.yaml
// should document at least 60 paths.
func TestOpenAPI_OpenAPIHasExpectedPathCount(t *testing.T) {
        paths := openAPIPaths(t)
        if len(paths) < 60 {
                t.Errorf("openapi.yaml path count dropped to %d — expected at least 60", len(paths))
        }
        t.Logf("openapi.yaml path count: %d", len(paths))
}

// TestOpenAPI_NoDuplicateMainGoRoutes verifies no route is registered
// twice in main.go. Duplicate registrations would silently overwrite each
// other (last-wins in net/http.ServeMux).
func TestOpenAPI_NoDuplicateMainGoRoutes(t *testing.T) {
        data, err := os.ReadFile(filepath.Join(workspaceRoot(t), "services", "api", "cmd", "main.go"))
        if err != nil {
                t.Fatalf("read main.go: %v", err)
        }
        src := stripLineComments(string(data))
        matches := routePattern.FindAllStringSubmatch(src, -1)
        seen := map[string]int{}
        for _, m := range matches {
                if len(m) >= 2 {
                        seen[m[1]]++
                }
        }
        for r, n := range seen {
                if n > 1 {
                        t.Errorf("route %q is registered %d times in main.go — duplicate registration", r, n)
                }
        }
}

// TestOpenAPI_HealthzAndReadyAreDocumented verifies the operational
// endpoints /healthz and /readyz are documented in openapi.yaml. These
// are the only operational endpoints that SHOULD be in the public spec
// (they are used by orchestrators + load balancers).
func TestOpenAPI_HealthzAndReadyAreDocumented(t *testing.T) {
        paths := openAPIPaths(t)
        for _, p := range []string{"/healthz", "/readyz"} {
                if !paths[p] {
                        t.Errorf("openapi.yaml is missing required operational path %q", p)
                }
        }
}
