// Package gazette adapts the Kenya Gazette source — official gazette notices.
//
// The Kenya Gazette is the official weekly publication of the Government of
// Kenya. It carries legal notices, appointments, name changes, electoral
// boundary changes, statutory instrument commencements, and other official
// acts required by law to be published.
//
// The canonical source is the Kenya Law portal at:
//
//	https://new.kenyalaw.org/kenya_law/gazette/
//
// That page lists Gazette issues (weekly volumes) as PDF links. Each PDF
// is the full issue for one week — individual notices live inside the PDF
// and require OCR or born-digital text extraction (delegated to the
// documents service). The adapter's job is to discover the issue-level
// PDFs; per-notice extraction happens downstream.
//
// The parser (parser.go) handles the HTML structure of the gazette index
// page. The DiscoverNotices function here walks the paginated index
// (?page=N, 0-indexed, 25 issues per page) and returns one GazetteEntry
// per issue found.
package gazette

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Roy-Wanyoike/civic-intelligence/packages/contracts"
	"golang.org/x/net/html"
)

// gazetteIndexURL is the canonical Kenya Gazette index page on the Kenya Law
// portal. Each row of the listing is a single weekly Gazette issue PDF.
const gazetteIndexURL = "https://new.kenyalaw.org/kenya_law/gazette/"

// gazetteMaxPages caps the crawl depth to avoid pulling the entire historical
// archive (which spans decades). 5 pages × 25 issues = 125 most-recent issues.
const gazetteMaxPages = 5

// defaultUserAgent is the polite identifier used when the caller passes an
// empty string. It MUST include the project URL so source administrators can
// reach the platform operators if the crawler misbehaves.
const defaultUserAgent = "CivicIntelligence/0.1 (+https://github.com/Roy-Wanyoike/civic-intelligence)"

// GazetteEntry is the parsed representation of a single Kenya Gazette notice.
// The parser (parser.go) produces these from the Gazette index page or from
// the plain text of a single notice.
type GazetteEntry struct {
	Volume      string
	NoticeNo    string
	Title       string
	Issuer      string
	PublishedAt time.Time
	URL         string
}

// HTTPClient is the interface gazette adapters use to fetch remote content.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// Adapter fetches from the Kenya Gazette source (new.kenyalaw.org).
type Adapter struct {
	client    HTTPClient
	userAgent string
	// indexURL is the canonical gazette index page.
	indexURL string
	// maxPages caps the crawl depth (default 5 pages × 25 issues = 125).
	maxPages int
}

// NewAdapter creates a Gazette adapter.
//
// If client is nil, a default http.Client with a 30-second timeout is used.
// If userAgent is empty, defaultUserAgent is used.
func NewAdapter(client HTTPClient, userAgent string) *Adapter {
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	if userAgent == "" {
		userAgent = defaultUserAgent
	}
	return &Adapter{
		client:    client,
		userAgent: userAgent,
		indexURL:  gazetteIndexURL,
		maxPages:  gazetteMaxPages,
	}
}

// Discover walks the gazette index page (paginated ?page=N) and returns one
// GazetteEntry per issue PDF found. Implements contracts.LegislativeSourceAdapter.
func (a *Adapter) Discover(ctx context.Context) ([]contracts.SourceItem, error) {
	entries, err := a.DiscoverNotices(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]contracts.SourceItem, 0, len(entries))
	now := time.Now().UTC()
	for _, e := range entries {
		out = append(out, contracts.SourceItem{
			URL:          e.URL,
			Title:        e.Title,
			DocumentType: "gazette_notice",
			SourceType:   contracts.SourceItemGazette,
			CountryCode:  "KE",
			SourceID:     "ke-kenyalaw-gazette-" + e.Volume + "-" + e.NoticeNo,
			House:        "Gazette",
			DiscoveredAt: now,
			Metadata: map[string]string{
				"volume":       e.Volume,
				"notice_no":    e.NoticeNo,
				"issuer":       e.Issuer,
				"published_at": e.PublishedAt.Format(time.RFC3339),
			},
		})
	}
	return out, nil
}

