// Package president — HTML / RSS parser for Presidential Assent announcements
// published on president.go.ke.
//
// The president.go.ke site is a WordPress instance. Two endpoints matter:
//
//  1. /search/assent/feed/rss2/  — returns RSS 2.0 of every post whose content
//     matches the word "assent". ParseAssentList consumes this feed.
//
//  2. /<post-slug>/              — the human-readable announcement page.
//     ParseAssentDetail consumes this page's HTML and extracts one
//     AssentCandidate per BILL mentioned in the announcement (a single
//     assent ceremony may cover several bills — e.g., the Sep 8 2026 ceremony
//     at State House Nairobi covered 4).
package president

import (
	"encoding/xml"
	"regexp"
	"strings"
	"time"
)

// AssentCandidate is a discovered or parsed Presidential Assent.
//
// A single announcement may contain MULTIPLE Bills (e.g., the Sep 8 2026
// State House ceremony covered 4). The adapter produces one AssentCandidate
// per BILL — multiple candidates produced from the same announcement share
// the same SourceURL.
type AssentCandidate struct {
	// BillName is the official short title of the Bill as it appears in the
	// announcement, e.g., "The Division of Revenue (Amendment) Bill, 2024".
	// When produced by ParseAssentList (announcement-level), this field holds
	// the announcement's og:title as a hint; ParseAssentDetail refines it to
	// the precise per-bill name.
	BillName string

	// AssentDate is the date the President signed the Bill(s), parsed from
	// the post's published date.
	AssentDate time.Time

	// SourceURL is the president.go.ke announcement URL.
	SourceURL string

	// SourceID is the platform-internal stable ID. Constructed from the
	// announcement slug (and bill name when known), so the same announcement
	// always yields the same SourceID across runs.
	SourceID string
}

// rssFeed mirrors the WordPress RSS 2.0 feed structure used by president.go.ke.
type rssFeed struct {
	XMLName xml.Name   `xml:"rss"`
	Channel rssChannel `xml:"channel"`
}

type rssChannel struct {
	Title string    `xml:"title"`
	Link  string    `xml:"link"`
	Items []rssItem `xml:"item"`
}

type rssItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	PubDate     string `xml:"pubDate"`
	Description string `xml:"description"`
}

// ogDescRe extracts the og:description meta tag — the cleanest summary of an
// announcement. WordPress always populates this for published posts.
var ogDescRe = regexp.MustCompile(`og:description"\s+content="([^"]+)"`)

// postDateRe extracts the publication date from the post-meta span.
// president.go.ke renders dates as "December 4, 2024" / "May 11, 2026".
var postDateRe = regexp.MustCompile(`class="item">([A-Z][a-z]+ \d{1,2}, \d{4})</span>`)

// paragraphRe captures the body paragraphs of the announcement.
var paragraphRe = regexp.MustCompile(`<p class="wp-block-paragraph">([^<]+)</p>`)

// introRe finds the verbs that introduce a list of bills in the announcement
// prose. The fragment AFTER each match (up to the next sentence-ending
// period) is the candidate region in which to look for bill names.
//
// The pattern is case-insensitive so it also matches title-case headlines
// such as "PRESIDENT RUTO ASSENTS TO THE X BILL" (announcement headlines) and
// "DURING THE ASSENT TO THE X BILL" (speeches_remarks headlines).
var introRe = regexp.MustCompile(`(?i)(?:assented to|assents to|assent to|signed into law|enacted into law|gave assent to)`)

// billNameRe matches a Kenyan Bill short title.
//
// A Bill short title is one or more Capitalized words, optionally followed by
// a parenthetical qualifier (e.g., "(Amendment)"), then the literal "Bill",
// optionally followed by ", YYYY" or " YYYY" year suffix, optionally followed
// by a second parenthetical (e.g., "(National Assembly Bills No. 13 of 2024)").
//
// The "Bill" suffix is matched case-insensitively ([Bb][Ii][Ll][Ll]) so the
// regex also works on all-caps WordPress headlines (e.g., "...Sovereign Wealth
// Fund BILL, 2026"). The plural "Bills" (5 letters) is rejected.
//
// Examples the regex captures:
//
//	Division of Revenue (Amendment) Bill, 2024
//	National Rating Bill 2022
//	Water (Amendment) Bill, 2024
//	Income Tax Bill
//	Special Economic Zones (Amendment) Bill
//	Technopolis Bill
//	Anti-Money Laundering and Combating of Terrorism Financing Laws (Amendment) Bill, 2025
//	Insurance Professionals Bill (National Assembly Bills No. 13 of 2024)
//
// The leading subject phrase is non-greedy (+?) so the engine expands it
// minimally until "Bill" can match.
var billNameRe = regexp.MustCompile(
	`([A-Z][A-Za-z'\- ]+?(?:\([^)]+\)\s*)?[Bb][Ii][Ll][Ll](?:\s*\([^)]+\))?)(?:,?\s*(\d{4}))?\b`,
)

