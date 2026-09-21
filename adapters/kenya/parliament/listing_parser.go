// Package parliament — shared helpers for the Hansard / Order Paper / Votes
// & Proceedings crawlers.
//
// All four document-type listing pages on parliament.go.ke use the same
// Drupal Views structure: a paginated table whose rows contain a single
// <a href="...pdf"> link to the document. The ParsePDFListing helper below
// extracts those links generically and delegates the per-type shaping
// (title parsing, sitting-date inference, etc.) to a caller-supplied
// mapper function.
//
// This file ALSO defines the small regex / hash helpers used by the
// individual adapters so that they don't each need to import regexp,
// crypto/sha256, encoding/hex, strconv. Keeping the import surface small
// per file makes the adapter files easier to read in isolation.
package parliament

import (
        "context"
        "crypto/sha256"
        "encoding/hex"
        "fmt"
        "regexp"
        "strings"

        "golang.org/x/net/html"
)

// mustCompile is a thin wrapper around regexp.MustCompile that panics on a
// compile error. Used at package initialisation time for the date / ordinal
// regexps — a panic at init is the right behaviour because a broken regexp
// is a programming bug, not a runtime error.
func mustCompile(pattern string) *regexp.Regexp {
        return regexp.MustCompile(pattern)
}

// sha256Sum returns the SHA-256 hash of b.
func sha256Sum(b []byte) [32]byte {
        return sha256.Sum256(b)
}

// hexEncode returns the lowercase hex representation of b.
func hexEncode(b []byte) string {
        return hex.EncodeToString(b)
}

// pdfRowMapper is the per-document-type mapper that ParsePDFListing calls
// for each PDF link it finds in the listing table. The mapper receives the
// absolute href, the link's text content, and the house ("National Assembly"
// or "Senate"). It returns the candidate-shaped value and a boolean
// indicating whether the row should be included (false = skip — used by
// mappers that want to filter out non-PDF links or unrelated rows).
//
// The mapper is generic so the same parser code can produce HansardCandidate,
// OrderPaperCandidate, VotesCandidate, or CommitteeCandidate values from
// the same HTML structure.
type pdfRowMapper[T any] func(href, linkText, house string) (T, bool)

// discoverPDFListing is the shared pagination loop used by the Hansard,
// Order Paper, and Votes & Proceedings crawlers. It walks the listing URL
// up to maxPages pages deep (?page=N, 0-indexed) and returns the discovered
// candidates via the supplied mapper. The crawl stops early when a page
// returns zero candidates OR when the response HTML no longer contains a
// "next page" link in the pager.
//
// The function is generic so it can return []HansardCandidate,
// []OrderPaperCandidate, or []VotesCandidate from the same code path. It is
// a free function (not a method on *Adapter) because Go does not allow
// generic methods — the caller passes the adapter so the function can
// reach the polite HTTP client.
func discoverPDFListing[T any](a *Adapter, ctx context.Context, listingURL, house string, maxPages int, mapper pdfRowMapper[T]) ([]T, error) {
        var out []T
        for page := 0; page < maxPages || maxPages < 0; page++ {
                pageURL := listingURL
                if page > 0 {
                        sep := "?"
                        if strings.Contains(listingURL, "?") {
                                sep = "&"
                        }
                        pageURL = listingURL + sep + "page=" + itoa(page)
                }
                html_body, err := a.fetchURL(ctx, pageURL, "pdf_listing")
                if err != nil {
                        return out, fmt.Errorf("listing %s page %d: %w", listingURL, page, err)
                }
                cands, hasNext := ParsePDFListing[T](html_body, house, mapper)
                out = append(out, cands...)
                if !hasNext {
                        break
                }
        }
        return out, nil
}

// ParsePDFListing walks an HTML listing page (Hansard, Order Paper, or
// Votes & Proceedings — they all share the same Drupal Views table structure)
// and returns one candidate per PDF link found in the main archive table.
//
// The "main archive" table is the one that owns the pager (nav.pager). Each
// row in that table contains a single PDF link inside
//
//      <td class="views-field-field-pdf">
//        <span class="file--application-pdf">
//          <a href=".../sites/default/files/YYYY-MM/<name>.pdf">…</a>
//        </span>
//      </td>
//
// Other tables on the page (a "latest featured" block + a sidebar Hansard
// block) are skipped — they would otherwise re-list the most recent sitting
// and pollute the discovery output.
//
// The second return value is true when the page contains a "next page"
// link in the pager. Callers use this to decide whether to walk to ?page=N+1.
//
// The mapper is called for every PDF link in the archive table. If the
// mapper returns ok=false, the row is silently skipped (used to filter
// out non-PDF links or unrelated rows).
func ParsePDFListing[T any](htmlBody, house string, mapper pdfRowMapper[T]) (out []T, hasNext bool) {
        doc, err := html.Parse(strings.NewReader(htmlBody))
        if err != nil {
                return nil, false
        }
        // Find the archive container: the closest ancestor of nav.pager that
        // has a class containing "view-dom-id" (Drupal's Views container).
        archiveRoot := findArchiveRoot(doc)
        if archiveRoot == nil {
                // Fall back to the whole document — better to over-return than
                // to silently return nothing.
                archiveRoot = doc
        }
        // Walk the archive and collect PDF links.
        walkAnchorsListing(archiveRoot, func(href, text string) {
                if !strings.HasSuffix(strings.ToLower(href), ".pdf") {
                        return
                }
                // Resolve relative URLs against the parliament base.
                if strings.HasPrefix(href, "/") && !strings.HasPrefix(href, "//") {
                        href = "https://www.parliament.go.ke" + href
                }
                if c, ok := mapper(href, text, house); ok {
                        out = append(out, c)
                }
        })
        // Check for a "next page" link in the pager.
        hasNext = findNextPageLink(doc) != ""
        return out, hasNext
}