// DiscoverNotices discovers Kenya Gazette issues from the paginated index.
// It walks the first a.maxPages pages of the gazette index, extracting
// one GazetteEntry per issue PDF link.
//
// The returned entries are deduplicated by URL (the "latest" sidebar on
// the gazette index can re-list the most recent issue).
func (a *Adapter) DiscoverNotices(ctx context.Context) ([]GazetteEntry, error) {
	var out []GazetteEntry
	for page := 0; page < a.maxPages; page++ {
		pageURL := a.indexURL
		if page > 0 {
			sep := "?"
			if strings.Contains(a.indexURL, "?") {
				sep = "&"
			}
			pageURL = a.indexURL + sep + "page=" + itoaGazette(page)
		}
		body, err := a.fetchURL(ctx, pageURL)
		if err != nil {
			return out, fmt.Errorf("gazette listing %s page %d: %w", a.indexURL, page, err)
		}
		entries, hasNext := ParseGazetteIndex(body)
		out = append(out, entries...)
		if !hasNext {
			break
		}
	}
	out = dedupeGazetteByURL(out)
	return out, nil
}

// Fetch implements contracts.LegislativeSourceAdapter. It downloads a single
// gazette issue PDF.
func (a *Adapter) Fetch(ctx context.Context, item contracts.SourceItem) (*contracts.RawDocument, error) {
	if item.URL == "" {
		return nil, fmt.Errorf("gazette.Fetch: empty URL")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, item.URL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", a.userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/pdf")
	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("gazette.Fetch: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gazette.Fetch: HTTP %d for %s", resp.StatusCode, item.URL)
	}
	body, err := io_ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	mime := resp.Header.Get("Content-Type")
	if mime == "" {
		mime = "application/octet-stream"
	}
	return &contracts.RawDocument{
		URL:         item.URL,
		Bytes:       body,
		MimeType:    mime,
		RetrievedAt: time.Now().UTC(),
	}, nil
}

// Parse implements contracts.LegislativeSourceAdapter. It parses a gazette
// issue PDF into a slice of ExtractedRecord values, one per notice in the
// issue. Real PDF text extraction + notice segmentation is delegated to the
// documents service (services/documents) which has the OCR + structure-
// extraction infrastructure. This HTML-only parser is intentionally a stub
// — it returns an empty slice so the discovery pipeline can complete and
// the document can be queued for downstream extraction.
func (a *Adapter) Parse(ctx context.Context, doc contracts.RawDocument) ([]contracts.ExtractedRecord, error) {
	return []contracts.ExtractedRecord{}, nil
}

// fetchURL performs the actual HTTP GET against url and returns the response
// body as a string.
func (a *Adapter) fetchURL(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", a.userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/pdf")
	resp, err := a.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("fetch %s: HTTP %d", url, resp.StatusCode)
	}
	body, err := io_ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", url, err)
	}
	return string(body), nil
}

// FetchNotice downloads a single gazette notice PDF.
func (a *Adapter) FetchNotice(ctx context.Context, url string) (*contracts.RawDocument, error) {
	return a.Fetch(ctx, contracts.SourceItem{URL: url, DocumentType: "gazette_notice"})
}

// GazetteNotice is a parsed gazette notice with full metadata.
type GazetteNotice struct {
	Volume          string
	NoticeNo        string
	Title           string
	Issuer          string
	PublicationDate time.Time
	URL             string
}

// ParseNotice parses a gazette notice into structured data.
//
// NOTE: real per-notice parsing requires PDF text extraction + notice
// structure heuristics. Delegated to the documents service. This stub
// reuses ParseNoticesHTML on the HTML body (works only for HTML gazette
// notices; PDFs return an empty list).
func (a *Adapter) ParseNotice(htmlBody, sourceURL string) (*GazetteNotice, error) {
	entries, err := ParseNoticesHTML(strings.NewReader(htmlBody))
	if err != nil || len(entries) == 0 {
		return nil, fmt.Errorf("parse notice: %w", err)
	}
	e := entries[0]
	return &GazetteNotice{
		Volume:          e.Volume,
		NoticeNo:        e.NoticeNo,
		Title:           e.Title,
		Issuer:          e.Issuer,
		PublicationDate: e.PublishedAt,
		URL:             sourceURL,
	}, nil
}

