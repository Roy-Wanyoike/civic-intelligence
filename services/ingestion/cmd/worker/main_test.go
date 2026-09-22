package main

import (
	"errors"
	"testing"

	"github.com/Roy-Wanyoike/civic-intelligence/adapters/registry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWorker_RegistryIsWired is the regression guard for issue #268: the
// Temporal worker used to construct its DiscoverActivity with
// `Registry: nil /* TODO: wire SourceRegistry */`. That made the worker
// compile but the discover activity panicked at the first call because it
// had no SourceRegistry to delegate to.
//
// This test constructs the worker exactly as main() does (via the extracted
// buildWorkflow helper) and verifies:
//  1. The Discover activity's Registry field is non-nil — i.e. a concrete
//     SourceRegistry has been wired in, not a nil placeholder.
//  2. The registry resolves the Kenya adapter ("KE") without error and
//     echoes back the canonical country code — proving the wiring reaches
//     the real adapters/registry package rather than a stub.
//  3. An unknown country code ("ZZ") returns an error wrapping
//     registry.ErrUnsupportedCountry — proving the registry genuinely
//     looks things up rather than always succeeding.
//  4. The startup log line's data source — registry.SupportedCountries() —
//     returns a non-empty list, so the worker's "registered N source
//     adapters" log line reports a non-zero count at boot.
//
// MustRegisterDefault is called defensively at the top of the test in case
// it runs before main() has had a chance to (it is idempotent — safe to call
// repeatedly).
func TestWorker_RegistryIsWired(t *testing.T) {
	// Register the default country adapters. Idempotent — if a previous test
	// (or main()) already registered them, this is a no-op.
	registry.MustRegisterDefault()

	wf := buildWorkflow()
	require.NotNil(t, wf, "buildWorkflow returned nil workflow — wiring helper broken")
	require.NotNil(t, wf.Discover, "workflow's Discover activity is nil")

	// 1. The Registry field must be non-nil — this is the literal fix for #268.
	require.NotNil(t, wf.Discover.Registry,
		"DiscoverActivity.Registry is nil — SourceRegistry not wired (issue #268 regression)")

	// 2. AdapterFor("KE") resolves the Kenya adapter without error and echoes
	//    back the canonical country code.
	code, err := wf.Discover.Registry.AdapterFor("KE")
	require.NoError(t, err,
		"AdapterFor(\"KE\") returned an error — Kenya adapter not registered")
	assert.Equal(t, "KE", code,
		"AdapterFor(\"KE\") returned %q, want \"KE\"", code)

	// 3. An unknown country code MUST error. Without this guard, a future
	//    regression that swaps the registry for a "always succeed" stub would
	//    slip through.
	_, err = wf.Discover.Registry.AdapterFor("ZZ")
	require.Error(t, err,
		"AdapterFor(\"ZZ\") returned nil error for unknown country code")
	assert.True(t, errors.Is(err, registry.ErrUnsupportedCountry),
		"AdapterFor(\"ZZ\") error should wrap registry.ErrUnsupportedCountry, got: %v", err)

	// 4. The worker's startup log line ("registered N source adapters")
	//    reads from registry.SupportedCountries(); assert it is non-empty so
	//    the boot log doesn't silently report "0".
	assert.Greater(t, len(registry.SupportedCountries()), 0,
		"registry.SupportedCountries() returned an empty list — adapters not registered")
}

// TestWorker_RegistryImplementsSourceRegistry is a compile-time-style test
// that the registrySourceRegistry value satisfies the temporal.SourceRegistry
// interface. The compile-time `var _ temporal.SourceRegistry =
// registrySourceRegistry{}` assertion in main.go catches signature drift
// at build time; this test catches it at unit-test time too, in case the
// package is built without running the compiler's interface check (e.g. a
// future split into a separate package).
func TestWorker_RegistryImplementsSourceRegistry(t *testing.T) {
	registry.MustRegisterDefault()

	var r interface{ AdapterFor(string) (string, error) } = registrySourceRegistry{}
	assert.NotNil(t, r, "registrySourceRegistry does not satisfy the SourceRegistry surface")

	code, err := r.AdapterFor("KE")
	require.NoError(t, err)
	assert.Equal(t, "KE", code)
}
