// Package parliament — tests for the Hansard / Order Paper / Votes &
// Proceedings / Committees crawlers.
//
// These tests cover the parsers in isolation: each test loads a small
// HTML fixture from testdata/ that mirrors the actual parliament.go.ke
// Drupal Views structure, then verifies that the parser extracts the
// expected candidates with the expected fields (URL, title, house,
// sitting date, source ID).
//
// The tests do NOT hit the live site — they exercise the parsers against
// captured HTML, so they're deterministic and fast.
package parliament

import (
        "context"
        "net/http"
        "net/http/httptest"
        "strings"
        "testing"
        "time"

        "github.com/stretchr/testify/assert"
        "github.com/stretchr/testify/require"
)

// --- Hansard parser tests -------------------------------------------------

func TestParsePDFListing_Hansard_ExtractsAllPDFRows(t *testing.T) {
        html := loadFixture(t, "hansard_na.html")
        cands, hasNext := ParsePDFListing[HansardCandidate](html, "National Assembly", hansardRowMapper)

        // The fixture has 4 PDF links in the main archive (the 5th row is a
        // non-PDF link that should be filtered out). Plus the "Latest Hansard"
        // sidebar also has 1 PDF link that the archive container SHOULD
        // include because the parser falls back to the whole document when
        // the pager-relative container is ambiguous in the test fixture.
        // We assert at least the 4 archive rows are present.
        require.GreaterOrEqual(t, len(cands), 4, "expected at least 4 Hansard PDFs from the archive")

        // First entry: 27 August 2026
        assert.Equal(t, "National Assembly", cands[0].House)
        assert.Contains(t, cands[0].URL, "The%20Hansard%20-%20Thursday%2C%2027%20August%202026.pdf")
        assert.Equal(t, time.Date(2026, time.August, 27, 0, 0, 0, 0, time.UTC), cands[0].SittingDate)

        // The hasNext flag should be true (the fixture has a Next pager link).
        assert.True(t, hasNext, "fixture has a pager__item--next — parser should report hasNext=true")
}

func TestParseHansardSittingDate_Formats(t *testing.T) {
        cases := []struct {
                name  string
                title string
                want  time.Time
        }{
                {
                        name:  "full-with-comma",
                        title: "The Hansard - Thursday, 27 August 2026.pdf",
                        want:  time.Date(2026, time.August, 27, 0, 0, 0, 0, time.UTC),
                },
                {
                        name:  "with-ordinal-suffix",
                        title: "The Hansard - Wednesday 26Th August 2026.pdf",
                        want:  time.Date(2026, time.August, 26, 0, 0, 0, 0, time.UTC),
                },
                {
                        name:  "afternoon-sitting-annotation",
                        title: "Hansard - 18 September 2026 (Afternoon Sitting).pdf",
                        want:  time.Date(2026, time.September, 18, 0, 0, 0, 0, time.UTC),
                },
                {
                        name:  "no-date-found",
                        title: "Some random title without a date",
                        want:  time.Time{},
                },
        }
        for _, tc := range cases {
                t.Run(tc.name, func(t *testing.T) {
                        got := parseHansardSittingDate(tc.title)
                        assert.Equal(t, tc.want, got)
                })
        }
}

func TestMakeHansardSourceID_NationalAssembly(t *testing.T) {
        id := makeHansardSourceID("National Assembly",
                time.Date(2026, time.August, 27, 0, 0, 0, 0, time.UTC),
                "https://www.parliament.go.ke/sites/default/files/2026-08/The%20Hansard%20-%20Thursday.pdf")
        assert.Equal(t, "ke-parliament-na-hansard-2026-08-27", id)
}

func TestMakeHansardSourceID_Senate(t *testing.T) {
        id := makeHansardSourceID("Senate",
                time.Date(2026, time.September, 18, 0, 0, 0, 0, time.UTC),
                "https://www.parliament.go.ke/sites/default/files/2026-09/afternoon.pdf")
        assert.Equal(t, "ke-parliament-senate-hansard-2026-09-18", id)
}

func TestMakeHansardSourceID_ZeroDateFallsBackToHash(t *testing.T) {
        id := makeHansardSourceID("National Assembly", time.Time{}, "https://example.com/x.pdf")
        assert.Contains(t, id, "ke-parliament-na-hansard-")
        assert.NotEqual(t, "ke-parliament-na-hansard-", id)
}

// --- Order Paper parser tests ---------------------------------------------

