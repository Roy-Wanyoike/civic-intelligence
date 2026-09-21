// Package gazette — tests for the gazette index parser + DiscoverNotices.
package gazette

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// loadFixture reads a testdata/ fixture into a string.
func loadFixture(t *testing.T, name string) string {
	t.Helper()
	path := filepath.Join("testdata", name)
	data, err := os.ReadFile(path)
	require.NoError(t, err, "fixture %s not found under testdata/", name)
	return string(data)
}

// TestParseGazetteIndex_ExtractsAllIssuePDFs verifies the parser extracts
// every PDF link from the gazette index page and ignores non-PDF links.
func TestParseGazetteIndex_ExtractsAllIssuePDFs(t *testing.T) {
	html := loadFixture(t, "gazette_index.html")
	entries, hasNext := ParseGazetteIndex(html)

	// 4 PDFs in the archive + 1 in the "Latest" sidebar = 5 total. The
	// "About" link must be filtered out.
	require.Len(t, entries, 5, "expected 5 gazette PDF links (4 archive + 1 sidebar)")

	// Verify URL resolution (relative URLs → absolute).
	for _, e := range entries {
		assert.True(t, strings.HasPrefix(e.URL, "https://new.kenyalaw.org/files/gazette/"),
			"URL should be resolved to absolute: %s", e.URL)
	}

	// First entry should be the latest issue with the highest notice number.
	assert.Contains(t, entries[0].URL, "Vol_CXXVIII-No.115.pdf")

	// hasNext should be true because the fixture has a "next page" link.
	assert.True(t, hasNext, "fixture has a pager__item--next — parser should report hasNext=true")
}

// TestParseGazetteIndex_FiltersNonPDFLinks verifies that non-PDF links
// (like an "About" link) are filtered out by the parser.
func TestParseGazetteIndex_FiltersNonPDFLinks(t *testing.T) {
	html := loadFixture(t, "gazette_index.html")
	entries, _ := ParseGazetteIndex(html)

	for _, e := range entries {
		assert.True(t, strings.HasSuffix(strings.ToLower(e.URL), ".pdf"),
			"non-PDF link should have been filtered out: %s", e.URL)
	}
}

// TestNewAdapter_Defaults verifies that NewAdapter fills in defaults.
func TestNewAdapter_Defaults(t *testing.T) {
	a := NewAdapter(nil, "")
	assert.NotNil(t, a.client, "client should default to a non-nil http.Client")
	assert.Equal(t, defaultUserAgent, a.userAgent, "userAgent should default to defaultUserAgent")
	assert.Equal(t, gazetteIndexURL, a.indexURL, "indexURL should default to gazetteIndexURL")
	assert.Equal(t, gazetteMaxPages, a.maxPages, "maxPages should default to gazetteMaxPages")
}

// TestDiscoverNotices_MockServer verifies the full DiscoverNotices flow
// against a local mock server that returns the test fixture. This is
// the closest we can get to a live-site test without hitting kenyalaw.org
// from CI.
func TestDiscoverNotices_MockServer(t *testing.T) {
	html := loadFixture(t, "gazette_index.html")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(html))
	}))
	defer srv.Close()

	a := NewAdapter(http.DefaultClient, "test-agent")
	a.indexURL = srv.URL + "/gazette/"

	entries, err := a.DiscoverNotices(context.Background())
	require.NoError(t, err)

	// The fixture has 5 PDF links (4 archive + 1 sidebar dup of #115).
	// After dedup the result is 4 unique URLs.
	require.Len(t, entries, 4, "expected 4 unique gazette entries (1 sidebar dup removed)")

	// Verify no duplicate URLs.
	seen := map[string]struct{}{}
	for _, e := range entries {
		_, dup := seen[e.URL]
		assert.False(t, dup, "duplicate URL in DiscoverNotices output: %s", e.URL)
		seen[e.URL] = struct{}{}
	}
}

// TestDiscover_ContractProjection verifies that Discover projects the
// GazetteEntry slice into contracts.SourceItem with the expected fields
// (URL, DocumentType, SourceType, CountryCode, SourceID, Metadata).
func TestDiscover_ContractProjection(t *testing.T) {
	html := loadFixture(t, "gazette_index.html")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(html))
	}))
	defer srv.Close()

	a := NewAdapter(http.DefaultClient, "test-agent")
	a.indexURL = srv.URL + "/gazette/"

	items, err := a.Discover(context.Background())
	require.NoError(t, err)
	require.NotEmpty(t, items)

	first := items[0]
	assert.Equal(t, "KE", first.CountryCode)
	assert.Equal(t, "gazette_notice", first.DocumentType)
	assert.NotEmpty(t, first.URL)
	assert.NotEmpty(t, first.SourceID)
	assert.Contains(t, first.SourceID, "ke-kenyalaw-gazette-")
	assert.Contains(t, first.Metadata, "volume")
}
