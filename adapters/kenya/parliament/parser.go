package parliament

import (
	"fmt"
	"io"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/Roy-Wanyoike/civic-intelligence/adapters/kenya/internal"
	"github.com/Roy-Wanyoike/civic-intelligence/packages/contracts"
	"golang.org/x/net/html"
)

// stageMapping maps the lowercased substrings Parliament uses on its Bill
// Tracker pages to the canonical stage codes from adapters/kenya/internal/stages.go.
// Order matters: more specific patterns MUST come first (e.g.,
// "committee of the whole house" before "committee").
var stageMapping = []struct {
	needle string
	code   internal.StageCode
}{
	{"first reading", internal.StageFirstReading},
	{"1st reading", internal.StageFirstReading},
	{"first time", internal.StageFirstReading}, // "read a First Time"
	{"read a first", internal.StageFirstReading},

	{"second reading", internal.StageSecondReading},
	{"2nd reading", internal.StageSecondReading},
	{"second time", internal.StageSecondReading}, // "read a Second Time"
	{"read a second", internal.StageSecondReading},

	{"committee of the whole house", internal.StageCommitteeOfWholeHouse},
	{"committee of whole house", internal.StageCommitteeOfWholeHouse},
	{"whole house", internal.StageCommitteeOfWholeHouse},

	{"committee stage", internal.StageCommitteeStage},
	{"departmental committee", internal.StageCommitteeStage},
	{"committee of the house", internal.StageCommitteeStage},

	{"report stage", internal.StageReportStage},

	{"third reading", internal.StageThirdReading},
	{"3rd reading", internal.StageThirdReading},
	{"third time", internal.StageThirdReading}, // "read a Third Time"
	{"read a third", internal.StageThirdReading},

	{"presidential assent", internal.StagePresidentialAssent},
	{"assented", internal.StagePresidentialAssent},
	{"assent to", internal.StagePresidentialAssent},
	{"assented to", internal.StagePresidentialAssent},

	{"commencement", internal.StageCommencement},
	{"in force", internal.StageCommencement},
	{"came into force", internal.StageCommencement},
	{"gazetted", internal.StageCommencement},

	{"mediation", internal.StageMediation},
	{"mediation committee", internal.StageMediation},

	{"rejected", internal.StageRejected},
	{"negatived", internal.StageRejected},

	{"withdrawn", internal.StageWithdrawn},
	{"withdraw", internal.StageWithdrawn},

	{"lapsed", internal.StageLapsed},

	// Bare "committee" — must come AFTER the more specific committee patterns.
	{"committee", internal.StageCommitteeStage},

	// Bare "assent" — must come AFTER the more specific assent patterns.
	{"assent", internal.StagePresidentialAssent},
}

// MapStageText maps Parliament's raw stage text to the platform's canonical
// stage codes (FIRST_READING, SECOND_READING, …). The match is
// case-insensitive and substring-based, so "Second Reading — 12 Mar 2026"
// → SECOND_READING. Returns "" if the input does not match a known stage.
//
// The canonical stage codes are defined in
// adapters/kenya/internal/stages.go and MUST NOT be duplicated here — this
// function imports them rather than re-declaring them.
func MapStageText(input string) string {
	if input == "" {
		return ""
	}
	lc := strings.ToLower(input)
	for _, m := range stageMapping {
		if strings.Contains(lc, m.needle) {
			return string(m.code)
		}
	}
	return ""
}

// MapStageTextWithConfidence is like MapStageText but also returns a
// confidence score (0.0-1.0) for the mapping. Exact stage-name matches score
// 0.95; substring matches score 0.80; no match returns ("", 0.0).
func MapStageTextWithConfidence(input string) (string, float64) {
	if input == "" {
		return "", 0.0
	}
	lc := strings.ToLower(strings.TrimSpace(input))
	for _, m := range stageMapping {
		if lc == m.needle {
			return string(m.code), 0.95
		}
	}
	for _, m := range stageMapping {
		if strings.Contains(lc, m.needle) {
			return string(m.code), 0.80
		}
	}
	return "", 0.0
}