// ParseAssentList parses an RSS feed (or HTML search results page) listing
// assent announcements and returns one AssentCandidate per announcement.
//
// The returned candidates carry the SourceURL + AssentDate + a title hint in
// BillName. Callers should FetchAssent + ParseAssentDetail on each candidate
// to extract the precise per-bill names from the full HTML.
//
// president.go.ke exposes the search-as-RSS endpoint at
// /search/assent/feed/rss2/, which returns a well-formed RSS 2.0 feed of all
// assent-related announcements.
func ParseAssentList(feedXML string) []AssentCandidate {
	var feed rssFeed
	if err := xml.Unmarshal([]byte(feedXML), &feed); err != nil {
		// Fall back: if the input is HTML rather than RSS, scan for anchor
		// links whose URL contains "assent" — this is the second discovery
		// path used when RSS is unavailable.
		return parseAssentListFromHTML(feedXML)
	}

	out := make([]AssentCandidate, 0, len(feed.Channel.Items))
	for _, item := range feed.Channel.Items {
		if item.Link == "" {
			continue
		}
		var date time.Time
		if item.PubDate != "" {
			// RFC 1123Z: "Wed, 08 Jul 2026 12:57:21 +0000"
			if t, err := time.Parse(time.RFC1123Z, item.PubDate); err == nil {
				date = t
			}
		}
		title := strings.TrimSpace(item.Title)
		out = append(out, AssentCandidate{
			BillName:   title,
			AssentDate: date,
			SourceURL:  item.Link,
			SourceID:   sourceIDFromURL(item.Link),
		})
	}
	return out
}

// anchorRe finds <a href="..."> links in HTML fallback mode.
var anchorRe = regexp.MustCompile(`<a[^>]+href="(https?://[^"]*assent[^"]*)"[^>]*>([^<]*)</a>`)

// parseAssentListFromHTML is the fallback path used when ParseAssentList
// receives HTML rather than RSS — it scans anchor tags whose URL contains
// "assent" and produces a candidate per unique URL.
func parseAssentListFromHTML(html string) []AssentCandidate {
	matches := anchorRe.FindAllStringSubmatch(html, -1)
	if matches == nil {
		return nil
	}
	seen := make(map[string]bool, len(matches))
	out := make([]AssentCandidate, 0, len(matches))
	for _, m := range matches {
		url := m[1]
		if seen[url] {
			continue
		}
		seen[url] = true
		title := strings.TrimSpace(decodeHTMLEntities(m[2]))
		out = append(out, AssentCandidate{
			BillName:  title,
			SourceURL: url,
			SourceID:  sourceIDFromURL(url),
		})
	}
	return out
}