// ParseGazetteIndex walks the gazette index HTML and returns one GazetteEntry
// per issue PDF link found. The parser looks for <a href> links whose URL
// ends in .pdf (case-insensitive). Each link is decorated with the issue's
// title (from the link text) and publication date (parsed from the title
// when possible).
//
// The function returns the parsed entries plus a boolean indicating whether
// the page contains a "next page" link in the pager. Callers use this to
// decide whether to walk to ?page=N+1.
func ParseGazetteIndex(htmlBody string) ([]GazetteEntry, bool) {
	doc, err := html.Parse(strings.NewReader(htmlBody))
	if err != nil {
		return nil, false
	}
	var out []GazetteEntry
	walkGazetteAnchors(doc, func(href, text string) {
		if !strings.HasSuffix(strings.ToLower(href), ".pdf") {
			return
		}
		// Resolve relative URLs against the kenyalaw.org base.
		if strings.HasPrefix(href, "/") && !strings.HasPrefix(href, "//") {
			href = "https://new.kenyalaw.org" + href
		}
		title := strings.TrimSpace(text)
		if title == "" {
			// Fall back to the URL-decoded filename.
			if decoded, err := url.QueryUnescape(href); err == nil {
				if idx := strings.LastIndex(decoded, "/"); idx >= 0 {
					title = decoded[idx+1:]
				} else {
					title = decoded
				}
			} else {
				title = href
			}
		}
		entry := GazetteEntry{
			Title: title,
			URL:   href,
		}
		// Reuse the parser.go regex helpers for volume + notice number
		// extraction on the link text.
		entries, _ := ParseNoticesHTML(strings.NewReader(title))
		if len(entries) > 0 {
			entry.Volume = entries[0].Volume
			entry.NoticeNo = entries[0].NoticeNo
			entry.Issuer = entries[0].Issuer
			entry.PublishedAt = entries[0].PublishedAt
		}
		out = append(out, entry)
	})
	hasNext := findGazetteNextPageLink(doc) != ""
	return out, hasNext
}

// walkGazetteAnchors visits every <a> node in the tree rooted at n and invokes
// fn with its href attribute and text content.
func walkGazetteAnchors(n *html.Node, fn func(href, text string)) {
	if n == nil {
		return
	}
	if n.Type == html.ElementNode && n.Data == "a" {
		href := ""
		text := ""
		for _, a := range n.Attr {
			if a.Key == "href" {
				href = a.Val
			}
		}
		text = textOfGazetteAnchor(n)
		if href != "" {
			fn(href, text)
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		walkGazetteAnchors(c, fn)
	}
}

// textOfGazetteAnchor returns the concatenated text content of an <a> element.
func textOfGazetteAnchor(n *html.Node) string {
	if n == nil {
		return ""
	}
	if n.Type == html.TextNode {
		return n.Data
	}
	var b strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		b.WriteString(textOfGazetteAnchor(c))
	}
	return b.String()
}

// findGazetteNextPageLink returns the URL of the "next page" link in the pager,
// or "" if there isn't one (i.e. we're on the last page). Looks for an <a>
// inside a list item with class pager__item--next or a rel="next" attribute.
func findGazetteNextPageLink(n *html.Node) string {
	var nextURL string
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if nextURL != "" || n == nil {
			return
		}
		if n.Type == html.ElementNode && n.Data == "a" {
			rel := ""
			parentClass := ""
			for _, a := range n.Attr {
				if a.Key == "rel" {
					rel = a.Val
				}
			}
			if n.Parent != nil {
				for _, a := range n.Parent.Attr {
					if a.Key == "class" {
						parentClass = a.Val
					}
				}
			}
			if strings.Contains(rel, "next") ||
				strings.Contains(parentClass, "pager__item--next") ||
				strings.Contains(parentClass, "next") {
				for _, a := range n.Attr {
					if a.Key == "href" {
						nextURL = a.Val
						return
					}
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return nextURL
}

// dedupeGazetteByURL removes candidates with duplicate URLs. First occurrence
// wins (the main archive is preferred over the "latest" sidebar block).
func dedupeGazetteByURL(in []GazetteEntry) []GazetteEntry {
	seen := make(map[string]struct{}, len(in))
	out := make([]GazetteEntry, 0, len(in))
	for _, e := range in {
		if e.URL == "" {
			continue
		}
		if _, ok := seen[e.URL]; ok {
			continue
		}
		seen[e.URL] = struct{}{}
		out = append(out, e)
	}
	return out
}

// itoaGazette is a tiny dependency-free int→string for the page parameter.
func itoaGazette(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [12]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
