// Package integration contains black-box integration tests for the Civic
// Intelligence API. The harness in this file builds the API server binary
// once, starts it on a random port, waits for /healthz to flip green, and
// exposes the base URL to every test in the package via the package-level
// serverURL variable.
//
// Each test file (api_bills_test.go, api_scenarios_test.go, ...) then issues
// real HTTP requests against serverURL and asserts on JSON response shape +
// status codes. The server is shared across all tests in the package for
// speed — building + starting the binary takes ~3s, so amortising it across
// the whole suite keeps `go test` under 10s end-to-end.
package integration

import (
        "context"
        "encoding/json"
        "fmt"
        "io"
        "net"
        "net/http"
        "os"
        "os/exec"
        "path/filepath"
        "runtime"
        "strconv"
        "strings"
        "testing"
        "time"
)

// serverURL is the base URL of the running API server (e.g.
// "http://127.0.0.1:33127"). It is populated by TestMain after the server
// has accepted its first /healthz request. Tests should call apiURL(path)
// to build a full URL from a route relative to /api/v1.
var serverURL string

// apiURL builds a full URL for a path relative to /api/v1. Examples:
//
//      apiURL("/bills")           -> http://127.0.0.1:PORT/api/v1/bills
//      apiURL("/governments/x")   -> http://127.0.0.1:PORT/api/v1/governments/x
func apiURL(path string) string {
        if !strings.HasPrefix(path, "/") {
                path = "/" + path
        }
        return serverURL + "/api/v1" + path
}

// rootURL returns the base URL without the /api/v1 prefix — used for
// /healthz and /metrics which sit on the outer mux.
func rootURL(path string) string {
        if !strings.HasPrefix(path, "/") {
                path = "/" + path
        }
        return serverURL + path
}

// workspaceRoot returns the absolute path to the monorepo root (the
// directory containing services/api/cmd/main.go). It walks up from this
// test file's location until it finds the marker file.
func workspaceRoot() (string, error) {
        _, thisFile, _, ok := runtime.Caller(0)
        if !ok {
                return "", fmt.Errorf("runtime.Caller failed")
        }
        dir := filepath.Dir(thisFile)
        for i := 0; i < 8; i++ {
                if _, err := os.Stat(filepath.Join(dir, "services", "api", "cmd", "main.go")); err == nil {
                        return dir, nil
                }
                dir = filepath.Dir(dir)
        }
        return "", fmt.Errorf("workspace root not found (services/api/cmd/main.go not located)")
}

// freePort asks the kernel for a free TCP port by opening a listener on
// :0 and returning its address. The listener is closed immediately so the
// port can be reused by the server subprocess.
func freePort() (int, error) {
        l, err := net.Listen("tcp", "127.0.0.1:0")
        if err != nil {
                return 0, err
        }
        defer l.Close()
        return l.Addr().(*net.TCPAddr).Port, nil
}

// waitForReady polls /healthz until it returns 200 or the timeout elapses.
// The API server starts in <1s once the binary is built; we allow 60s to
// accommodate the first build (which downloads + compiles deps).
func waitForReady(baseURL string, timeout time.Duration, stderr *strings.Builder) error {
        deadline := time.Now().Add(timeout)
        client := &http.Client{Timeout: 2 * time.Second}
        var lastErr error
        for time.Now().Before(deadline) {
                resp, err := client.Get(baseURL + "/api/v1/healthz")
                if err == nil {
                        body, _ := io.ReadAll(resp.Body)
                        resp.Body.Close()
                        if resp.StatusCode == 200 && strings.Contains(string(body), "ok") {
                                return nil
                        }
                        lastErr = fmt.Errorf("healthz returned %d: %s", resp.StatusCode, string(body))
                } else {
                        lastErr = err
                }
                time.Sleep(200 * time.Millisecond)
        }
        if stderr != nil {
                return fmt.Errorf("server not ready within %s: last error: %v\nserver stderr:\n%s",
                        timeout, lastErr, stderr.String())
        }
        return fmt.Errorf("server not ready within %s: last error: %v", timeout, lastErr)
}

