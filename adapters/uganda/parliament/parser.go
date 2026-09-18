package parliament

import (
        "fmt"
        "regexp"
        "strings"
        "time"

        "github.com/Roy-Wanyoike/civic-intelligence/packages/contracts"
        "golang.org/x/net/html"
)

// stageMapping maps the lowercased substrings the Parliament of Uganda uses
// on its Bills listing pages to the canonical stage codes from
// adapters/uganda/internal/uganda_data.go (UgandaBillStages).
//
// Order matters: more specific patterns MUST come first (e.g.,
// "committee stage" before the bare "committee"; "presidential assent"
// before the bare "assent").
var stageMapping = []struct {
        needle string
        code   string
}{
        {"first reading", "FIRST_READING"},
        {"1st reading", "FIRST_READING"},

        {"second reading", "SECOND_READING"},
        {"2nd reading", "SECOND_READING"},

        {"committee stage", "COMMITTEE_STAGE"},
        {"committee of the whole house", "COMMITTEE_STAGE"},
        {"committee", "COMMITTEE_STAGE"},

        {"report stage", "REPORT_STAGE"},

        {"third reading", "THIRD_READING"},
        {"3rd reading", "THIRD_READING"},

        {"presidential assent", "PRESIDENTIAL_ASSENT"},
        {"assented to", "PRESIDENTIAL_ASSENT"},
        {"assented", "PRESIDENTIAL_ASSENT"},
        {"assent", "PRESIDENTIAL_ASSENT"},

        {"commencement", "COMMENCEMENT"},
        {"in force", "COMMENCEMENT"},
        {"came into force", "COMMENCEMENT"},
        {"gazetted", "COMMENCEMENT"},

        {"rejected", "REJECTED"},
        {"negatived", "REJECTED"},
}

// MapStageText maps Parliament's raw stage text (e.g., "Second Reading",
// "Committee Stage", "Presidential Assent") to the canonical Uganda stage
// codes (FIRST_READING, SECOND_READING, …). The match is case-insensitive and
// substring-based, so "Second Reading — 12 March 2024" → SECOND_READING.
// Returns "" if the input does not match a known stage.
func MapStageText(input string) string {
        if input == "" {
                return ""
        }
        lc := strings.ToLower(input)
        for _, m := range stageMapping {
                if strings.Contains(lc, m.needle) {
                        return m.code
                }
        }
        return ""
}

// billNumberRe matches Uganda's official Bill number format. Example:
//
//      "The Bill No. 12 of 2024" → matches "Bill No. 12 of 2024".
var billNumberRe = regexp.MustCompile(`(?i)bill\s*no\.?\s*\d+\s*(?:of\s*)?\d{4}`)

// dateLayouts are the date formats the parser tries, in order. The Uganda
// Parliament listing page typically uses ISO "2006-01-02", but the parser
// tolerates common variants.
var dateLayouts = []string{
        "2006-01-02",
        "02/01/2006",
        "01/02/2006",
        "2 January 2006",
        "January 2, 2006",
        "2 Jan 2006",
}

// ParseBillsListing parses the HTML of the Parliament of Uganda Bills listing
// page (https://www.parliament.go.ug/business/bills) into a slice of
// BillCandidate records.
//
// The page uses a <table class="bills-listing"> with one <tr> per Bill. Each
// row has up to five <td> cells: Bill No., Title (as an <a> link), Sponsor,
// Stage, Publication Date. The parser is deliberately tolerant: missing cells
// produce empty fields, not errors, and any <tr> with at least one <td> is
// considered (rows that don't look like Bill rows are filtered out by the
// content heuristics in parseBillRow).
func ParseBillsListing(pageHTML string) []BillCandidate {
        doc, err := html.Parse(strings.NewReader(pageHTML))
        if err != nil {
                return nil
        }
        var out []BillCandidate
        walkBillRows(doc, func(row *html.Node) {
                c, ok := parseBillRow(row)
                if !ok {
                        return
                }
                out = append(out, c)
        })
        // De-duplicate by URL (a Bill may be linked more than once).
        seen := make(map[string]bool, len(out))
        deduped := out[:0]
        for _, c := range out {
                key := c.URL
                if key == "" {
                        key = c.Title
                }
                if key == "" {
                        key = c.Number
                }
                if seen[key] {
                        continue
                }
                seen[key] = true
                deduped = append(deduped, c)
        }
        return deduped
}

// ParseBillDetail extracts Bill metadata from a Parliament of Uganda HTML
// page. Returns one ExtractedRecord per Bill found (a detail page yields 1;
// a listing page yields N).
//
// The parser never returns a parse error for a partial extraction — it returns
// whatever records could be extracted with lowered Confidence. An error is
// returned only when no Bills can be extracted at all (e.g., the page is empty
// or contains no Bill rows).
func ParseBillDetail(pageHTML, sourceURL string) ([]contracts.ExtractedRecord, error) {
        cands := ParseBillsListing(pageHTML)
        if len(cands) == 0 {
                return nil, fmt.Errorf("uganda.ParseBillDetail: no Bills found in %s", sourceURL)
        }
        now := time.Now().UTC()
        out := make([]contracts.ExtractedRecord, 0, len(cands))
        for _, c := range cands {
                rec := contracts.ExtractedRecord{
                        Kind:          "bill",
                        Identifier:    c.Number,
                        Title:         c.Title,
                        Sponsor:       c.Sponsor,
                        Stage:         c.Stage,
                        House:         parliamentHouse,
                        PublishedAt:   c.PublicationDate,
                        SourceURL:     sourceURL,
                        RetrievedAt:   now,
                        ExtractedAt:   now,
                        ExtractorName: "uganda.parliament.HTMLParser",
                        Confidence:    0.85,
                }
                // Lower confidence when essential fields are missing.
                switch {
                case c.Title == "":
                        rec.Confidence = 0.3
                case c.Number == "" || c.Stage == "":
                        rec.Confidence = 0.6
                }
                extra := map[string]interface{}{}
                if c.StageRaw != "" {
                        extra["stage_raw"] = c.StageRaw
                }
                if c.URL != "" && c.URL != sourceURL {
                        extra["bill_url"] = c.URL
                }
                if len(extra) > 0 {
                        rec.Extra = extra
                }
                out = append(out, rec)
        }
        return out, nil
}

