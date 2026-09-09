package parliament

import (
        "fmt"
        "io"
        "regexp"
        "strings"
        "time"

        "golang.org/x/net/html"
)

// BillRow is a parsed row from the Kenya Parliament Bill Tracker. Field
// names mirror the column headers used on parliament.go.ke.
type BillRow struct {
        BillNo         string // e.g. "National Assembly Bill No. 23 of 2023"
        Title          string
        Sponsor        string // Mover / originating department
        Stage          string // raw text from the tracker (e.g. "Second Reading")
        House          string // "National Assembly" / "Senate"
        PublicationDate time.Time
        URL            string
}

// HansardEntry is a parsed Hansard report listing from a house's Hansard page.
type HansardEntry struct {
        Title     string
        Sitting   string // e.g. "Morning Sitting"
        Date      time.Time
        URL       string
        House     string
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
        Code     string
        Name     string
        House    string
        Type     string // "departmental", "sessional", "select", "standing"
        URL      string
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
                        Name: strings.TrimSpace(text),
                        URL:  href,
                        House: houseFromHref(href),
                        Type: committeeTypeFromText(text),
                        Code: committeeCodeFromName(text),
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