// ParseBillsListing parses the HTML of a Parliament of Kenya Bills listing
// page (https://www.parliament.go.ke/the-national-assembly/house-business/bills
// or /the-senate/senate-bills) into a slice of BillCandidate records.
//
// The pages use a Drupal 8 "views" rendering with one <div class="post-block">
// per Bill. Each post-block contains:
//   - <div class="post-title"><a href="…PDF" title="…">Bill Title</a>
//   - <div class="post-meta"> with three optional spans:
//   - <span class="post-digest">Bill Digest: <a href="…">…</a>
//   - <span class="post-billtracker">Bill Tracker: <a href="…">…</a>
//   - <span class="post-petition"><a href="…/contact/…_petition?bill=…">
//
// The parser is deliberately tolerant: a Bill with no digest or tracker link
// is still returned, with the corresponding fields left empty.
func ParseBillsListing(pageHTML string) []BillCandidate {
	doc, err := html.Parse(strings.NewReader(pageHTML))
	if err != nil {
		return nil
	}
	var out []BillCandidate
	walkPostBlocks(doc, func(block *html.Node) {
		c := parsePostBlock(block)
		if c.URL != "" || c.Title != "" {
			out = append(out, c)
		}
	})
	// De-duplicate by URL (a Bill may be linked more than once).
	seen := make(map[string]bool, len(out))
	deduped := out[:0]
	for _, c := range out {
		key := c.URL
		if key == "" {
			key = c.Title
		}
		if seen[key] {
			continue
		}
		seen[key] = true
		deduped = append(deduped, c)
	}
	return deduped
}

// ParseBillDetail extracts Bill metadata from a Parliament Bill detail page
// (or, in the absence of a per-Bill detail page, from the Bill listing's
// post-block markup). Returns a single ExtractedRecord of kind "bill".
//
// The current Parliament website does not publish a per-Bill HTML detail
// page (the canonical Bill artefact is the PDF itself); this function
// therefore tolerates either a Bill listing's post-block markup or a
// hypothetical detail page containing <title>…</title> + a single
// post-title link.
func ParseBillDetail(pageHTML, sourceURL string) ([]contracts.ExtractedRecord, error) {
	// Find the first PDF link / post-title link in the page.
	pdfURL, title := firstBillLink(pageHTML)
	if title == "" {
		return nil, fmt.Errorf("parliament.ParseBillDetail: no Bill title found in %s", sourceURL)
	}
	house := houseFromHref(sourceURL)
	pubDate := parseDateInText(title)

	rec := contracts.ExtractedRecord{
		Kind:          "bill",
		Title:         title,
		House:         house,
		PublishedAt:   pubDate,
		SourceURL:     sourceURL,
		RetrievedAt:   time.Now().UTC(),
		Confidence:    0.85,
		ExtractorName: "kenya.parliament.HTMLParser",
	}
	if pdfURL != "" {
		rec.Extra = map[string]interface{}{"pdf_url": pdfURL}
	}
	return []contracts.ExtractedRecord{rec}, nil
}

// ParseBillTrackerHTML parses a Bill Tracker page or per-Bill tracker page
// and returns a BillTracker with the most recent stage, status, and date
// found. If no stage information is available on the page (e.g., the page
// is a weekly tracker index), an empty Stage is returned with low
// Confidence — the caller can then decide to schedule a PDF-parsing job.
//
// sourceURL is stored verbatim on the returned BillTracker.
func ParseBillTrackerHTML(pageHTML, sourceURL string) *BillTracker {
	bt := &BillTracker{
		SourceURL:   sourceURL,
		RetrievedAt: time.Now().UTC(),
		Confidence:  0.0,
	}
	// Find the first stage keyword in the HTML text. The weekly tracker PDF
	// pages embed stage rows as text; per-Bill tracker pages (if/when
	// published) would do the same.
	stageCode, conf := MapStageTextWithConfidence(extractText(pageHTML))
	if stageCode != "" {
		bt.Stage = stageCode
		bt.Confidence = conf
		// Best-effort status: try to find a line containing the stage text.
		bt.Status = firstLineContaining(pageHTML, stageKeywords(stageCode))
		bt.Date = parseDateInText(bt.Status)
		return bt
	}
	// No stage info in the HTML — return empty stage with the source URL.
	bt.Status = "no stage information in HTML (likely a weekly tracker PDF)"
	bt.Confidence = 0.1
	return bt
}