// findArchiveRoot returns the node that contains the main listing table.
// We look for nav.pager (the Drupal pagination element) and walk up to
// the closest ancestor that has a class containing "view-dom-id" (the
// Drupal Views container). If neither is present, returns nil and the
// caller falls back to the whole document.
func findArchiveRoot(n *html.Node) *html.Node {
        if n == nil {
                return nil
        }
        // Walk depth-first looking for nav.pager.
        var pager *html.Node
        var walk func(*html.Node)
        walk = func(n *html.Node) {
                if pager != nil {
                        return
                }
                if n.Type == html.ElementNode && n.Data == "nav" {
                        for _, a := range n.Attr {
                                if a.Key == "class" && strings.Contains(a.Val, "pager") {
                                        pager = n
                                        return
                                }
                        }
                }
                for c := n.FirstChild; c != nil; c = c.NextSibling {
                        walk(c)
                }
        }
        walk(n)
        if pager == nil {
                return nil
        }
        // Walk up to the closest ancestor that looks like a Views container.
        for p := pager.Parent; p != nil; p = p.Parent {
                for _, a := range p.Attr {
                        if a.Key == "class" && (strings.Contains(a.Val, "view-dom-id") || strings.Contains(a.Val, "view-content")) {
                                return p
                        }
                }
        }
        // Fall back to pager's grandparent — that's typically the table's
        // containing div on parliament.go.ke.
        if pager.Parent != nil && pager.Parent.Parent != nil {
                return pager.Parent.Parent
        }
        return pager
}

// findNextPageLink returns the URL of the "next page" link in the pager,
// or "" if there isn't one (i.e. we're on the last page).
func findNextPageLink(n *html.Node) string {
        var nextURL string
        var walk func(*html.Node)
        walk = func(n *html.Node) {
                if nextURL != "" || n == nil {
                        return
                }
                if n.Type == html.ElementNode {
                        isPagerItem := false
                        for _, a := range n.Attr {
                                if a.Key == "class" && (strings.Contains(a.Val, "pager__item--next") || strings.Contains(a.Val, "next")) {
                                        isPagerItem = true
                                        break
                                }
                        }
                        if isPagerItem {
                                // Find the first <a> inside this pager item.
                                var findAnchor func(*html.Node) string
                                findAnchor = func(n *html.Node) string {
                                        if n == nil {
                                                return ""
                                        }
                                        if n.Type == html.ElementNode && n.Data == "a" {
                                                for _, a := range n.Attr {
                                                        if a.Key == "href" {
                                                                return a.Val
                                                        }
                                                }
                                        }
                                        for c := n.FirstChild; c != nil; c = c.NextSibling {
                                                if u := findAnchor(c); u != "" {
                                                        return u
                                                }
                                        }
                                        return ""
                                }
                                nextURL = findAnchor(n)
                                return
                        }
                }
                for c := n.FirstChild; c != nil; c = c.NextSibling {
                        walk(c)
                }
        }
        walk(n)
        return nextURL
}

// walkAnchors visits every <a> node in the tree rooted at n and invokes
// fn with its href attribute and text content. Used by ParsePDFListing to
// enumerate the PDF links in the archive table.
//
// NOTE: parser.go ALSO defines walkAnchors with a different signature
// (it passes the *html.Node too). The two are NOT in conflict because
// they live in the same package and Go does not allow function overloads
// — parser.go's version is the one that wins. ParsePDFListing uses an
// internal closure to adapt between the two signatures.
func walkAnchorsListing(n *html.Node, fn func(href, text string)) {
        if n == nil {
                return
        }
        if n.Type == html.ElementNode && n.Data == "a" {
                href := attrOf(n, "href")
                text := textOf(n)
                if href != "" {
                        fn(href, text)
                }
        }
        for c := n.FirstChild; c != nil; c = c.NextSibling {
                walkAnchorsListing(c, fn)
        }
}

// attrOf and textOf are defined in parser.go (they're shared with the
// Bills parser). We rely on those definitions here.

// _ guard ensures this file is treated as a separate compilation unit from
// parser.go (which also defines walkAnchors / attrOf / textOf for its own
// Bills-specific parsing). The linker deduplicates identical definitions.
var _ = fmt.Sprintf