// ParseAssentDetail parses a single assent announcement HTML page and returns
// one AssentCandidate per BILL mentioned in the announcement.
//
// The parser extracts bill names from the fragment that follows an
// introductory verb ("assented to", "signed into law", "enacted into law",
// "gave assent to", "assents to"). Searching only in this fragment prevents
// false positives where the regex would otherwise capture the entire
// "The President assented to the X Bill" sentence as a single (wrong) match.
//
// The fragment is searched first in the og:description meta tag (the
// cleanest summary), then the first few body paragraphs as fallback. If no
// bills are found in those primary sources, the parser retries the same
// pipeline on the announcement's headline (og:title) — this handles posts
// that mention the bill only in the title. As a last resort, the title is
// used verbatim.
//
// The assent date is parsed from the post-meta span (format "December 4, 2024").
// For posts that lack the post-meta span (some speeches_remarks posts), the
// assent date is zero — callers can fall back to the RSS pubDate.
func ParseAssentDetail(html, sourceURL string) []AssentCandidate {
	assentDate := extractPostDate(html)

	// Primary source: og:description + body paragraphs.
	sources := extractSearchFragments(html)
	billNames := extractBillNames(sources)

	// Fallback 1: if no bills were found in the primary sources, run the
	// same introRe + billNameRe pipeline on the announcement's headline
	// (og:title). This handles posts that don't have an og:description or
	// body paragraphs (e.g., speeches_remarks posts), but DO have the bill
	// name in their title (e.g., "DURING THE ASSENT TO THE SOVEREIGN WEALTH
	// FUND BILL, 2026").
	if len(billNames) == 0 {
		if title := extractTitle(html); title != "" {
			billNames = extractBillNames([]string{title})
		}
	}

	// Fallback 2: if STILL no bills were extracted, use the title verbatim.
	// This handles edge cases where the title doesn't match the introRe +
	// billNameRe pipeline (e.g., a short title like "PRESIDENT RUTO
	// ADDRESSES THE NATION" with no bill mention at all).
	if len(billNames) == 0 {
		if title := extractTitle(html); title != "" {
			billNames = []string{title}
		}
	}

	if len(billNames) == 0 {
		return nil
	}

	out := make([]AssentCandidate, 0, len(billNames))
	for _, name := range billNames {
		c := AssentCandidate{
			BillName:   normaliseBillName(name),
			AssentDate: assentDate,
			SourceURL:  sourceURL,
			SourceID:   sourceIDFromURL(sourceURL) + "-" + slugify(name),
		}
		out = append(out, c)
	}
	return out
}

// extractSearchFragments returns the candidate text regions in which to look
// for bill names: the og:description meta content and the first handful of
// body paragraphs. The og:title is intentionally NOT included here — it's
// only used as a fallback (in ParseAssentDetail) when no bills are found in
// the primary sources, to avoid duplicates between og:description and the
// title.
func extractSearchFragments(html string) []string {
	var out []string
	// og:description is the cleanest summary and usually contains the
	// introductory verb ("assented to the X Bill, YYYY") followed by the
	// bill list. Try it first.
	if m := ogDescRe.FindStringSubmatch(html); len(m) >= 2 {
		out = append(out, decodeHTMLEntities(m[1]))
	}
	// Body paragraphs are the fallback when the og:description is missing
	// or doesn't mention the bills by name.
	for _, m := range paragraphRe.FindAllStringSubmatch(html, 5) {
		if len(m) >= 2 {
			out = append(out, decodeHTMLEntities(m[1]))
		}
	}
	return out
}

// extractTitle returns the announcement's headline, taken from og:title (which
// is cleaner than the <title> tag — the latter has the site suffix).
func extractTitle(html string) string {
	re := regexp.MustCompile(`og:title"\s+content="([^"]+)"`)
	if m := re.FindStringSubmatch(html); len(m) >= 2 {
		return strings.TrimSpace(decodeHTMLEntities(m[1]))
	}
	return ""
}

// extractPostDate parses the post-meta span's publication date.
func extractPostDate(html string) time.Time {
	m := postDateRe.FindStringSubmatch(html)
	if len(m) < 2 {
		return time.Time{}
	}
	dateStr := strings.TrimSpace(m[1])
	// "December 4, 2024" — Go reference date "January 2, 2006".
	if t, err := time.Parse("January 2, 2006", dateStr); err == nil {
		return t
	}
	return time.Time{}
}

// extractBillNames scans the candidate text fragments and returns the unique
// bill names found, in first-seen order.
//
// For each fragment, the parser first locates the introductory verb
// ("assented to", "signed into law", ...) and takes the substring from the
// end of the verb to the next sentence-ending period. Within that substring,
// the bill-name regex is applied. Restricting the search to post-verb
// fragments prevents the regex from capturing the entire "The President
// assented to the X Bill" sentence as a single false-positive match.
func extractBillNames(fragments []string) []string {
	seen := make(map[string]bool)
	var out []string
	for _, src := range fragments {
		for _, frag := range fragmentsAfterIntro(src) {
			for _, m := range billNameRe.FindAllStringSubmatch(frag, -1) {
				// m[0] is the entire bill name including the year suffix (if any)
				// with its ORIGINAL separator ("Bill, 2024" or "Bill 2022"). Use it
				// verbatim — the source-of-truth spacing is part of the bill's
				// official short title (e.g., "National Rating Bill 2022" has no comma).
				name := normaliseBillName(m[0])
				// The "Bill" suffix check is case-insensitive because the regex
				// matches all-caps titles too (e.g., "...Sovereign Wealth Fund
				// BILL, 2026").
				if name == "" || !strings.Contains(strings.ToLower(name), "bill") || seen[name] {
					continue
				}
				// Reject obvious false positives: a bill name cannot contain
				// the words "President", "State House" or "Nairobi" — those
				// would indicate the regex has matched the entire intro
				// sentence, not a real bill name.
				lower := strings.ToLower(name)
				if strings.Contains(lower, "president") ||
					strings.Contains(lower, "state house") ||
					strings.Contains(lower, "nairobi") ||
					strings.Contains(lower, "the president") {
					continue
				}
				seen[name] = true
				out = append(out, name)
			}
		}
	}
	return out
}

