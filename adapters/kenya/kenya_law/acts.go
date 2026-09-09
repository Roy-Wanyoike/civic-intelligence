package kenya_law

import (
	"context"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	"golang.org/x/net/html"
)

// ActEntry is a parsed record from the kenyalaw.org Acts of Parliament index.
type ActEntry struct {
	Cap      string // Cap number (e.g. "Cap 2" for the Interpretation and General Provisions Act)
	Title    string
	ActNo    string // e.g. "No. 18 of 2010" where applicable
	Year     int
	URL      string
	Updated  time.Time
}

// DiscoverActs fetches the kenyalaw.org Acts of Parliament index and parses
// it into a slice of ActEntry records.
func (k *KenyaLawAdapter) DiscoverActs(ctx context.Context) ([]ActEntry, error) {
	body, err := k.fetchSource(ctx, SourceActsIndex, "acts")
	if err != nil {
		return nil, err
	}
	if body == nil {
		return nil, nil
	}
	entries, err := ParseActsHTML(strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrParse, err)
	}
	return entries, nil
}

// ParseActsHTML parses the kenyalaw.org Acts index page into ActEntry
// records.
//
// The page is a list of links to individual Act pages, each link's text
// following the pattern "Cap XX — <Title>" or "<Title> (No. NN of YYYY)".
// The parser is tolerant of layout drift; missing fields are left blank.
func ParseActsHTML(r io.Reader) ([]ActEntry, error) {
	doc, err := html.Parse(r)
	if err != nil {
		return nil, fmt.Errorf("kenya_law parser: %w", err)
	}
	var out []ActEntry
	walkAnchors(doc, func(href, text string) {
		if !looksLikeAct(href, text) {
			return
		}
		out = append(out, ActEntry{
			Cap:     extractCap(text),
			Title:   extractActTitle(text),
			ActNo:   extractActNo(text),
			Year:    extractYear(text),
			URL:     href,
			Updated: time.Time{},
		})
	})
	return out, nil
}

// --- helpers ---

var (
	capRe     = regexp.MustCompile(`(?i)\bcap\.?\s*(\d+[a-z]?)\b`)
	actNoRe   = regexp.MustCompile(`(?i)\bno\.?\s*(\d+)\s+of\s+(\d{4})\b`)
	yearRe    = regexp.MustCompile(`\b(19|20)\d{2}\b`)
	dashSplit = regexp.MustCompile(`\s+[-–—]\s+`)
)

func walkAnchors(n *html.Node, fn func(href, text string)) {
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
		walkAnchors(c, fn)
	}
}

func attrOf(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}

func textOf(n *html.Node) string {
	if n == nil {
		return ""
	}
	if n.Type == html.TextNode {
		return n.Data
	}
	var sb strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		sb.WriteString(textOf(c))
	}
	return sb.String()
}

func looksLikeAct(href, text string) bool {
	lc := strings.ToLower(href + " " + text)
	if !strings.Contains(lc, "act") {
		return false
	}
	// Filter out navigation noise.
	if strings.Contains(lc, "login") || strings.Contains(lc, "subscribe") {
		return false
	}
	return capRe.MatchString(text) || actNoRe.MatchString(text) || strings.Contains(lc, "cap ")
}

func extractCap(text string) string {
	m := capRe.FindStringSubmatch(text)
	if len(m) >= 2 {
		return "Cap " + strings.ToUpper(m[1])
	}
	return ""
}

func extractActTitle(text string) string {
	// Try "Cap XX — Title" first.
	if parts := dashSplit.Split(text, 2); len(parts) == 2 {
		return strings.TrimSpace(parts[1])
	}
	// Fallback to the whole text minus a leading "Cap XX — " token.
	return strings.TrimSpace(text)
}

func extractActNo(text string) string {
	m := actNoRe.FindStringSubmatch(text)
	if len(m) >= 2 {
		return "No. " + m[1] + " of " + m[2]
	}
	return ""
}

func extractYear(text string) int {
	m := yearRe.FindString(text)
	if m == "" {
		return 0
	}
	var y int
	_, err := fmt.Sscanf(m, "%d", &y)
	if err != nil {
		return 0
	}
	return y
}