// stageKeywords returns English keywords for a stage code, used to find a
// matching line in the tracker HTML for the status text.
func stageKeywords(code string) []string {
	switch strings.ToUpper(code) {
	case string(internal.StageFirstReading):
		return []string{"first reading", "1st reading"}
	case string(internal.StageSecondReading):
		return []string{"second reading", "2nd reading"}
	case string(internal.StageCommitteeStage):
		return []string{"committee stage", "departmental committee"}
	case string(internal.StageCommitteeOfWholeHouse):
		return []string{"committee of the whole house", "whole house"}
	case string(internal.StageReportStage):
		return []string{"report stage"}
	case string(internal.StageThirdReading):
		return []string{"third reading", "3rd reading"}
	case string(internal.StagePresidentialAssent):
		return []string{"assent", "assented"}
	case string(internal.StageCommencement):
		return []string{"commencement", "in force"}
	case string(internal.StageMediation):
		return []string{"mediation"}
	case string(internal.StageRejected):
		return []string{"rejected", "negatived"}
	case string(internal.StageWithdrawn):
		return []string{"withdrawn", "withdraw"}
	case string(internal.StageLapsed):
		return []string{"lapsed"}
	}
	return nil
}

// extractText returns the trimmed concatenation of all text in the HTML.
// Used by ParseBillTrackerHTML to scan for stage keywords.
func extractText(pageHTML string) string {
	doc, err := html.Parse(strings.NewReader(pageHTML))
	if err != nil {
		return pageHTML
	}
	return strings.TrimSpace(textOf(doc))
}

// firstLineContaining returns the first line of text in pageHTML that
// contains any of the (case-insensitive) keywords. Returns "" if none.
func firstLineContaining(pageHTML string, keywords []string) string {
	if len(keywords) == 0 {
		return ""
	}
	lowers := make([]string, len(keywords))
	for i, k := range keywords {
		lowers[i] = strings.ToLower(k)
	}
	for _, line := range strings.Split(pageHTML, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		lc := strings.ToLower(trimmed)
		for _, k := range lowers {
			if strings.Contains(lc, k) {
				return trimmed
			}
		}
	}
	return ""
}

// firstBillLink returns the first Bill PDF link + title found in the HTML,
// by walking the document's anchors and selecting the first that looks like
// a Bill (sites/default/files/.../*.pdf with a "bill" keyword in the URL
// or title text).
func firstBillLink(pageHTML string) (pdfURL, title string) {
	doc, err := html.Parse(strings.NewReader(pageHTML))
	if err != nil {
		return "", ""
	}
	var foundURL, foundTitle string
	walkAnchors(doc, func(_ *html.Node, href, text string) {
		if foundURL != "" {
			return
		}
		lc := strings.ToLower(href + " " + text)
		if strings.Contains(lc, "sites/default/files") &&
			strings.Contains(lc, ".pdf") &&
			(strings.Contains(lc, "bill") || strings.Contains(lc, "act")) {
			foundURL = href
			foundTitle = strings.TrimSpace(text)
		}
	})
	return foundURL, foundTitle
}