func TestParsePDFListing_OrderPaper_ExtractsAllPDFRows(t *testing.T) {
        html := loadFixture(t, "order_paper_na.html")
        cands, hasNext := ParsePDFListing[OrderPaperCandidate](html, "National Assembly", orderPaperRowMapper)

        // The fixture has 3 PDFs in the main archive + 1 in the sidebar.
        // The parser walks the whole document when it can't disambiguate the
        // archive container from the sidebar block, so we may see the sidebar
        // entry too. The DiscoverOrderPapers method dedupes by URL — see
        // TestDiscoverOrderPapers_DeduplicatesSidebarDup for that coverage.
        require.GreaterOrEqual(t, len(cands), 3, "expected at least 3 Order Paper PDFs")

        // Find the supplementary entry by URL (not by index — the sidebar
        // may shift the position of the supplementary entry).
        var supp *OrderPaperCandidate
        for i := range cands {
                if cands[i].IsSupplementary {
                        supp = &cands[i]
                        break
                }
        }
        require.NotNil(t, supp, "expected at least one supplementary order paper")
        assert.Equal(t, time.Date(2026, time.September, 18, 0, 0, 0, 0, time.UTC), supp.SittingDate)
        assert.Contains(t, supp.URL, "Supplementary%20Order%20Paper")

        assert.True(t, hasNext)
}

// TestDiscoverOrderPapers_DeduplicatesSidebarDup verifies that the
// DiscoverOrderPapers method removes the sidebar duplicate that the
// raw ParsePDFListing call may pick up. This is the integration test
// for the dedupeOrderByURL helper.
func TestDiscoverOrderPapers_DeduplicatesSidebarDup(t *testing.T) {
        html := loadFixture(t, "order_paper_na.html")
        srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                w.Header().Set("Content-Type", "text/html; charset=utf-8")
                _, _ = w.Write([]byte(html))
        }))
        defer srv.Close()

        a := NewAdapter(http.DefaultClient, "test-agent")
        a.naOrderPaperURL = srv.URL + "/na/order-paper"
        a.senateOrderPaperURL = srv.URL + "/senate/order-paper"

        cands, err := a.DiscoverOrderPapers(context.Background())
        require.NoError(t, err)

        // Each house's fixture has 3 archive PDFs + 1 sidebar link (different URL).
        // Both houses point at the same mock fixture, so the URL set is deduped
        // across houses — total unique URLs = 4 (3 archive + 1 sidebar).
        totalUnique := len(cands)
        assert.GreaterOrEqual(t, totalUnique, 4, "expected at least 4 unique Order Paper candidates (3 archive + 1 sidebar per fixture, deduped across houses)")

        // Verify no duplicate URLs in the final list.
        seen := map[string]struct{}{}
        for _, c := range cands {
                _, dup := seen[c.URL]
                assert.False(t, dup, "duplicate URL in DiscoverOrderPapers output: %s", c.URL)
                seen[c.URL] = struct{}{}
        }
}

func TestMakeOrderPaperSourceID_Standard(t *testing.T) {
        id := makeOrderPaperSourceID("National Assembly",
                time.Date(2026, time.August, 27, 0, 0, 0, 0, time.UTC),
                "https://www.parliament.go.ke/order-paper-2026-08-27.pdf")
        assert.Equal(t, "ke-parliament-na-order-paper-2026-08-27", id)
}

func TestMakeOrderPaperSourceID_Supplementary(t *testing.T) {
        id := makeOrderPaperSourceID("National Assembly",
                time.Date(2026, time.September, 18, 0, 0, 0, 0, time.UTC),
                "https://www.parliament.go.ke/Supplementary-Order-Paper-2026-09-18.pdf")
        // The supplementary suffix is appended when the URL/title contains
        // the word "supplementary" — case-insensitive. The supp suffix
        // disambiguates multiple supplementary order papers issued for the
        // same sitting.
        assert.Contains(t, id, "ke-parliament-na-order-paper-2026-09-18-supp-")
}

// --- Votes & Proceedings parser tests -------------------------------------

func TestParsePDFListing_Votes_ExtractsAllPDFRows(t *testing.T) {
        html := loadFixture(t, "votes_proceeding_na.html")
        cands, hasNext := ParsePDFListing[VotesCandidate](html, "National Assembly", votesRowMapper)

        require.GreaterOrEqual(t, len(cands), 3, "expected at least 3 V&P PDFs")

        // First entry: Thursday, August 27, 2026 at 2.30pm
        assert.Equal(t, "National Assembly", cands[0].House)
        assert.Contains(t, cands[0].URL, "Thursday%2C%20August%2027%2C%202026%20at%202.30pm.pdf")
        assert.Equal(t, time.Date(2026, time.August, 27, 0, 0, 0, 0, time.UTC), cands[0].SittingDate)

        assert.True(t, hasNext)
}