// fragmentsAfterIntro returns the substrings of text that follow each
// occurrence of an introductory verb, each ending at the next sentence-ending
// period. This is the search region in which the bill-name regex is applied.
func fragmentsAfterIntro(text string) []string {
	var fragments []string
	pos := 0
	for {
		loc := introRe.FindStringIndex(text[pos:])
		if loc == nil {
			break
		}
		start := pos + loc[1]
		// Find the next sentence-ending period. We treat "." at word
		// boundaries (space, end-of-string, or after a closing paren) as a
		// sentence terminator.
		endIdx := nextSentenceEnd(text[start:])
		var end int
		if endIdx >= 0 {
			end = start + endIdx
		} else {
			end = len(text)
		}
		if start < end {
			fragments = append(fragments, text[start:end])
		}
		if end >= len(text) {
			break
		}
		pos = end
	}
	return fragments
}

// nextSentenceEnd returns the byte index of the next sentence-ending period
// in s, or -1 if none. A period is treated as sentence-ending if it's
// followed by whitespace, end-of-string, or a closing quote.
func nextSentenceEnd(s string) int {
	for i := 0; i < len(s); i++ {
		if s[i] != '.' {
			continue
		}
		// Allow trailing digits (e.g., "2024. " — the year ends with 4, not .)
		// and check what follows the period.
		if i+1 >= len(s) {
			return i
		}
		next := s[i+1]
		if next == ' ' || next == '\t' || next == '\n' || next == '"' || next == '&' {
			return i
		}
		// "&hellip;" or other entity — treat as sentence end.
		if i+7 <= len(s) && s[i:i+7] == ".&hell" {
			return i
		}
	}
	return -1
}

// normaliseBillName cleans up a captured bill name: trims whitespace,
// collapses whitespace runs, ensures a space after any comma, and strips
// trailing punctuation.
func normaliseBillName(raw string) string {
	s := strings.TrimSpace(raw)
	s = collapseWhitespace(s)
	// Ensure a single space after each comma.
	s = regexp.MustCompile(`,\s*`).ReplaceAllString(s, ", ")
	// Strip a trailing comma or period.
	s = strings.TrimRight(s, ",.")
	return strings.TrimSpace(s)
}

func collapseWhitespace(s string) string {
	return regexp.MustCompile(`\s+`).ReplaceAllString(s, " ")
}

func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}

// sourceIDFromURL builds a stable platform ID from a president.go.ke
// announcement URL by extracting the slug. Example:
//
//	https://www.president.go.ke/president-ruto-assents-to-parliamentary-bills/
//	  → "ke-assent-president-ruto-assents-to-parliamentary-bills"
func sourceIDFromURL(url string) string {
	u := strings.TrimRight(url, "/")
	if i := strings.LastIndex(u, "/"); i >= 0 {
		return "ke-assent-" + u[i+1:]
	}
	return "ke-assent-" + u
}

// decodeHTMLEntities replaces the common HTML entities that WordPress emits
// (smart quotes, ellipsis, em dash, ampersand). This is intentionally a small
// subset — full entity decoding is the documents service's job.
func decodeHTMLEntities(s string) string {
	r := strings.NewReplacer(
		"&#8217;", "'",
		"&#8216;", "'",
		"&#8220;", "\"",
		"&#8221;", "\"",
		"&#8230;", "...",
		"&#8211;", "-",
		"&#8212;", "-",
		"&#038;", "&",
		"&#039;", "'",
		"&amp;", "&",
		"&nbsp;", " ",
		"&hellip;", "...",
		"&quot;", "\"",
		"&apos;", "'",
	)
	return r.Replace(s)
}