// walkPostBlocks visits every <div class="post-block"> node and invokes fn on it.
func walkPostBlocks(n *html.Node, fn func(*html.Node)) {
	if n == nil {
		return
	}
	if n.Type == html.ElementNode && n.Data == "div" {
		if hasClass(n, "post-block") {
			fn(n)
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		walkPostBlocks(c, fn)
	}
}

// parsePostBlock extracts a single BillCandidate from a <div class="post-block">.
func parsePostBlock(block *html.Node) BillCandidate {
	var c BillCandidate
	walkAnchors(block, func(_ *html.Node, href, text string) {
		lc := strings.ToLower(href + " " + text)
		switch {
		case strings.Contains(lc, "sites/default/files") && strings.Contains(lc, ".pdf"):
			// The first PDF link inside post-title is the Bill itself.
			if c.URL == "" {
				c.URL = href
				c.Title = strings.TrimSpace(text)
			} else if strings.Contains(lc, "digest") && c.BillDigestURL == "" {
				c.BillDigestURL = href
			}
		case strings.Contains(lc, "national_assembly_petition") || strings.Contains(lc, "senate_petition"):
			if c.PetitionURL == "" {
				c.PetitionURL = href
				// If we don't have a Bill title yet, fall back to the ?bill= query param.
				if c.Title == "" {
					if name := billNameFromPetitionURL(href); name != "" {
						c.Title = name
					}
				}
			}
		}
	})
	// Look for a per-Bill tracker link inside <span class="post-billtracker">.
	if tracker := findTrackerLink(block); tracker != "" {
		c.BillTrackerURL = tracker
	}
	return c
}

// findTrackerLink finds an <a> inside the <span class="post-billtracker"> of
// the given post-block, if any. Returns "" when the tracker slot is empty
// (which is the common case on the live site for recently published Bills).
func findTrackerLink(block *html.Node) string {
	var tracker string
	var visit func(*html.Node)
	visit = func(n *html.Node) {
		if tracker != "" || n == nil {
			return
		}
		if n.Type == html.ElementNode && n.Data == "span" && hasClass(n, "post-billtracker") {
			// Find the first <a> descendant.
			walkAnchors(n, func(_ *html.Node, href, _ string) {
				if tracker == "" && href != "" {
					tracker = href
				}
			})
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			visit(c)
		}
	}
	visit(block)
	return tracker
}

// billNameFromPetitionURL extracts the ?bill=… query parameter value from a
// /contact/national_assembly_petition?bill=… URL, URL-decoded and trimmed.
func billNameFromPetitionURL(u string) string {
	// Cheap URL parse — these URLs are well-formed.
	idx := strings.Index(u, "?bill=")
	if idx < 0 {
		return ""
	}
	raw := u[idx+len("?bill="):]
	// URL-decode percent-encoded chars.
	dec, err := url.QueryUnescape(raw)
	if err != nil {
		dec = raw
	}
	return strings.TrimSpace(dec)
}

// hasClass reports whether n has the given class in its class attribute.
func hasClass(n *html.Node, want string) bool {
	for _, a := range n.Attr {
		if a.Key != "class" {
			continue
		}
		for _, c := range strings.Fields(a.Val) {
			if c == want {
				return true
			}
		}
	}
	return false
}

// BillRow is a parsed row from the Kenya Parliament Bill Tracker. Field
// names mirror the column headers used on parliament.go.ke.
type BillRow struct {
	BillNo          string // e.g. "National Assembly Bill No. 23 of 2023"
	Title           string
	Sponsor         string // Mover / originating department
	Stage           string // raw text from the tracker (e.g. "Second Reading")
	House           string // "National Assembly" / "Senate"
	PublicationDate time.Time
	URL             string
}

// HansardEntry is a parsed Hansard report listing from a house's Hansard page.
type HansardEntry struct {
	Title   string
	Sitting string // e.g. "Morning Sitting"
	Date    time.Time
	URL     string
	House   string
}

// OrderPaperEntry is a parsed entry from a house's Order Paper listing.
type OrderPaperEntry struct {
	Date  time.Time
	URL   string
	Title string
	House string
}

// VotesProceedingsEntry is a parsed entry from a house's Votes and Proceedings listing.
type VotesProceedingsEntry struct {
	Date  time.Time
	URL   string
	House string
}

// CommitteeEntry is a parsed committee listing from a house's committees page.
type CommitteeEntry struct {
	Code  string
	Name  string
	House string
	Type  string // "departmental", "sessional", "select", "standing"
	URL   string
}

// ParseBillsHTML parses the HTML returned by parliament.go.ke's Bill Tracker
// page into a slice of BillRow records.
//
// The Bill Tracker is a table-driven page; each row has cells like
// "Bill No.", "Title", "Sponsor", "Stage", "Publication Date", and a link
// to the Bill's detail page. The parser is deliberately tolerant of layout
// drift — if a cell is missing we leave the corresponding field blank rather
// than failing the entire parse.
func ParseBillsHTML(r io.Reader) ([]BillRow, error) {
	doc, err := html.Parse(r)
	if err != nil {
		return nil, fmt.Errorf("parliament parser: %w", err)
	}
	var rows []BillRow
	walkTables(doc, func(tbl *html.Node) {
		rr := parseBillTable(tbl)
		rows = append(rows, rr...)
	})
	return rows, nil
}

// ParseHansardHTML parses a house's Hansard listing page.
func ParseHansardHTML(r io.Reader) ([]HansardEntry, error) {
	doc, err := html.Parse(r)
	if err != nil {
		return nil, fmt.Errorf("parliament parser: %w", err)
	}
	var out []HansardEntry
	walkAnchors(doc, func(a *html.Node, href, text string) {
		if !looksLikeHansard(href, text) {
			return
		}
		house := houseFromHref(href)
		out = append(out, HansardEntry{
			Title:   strings.TrimSpace(text),
			URL:     href,
			House:   house,
			Date:    parseDateInText(text),
			Sitting: sittingFromText(text),
		})
	})
	return out, nil
}

// ParseOrderPaperHTML parses a house's Order Paper listing page.
func ParseOrderPaperHTML(r io.Reader) ([]OrderPaperEntry, error) {
	doc, err := html.Parse(r)
	if err != nil {
		return nil, fmt.Errorf("parliament parser: %w", err)
	}
	var out []OrderPaperEntry
	walkAnchors(doc, func(a *html.Node, href, text string) {
		if !looksLikeOrderPaper(href, text) {
			return
		}
		out = append(out, OrderPaperEntry{
			Title: strings.TrimSpace(text),
			URL:   href,
			Date:  parseDateInText(text),
			House: houseFromHref(href),
		})
	})
	return out, nil
}

// ParseVotesProceedingsHTML parses a house's Votes and Proceedings listing page.
func ParseVotesProceedingsHTML(r io.Reader) ([]VotesProceedingsEntry, error) {
	doc, err := html.Parse(r)
	if err != nil {
		return nil, fmt.Errorf("parliament parser: %w", err)
	}
	var out []VotesProceedingsEntry
	walkAnchors(doc, func(a *html.Node, href, text string) {
		if !looksLikeVotesProceedings(href, text) {
			return
		}
		out = append(out, VotesProceedingsEntry{
			URL:   href,
			Date:  parseDateInText(text),
			House: houseFromHref(href),
		})
	})
	return out, nil
}

// ParseCommitteesHTML parses a house's committees listing page.
func ParseCommitteesHTML(r io.Reader) ([]CommitteeEntry, error) {
	doc, err := html.Parse(r)
	if err != nil {
		return nil, fmt.Errorf("parliament parser: %w", err)
	}
	var out []CommitteeEntry
	walkAnchors(doc, func(a *html.Node, href, text string) {
		if !looksLikeCommittee(href, text) {
			return
		}
		out = append(out, CommitteeEntry{
			Name:  strings.TrimSpace(text),
			URL:   href,
			House: houseFromHref(href),
			Type:  committeeTypeFromText(text),
			Code:  committeeCodeFromName(text),
		})
	})
	return out, nil
}

// --- internal walker helpers ---

// walkTables visits every <table> node and invokes fn on it.
func walkTables(n *html.Node, fn func(*html.Node)) {
	if n == nil {
		return
	}
	if n.Type == html.ElementNode && n.Data == "table" {
		fn(n)
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		walkTables(c, fn)
	}
}

// walkAnchors visits every <a> node and invokes fn with its href and text.
func walkAnchors(n *html.Node, fn func(a *html.Node, href, text string)) {
	if n == nil {
		return
	}
	if n.Type == html.ElementNode && n.Data == "a" {
		href := attrOf(n, "href")
		text := textOf(n)
		if href != "" {
			fn(n, href, text)
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		walkAnchors(c, fn)
	}
}

// parseBillTable converts a <table> node into a slice of BillRow records,
// using the first row as a header to map column positions.
func parseBillTable(tbl *html.Node) []BillRow {
	var headers []string
	var rows [][]string
	walkRows(tbl, func(tr *html.Node) {
		cells := cellsOf(tr)
		if len(cells) == 0 {
			return
		}
		// Heuristic: a header row has at least one cell that contains
		// the word "Bill" or "Stage" or "Title".
		if len(headers) == 0 && isHeaderRow(cells) {
			headers = normaliseHeaders(cells)
			return
		}
		rows = append(rows, cells)
	})
	if len(headers) == 0 || len(rows) == 0 {
		return nil
	}
	idx := indexMap(headers)
	out := make([]BillRow, 0, len(rows))
	for _, r := range rows {
		row := BillRow{}
		row.BillNo = cellAt(r, idx, "bill no", "billno", "no")
		row.Title = cellAt(r, idx, "title", "short title", "bill title")
		row.Sponsor = cellAt(r, idx, "sponsor", "mover", "originator", "department")
		row.Stage = cellAt(r, idx, "stage", "status")
		row.House = cellAt(r, idx, "house", "chamber")
		row.URL = "" // per-cell link extraction is not yet implemented; caller may post-process the table to populate URLs
		// Date parsing: try "publication date", "published", "date".
		if d := cellAt(r, idx, "publication date", "published", "date", "pub date"); d != "" {
			row.PublicationDate = parseDateInText(d)
		}
		if row.BillNo == "" && row.Title == "" {
			continue
		}
		out = append(out, row)
	}
	return out
}

// walkRows invokes fn on every <tr> descendant of n.
func walkRows(n *html.Node, fn func(*html.Node)) {
	if n == nil {
		return
	}
	if n.Type == html.ElementNode && n.Data == "tr" {
		fn(n)
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		walkRows(c, fn)
	}
}

// cellsOf returns the trimmed text of every <td>/<th> in a <tr>.
func cellsOf(tr *html.Node) []string {
	var cells []string
	for c := tr.FirstChild; c != nil; c = c.NextSibling {
		if c.Type != html.ElementNode {
			continue
		}
		if c.Data == "td" || c.Data == "th" {
			cells = append(cells, strings.TrimSpace(textOf(c)))
		}
	}
	return cells
}

func isHeaderRow(cells []string) bool {
	for _, c := range cells {
		lc := strings.ToLower(c)
		if strings.Contains(lc, "bill") || strings.Contains(lc, "stage") || strings.Contains(lc, "title") {
			return true
		}
	}
	return false
}

func normaliseHeaders(cells []string) []string {
	out := make([]string, len(cells))
	for i, c := range cells {
		out[i] = strings.ToLower(strings.TrimSpace(c))
	}
	return out
}

func indexMap(headers []string) map[string]int {
	m := make(map[string]int, len(headers))
	for i, h := range headers {
		m[h] = i
	}
	return m
}

func cellAt(row []string, idx map[string]int, names ...string) string {
	for _, n := range names {
		if i, ok := idx[n]; ok && i < len(row) {
			return row[i]
		}
	}
	return ""
}

func attrOf(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}

// textOf returns the trimmed concatenation of all text in n's subtree.
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

// houseFromHref classifies a URL into "National Assembly" or "Senate" based
// on its path. Falls back to "" if it cannot tell.
func houseFromHref(href string) string {
	lc := strings.ToLower(href)
	switch {
	case strings.Contains(lc, "national-assembly") || strings.Contains(lc, "nationalassembly"):
		return "National Assembly"
	case strings.Contains(lc, "senate"):
		return "Senate"
	}
	return ""
}

// looksLikeHansard returns true if href or text suggests a Hansard report.
func looksLikeHansard(href, text string) bool {
	lc := strings.ToLower(href + " " + text)
	return strings.Contains(lc, "hansard") || strings.Contains(lc, "official-report")
}

// looksLikeOrderPaper returns true if href or text suggests an Order Paper.
func looksLikeOrderPaper(href, text string) bool {
	lc := strings.ToLower(href + " " + text)
	return strings.Contains(lc, "order-paper") || strings.Contains(lc, "order_paper") || strings.Contains(lc, "order paper")
}

// looksLikeVotesProceedings returns true if href or text suggests a Votes
// and Proceedings document.
func looksLikeVotesProceedings(href, text string) bool {
	lc := strings.ToLower(href + " " + text)
	return strings.Contains(lc, "votes-and-proceedings") ||
		strings.Contains(lc, "votes_and_proceedings") ||
		strings.Contains(lc, "votes and proceedings")
}

// looksLikeCommittee returns true if href or text suggests a committee page.
func looksLikeCommittee(href, text string) bool {
	lc := strings.ToLower(href + " " + text)
	return strings.Contains(lc, "committee") || strings.Contains(lc, "departmental")
}

// committeeTypeFromText attempts to classify a committee listing.
func committeeTypeFromText(text string) string {
	lc := strings.ToLower(text)
	switch {
	case strings.Contains(lc, "departmental") || strings.Contains(lc, "standing"):
		return "departmental"
	case strings.Contains(lc, "sessional"):
		return "sessional"
	case strings.Contains(lc, "select"):
		return "select"
	}
	return ""
}

// committeeCodeFromName derives a short, stable committee code from a name.
// The platform never looks at this code; it's just used internally for
// stable cross-references between crawler runs.
var nonAlnum = regexp.MustCompile(`[^a-z0-9]+`)

func committeeCodeFromName(name string) string {
	lc := strings.ToLower(strings.TrimSpace(name))
	if lc == "" {
		return ""
	}
	lc = nonAlnum.ReplaceAllString(lc, "_")
	lc = strings.Trim(lc, "_")
	return lc
}

// sittingFromText attempts to extract "Morning Sitting" / "Afternoon Sitting"
// markers from a Hansard title.
func sittingFromText(text string) string {
	lc := strings.ToLower(text)
	switch {
	case strings.Contains(lc, "morning"):
		return "Morning Sitting"
	case strings.Contains(lc, "afternoon"):
		return "Afternoon Sitting"
	case strings.Contains(lc, "whole day") || strings.Contains(lc, "whole-day"):
		return "Whole Day Sitting"
	}
	return ""
}

// parseDateInText tries to find a date in a free-text string using several
// common Kenyan parliamentary layouts.
func parseDateInText(s string) time.Time {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}
	}
	layouts := []string{
		"2006-01-02",
		"02/01/2006",
		"02 January 2006",
		"02 Jan 2006",
		"January 02, 2006",
		"Jan 02, 2006",
		time.RFC3339,
	}
	for _, l := range layouts {
		if t, err := time.Parse(l, s); err == nil {
			return t.UTC()
		}
	}
	// Try to extract the first date-like substring.
	for _, l := range []string{"02 January 2006", "January 02, 2006", "02/01/2006", "2006-01-02"} {
		re := dateRegexForLayout(l)
		if m := re.FindString(s); m != "" {
			if t, err := time.Parse(l, m); err == nil {
				return t.UTC()
			}
		}
	}
	return time.Time{}
}

// dateRegexForLayout returns a regex that matches a date in the given layout.
// Used as a tolerant fallback when the exact layout isn't directly parseable.
func dateRegexForLayout(layout string) *regexp.Regexp {
	switch layout {
	case "02 January 2006":
		return regexp.MustCompile(`\b\d{1,2}\s+(January|February|March|April|May|June|July|August|September|October|November|December)\s+\d{4}\b`)
	case "January 02, 2006":
		return regexp.MustCompile(`\b(January|February|March|April|May|June|July|August|September|October|November|December)\s+\d{1,2},\s+\d{4}\b`)
	case "02/01/2006":
		return regexp.MustCompile(`\b\d{1,2}/\d{1,2}/\d{4}\b`)
	case "2006-01-02":
		return regexp.MustCompile(`\b\d{4}-\d{2}-\d{2}\b`)
	}
	return regexp.MustCompile(`.`)
}