// TestMain builds + starts the API server once for the whole package, runs
// all tests, then tears the server down. Sharing the server across tests is
// ~10x faster than starting per-test because the binary build cost is
// amortised and the in-memory stores (actRepo, debtRepo, scenarioSvc) are
// already seeded when the server starts.
func TestMain(m *testing.M) {
        root, err := workspaceRoot()
        if err != nil {
                fmt.Fprintf(os.Stderr, "integration TestMain: %v\n", err)
                os.Exit(1)
        }

        port, err := freePort()
        if err != nil {
                fmt.Fprintf(os.Stderr, "integration TestMain: free port: %v\n", err)
                os.Exit(1)
        }

        // Build the API server binary from its own module (services/api has
        // its own go.mod, so the build must run from that directory rather
        // than from the workspace root).
        bin := filepath.Join(os.TempDir(), "civic-api-integration")
        build := exec.Command("go", "build", "-o", bin, "./cmd")
        build.Dir = filepath.Join(root, "services", "api")
        build.Stdout = os.Stdout
        build.Stderr = os.Stderr
        if err := build.Run(); err != nil {
                fmt.Fprintf(os.Stderr, "integration TestMain: go build services/api: %v\n", err)
                os.Exit(1)
        }

        cmd := exec.Command(bin)
        // env: dev mode (no signature check), empty AI URL (handlers fall back
        // to structured stubs so tests are deterministic without a Python sidecar).
        cmd.Env = append(os.Environ(),
                "DEV_MODE=true",
                "AI_SERVICE_URL=",
                "API_SERVICE_ADDR=127.0.0.1:"+strconv.Itoa(port),
        )
        var stderr strings.Builder
        cmd.Stderr = &stderr
        if err := cmd.Start(); err != nil {
                fmt.Fprintf(os.Stderr, "integration TestMain: start: %v\n", err)
                os.Exit(1)
        }

        serverURL = fmt.Sprintf("http://127.0.0.1:%d", port)
        if err := waitForReady(serverURL, 60*time.Second, &stderr); err != nil {
                _ = cmd.Process.Kill()
                fmt.Fprintf(os.Stderr, "integration TestMain: %v\n", err)
                os.Exit(1)
        }

        // Run the test suite.
        code := m.Run()

        // Tear down. SIGINT triggers the server's graceful shutdown path which
        // drains in-flight requests (up to ShutdownTimeout=15s).
        _ = cmd.Process.Signal(os.Interrupt)
        done := make(chan struct{})
        go func() {
                _ = cmd.Wait()
                close(done)
        }()
        select {
        case <-done:
        case <-time.After(15 * time.Second):
                _ = cmd.Process.Kill()
                <-done
        }

        // If any test failed, dump the server stderr to aid debugging.
        if code != 0 {
                fmt.Fprintf(os.Stderr, "\n--- server stderr ---\n%s\n--- end server stderr ---\n", stderr.String())
        }

        os.Exit(code)
}

// doJSON issues an HTTP request and decodes the JSON body into out. Returns
// the status code so callers can assert on it independently. The decoded
// body is best-effort — if the body is not JSON (or empty), out is left
// untouched and no error is returned.
func doJSON(t *testing.T, method, url string, body io.Reader, out any) int {
        t.Helper()
        req, err := http.NewRequestWithContext(context.Background(), method, url, body)
        if err != nil {
                t.Fatalf("new request %s %s: %v", method, url, err)
        }
        if body != nil {
                req.Header.Set("Content-Type", "application/json")
        }
        resp, err := http.DefaultClient.Do(req)
        if err != nil {
                t.Fatalf("do %s %s: %v", method, url, err)
        }
        defer resp.Body.Close()
        raw, _ := io.ReadAll(resp.Body)
        if out != nil && len(raw) > 0 {
                _ = json.Unmarshal(raw, out)
        }
        return resp.StatusCode
}

// assertStatus fails the test if the actual status code doesn't match expected.
func assertStatus(t *testing.T, url string, expected, actual int) {
        t.Helper()
        if actual != expected {
                t.Errorf("%s: expected status %d, got %d", url, expected, actual)
        }
}

// mustGet issues a GET and returns the status code + decoded body. Fails the
// test immediately if the request itself errors (network failure).
func mustGet(t *testing.T, url string, out any) int {
        t.Helper()
        return doJSON(t, http.MethodGet, url, nil, out)
}

// mustPost issues a POST with the given JSON body and returns the status
// code + decoded response.
func mustPost(t *testing.T, url, body string, out any) int {
        t.Helper()
        return doJSON(t, http.MethodPost, url, strings.NewReader(body), out)
}