// walkBillRows traverses the HTML tree, invoking fn for every <tr> element
// that contains at least one <td> child (i.e., a data row, not a header row).
func walkBillRows(n *html.Node, fn func(row *html.Node)) {
        if n.Type == html.ElementNode && n.Data == "tr" && hasChildElement(n, "td") {
                fn(n)
        }
        for c := n.FirstChild; c != nil; c = c.NextSibling {
                walkBillRows(c, fn)
        }
}

// hasChildElement reports whether n has a direct child element with the given tag.
func hasChildElement(n *html.Node, tag string) bool {
        for c := n.FirstChild; c != nil; c = c.NextSibling {
                if c.Type == html.ElementNode && c.Data == tag {
                        return true
                }
        }
        return false
}

// parseBillRow extracts a BillCandidate from a single <tr> row. The second
// return value is false when the row does not look like a Bill row (e.g., a
// navigation table row).
//
// Column identification is content-based (not position-based) so the parser
// tolerates column reordering:
//   - Bill number: matches billNumberRe.
//   - Stage: matches MapStageText.
//   - Publication date: parses with one of dateLayouts.
//   - Sponsor: a cell containing "Hon.", "Minister", or "Member" that is not
//     the number, stage, or title.
//   - Title: the text of the first <a> link in the row; falls back to the
//     first non-empty cell if no link is present.
func parseBillRow(row *html.Node) (BillCandidate, bool) {
        var c BillCandidate
        var cells []string
        var linkURL, linkText string
        for td := row.FirstChild; td != nil; td = td.NextSibling {
                if td.Type != html.ElementNode || td.Data != "td" {
                        continue
                }
                text := strings.TrimSpace(textOf(td))
                cells = append(cells, text)
                // Capture the first <a href> in this cell as the Bill link.
                if linkURL == "" {
                        if href, txt := firstAnchor(td); href != "" {
                                linkURL = href
                                linkText = txt
                        }
                }
        }
        if len(cells) < 2 {
                // A Bill row has at least 2 cells (number + title). Filter out
                // single-cell rows from nav tables, etc.
                return c, false
        }
        // Identify columns by content.
        for _, cell := range cells {
                if c.Number == "" && billNumberRe.MatchString(cell) {
                        c.Number = strings.TrimSpace(cell)
                        continue
                }
                if c.Stage == "" {
                        if code := MapStageText(cell); code != "" {
                                c.Stage = code
                                c.StageRaw = strings.TrimSpace(cell)
                                continue
                        }
                }
                if c.PublicationDate.IsZero() {
                        c.PublicationDate = tryParseDate(cell)
                }
        }
        c.Title = strings.TrimSpace(linkText)
        c.URL = strings.TrimSpace(linkURL)
        // Sponsor heuristic: a cell containing "Hon." or "Minister" or "Member"
        // that is not the number, stage, or title.
        for _, cell := range cells {
                lc := strings.ToLower(cell)
                if strings.Contains(lc, "hon.") ||
                        strings.Contains(lc, "minister") ||
                        strings.Contains(lc, "attorney general") ||
                        strings.Contains(lc, "member of parliament") {
                        if cell != c.Number && cell != c.StageRaw && cell != c.Title && c.Sponsor == "" {
                                c.Sponsor = strings.TrimSpace(cell)
                                break
                        }
                }
        }
        // Only keep rows that look like Bill rows: must have a bill number OR a
        // link OR a title that contains "Bill".
        if c.Number == "" && c.URL == "" {
                if !strings.Contains(strings.ToLower(c.Title), "bill") {
                        return c, false
                }
        }
        return c, true
}

// tryParseDate attempts to parse cell with each layout in dateLayouts. Returns
// the zero time if none match.
func tryParseDate(cell string) time.Time {
        cell = strings.TrimSpace(cell)
        if cell == "" {
                return time.Time{}
        }
        for _, layout := range dateLayouts {
                if t, err := time.Parse(layout, cell); err == nil {
                        return t
                }
        }
        return time.Time{}
}

// firstAnchor returns the href and trimmed text of the first <a> element
// under n (depth-first search). Returns ("", "") if no anchor is found.
func firstAnchor(n *html.Node) (href, text string) {
        var find func(*html.Node)
        find = func(node *html.Node) {
                if href != "" {
                        return
                }
                if node.Type == html.ElementNode && node.Data == "a" {
                        for _, attr := range node.Attr {
                                if attr.Key == "href" {
                                        href = attr.Val
                                        break
                                }
                        }
                        text = strings.TrimSpace(textOf(node))
                        return
                }
                for c := node.FirstChild; c != nil; c = c.NextSibling {
                        find(c)
                        if href != "" {
                                return
                        }
                }
        }
        find(n)
        return href, text
}

// textOf returns the concatenated text of all descendant text nodes of n.
// Used to extract the visible text of a cell.
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