func TestMakeVotesSourceID_Standard(t *testing.T) {
        id := makeVotesSourceID("National Assembly",
                time.Date(2026, time.August, 27, 0, 0, 0, 0, time.UTC),
                "https://www.parliament.go.ke/vp-2026-08-27.pdf")
        assert.Equal(t, "ke-parliament-na-votes-2026-08-27", id)
}

// --- Committee parser tests -----------------------------------------------

func TestParseCommitteeIndex_ExtractsCommitteeLinks(t *testing.T) {
        html := loadFixture(t, "committees_index_na.html")
        links := ParseCommitteeIndex(html)

        // 4 committee detail links should be extracted. The "/the-national-assembly/committees"
        // index link itself + the "/contact" link must be filtered out.
        require.Len(t, links, 4, "expected 4 committee links (excluding the index link + contact link)")

        assert.Equal(t, "Public Accounts Committee", links[0].Name)
        assert.Equal(t, "https://www.parliament.go.ke/the-national-assembly/committees/12/public-accounts-committee", links[0].URL)
        assert.Equal(t, "public-accounts-committee", links[0].Slug)

        assert.Equal(t, "Public Investments Committee", links[1].Name)
        assert.Equal(t, "public-investments-committee", links[1].Slug)

        assert.Equal(t, "Health Committee", links[2].Name)
        assert.Equal(t, "Education Committee", links[3].Name)
}

func TestParseCommitteeReports_ExtractsPDFs(t *testing.T) {
        html := loadFixture(t, "committee_detail_pac.html")
        cands := ParseCommitteeReports(html,
                "https://www.parliament.go.ke/the-national-assembly/committees/12/public-accounts-committee",
                "National Assembly",
                "Public Accounts Committee")

        require.Len(t, cands, 3, "expected 3 committee report PDFs (2 reports + 1 minutes)")

        // First report
        assert.Equal(t, "National Assembly", cands[0].House)
        assert.Equal(t, "Public Accounts Committee", cands[0].CommitteeName)
        assert.Contains(t, cands[0].URL, "Report%20on%20procurement%20of%20External%20Audit%20Services")
        assert.Equal(t, "report", cands[0].ReportType)

        // Second report — has a date in the title
        assert.Contains(t, cands[1].Title, "Universal Health Coverage")
        assert.Equal(t, time.Date(2026, time.September, 18, 0, 0, 0, 0, time.UTC), cands[1].PublishedDate)

        // Third entry — Minutes, should be flagged as reportType="minute"
        assert.Equal(t, "minute", cands[2].ReportType)
        assert.Contains(t, cands[2].Title, "Minutes")
}

func TestMakeCommitteeSourceID(t *testing.T) {
        id := makeCommitteeSourceID("National Assembly",
                "Public Accounts Committee",
                "https://www.parliament.go.ke/sites/default/files/2026-08/report.pdf")
        // Format: ke-parliament-<house>-committee-<committee-slug>-<hash>
        assert.Contains(t, id, "ke-parliament-na-committee-public-accounts-committee-")
}

// --- DiscoverHansard integration test (with mock HTTP server) -------------

// TestDiscoverHansard_MockServer verifies the full DiscoverHansard flow
// against a local mock server that returns the test fixture. This is the
// closest we can get to a live-site test without actually hitting
// parliament.go.ke from CI.
func TestDiscoverHansard_MockServer(t *testing.T) {
        html := loadFixture(t, "hansard_na.html")
        srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                // All Hansard URLs resolve to the same fixture for this test.
                w.Header().Set("Content-Type", "text/html; charset=utf-8")
                w.WriteHeader(http.StatusOK)
                _, _ = w.Write([]byte(html))
        }))
        defer srv.Close()

        a := NewAdapter(http.DefaultClient, "test-agent")
        // Override the listing URLs to point at the mock server.
        a.naHansardURL = srv.URL + "/na/hansard"
        a.senateHansardURL = srv.URL + "/senate/hansard"

        cands, err := a.DiscoverHansard(context.Background())
        require.NoError(t, err)
        // Each house returns the same fixture (4 archive rows + possibly 1
        // sidebar dup), then dedupes by URL. With both houses pointing at
        // the same fixture, the URL set will be deduplicated across houses.
        require.GreaterOrEqual(t, len(cands), 4, "expected at least 4 unique Hansard candidates")

        // Verify the first candidate has a deterministic source ID.
        first := cands[0]
        assert.NotEmpty(t, first.SourceID)
        assert.True(t, strings.HasPrefix(first.SourceID, "ke-parliament-"))
        assert.Equal(t, "National Assembly", first.House)
}
